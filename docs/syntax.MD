# Zumbra Syntax Guide

Zumbra is a functional programming language, compiled and executed on top of a virtual machine (VM). Its goal is to make code more accessible and easier to understand, with a light and direct syntax.

---

## Variables

Zumbra uses `var` to declare variables. Assignment (and reassignment) is done with `<<`.

```zumbra
var x << 10;
x << 8;
x << "some text";
```

There are no constants.

---

## Functions

Functions are declared using `fct` and assigned to variables. `return` is optional: the last expression is returned automatically.

```zumbra
var sum << fct(a, b) {
    return a + b;
};

var sub << fct(a, b) {
    a - b;
};

show(sum(10, 10)); // 20
show(sub(30, 10)); // 20
```

---

## Flow Control

`if-else` in Zumbra programming language to make a decision in fulfillment is fulfilled (true) and run other code if it is not fulfilled or false.

### `if` and `else`

```zumbra
var x << 10;

if (x == 10) {
    show("X is equal to 10");
} else {
    show("X is not equal to 10");
}
```

Logical operators are supported:

```zumbra
if (x == 10 and y == 4) {
    show("AND case");
}

if (x == 10 or y == 10) {
    show("OR case");
}
```

### `while`

```zumbra
var x << 0;

while (x < 3) {
    show(x);
    x << x + 1;
    show(x);
}
```

---

## Operators

In the Zumbra programming language, operators are special symbols or characters used to perform operations or actions on values ​​(data) or variables. Just like mathematics.

Zumbra supports all common operators:

* Arithmetic: `+`, `-`, `*`, `/`, `%`
* Comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`
* Logical: `and`, `or`, `not`

---

## Collections

Data structures are used to store the values ​​of variables and process a set of data. Lets you store multiple values ​​in one variable, formatted in Zumbra language:

### Arrays

```zumbra
var arr << [1, 2, 3];
```

### Dictionaries

```zumbra
var dict << {"a": "v", "b": "o"};
```

---

## Output / Debugging

Use `show(...)` to display output:

```zumbra
show("Zumbra"); // Zumbra
show("{} + {} = {}", 1, 1, 2); // 1 + 1 = 2
show("{}", 1); // 1
show("{}"); // {}
```

---

## Comments

Use `//` to write single-line comments:

```zumbra
// This is a comment
var x << 1; // Inline comment
```

---

## Full Example

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
