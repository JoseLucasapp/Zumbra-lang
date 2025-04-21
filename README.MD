# VAJA Programming Language

> ⚙️ *VAJA is a simple, fun, and expressive programming language made for learning, experimenting, and enjoying the beauty of code.*

---

## ✨ What is VAJA?

VAJA is a custom programming language built with its own parser, compiler, and virtual machine. It supports function definitions, scoped variables, custom operators, and more — all with a minimalist syntax and a playful soul.

Not your typical cup of J..., try a shot of VAJA.

---

## 📦 Features

- 🧠 **User-defined functions** with `fct`
- 📦 **Global and local variables** with `var`
- 🧮 Arithmetic and logical operations: `+ - * / % = <= >=`
- 🔁 Conditional logic: `if`, `else`, `return`
- 🔧 Custom assignment operator: `<<`
- 📚 Built-in functions:
  - `sizeOf`
  - `addToArray`
  - `removeFromArray`
  - `first`, `last`, `allButFirst`
  - `show`
  - `input`
- 🧠 **Don't forget the ";"**

---

## 🛠 Code examples

```vaja
show("Whats your name ?");
var name << input();

show("My name is " + name);
```

```vaja
var sum << fct(a, b){
    return a + b;
};

var sub << fct(a,b){
    return a - b;
};

show(sum(1, 2));
show(sub(2, 5));
```

---

## 📥 How to Run

You can execute `.vaja` code using the compiled VAJA binary on different platforms.

### **Windows** 🖥️

1. Download the `vaja.exe` file for Windows from the [GitHub Release](https://github.com/JoseLucasapp/VAJA/releases/download/v0.0.1/vaja.exe).
2. Open a command prompt or PowerShell window and run the following command:

   ```bash
   vaja.exe myscript.vaja
   ```

### **Linux** 🐧

1. Download the `vaja-linux` file for Linux from the [GitHub Release](https://github.com/JoseLucasapp/VAJA/releases/download/linuxv0.0.1/vaja-linux).
2. Make the file executable by running:

   ```bash
   chmod +x vaja-linux
   ```

3. Run the script with the following command in your terminal:

   ```bash
   ./vaja-linux myscript.vaja
   ```

---

### ⚡ Additional Notes:
- Make sure the VAJA binary file (`vaja-linux`, `vaja.exe`) has execution permissions on your system.
- For **Windows**, just run the command without additional configuration.
- For **Linux**, use the `chmod` command to make the file executable.

---

## 🦝 Meet the Mascot

VAJA is proudly represented by its friendly mascot: **Pinguelim**, the coding gambazin!

<p align="center">
    <img src="./assets/mascot.png" width="200" alt="Pinguelim">
</p>

Feel free to include Pinguelim in your projects or documentation! He loves code and caffeine ☕.

---

## 💻 Developing VAJA

VAJA is written in Go and includes a custom-built:

- Lexer
- Parser (Pratt-style)
- Bytecode Compiler
- Virtual Machine (Stack-based)

### Compile it:

```bash
go build -o vaja
```

---

## 📎 VS Code Integration

We provide a [VAJA VS Code Extension](https://marketplace.visualstudio.com/items/?itemName=joselucasapp.vaja-lang-support) for syntax highlighting and `.vaja` file support.

> Includes keyword coloring, strings, and built-in function recognition.

---

## 📄 License

MIT License. See `LICENSE` for details.

---

### 👨‍💻 Creator

---

## Joselucasapp
Bringing VAJA to life with love, logic, and a hint of rebellion.