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