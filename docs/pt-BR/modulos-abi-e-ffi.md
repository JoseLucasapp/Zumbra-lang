# Módulos, ABI nativa e FFI C

O Z8 introduz isolamento real entre arquivos Zumbra e uma ponte tipada para bibliotecas C. O objetivo é manter o código comum simples e seguro, deixando operações dependentes da máquina explícitas.

## Módulos com alias

Um módulo é um arquivo `.zum`. Importe-o com um alias:

```zumbra
import "../modules/math.zum" as math;

show(math.add(20, 22));
```

O caminho é resolvido a partir do arquivo que contém o `import`, não do diretório em que o comando foi executado.

Cada arquivo possui namespace próprio. Declarações são privadas por padrão:

```zumbra
const INTERNAL_LIMIT << 64;

pub const DEFAULT_LIMIT << 32;

pub fct add(left, right) {
    left + right;
}
```

Somente declarações marcadas com `pub` podem ser acessadas pelo alias:

```zumbra
show(math.DEFAULT_LIMIT); // permitido
show(math.INTERNAL_LIMIT); // erro: membro privado
```

`pub` pode ser usado em:

- funções nomeadas;
- variáveis e constantes no nível do módulo;
- aliases de tipo;
- structs;
- enums;
- blocos `extern`.

No Z8, campos e métodos pertencentes a uma struct pública acompanham o tipo. Visibilidade individual de membros será avaliada quando o sistema de pacotes evoluir.

## Funções nomeadas

O Z8 aceita uma forma direta para funções de módulo:

```zumbra
pub fct multiply(left, right) {
    left * right;
}
```

Internamente ela continua usando o mesmo modelo de função já existente. A forma anterior continua válida:

```zumbra
pub var multiply << fct(left, right) {
    left * right;
};
```

## Grafo de módulos

Inspecione exatamente quais arquivos entram no pipeline:

```bash
zumbra modules app.zum
```

A saída informa:

- arquivo de entrada;
- dependências carregadas;
- aliases;
- símbolos públicos;
- arquivos nativos que serão linkados.

Cada dependência é carregada apenas uma vez, mesmo quando alcançada por mais de um caminho. Importações cíclicas são rejeitadas.

O pipeline carrega somente módulos alcançáveis a partir do arquivo de entrada. O Z8 ainda não elimina declarações individuais não utilizadas dentro de um módulo carregado.

## Compatibilidade com imports antigos

A forma antiga sem alias continua temporariamente aceita:

```zumbra
import "math.zum";
```

Ela mistura as declarações do outro arquivo no escopo atual e gera um warning. Para isolamento, visibilidade e diagnósticos melhores, use sempre `as`.

## Declarações C

Declare funções de uma ABI C com `extern`:

```zumbra
extern "C" from "../native/math.c" {
    fct add(left: i32, right: i32) -> i32;
}
```

O caminho em `from` também é relativo ao módulo que contém a declaração. Ele pode apontar para:

- fonte `.c`;
- objeto `.o`;
- biblioteca estática `.a`;
- biblioteca compartilhada `.so`, `.dylib` ou `.dll`, conforme o toolchain da plataforma.

Para exportar as funções por um módulo de bindings:

```zumbra
pub extern "C" from "../native/math.c" {
    fct add(left: i32, right: i32) -> i32;
}
```

Outro arquivo pode então importar o binding:

```zumbra
import "math_binding.zum" as native;

unsafe {
    show(native.add(20i32, 22i32));
}
```

## Nome Zumbra diferente do símbolo C

Use `as` depois do retorno:

```zumbra
extern "C" from "native.c" {
    fct add(left: i32, right: i32) -> i32 as "project_add_i32";
}
```

O programa chama `add`, enquanto o linker procura `project_add_i32`.

## Tipos compatíveis

| Tipo na assinatura Zumbra | Tipo C gerado |
|---|---|
| `void` | `void` |
| `bool` | `bool` |
| `int` | `int` |
| `i8` | `int8_t` |
| `i16` | `int16_t` |
| `i32` | `int32_t` |
| `i64` | `int64_t` |
| `u8` | `uint8_t` |
| `u16` | `uint16_t` |
| `u32` | `uint32_t` |
| `u64` | `uint64_t` |
| `usize` | `size_t` |
| `float` | `double` |
| `string` ou `cstring` | `const char *` |
| `ptr` | `void *` |
| `callback(...) -> ...` | ponteiro de função C |

