# Z17 — Programação de sistemas completa

O Z17 transforma a Zumbra em uma linguagem capaz de manipular memória explícita e recursos do sistema operacional sem remover as proteções existentes do evaluator, da VM e do pipeline tipado.

A versão estável do marco é **Zumbra 0.13.0**. O marco foi entregue como uma única versão, sem subversões Z17.1, Z17.2 ou equivalentes.

## Objetivos do marco

O Z17 fecha os seguintes blocos:

- ponteiros tipados e memória explícita;
- ownership, empréstimos e movimentação;
- arenas e diagnóstico de memória;
- layout ABI nativo;
- arquivos mapeados e memória compartilhada;
- acesso volatile, atomics e barreiras de memória;
- proteção e bloqueio de páginas;
- carregamento dinâmico e chamadas por símbolo;
- informações do sistema, syscalls controladas e profiling;
- integração com processos;
- otimização de constantes de layout nativo;
- builds nativos com sanitizers;
- paridade entre evaluator, VM e backend C11.

## Ponteiros tipados

A alocação recebe o tipo nativo e a quantidade de elementos:

```zumbra
var pointer << alloc("i32", 4);

pointer[0] << 10i32;
pointer[1] << 20i32;
pointer[2] << 30i32;
pointer[3] << 40i32;

show(pointer[2]);
free(pointer);
```

O type checker representa o resultado como `ptr<i32>`. O tipo do elemento permanece concreto em indexação, `pointerRead`, `pointerWrite`, empréstimos e `realloc`.

Tipos de memória suportados:

| Nome | Representação |
|---|---|
| `u8`, `i8` | 8 bits |
| `u16`, `i16` | 16 bits |
| `u32`, `i32` | 32 bits |
| `u64`, `i64` | 64 bits |
| `int` | inteiro Zumbra de 64 bits |
| `float` | `double` nativo |
| `bool` | byte booleano |
| `ptr` | endereço nativo |

APIs principais:

```text
alloc, calloc, realloc, free
nullPointer
addressOf, pointerFromAddress
pointerRead, pointerWrite, dereference
pointerOffset, pointerLength, pointerByteLength
pointerType, pointerAddress
pointerEqual, pointerCompare, pointerIsAligned
pointerIsNull, pointerIsValid
pointerOwned, pointerBorrowed, pointerMutable
pointerCopy, pointerFill
```

## Segurança no evaluator e na VM

Alocações gerenciadas mantêm metadados de vida útil. Erros são detectados antes de acessar memória inválida:

- desreferência de ponteiro nulo;
- acesso fora dos limites;
- use-after-free;
- double free;
- escrita por ponteiro imutável;
- uso de ponteiro movido;
- uso de empréstimo já liberado;
- `free`, `realloc` ou `move` durante empréstimos ativos.

Ponteiros criados com `pointerFromAddress` são views de memória externa. Como a validade dessa memória não pode ser provada pela Zumbra, a operação exige `unsafe`.

```zumbra
var raw << nullPointer("u8");

unsafe {
    raw << pointerFromAddress("u8", address, length, true);
    raw[0] << 42u8;
}
```

## Ownership e empréstimos

Uma alocação possui exatamente um ponteiro proprietário.

```zumbra
var owner << alloc("u64", 1);
var view << borrowPointer(owner);

show(pointerRead(view));
releaseBorrow(view);

var nextOwner << movePointer(owner);
free(nextOwner);
```

APIs:

```text
borrowPointer
borrowPointerMut
releaseBorrow
movePointer
```

Regras:

- múltiplos empréstimos imutáveis são permitidos;
- um empréstimo mutável é exclusivo;
- empréstimo mutável não pode coexistir com outro empréstimo;
- o proprietário não pode ser liberado, redimensionado ou movido enquanto existirem empréstimos;
- `movePointer` invalida o ponteiro de origem.

## Arenas

Arenas agrupam várias alocações e permitem liberação em lote:

```zumbra
var arena << arenaCreate();
var temporary << arenaAlloc(arena, "u16", 128);

arenaReset(arena);
arenaFree(arena);
```

`arenaReset` invalida todos os ponteiros criados pela arena e mantém a arena aberta. `arenaFree` encerra a arena definitivamente.

APIs:

```text
arenaCreate
arenaAlloc
arenaReset
arenaFree
arenaStats
```

## Diagnóstico de memória

```zumbra
show(memoryStats());
show(memoryLeaks());
show(memoryValidate());
memoryResetStats();
```

As métricas incluem:

- alocações e liberações;
- bytes totais, ativos e pico;
- blocos ativos;
- acessos inválidos;
- double frees;
- lista de blocos ainda ativos.

## Layout ABI

```zumbra
show(sizeOfType("i32"));
show(alignOfType("u64"));

var layout << nativeStructLayout([
    {"name": "tag", "type": "u8"},
    {"name": "value", "type": "u64"}
]);

show(layout);
```

