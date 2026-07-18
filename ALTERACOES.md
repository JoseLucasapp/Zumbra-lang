# Alterações — inteiros de tamanho fixo v1

## Objetivo

Adicionar inteiros de largura explícita para programação de sistemas, games e emulação, preservando o principal pilar do Zumbra: simplicidade.

A linguagem continua usando `int` como padrão. O tipo fixo aparece somente quando o programador precisa dele, usando um sufixo ou uma conversão:

```zumbra
var opcode << 0xA9u8;
var address << u16(0x8000);
```

## O que foi implementado

- tipos `u8`, `u16`, `u32`, `u64`;
- tipos `i8`, `i16`, `i32`, `i64`;
- literais tipados decimais, hexadecimais, binários e octais;
- suporte completo ao intervalo de `u64` em literais;
- conversões explícitas e seguras por meio de `u8(...)`, `i16(...)` e equivalentes;
- representação própria de inteiros fixos no runtime e na VM;
- aritmética normal com wrap na largura do tipo, como em registradores de hardware;
- `wrapAdd`, `wrapSub`, `wrapMul`;
- `checkedAdd`, `checkedSub`, `checkedMul`;
- `satAdd`, `satSub`, `satMul`;
- operações bit a bit preservando o tipo;
- shifts limitados à largura do operando esquerdo;
- comparação e validação entre tipos;
- uso de inteiros fixos como índices de arrays;
- conversão por `toInt`, `toFloat`, `toString` e `toBool`;
- suporte no transpiler experimental e no runtime Go;
- testes de conformidade entre evaluator e VM;
- documentação e exemplo executável.

## Semântica definida

### Aritmética normal

Inteiros fixos fazem wrap por padrão:

```zumbra
var value << 255u8 + 1; // 0u8
```

Isso atende diretamente ao comportamento necessário para registradores e memória de consoles.

### Conversão

Conversões explícitas verificam a faixa e retornam erro quando o valor não cabe:

```zumbra
var valid << u8(255);
var invalid << u8(256); // erro
```

### Compatibilidade

- `int` permanece inalterado;
- a atribuição continua usando `<<`;
- nenhum código existente precisa declarar tipos;
- não foram adicionadas palavras-chave;
- não foram adicionados novos opcodes: os opcodes aritméticos existentes agora reconhecem valores fixos;
- evaluator e VM compartilham a mesma implementação de semântica fixa no pacote `numeric`.

## Arquivos alterados

### Implementação

- `ast/integer_literal.go`
- `compiler/compiler.go`
- `evaluator/builtins.go`
- `evaluator/evaluator.go`
- `lexer/lexer.go`
- `object/builtins/builtins.go`
- `object/builtins/parser_types_builtin.go`
- `parser/parser.go`
- `runtime/runtime.go`
- `transpiler/transpiler.go`
- `types/builtins.go`
- `types/checker.go`
- `types/types.go`
- `vm/vm.go`

### Documentação existente atualizada

- `README.MD`
- `ROADMAP.md`
- `docs/syntax.MD`
- `ALTERACOES.md`

## Arquivos criados

### Implementação

- `numeric/fixed_integer.go`
- `object/fixed_integer.go`
- `object/builtins/fixed_integer_builtins.go`

### Testes

- `ast/fixed_integer_test.go`
- `compiler/fixed_integers_test.go`
- `conformance/fixed_integers_test.go`
- `evaluator/fixed_integers_test.go`
- `lexer/fixed_integers_test.go`
- `numeric/fixed_integer_test.go`
- `object/fixed_integer_test.go`
- `object/builtins/fixed_integer_builtins_test.go`
- `parser/fixed_integers_test.go`
- `runtime/fixed_integers_test.go`
- `transpiler/fixed_integers_test.go`
- `types/fixed_integers_test.go`
- `vm/fixed_integers_test.go`

### Documentação, exemplo e automação

- `docs/pt-BR/inteiros-de-tamanho-fixo.md`
- `code_examples/core/fixed_integers.zum`
- `scripts/test-fixed-integers.sh`

## Onde colocar

O ZIP completo já contém todos os arquivos nas posições corretas.

Para aplicar somente o patch em outra cópia do rebuild, copie o conteúdo do ZIP de patch para a raiz do repositório, preservando os caminhos relativos.

Branch recomendada:

```bash
git checkout -b feature/fixed-integers-v1
```

## Como testar

### Suíte específica

```bash
./scripts/test-fixed-integers.sh
```

### Suíte completa

```bash
go test ./...
```

### Exemplo manual

```bash
go run . run code_examples/core/fixed_integers.zum
```

Saída esperada:

```text
169
32768
0
0
255
169
-128
```

## Validação realizada durante a implementação

Passaram diretamente no ambiente de geração:

```text
ast
object
numeric
lexer
parser
types
runtime
transpiler
```

Também passaram em um ambiente isolado das integrações externas:

```text
compiler
evaluator
vm
conformance
```

O exemplo executável foi validado com o pipeline compiler + VM e produziu a saída documentada.

A suíte completa `go test ./...` deve ser executada no ambiente do projeto, pois o ambiente de geração não tem acesso à internet para baixar as dependências externas de banco, Redis, JWT e Twilio.

## Limitações conscientes

- ainda não há declaração de variável no formato `var value: u8`; o tipo é definido pelo literal ou conversão para manter a sintaxe simples;
- para o menor valor assinado, como `-128i8`, use `i8(-128)`;
- arrays ainda armazenam objetos genéricos;
- mutação por índice ainda não foi implementada;
- `ByteArray`, slices e arquivos binários pertencem à próxima etapa;
- o transpiler continua experimental, embora agora reconheça os inteiros fixos e operadores de sistemas.

## Próxima etapa

- arrays tipados;
- mutação por índice;
- `ByteArray` compacto;
- slices;
- leitura e escrita de arquivos binários.