Use os tipos de largura fixa quando a ABI exigir tamanho exato. O tamanho de `int` segue o compilador C e não deve ser usado para formatos binários ou protocolos que exigem 32 bits garantidos.

## Código `unsafe`

Chamadas externas exigem um bloco explícito:

```zumbra
unsafe {
    var result << native.add(20i32, 22i32);
    show(result);
}
```

O bloco não desativa o type checker. Ele apenas declara que o programador aceita os riscos da fronteira nativa:

- assinatura C incorreta;
- ponteiro inválido;
- biblioteca incompatível;
- vida útil de strings ou callbacks controlada pela biblioteca;
- comportamento dependente da plataforma.

Uma chamada externa fora de `unsafe` falha durante a análise semântica.

## Ponteiros opacos

`ptr` representa `void *`. No Z8 ele pode ser recebido de C, comparado, armazenado e devolvido para C, mas não pode ser desreferenciado diretamente em Zumbra.

Isso permite usar handles de SDL, bancos e APIs do sistema sem introduzir ainda aritmética de ponteiros. Leitura, escrita, alocação manual e `Pointer<T>` pertencem ao marco posterior de systems programming.

## Callbacks síncronos

Callbacks C básicos são declarados na própria assinatura:

```zumbra
extern "C" from "native.c" {
    fct apply(
        value: i32,
        transform: callback(i32) -> i32
    ) -> i32;
}

var double << fct(value) {
    value * 2i32;
};

unsafe {
    show(apply(21i32, double));
}
```

O Z8 suporta callbacks que:

- são chamados antes de a função C retornar;
- não são armazenados pela biblioteca;
- não são chamados por outra thread;
- não reentram simultaneamente no mesmo slot de callback.

Callbacks assíncronos, persistentes ou multithread exigirão handles de vida útil e sincronização no runtime.

## Linking adicional

Quando a declaração não usa `from`, ou quando a biblioteca depende de outros arquivos:

```bash
zumbra build --link native/math.o app.zum
zumbra build --link native/libmath.a app.zum
zumbra build --include native/include app.zum
zumbra build --library-dir native/lib -l project_math app.zum
```

Essas opções podem ser combinadas. Arquivos declarados em `from` são adicionados automaticamente.

## Gerador conservador de bindings

Para headers C simples:

```bash
zumbra bind-c \
  --pub \
  --link ../native/math.c \
  -o bindings/math.zum \
  native/math.h
```

Sem `-o`, o binding é escrito no terminal.

O gerador reconhece protótipos portáveis com escalares e ponteiros. Ele não tenta adivinhar ABI para:

- structs e unions;
- macros;
- funções variádicas;
- arrays em parâmetros;
- ponteiros de função;
- typedefs dependentes da plataforma;
- atributos específicos do compilador.

Declarações não suportadas produzem warnings e devem receber bindings escritos manualmente. Essa abordagem é intencional: um binding incompleto é melhor que código silenciosamente incompatível.

## ABI do runtime nativo

O runtime C define `ZUMBRA_NATIVE_ABI_VERSION` e o código gerado contém uma verificação estática. O Z8 inicia essa ABI na versão `1`.

Essa versão protege a comunicação entre o C gerado e `zumbra_runtime`. Ela não altera a ABI das bibliotecas externas, que continua sendo a ABI C da plataforma.

## VM, evaluator e transpiler Go

Módulos Zumbra sem FFI podem ser executados pela VM:

```bash
zumbra run app.zum
```

Declarações `extern "C"` são aceitas pelo frontend, mas a execução externa exige:

```bash
zumbra build --release app.zum
```

O compiler de bytecode, evaluator e transpiler Go retornam um diagnóstico explícito em vez de simular uma chamada nativa.

## Exemplo completo

Módulos:

```bash
zumbra run code_examples/core/modules.zum
zumbra modules code_examples/core/modules.zum
```

FFI:

```bash
zumbra build --release code_examples/core/ffi.zum
./build/ffi
```

Validação completa do marco:

```bash
./scripts/test-modules-ffi.sh
```

## Limites atuais

O Z8 não inclui ainda:

- registry ou gerenciador de pacotes;
- imports por nome de pacote;
- versionamento de dependências;
- C structs/unions por valor;
- ponteiros tipados e dereference;
- ownership de recursos C;
- callbacks assíncronos;
- carregamento dinâmico em runtime com `dlopen`;
- tree shaking por símbolo;
- ABI estável entre bibliotecas escritas em Zumbra.

Esses limites são explícitos para que a linguagem permaneça pequena e previsível enquanto a base nativa amadurece.
