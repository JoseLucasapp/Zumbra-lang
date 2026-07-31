# IR tipada e pipeline canônico — Z6

O Z6 introduz um pipeline único para arquivos Zumbra sem adicionar sintaxe nova ao código do usuário.

## Objetivo

Antes do Z6, cada caminho da implementação podia iniciar seu próprio fluxo de parsing ou análise. Isso aumentava o risco de o evaluator, a VM e o transpiler aceitarem programas diferentes.

O fluxo canônico passa a ser:

```text
source
  ↓
lexer
  ↓
parser
  ↓
semantic resolver
  ↓
type checker + Type Analysis
  ↓
HIR tipada
  ↓
MIR estruturada
  ↓
passes de otimização
  ↓
compiler / evaluator / transpiler
```

A sintaxe do Zumbra continua simples. HIR e MIR são detalhes internos, mas podem ser inspecionadas por ferramentas oficiais.

## Type Analysis

O type checker agora produz um resultado reutilizável contendo:

- tipo inferido de cada expressão da AST;
- símbolos globais;
- aliases;
- structs;
- enums;
- assinaturas inferidas de métodos e funções.

Isso evita recalcular tipos em cada backend.

## HIR

HIR significa **High-level Intermediate Representation**.

Ela mantém construções próximas ao código original:

- variáveis e constantes;
- structs, campos e métodos;
- enums;
- chamadas;
- `if`;
- `match`;
- loops;
- arrays, dicionários e buffers;
- atribuições por índice e por campo.

Cada expressão recebe um tipo resolvido.

Exemplo de dump:

```text
%17 var name="folded" : int
  %18 binary op="+" : int
    %19 integer value="2" : int
    %20 binary op="*" : int
      %21 integer value="3" : int
      %22 integer value="4" : int
```

## MIR

MIR significa **Middle-level Intermediate Representation**.

Ela transforma expressões em valores virtuais explícitos:

```text
%7 = const value="2" : int
%8 = const value="3" : int
%9 = const value="4" : int
%10 = binary op="*" (%8, %9) : int
%11 = binary op="+" (%7, %10) : int
declare "folded" (%11) : int
```

Controle de fluxo usa regiões estruturadas. Isso mantém a implementação legível e facilita a conversão futura para blocos básicos ou código nativo.

## Otimizações iniciais

### Constant folding

Operações com valores conhecidos em compilação são calculadas previamente:

```zumbra
var folded << 2 + 3 * 4;
```

MIR otimizada:

```text
%11 = const value="14" : int
declare "folded" (%11) : int
```

O folding inicial cobre:

- inteiros;
- floats;
- booleanos;
- concatenação de strings;
- comparações;
- operações bitwise;
- shifts seguros;
- parte das operações com inteiros fixos.

Quando uma operação não pode ser dobrada com segurança, ela permanece na MIR e é executada normalmente.

### Dead-value elimination

Constantes, loads e operações puras sem uso são removidos.

Chamadas e operações com efeitos colaterais são preservadas.

### Unreachable-code elimination

Instruções depois de:

- `return`;
- `break`;
- `continue`;

são removidas da região correspondente.

### Verificação da MIR

Antes e depois da otimização, o verifier confirma:

- cada valor é definido antes de ser usado;
- não há resultados duplicados;
- resultados de regiões existem;
- funções e regiões são estruturalmente válidas.

## Comandos

### Validar sem executar

```bash
zumbra check app.zum
```

Ou durante o desenvolvimento do próprio Zumbra:

```bash
go run . check app.zum
```

Saída esperada:

```text
OK: app.zum
```

### Mostrar HIR

```bash
zumbra ir app.zum hir
```

### Mostrar MIR sem otimização

```bash
zumbra ir app.zum mir
```

### Mostrar MIR otimizada

```bash
zumbra ir app.zum optimized
```

`optimized` também é o modo padrão:

```bash
zumbra ir app.zum
```

## Integração com os backends

O Z6 adiciona adaptadores oficiais:

- `compiler.CompilePipeline`;
- `evaluator.EvalPipeline`;
- `transpiler.ZumbraTranspilerPipeline`.

Todos recebem o mesmo `pipeline.Result`, que já contém AST validada, tipos, HIR e MIR.

Nesta etapa, os backends existentes ainda reutilizam sua implementação madura baseada em AST depois de validar o pipeline. O Z7 moverá a geração nativa e os novos backends diretamente para MIR. Essa separação evita reescrever a VM e o transpiler no mesmo corte em que a IR é introduzida.

## Builtins sem dependências pesadas no frontend

Os nomes de builtins usados pelo semantic resolver foram movidos para `builtinspec`.

Isso permite que lexer, parser, semantic, type checker, HIR, MIR e pipeline sejam compilados e testados sem carregar drivers de banco, Redis, Twilio ou outras integrações.

Builtins podem ser sombreados por variáveis do usuário:

```zumbra
var sum << fct(a, b) {
    a + b;
};
```

Nomes de builtins são conveniências predeclaradas, não palavras reservadas.

## Simplicidade preservada

O usuário não precisa declarar HIR, MIR, blocos básicos ou registradores virtuais.

O código continua sendo:

```zumbra
var result << 2 + 3 * 4;
show(result);
```

A complexidade fica dentro do compilador, enquanto a superfície da linguagem permanece pequena.

## Limite desta etapa

O Z6 cria e valida a infraestrutura de IR e torna o pipeline canônico.

O Z7 será responsável por:

- backend nativo baseado em MIR;
- geração C inicial;
- otimizações orientadas a release;
- executáveis que linkam apenas módulos usados;
- benchmarks VM versus nativo.

## Benchmark do pipeline

Para medir parsing, análise, lowering e otimização:

```bash
go test ./benchmark -bench TypedPipeline -benchmem
```

Existem benchmarks com e sem otimização para acompanhar custo e regressões ao longo do rebuild.
