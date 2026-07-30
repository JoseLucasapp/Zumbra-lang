
<p align="center">
<img src="./assets/zumbra.png" width="100%"/>
</p>

> ⚙️ *ZUMBRA is a simple, fun, and expressive programming language made for learning, experimenting, and enjoying the beauty of code.*

<br />
<a href="https://zumbra-web.vercel.app/"><p style="font-size: 30px; font-weight: 600;">🖥️ Official website</p></a>

<p style="font-size: 20px; font-weight: 600; color: red">This readme will no longer be updated, all updates will be on the website</p>

<br />
<p style="font-size: 30px; font-weight: 600;">✨ What is ZUMBRA?</p>

Zumbra is a lightweight, expressive programming language designed and built entirely from scratch. It features a fully custom lexer, parser, compiler, and virtual machine, providing an educational yet powerful platform to explore language design concepts.  

Zumbra supports basic types like integers, floats, booleans, and strings, control flow structures like if and for, and logical operators with proper short-circuit evaluation.

Designed for clarity, simplicity, and extensibility, Zumbra is the perfect playground for learning how modern programming languages work internally.

The rebuild now follows a clear direction: systems, emulation, and game development without abandoning Zumbra's simplicity. See the Portuguese design and feature documentation in [`docs/pt-BR/`](./docs/pt-BR/).

**Zumbra 0.12.6** corrige a composição de scrollbars com modais: barras da interface de fundo permanecem abaixo do backdrop e não atravessam diálogos. Consulte [`docs/releases/0.12.6.md`](./docs/releases/0.12.6.md).

Current systems features include readable bitwise operators, fixed-width integers (`u8` through `u64`, `i8` through `i64`), mutable indexes, compact byte memory, typed integer arrays, zero-copy mutable slices, binary file I/O, endian-aware integer access, buffer copying/comparison, SHA-256 hashing, immutable constants, type aliases, structs with methods, enums, and `match`. Z6 adds a canonical typed pipeline with HIR and MIR. Z7 adds a C11 backend that consumes optimized MIR and produces standalone native executables with Clang or GCC. Z8 adds isolated modules with `pub` exports, module graph inspection, typed C FFI, opaque pointers, synchronous callbacks, native linking, and a conservative C-header binding generator. Z8.1 adds contextual callback inference, so callback parameters and returns become concrete in type analysis, HIR, MIR, and native builds without extra annotations. Z9 adds real tasks, `spawn`, `async`/`await`, inferred typed channels, timeouts, cooperative cancellation, mutexes, wait groups, semaphores, and atomics across the VM and pthread-based native runtime. Z9.1 extends contextual inference to ordinary calls and `spawn`, so user functions, methods, `Channel<T>`, `Task<T>`, atomics, and return values become concrete without annotations. Z10 adds local-first TCP, UDP, IPv4/IPv6 DNS, TLS 1.2+, common byte streams, accept/connect/read timeouts, half-close, keepalive, and conditional native linking through POSIX sockets and OpenSSL. Z11 adds typed HTTP/1.1 applications, routing, middleware, JSON, cookies, stateless JWT sessions, multipart uploads, static files, gzip, streaming/SSE, HTTP clients, graceful shutdown and RFC 6455 WebSockets/WSS across the VM and native runtime. Z12 completes the data layer with advanced SQLite, PostgreSQL, Redis, safe SQL parameters, migrations, streaming rows, persistent sessions, typed configuration, structured logs, metrics, tracing, rate limiting, JSON files, and versioned binary serialization across the VM and native backend. Z13 adds a Linux-first desktop runtime with SDL3 windows, lifecycle and events, multiple windows, High DPI, clipboard, drag and drop, pickers, notifications, tray, shortcuts, system paths, processes, and a deterministic headless backend across the evaluator, VM, Clang, and GCC. Z14 adds a retained-mode GUI toolkit with layouts, controls, tables, lists, trees, tabs, menus, modals, tooltips, progress, images, canvas, themes, reactive state, bidirectional data binding, custom components, keyboard navigation and accessibility across the VM and native backend. See [`docs/pt-BR/modulos-abi-e-ffi.md`](./docs/pt-BR/modulos-abi-e-ffi.md), [`docs/pt-BR/inferencia-contextual-de-callbacks.md`](./docs/pt-BR/inferencia-contextual-de-callbacks.md), [`docs/pt-BR/concorrencia-e-tarefas.md`](./docs/pt-BR/concorrencia-e-tarefas.md), [`docs/pt-BR/inferencia-contextual-de-chamadas.md`](./docs/pt-BR/inferencia-contextual-de-chamadas.md), and [`docs/pt-BR/rede-tcp-udp-dns-tls.md`](./docs/pt-BR/rede-tcp-udp-dns-tls.md), and [`docs/pt-BR/dados-e-persistencia-z12.md`](./docs/pt-BR/dados-e-persistencia-z12.md), and [`docs/pt-BR/runtime-desktop-z13.md`](./docs/pt-BR/runtime-desktop-z13.md), and [`docs/pt-BR/toolkit-interface-grafica-z14.md`](./docs/pt-BR/toolkit-interface-grafica-z14.md).

