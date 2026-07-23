# Compilação nativa com o backend C

O Z7 adiciona o primeiro backend nativo do Zumbra. Ele consome diretamente a MIR otimizada, gera C11 portátil e usa Clang ou GCC para produzir um executável independente da VM e do toolchain Go.

## Uso básico

Build de desenvolvimento:

```bash
zumbra build app.zum
./build/app
```

Build otimizado:

```bash
zumbra build --release app.zum
./build/app
```

Escolher o compilador:

```bash
zumbra build --compiler clang app.zum
zumbra build --compiler gcc app.zum
```

O modo automático procura, nesta ordem:

1. compilador definido em `CC`;
2. `clang`;
3. `gcc`;
4. `cc`.

Definir outro arquivo de saída:

```bash
zumbra build --release -o dist/meu-jogo app.zum
```

Gerar C sem compilar:

```bash
zumbra build --emit-c app.zum
```

Os arquivos são gravados em:

```text
build/native/<programa>/main.c
build/native/<programa>/zumbra_runtime.c
build/native/<programa>/zumbra_runtime.h
```

## Pipeline

```text
Código Zumbra
    ↓
Parser e análise semântica
    ↓
Type Analysis
    ↓
HIR
    ↓
MIR otimizada
    ↓
Backend C do Zumbra
    ↓
C11 + runtime mínimo
    ↓
Clang/GCC
    ↓
Executável nativo
```

O backend não usa o transpiler Go. A MIR é a fonte de verdade para a geração nativa.

## Recursos cobertos no Z7

- inteiros, inteiros fixos, floats, booleanos, strings e `null`;
- funções, recursão e retorno implícito;
- variáveis globais e locais;
- `if`, `match`, `while`, `for`, ranges, `break` e `continue`;
- arrays, dicionários, indexação e mutação;
- structs, construção posicional ou nomeada, campos e métodos;
- enums;
- `ByteArray`, arrays tipados, slices compartilhadas e `fill`;
- leitura e escrita de arquivos binários;
- leitura e escrita little-endian e big-endian;
- cópia, comparação e SHA-256 de buffers;
- casts de inteiros fixos e aritmética wrapping, checked e saturating.

## Runtime nativo

O runtime C usa uma representação uniforme chamada `ZValue`. Valores compostos são alocados em uma arena e liberados juntos ao final do processo.

Esse desenho inicial oferece:

- implementação pequena;
- sem garbage collector durante a execução;
- slices sem cópia;
- interoperabilidade simples com C;
- comportamento previsível entre plataformas.

O executável já é nativo, mas alguns valores ainda usam representação tagged/boxed. O Z7 não afirma que todo programa já atingirá o desempenho de C ou Rust. Os próximos passes poderão especializar operações escalares da MIR e remover boxing quando os tipos forem conhecidos.

## Diagnósticos explícitos

O backend rejeita antes da compilação recursos sem implementação nativa, em vez de gerar código parcialmente correto.

No Z7, continuam exclusivos da VM:

- imports, que serão ligados pelo sistema modular do Z8;
- `async` e `await`;
- handlers `try ... or`;
- closures que capturam variáveis locais externas;
- integrações como bancos, HTTP legado, Redis, Supabase e Twilio.

Exemplo de erro:

```text
native backend rejected the program:
    builtin "randomInteger" is not available in the Z7 native runtime
```

## Benchmark

Geração de C:

```bash
go test ./benchmark -bench NativeCGeneration -benchmem
```

Comparação da VM:

```bash
go run ./benchmark -engine vm
```

Execução nativa, incluindo o tempo de compilação separado do tempo do processo:

```bash
go run ./benchmark -engine native -release=true
```

Os números devem ser comparados na mesma máquina e com o mesmo código. Tempo de build e tempo de execução são métricas diferentes.

## Linking de módulos nativos no Z8

Arquivos declarados por um bloco `extern "C" from "..."` entram automaticamente no comando do compilador:

```zumbra
extern "C" from "../native/math.c" {
    fct add(left: i32, right: i32) -> i32;
}
```

Dependências adicionais podem ser informadas no CLI:

```bash
zumbra build --link native/math.o app.zum
zumbra build --link native/libmath.a app.zum
zumbra build --include native/include --library-dir native/lib -l math app.zum
```

Use `zumbra modules app.zum` para verificar os links descobertos pelo grafo. A sintaxe, os tipos C e as regras de `unsafe` estão documentados em [`modulos-abi-e-ffi.md`](./modulos-abi-e-ffi.md).
