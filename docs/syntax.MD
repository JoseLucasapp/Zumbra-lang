# Zumbra Syntax Guide

Zumbra is a programming language executed through this official path:

`lexer -> parser -> compiler -> VM`

The REPL uses this same path.

This document describes the current practical syntax supported by the repository.

---

## Variables

Variables are declared with `var`.

Assignment and reassignment use `<<`.

```zumbra
var x << 10;
x << 20;
x << x + 1;
```

---

## Functions

Functions are declared with `fct`.

The last expression can be returned implicitly.

```zumbra
var sum << fct(a, b) {
    a + b;
};

var explicit << fct(a, b) {
    return a + b;
};
```

---

## Conditionals

### if / else

```zumbra
if (x == 10) {
    show("x is 10");
} else {
    show("x is not 10");
}
```

Logical operators:

```zumbra
if (x == 10 and y == 4) {
    show("and");
}

if (x == 10 or y == 10) {
    show("or");
}
```

---

## While

```zumbra
var x << 0;

while (x < 3) {
    show(x);
    x << x + 1;
}
```

---

## For

### Range loop

```zumbra
for i in 1..5 {
    show(i);
}
```

### Array iteration

```zumbra
var arr << [10, 20, 30];

for item in arr {
    show(item);
}
```

### Dictionary iteration

```zumbra
var dict << {"a": 1, "b": 2};

for key, value in dict {
    show(key);
    show(value);
}
```

### Infinite loop

```zumbra
for {
    break;
}
```

---

## break and continue

```zumbra
for i in 1..10 {
    if (i == 3) {
        continue;
    }

    if (i == 8) {
        break;
    }

    show(i);
}
```

---

## Number literals

Zumbra supports decimal, hexadecimal, binary and octal integers. `_` can be used as a visual separator.

```zumbra
var decimal << 255;
var hexadecimal << 0xFF;
var binary << 0b1111_0000;
var octal << 0o755;
var readable << 1_000_000;
```

## Operators

### Arithmetic

- `+`
- `-`
- `*`
- `/`
- `%`
- `**`

### Comparison

- `==`
- `!=`
- `<`
- `<=`
- `>`
- `>=`

### Logical

- `and`
- `or`
- `!`

### Bitwise

- `band` — bitwise AND
- `bor` — bitwise OR
- `bxor` — bitwise XOR
- `bnot` — bitwise NOT
- `shl` — shift left
- `shr` — shift right

Bitwise operators currently accept integers only. Shift counts must be between `0` and `63`.

---

## Arrays

```zumbra
var arr << [1, 2, 3];
show(arr[0]);
```

---

## Dictionaries

```zumbra
var dict << {"name": "zumbra", "year": 2025};
show(dict["name"]);
```

---

## Attribute access

Zumbra supports dot-based attribute access in the language surface.

```zumbra
user.name;
```

Use dictionary indexing when working with dictionary keys:

```zumbra
user["name"];
```

---

## Imports

```zumbra
import "module.zum";
```

Imports are compiled through the same compiler pipeline and are cached during the same execution session.

---

## Async / await

```zumbra
var fetchValue << async fct() {
    42;
};

show(await fetchValue());
```

---

## Error helpers

### try

```zumbra
var result << try something();
```

### or block

```zumbra
var result << try something() or err {
    show(err);
};
```

### panic

```zumbra
panic("fatal");
```

---

## Strings

```zumbra
var name << "Zumbra";
show(name);
```

---

## Comments

Single-line comments use `//`.

```zumbra
// comment
var x << 1; // inline comment
```

---

## Example

```zumbra
var x << 0;

while (x < 5) {
    if (x % 2 == 0) {
        show("{} is even", x);
    } else {
        show("{} is odd", x);
    }

    x << x + 1;
}
```

---

## Practical note

The repository examples under `code_examples/` should be treated as the practical reference for currently supported constructs.
---

## Fixed-width integers

Zumbra supports compact fixed-width integer literals for systems, games, and emulation:

```zumbra
var opcode << 0xA9u8;
var address << 0x8000u16;
var signedOffset << i8(-10);
```

Available types:

```text
u8 u16 u32 u64
i8 i16 i32 i64
```

Type names are also safe conversion functions:

```zumbra
var byte << u8(200);
```

Normal fixed-width arithmetic wraps like hardware. Use `checkedAdd`, `checkedSub`, and `checkedMul` for overflow errors, or `satAdd`, `satSub`, and `satMul` for saturating arithmetic.

Complete documentation: [`docs/pt-BR/inteiros-de-tamanho-fixo.md`](./pt-BR/inteiros-de-tamanho-fixo.md).


---

## Indexed assignment

Arrays and dictionaries can be changed with the same readable assignment operator used by variables:

```zumbra
var memory << [0u8, 0u8, 0u8];
memory[1] << 0xA9u8;

var player << {"score": 10};
player["score"] << 25;
player["lives"] << 3;
```

Nested arrays are supported:

```zumbra
var matrix << [[0, 0], [0, 0]];
matrix[1][0] << 7;
```

Array writes must stay inside the existing bounds. Dictionary writes may update or insert a key. Strings are immutable.

Complete documentation: [`docs/pt-BR/mutacao-por-indice.md`](./pt-BR/mutacao-por-indice.md).


---

## Compact memory, typed arrays, and slices

Create compact byte memory with `bytes(size)`:

```zumbra
var memory << bytes(4096);
memory[0x200] << 0xA9u8;
```

Create compact fixed-integer arrays with `arrayOf(type, size)`:

```zumbra
var registers << arrayOf("u16", 16);
registers[0] << 0x8000u16;
```

Create a mutable view without copying data:

```zumbra
var page << slice(memory, 0x200, 0x300);
page[0] << 0x42u8;
```

Fill a whole collection or view:

```zumbra
fill(page, 0u8);
```

Complete documentation: [`docs/pt-BR/memoria-compacta.md`](./pt-BR/memoria-compacta.md).


---

## Binary files and endianness

Read a file as compact bytes:

```zumbra
var rom << readBytes("game.nes");
```

Write a byte-compatible buffer and receive the number of bytes written:

```zumbra
var written << writeBytes("copy.nes", rom);
```

Read and write unsigned integers with explicit byte order:

```zumbra
var data << bytes(16);
writeU16LE(data, 0, 0x1234u16);
writeU32BE(data, 2, 0x89ABCDEFu32);

var first << readU16LE(data, 0);
var second << readU32BE(data, 2);
```

Available endian functions:

```text
readU16LE  readU16BE  writeU16LE  writeU16BE
readU32LE  readU32BE  writeU32LE  writeU32BE
readU64LE  readU64BE  writeU64LE  writeU64BE
```

Copy, compare, and hash buffers:

```zumbra
var target << bytes(sizeOf(rom));
copyBytes(target, 0, rom, 0, sizeOf(rom));
show(bytesEqual(rom, target));
show(sha256(rom));
```

Complete documentation: [`docs/pt-BR/arquivos-binarios-e-endianness.md`](./pt-BR/arquivos-binarios-e-endianness.md).


---

## Constants, structs, enums, and match

Declare immutable values with `const`:

```zumbra
const RESET_VECTOR << 0xFFFCu16;
```

Create a readable alias for an existing type:

```zumbra
type Byte << u8;
```

Declare data and methods with a compact struct:

```zumbra
struct Player {
    x: int;
    y: int;

    fct move(dx, dy) {
        self.x << self.x + dx;
        self.y << self.y + dy;
    }
}

var player << Player(10, 20);
player.move(5, -2);
player.x << 30;
```

Named initialization is also supported:

```zumbra
var player << Player({"x": 10, "y": 20});
```

Declare a finite set of values and select with `match`:

```zumbra
enum Direction { Up; Down; Left; Right; }

var direction << Direction.Right;
var label << match(direction) {
    case Direction.Left { "left"; }
    case Direction.Right { "right"; }
    else { "other"; }
};
```