`nativeStructLayout` retorna tamanho, alinhamento e offset de cada campo segundo as regras do backend nativo. Chamadas constantes a `sizeOfType` e `alignOfType` são dobradas pela otimização da MIR.

## Arquivos mapeados

```zumbra
var mapping << mmapOpen("data.bin", "readwrite", 4096);
var memory << mmapPointer(mapping);

memory[0] << 42u8;
mmapFlush(mapping);
mmapClose(mapping);
```

Modos:

- `read`;
- `readwrite`;
- `private`.

APIs:

```text
mmapOpen
mmapPointer
mmapFlush
mmapClose
mmapSize
```

## Memória compartilhada

```zumbra
var shared << sharedMemoryOpen("example", 4096, true);
var memory << sharedMemoryPointer(shared);

memory[0] << 7u8;
sharedMemoryClose(shared);
sharedMemoryUnlink("example");
```

## Volatile, atomics e ordenação

Operações volatile exigem `unsafe`:

```zumbra
unsafe {
    volatileWrite(pointer, 0, 1u32);
    show(volatileRead(pointer, 0));
}
```

Atomics:

```text
atomicPointerLoad
atomicPointerStore
atomicPointerAdd
atomicPointerSwap
atomicPointerCompareSwap
memoryFence
```

Ordens aceitas por `memoryFence`:

```text
relaxed
acquire
release
acq_rel
seq_cst
```

As operações atômicas exigem tamanho e alinhamento compatíveis com o tipo do ponteiro.

## Páginas de memória

```zumbra
unsafe {
    memoryProtect(pointer, "read");
}

memoryLock(pointer);
memoryUnlock(pointer);
```

`memoryProtect` exige `unsafe`. Em Linux, a faixa é arredondada para os limites de página antes de chamar `mprotect`.

## Bibliotecas dinâmicas

```zumbra
var library << dynamicOpen("libc.so.6");

unsafe {
    var getpid << dynamicSymbol(library, "getpid");
    show(dynamicCall(getpid, "i32", []) > 0i32);
}

dynamicClose(library);
```

`dynamicCall` oferece uma ABI mínima para funções com até seis argumentos inteiros, booleanos ou ponteiros e retorno inteiro, booleano, ponteiro ou `void`. Retornos e argumentos de ponto flutuante não fazem parte dessa ABI mínima; para assinaturas C complexas, use o FFI tipado do Z8.

## Processos

O módulo `std/system.zum` reutiliza o runtime portátil de processos do Z13:

```zumbra
import "../../std/system.zum" as system;

var process << system.spawnProcess("/bin/sh", {
    "args": ["-c", "exit 7"]
});

show(system.processId(process));
show(system.waitProcess(process));
```

Também estão disponíveis `killProcess` e `processRunning`. Diretório de trabalho e ambiente podem ser definidos nas opções de criação.

## Informações do sistema e profiling

```zumbra
show(systemInfo());
show(pageSize());
show(cpuCount());

var start << profileNowNs();
// trabalho
show(profileElapsedNs(start));
```

## Syscalls brutas

`rawSyscall` é uma saída de emergência Linux, exige `unsafe` e permanece desativada por padrão.

```bash
export ZUMBRA_ALLOW_RAW_SYSCALLS=1
```

```zumbra
unsafe {
    var result << rawSyscall(number, [arg0, arg1]);
    show(result);
}
```

O resultado contém `ok`, `value`, `errno` e `error`.

## Sanitizers no backend C11

```bash
zumbra build --release --sanitize address app.zum
zumbra build --release --sanitize address,undefined app.zum
zumbra build --release --sanitize thread app.zum
zumbra build --release --sanitize leak app.zum
```

`thread` não pode ser combinado com `address` ou `leak`. O backend adiciona `-fno-omit-frame-pointer` automaticamente.

## Módulo padrão

O módulo [`std/system.zum`](../../std/system.zum) fornece nomes organizados para memória, arenas, mapeamentos, atomics, bibliotecas, sistema, profiling e processos. Operações que obrigatoriamente exigem `unsafe` permanecem como builtins diretos para que o checker veja o bloco inseguro no local exato da chamada.

## Paridade de backends

O Z17 é validado em:

- evaluator;
- VM bytecode;
- backend C11 com Clang;
- backend C11 com GCC;
- AddressSanitizer e UndefinedBehaviorSanitizer.

A camada de memória gerenciada e os diagnósticos funcionam nos três backends. Integrações de sistema operacional são Linux-first; plataformas sem implementação retornam erros explícitos em vez de simular sucesso.

## Exemplos oficiais

```text
code_examples/core/systems_programming.zum
code_examples/core/systems_mapping.zum
code_examples/core/systems_process.zum
code_examples/core/systems_dynamic.zum
```

## Gate do marco

```bash
scripts/test-z17-foundation.sh
```

O script valida type checker, compiler, MIR, evaluator, VM, C11, Clang, GCC, mapeamento, processos, carregamento dinâmico e build com ASan/UBSan.