<br />
<a href="./code_examples/"><p style="font-size: 30px; font-weight: 600;">🛠 Code_examples</p></a>
<br />

<p style="font-size: 30px; font-weight: 600;">📥 How to Run</p>


You can execute `.zum` code using the compiled ZUMBRA binary on different platforms.

<br />
<p style="font-size: 25px">🖥️ Windows</p>

1. Download the `zumbra.exe` file for Windows from the [GitHub Release](https://github.com/JoseLucasapp/Zumbra-lang/releases).
2. Open a command prompt or PowerShell window and run the following command:

   ```bash
   zumbra.exe myscript.zum
   ```

<br />
<p style="font-size: 25px">🐧Linux</p>

1. Download the `zumbra-linux` file for Linux from the [GitHub Release](https://github.com/JoseLucasapp/Zumbra-lang/releases).
2. Make the file executable by running:

   ```bash
   chmod +x zumbra-linux
   ```

3. Run the script with the following command in your terminal:

   ```bash
   ./zumbra-linux myscript.zum
   ```

<br />

<p style="font-size: 25px">🔎 Validation and IR:</p>

```bash
zumbra check app.zum
zumbra modules app.zum
zumbra ir app.zum hir
zumbra ir app.zum optimized
```

<p style="font-size: 25px">🚀 Native build:</p>

```bash
zumbra build app.zum
zumbra build --release app.zum
zumbra build --emit-c app.zum
./build/app
```

Native builds require Clang or GCC. Use `--compiler`, `CC`, and `-o` to control the toolchain and output. Z8 also supports `--link`, `--include`, `--library-dir`, and `-l` for native dependencies.

<p style="font-size: 25px">🧩 Modules and C bindings:</p>

```bash
zumbra modules app.zum
zumbra bind-c --pub --link ../native/library.c -o bindings/library.zum native/library.h
zumbra build --release app.zum
```


<p style="font-size: 25px">🧠 Contextual callbacks:</p>

```zumbra
extern "C" {
    fct apply(value: i32, cb: callback(i32) -> i32) -> i32;
}

var double << fct(value) { value * 2i32; };
unsafe { show(apply(21i32, double)); }
```

The callback is inferred as `fct(i32) -> i32` and remains concrete through HIR, MIR, and native code generation.

<p style="font-size: 25px">🧵 Tasks and channels:</p>

```zumbra
fct calculate(value) { value * value; }

var task << spawn calculate(6);
show(await task);

var messages << channel(4);
send(messages, "ready");
show(receive(messages));
closeChannel(messages);
```

Native builds use pthreads and C11 atomics. Ordinary arrays, dictionaries, structs, and buffers stay lightweight; protect shared mutation with channels, mutexes, or atomics.


<p style="font-size: 25px">🔎 Contextual call inference:</p>

```zumbra
fct square(value) { value * value; }
var task << spawn square(6);
```

The first concrete call specializes `square` as `fct(int) -> int`, so the task becomes `Task<int>`. The same propagation applies to methods, channels, atomics, and ordinary calls. A function remains monomorphic: reusing it later with an incompatible argument type is rejected.

<p style="font-size: 25px">⚡ Additional Notes:</p>

- Make sure the ZUMBRA binary file (`zumbra-linux`, `zumbra.exe`) has execution permissions on your system.

- For **Windows**, just run the command without additional configuration.
- For **Linux**, use the `chmod` command to make the file executable.

<br />
<p style="font-size: 30px; font-weight: 600;">📎 VS Code Integration</p>

- We provide a [ZUMBRA VS Code Extension](https://marketplace.visualstudio.com/items/?itemName=joselucasapp.zumbra-lang-support) for syntax highlighting and `.zum` file support.

- > Includes keyword coloring, strings, and built-in function recognition.

<br />
<p style="font-size: 30px; font-weight: 600;">🤝 Contributing</p>
<p style="font-size: 16px;">We welcome contributions! Please read our <a href="./CONTRIBUTING">Contributing Guide</a> and <a href="./CODE_OF_CONDUCT">Code of Conduct</a> before submitting pull requests.</p>

<br />
<p style="font-size: 30px; font-weight: 600;">👨‍💻 Creator</p>
<p style="font-size: 25px">Joselucasapp</p>
<p style="font-size: 17px;">Bringing ZUMBRA to life with love, logic, and a hint of rebellion.</p>


## Z10 network quick start

```zumbra
import "std/net.zum" as net;

var listener << net.listen("127.0.0.1", 8080);
var connection << net.accept(listener);
var data << net.read(connection, 4096);
net.writeBytes(connection, data);
net.closeStream(connection);
```

The repository examples use a relative import (`../../std/net.zum`) because package-name imports are reserved for the future package manager. TCP, UDP, DNS and TLS are available in the evaluator, VM and native backend. Native builds compile network support only when the MIR references a network builtin, and OpenSSL is linked only for TLS programs.

## Z11 HTTP and WebSocket quick start

```zumbra
import "std/http.zum" as http;

var api << http.app();
api.get("/health", fct(request, response) {
    request;
    response;
    http.json(200, {"status": "ok"});
});

var server << api.listen("127.0.0.1", 8080);
```

Z11 provides typed HTTP applications, requests and responses; routing, middleware, CORS, cookies, stateless JWT sessions, JSON, multipart uploads, static files, gzip, chunked streams, SSE, HTTP clients, graceful shutdown, WebSockets and WSS. It runs in the evaluator, VM and MIR-based native backend. HTTP reuses Z10 streams and TLS, and native dependencies are enabled only when referenced by the program. See [`docs/pt-BR/http-websockets-e-apis.md`](./docs/pt-BR/http-websockets-e-apis.md).

Z11.1 restores the shared Go/Zumbra value conversion layer used by PostgreSQL, Supabase and platform integrations. Nested JSON values, database driver values, byte buffers and fixed-width integers now convert recursively without depending on private HTTP helpers. See [`docs/pt-BR/correcao-conversores-plataforma-z11-1.md`](./docs/pt-BR/correcao-conversores-plataforma-z11-1.md).


## Z12 data and persistence quick start

```zumbra
import "std/database.zum" as database;

var db << database.memory();
database.migrate(db, [
    {"version": 1, "name": "users", "sql": "CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT NOT NULL)"}
]);
database.exec(db, "INSERT INTO users(name) VALUES (:name)", {"name": "Lucas"});
show(database.queryOne(db, "SELECT name FROM users", {})["name"]);
database.close(db);
```

Z12 adds advanced SQLite, PostgreSQL, Redis, persistent sessions, typed configuration, structured observability, rate limiting, atomic JSON files and versioned binary serialization. Native dependencies are enabled and linked only when referenced. See [`docs/pt-BR/dados-e-persistencia-z12.md`](./docs/pt-BR/dados-e-persistencia-z12.md).

## Z13 desktop runtime quick start

```zumbra
import "std/desktop.zum" as desktop;

var app << desktop.app({
    "name": "Zumbra Desktop",
    "identifier": "dev.zumbra.desktop"
});
var window << app.window({
    "title": "Hello",
    "width": 900,
    "height": 600,
    "resizable": true,
    "highDPI": true
});
app.shortcut("Ctrl+Q", fct(event) { event; app.quit(); });
app.run();
app.close();
```

Z13 provides a Linux-first SDL3 backend and a deterministic headless backend for CI. It includes lifecycle, multiple windows, event dispatch, High DPI, clipboard, drop events, file/folder pickers, notifications, tray, shortcuts, XDG paths, external opening, and process management. See [`docs/pt-BR/runtime-desktop-z13.md`](./docs/pt-BR/runtime-desktop-z13.md).

Zumbra 0.7.0 completes Z13 and establishes the baseline for Z14 GUI toolkit development. See [`docs/releases/0.7.0.md`](./docs/releases/0.7.0.md).

## Z14 GUI toolkit quick start

```zumbra
import "std/desktop.zum" as desktop;
import "std/ui.zum" as ui;

var app << desktop.app({"name": "Zumbra GUI"});
var window << app.window({"title": "Hello", "width": 900, "height": 650});
var message << ui.state("Ready");
var label << ui.text("");
ui.bind(label, "text", message);

var root << ui.columnWith({"padding": 20, "gap": 12}, [
    ui.textWith("Zumbra GUI Toolkit", {"fontSize": 20}),
    ui.input({"placeholder": "Project name"}),
    ui.button("Save", fct(event) { event; ui.update(message, "Saved"); }),
    label
]);

var context << ui.mount(app, window, root, {"theme": ui.theme("light")});
window.show();
app.run();
context.close();
app.close();
```

Z14 provides a retained component tree, deterministic row/column layout, controls, collections, overlays, canvas drawing, themes, reactive state, bidirectional binding, custom components, focus navigation and accessibility. The headless backend supports deterministic CI tests; the graphical backend renders through dynamically loaded SDL3 on Linux, Windows and macOS. See [`docs/pt-BR/toolkit-interface-grafica-z14.md`](./docs/pt-BR/toolkit-interface-grafica-z14.md).

Zumbra 0.8.1 completes Z14 and establishes the baseline for Z15 desktop distribution. See [`docs/releases/0.8.1.md`](./docs/releases/0.8.1.md).


Zumbra 0.5.1 widens the VM builtin index operand from 8 to 16 bits. This prevents overflow now that Z12.1 raises the builtin table above 255 entries and restores binary I/O parity between the evaluator and VM. See [`docs/pt-BR/correcao-indice-builtins-0.5.1.md`](./docs/pt-BR/correcao-indice-builtins-0.5.1.md).

## Z15 desktop distribution

```bash
zumbra app inspect --manifest zumbra.toml
zumbra app doctor --manifest zumbra.toml
zumbra app build --manifest zumbra.toml
zumbra app package --manifest zumbra.toml
```

Z15 adds one target-aware packaging command for Linux, Windows and macOS. It creates Linux bundles, `.deb`, AppDir/AppImage, Windows portable ZIP/NSIS installers, and macOS `.app`/ZIP bundles. Packages include platform metadata, icons, embedded assets, dependency audits, deterministic timestamps, SHA-256 checksums, update descriptors, optional signatures and optional debug symbols. See [`docs/pt-BR/distribuicao-desktop-z15.md`](./docs/pt-BR/distribuicao-desktop-z15.md).

Zumbra 0.10.4 completes the Z15 implementation and maintenance pass with Linux, Windows and macOS desktop/GUI runtime paths, valid AppStream metadata, strict external-tool precedence, reproducible packages, and repository cleanup. See [`docs/releases/0.10.4.md`](./docs/releases/0.10.4.md).

AppImage tooling is external and is never stored in the repository. Install it in the Zumbra cache with:

```bash
scripts/setup-appimage-tools.sh
```



## Repository hygiene

Generated binaries, native C output, package directories, AppImages, installers, downloaded toolchains, profiles, and coverage files are not source code and must not be committed.

```bash
scripts/clean-generated.sh
scripts/clean-generated.sh --local-tools
scripts/check-repository-hygiene.sh
```

Release packages belong in the release service or CI artifacts, not in the Git tree. The source archive should contain only language sources, tests, documentation, examples, and small public assets. See [`docs/pt-BR/higiene-do-repositorio.md`](./docs/pt-BR/higiene-do-repositorio.md).

## Z16 reference desktop application

The first official product built with Zumbra is maintained in the separate `zumbra-game-collection` repository. It is a local SQLite desktop application for cataloging, searching, sorting, editing, and deleting games from a physical or digital collection.

Zumbra 0.11.0 adds UI target identity (`targetId` and `targetKind`) and native code-generation fixes required by dynamic database-backed component lists. See [`docs/pt-BR/aplicativo-referencia-z16.md`](./docs/pt-BR/aplicativo-referencia-z16.md) and [`docs/releases/0.11.0.md`](./docs/releases/0.11.0.md).

## Zumbra 0.11.1

Zumbra 0.11.1 enables SDL3 text input only while an editable UI control is focused, restoring real keyboard editing for desktop `input` and `textarea` components. See [`docs/releases/0.11.1.md`](./docs/releases/0.11.1.md).

## Zumbra 0.11.2

Zumbra 0.11.2 corrects retained UI layout alignment, adds clipped vertical scrolling for overflowing rows, columns and containers, and hardens native C11 generation for dynamically typed string literals and discarded control-flow results. See [`docs/releases/0.11.2.md`](./docs/releases/0.11.2.md).

## Zumbra 0.11.3

Zumbra 0.11.3 restricts scrollbar rendering to nodes that explicitly enable vertical overflow, eliminating stray scrollbar fragments on labels, cards, buttons and ordinary containers. See [`docs/pt-BR/correcao-scrollbars-0.11.3.md`](./docs/pt-BR/correcao-scrollbars-0.11.3.md).
## Zumbra 0.12.1 — fechamento do Z16

Zumbra 0.12.1 fecha a fundação do primeiro aplicativo desktop oficial. A versão adiciona operações recuperáveis de JSON e CSV, backup/restauração/verificação de integridade SQLite, file picker em modo salvar, notificações não fatais, modais reais com isolamento de foco, acessibilidade e ponteiro, além de paridade nativa para mutação incremental de arrays. Veja [`docs/releases/0.12.1.md`](./docs/releases/0.12.1.md) e [`docs/pt-BR/aplicativo-referencia-z16.md`](./docs/pt-BR/aplicativo-referencia-z16.md).


## Zumbra 0.12.2 — navegação e gráficos

Zumbra 0.12.2 corrige alinhamento e overflow de texto em controles, substitui o `v` textual dos selects por um chevron, adiciona regiões de navegação recolhíveis em qualquer borda e inclui gráficos portáteis de pizza, barras e linhas, além de tabelas estruturadas. Veja [`docs/releases/0.12.2.md`](./docs/releases/0.12.2.md).

## Zumbra 0.12.3 — interação e reflow final do Z16

Zumbra 0.12.3 finaliza a interação visual do toolkit com cursores de sistema, caret visível, foco interno, ícones vetoriais, scrollbar sobreposta e remoção real de componentes invisíveis do fluxo de layout. Isso corrige o clipping dos botões de fechar, preserva margens simétricas e permite que sidebars ocultas liberem integralmente o espaço para o conteúdo principal. Veja [`docs/releases/0.12.3.md`](./docs/releases/0.12.3.md).


## Zumbra 0.12.4 — sidebar e gráficos de linha

Navegações laterais preservam toda a altura disponível. Gráficos de linha mostram valores e rótulos categóricos nos backends SDL3 Go/cgo e C11. Veja [`docs/releases/0.12.4.md`](./docs/releases/0.12.4.md).

## Zumbra 0.12.5 — edição de texto e scrollbars seguras

Inputs e textareas agora oferecem edição completa com caret UTF-8, seleção por teclado e mouse, navegação por setas/Home/End, movimentos por palavra, copiar, recortar, colar e seleção total. Scrollbars podem reservar um gutter interno sem atravessar controles, e imagens com `fit: "cover"` usam recorte proporcional. A auditoria de mensagens padrão também removeu o último diagnóstico hardcoded em português. Veja [`docs/releases/0.12.5.md`](./docs/releases/0.12.5.md).


## Zumbra 0.12.6 — scrollbars respeitam camadas de modal

O renderer SDL3 separa a camada normal da camada modal antes de desenhar scrollbars. Assim, barras da sidebar e de listas permanecem no fundo, são desfocadas com o restante do aplicativo e nunca atravessam diálogos. Veja [`docs/releases/0.12.6.md`](./docs/releases/0.12.6.md).