Complete documentation: [`docs/pt-BR/tipos-estruturados.md`](./pt-BR/tipos-estruturados.md).

---

## Typed IR and pipeline tools

Z6 does not change source syntax. It adds validation and inspection commands:

```bash
zumbra check app.zum
zumbra ir app.zum hir
zumbra ir app.zum mir
zumbra ir app.zum optimized
```

`hir` shows the typed high-level representation. `mir` shows explicit virtual values without optimization. `optimized` applies constant folding, dead-value elimination, and unreachable-code elimination.

Complete documentation: [`docs/pt-BR/ir-tipada-e-pipeline.md`](./pt-BR/ir-tipada-e-pipeline.md).


## Native builds

Z7 compiles optimized MIR to portable C11 and then invokes Clang or GCC:

```bash
zumbra build app.zum
zumbra build --release app.zum
zumbra build --compiler clang -o dist/app app.zum
zumbra build --emit-c app.zum
```

Generated sources are placed under `build/native/<program>/`. Unsupported native features fail with explicit diagnostics; the VM remains available through `zumbra run`.

---

## Modules, visibility, and C FFI

Declarations are private to their file by default. Export a top-level declaration with `pub`:

```zumbra
pub const BASE << 40;

pub fct add(left, right) {
    left + right;
}
```

Import a module through an alias:

```zumbra
import "../modules/math.zum" as math;
show(math.add(math.BASE, 2));
```

Inspect the resolved graph:

```bash
zumbra modules app.zum
```

Declare native C functions with typed signatures:

```zumbra
pub extern "C" from "../native/math.c" {
    fct add(left: i32, right: i32) -> i32 as "project_add";
    fct apply(value: i32, cb: callback(i32) -> i32) -> i32;
}
```

External calls must be explicit:

```zumbra
unsafe {
    show(add(20i32, 22i32));
}
```

Generate conservative bindings for simple C headers:

```bash
zumbra bind-c --pub --link ../native/math.c -o bindings/math.zum native/math.h
```

Complete documentation: [`docs/pt-BR/modulos-abi-e-ffi.md`](./pt-BR/modulos-abi-e-ffi.md).

---

## Contextual callback inference

A callback can remain concise when the receiving function already declares its signature:

```zumbra
extern "C" {
    fct apply(value: i32, transform: callback(i32) -> i32) -> i32;
}

var double << fct(value) {
    value * 2i32;
};

unsafe {
    show(apply(21i32, double));
}
```

The expected `callback(i32) -> i32` type is applied before checking the callback body. The parameter `value`, the multiplication, the function return, and every HIR/MIR reference become `i32`.

Inline callbacks are inferred the same way:

```zumbra
unsafe {
    show(apply(21i32, fct(value) { value * 2i32; }));
}
```

A callback with the wrong arity or return type is rejected. A named callback that has already been inferred with one concrete signature cannot later be reused with an incompatible signature.

Complete documentation: [`docs/pt-BR/inferencia-contextual-de-callbacks.md`](./pt-BR/inferencia-contextual-de-callbacks.md).

---

## Concorrência e tarefas

Inicie uma chamada comum em paralelo com `spawn`:

```zumbra
fct calculate(value) {
    value * value;
}

var task << spawn calculate(6);
show(await task);
```

Uma função declarada com `async` retorna uma `Task` em todas as chamadas:

```zumbra
var calculate << async fct(value) {
    value * value;
};

show(await calculate(6));
```

Use channels para comunicação:

```zumbra
var messages << channel(8);
send(messages, "ready");
show(receive(messages));
closeChannel(messages);
```

O tipo do channel é inferido pelo primeiro envio conhecido. Depois de enviar uma `string`, envios de outro tipo são rejeitados.

Primitivas adicionais:

