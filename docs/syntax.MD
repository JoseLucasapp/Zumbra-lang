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