```zumbra
join(task);
joinTimeout(task, 100);
cancel(task);
taskDone(task);
taskCancelled(task);
sleepMs(10);

receiveOk(messages);
receiveTimeout(messages, 100);
channelClosed(messages);
channelLen(messages);
channelCap(messages);

var guard << mutex();
lock(guard);
unlock(guard);

var group << waitGroup();
wgAdd(group, 1);
wgDone(group);
wgWait(group);

var slots << semaphore(2);
acquire(slots);
release(slots);

var counter << atomicInt(0);
atomicAdd(counter, 1);
```

Complete documentation: [`docs/pt-BR/concorrencia-e-tarefas.md`](./pt-BR/concorrencia-e-tarefas.md).

---

## Inferência contextual de chamadas — Z9.1

Funções sem anotações podem receber tipos concretos a partir da primeira chamada válida:

```zumbra
fct identity(value) {
    value;
}

var result << identity(42);
```

O tipo inferido é:

```text
fct(int) -> int
```

A inferência também atravessa `spawn`:

```zumbra
fct square(value) {
    value * value;
}

var task << spawn square(6);
var result << await task;
```

Tipos resultantes:

```text
square: fct(int) -> int
task: Task<int>
result: int
```

Channels e primitivas concorrentes são propagados para os parâmetros:

```zumbra
fct publish(output, value) {
    send(output, value);
    return;
}

var messages << channel(1);
var producer << spawn publish(messages, 7);
```

Depois da chamada:

```text
publish: fct(Channel<int>, int) -> null
messages: Channel<int>
producer: Task<null>
```

Métodos seguem a mesma regra:

```zumbra
struct Box {
    value: int;

    fct echo(input) {
        input;
    }
}

var box << Box(1);
var result << box.echo(9);
```

`echo` é inferido como `fct(int) -> int`.

A especialização atual é monomórfica. Depois que uma função é concretizada, reutilizá-la com uma assinatura incompatível produz erro:

```zumbra
var identity << fct(value) { value; };
identity(42);
identity("zumbra"); // erro
```

A sintaxe não introduz generics explícitos. Polimorfismo real será projetado futuramente sem alterar silenciosamente o contrato monomórfico atual.

Documentação completa: [`docs/pt-BR/inferencia-contextual-de-chamadas.md`](./pt-BR/inferencia-contextual-de-chamadas.md).

---

## Rede, streams e TLS — Z10

O módulo de conveniência atual é importado por caminho:

```zumbra
import "../../std/net.zum" as net;
```

Servidor e cliente TCP local:

```zumbra
fct echo(listener) {
    var connection << net.accept(listener);
    var payload << net.readExact(connection, 4);
    net.writeBytes(connection, payload);
    net.closeStream(connection);
    return;
}

var listener << net.listen("127.0.0.1", 0);
var server << spawn echo(listener);
var client << net.connectTimeout(
    "127.0.0.1",
    net.listenerPortNumber(listener),
    1000
);

net.writeText(client, "ping");
var response << net.readExact(client, 4);
```

APIs de baixo nível:

```zumbra
tcpListen(host, port);
tcpConnect(host, port);
tcpConnectTimeout(host, port, timeoutMs);
listenerAccept(listener);
listenerAcceptTimeout(listener, timeoutMs);

streamRead(stream, maximumBytes);
streamReadExact(stream, amount);
streamReadTimeout(stream, maximumBytes, timeoutMs);
streamWrite(stream, data);
streamWriteAll(stream, data);
streamShutdownRead(stream);
streamShutdownWrite(stream);
streamClose(stream);

dnsLookup(host);
dnsLookupTimeout(host, timeoutMs);

udpBind(host, port);
udpSendTo(socket, host, port, data);
udpReceiveFrom(socket, maximumBytes);
udpReceiveFromTimeout(socket, maximumBytes, timeoutMs);

tlsListen(host, port, certificate, key);
tlsConnect(host, port, serverName, insecure);
tlsConnectTimeout(host, port, serverName, insecure, timeoutMs);
```

Documentação completa: [`docs/pt-BR/rede-tcp-udp-dns-tls.md`](./pt-BR/rede-tcp-udp-dns-tls.md).
