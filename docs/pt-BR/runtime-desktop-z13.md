# Z13 — Runtime Desktop

A versão 0.7.0 introduz o runtime desktop generalista do Zumbra. O primeiro alvo oficial é Linux, especialmente Debian. A API pública pertence ao módulo `std/desktop.zum`; SDL3 é um detalhe de implementação do backend gráfico, não uma dependência conceitual dos programas Zumbra.

## Arquitetura

O runtime possui dois backends:

- `sdl3`: cria janelas reais e recebe eventos do sistema no Linux;
- `headless`: backend determinístico para CI, servidores sem interface gráfica e testes automatizados.

O backend SDL3 é carregado dinamicamente. Programas headless não exigem uma sessão gráfica nem a biblioteca SDL3 instalada. Um aplicativo gráfico deve ser criado e executado na thread principal.

## Aplicação e lifecycle

```zumbra
import "../../std/desktop.zum" as desktop;

var app << desktop.app({
    "name": "Meu aplicativo",
    "identifier": "dev.exemplo.app",
    "backend": "auto",
    "quitOnLastWindow": true,
    "closeOnRequest": true
});

var window << app.window({
    "title": "Zumbra Desktop",
    "width": 900,
    "height": 600,
    "resizable": true,
    "highDPI": true
});

app.on("quit", fct(event) {
    event;
    app.quit();
});

app.run();
app.close();
```

`run()` processa o event loop até `quit()`, fechamento da última janela ou evento de término. `close()` fecha trays, janelas e o backend de maneira idempotente.

## Janelas e múltiplas janelas

```zumbra
var primary << app.window({"title": "Principal", "width": 900, "height": 600});
var inspector << app.window({"title": "Inspector", "width": 480, "height": 600});

primary.setSize(1280, 720);
primary.setPosition(100, 80);
primary.setFullscreen(true);
primary.setFullscreen(false);
primary.maximize();
primary.restore();
primary.focus();

var logical << primary.size();
var pixels << primary.pixelSize();
show(primary.displayScale());
show(primary.pixelDensity());
```

Métodos disponíveis:

```text
show hide close isOpen id
title setTitle
size pixelSize setSize
position setPosition
fullscreen setFullscreen
maximize minimize restore focus
displayScale pixelDensity
setIcon
```

O ícone gráfico usa atualmente arquivos BMP no backend SDL3. Outros formatos poderão ser incorporados no toolkit e no empacotamento desktop.

## Eventos

Handlers recebem um dicionário:

```zumbra
app.on("resized", fct(event) {
    show(event["windowId"]);
    show(event["width"]);
    show(event["height"]);
});
```

Campos comuns:

```text
type
windowId
timestamp
data
```

Os dados mais usados também são expostos diretamente no evento. Eventos suportados incluem lifecycle da aplicação, janela, teclado, texto, mouse, scroll, arquivos arrastados e itens de tray.

Eventos personalizados permitem integrar subsistemas internos:

```zumbra
app.on("sync_finished", fct(event) {
    show(event["records"]);
});

app.emit({
    "type": "sync_finished",
    "data": {"records": 42}
});
```

## Atalhos de teclado

```zumbra
app.shortcut("Ctrl+Shift+S", fct(event) {
    event;
    show("save as");
});
```

Aliases como `Control`, `Command`, `Cmd`, `Meta`, `Option`, `Return` e `Esc` são normalizados. Modificadores seguem uma forma canônica como `ctrl+shift+s`.

## Clipboard

```zumbra
app.setClipboard("texto copiado");
show(app.clipboard());
```

## Drag and drop

O backend gráfico traduz eventos de drop para eventos Zumbra. O caminho recebido fica em `event["path"]` ou nos dados do evento, conforme o tipo emitido pelo sistema.

```zumbra
app.on("drop_file", fct(event) {
    show(event["path"]);
});
```

## File picker e folder picker

```zumbra
var files << app.pickFile({
    "title": "Escolha arquivos",
    "defaultPath": "/home/user",
    "multiple": true
});

var folder << app.pickFolder({
    "title": "Escolha uma pasta",
    "defaultPath": "/home/user"
});
```

No Linux, a primeira implementação usa `zenity` e recorre a `kdialog`. Cancelamento retorna array vazio para arquivos e `null` para pasta. O backend headless usa `defaultPath` ou variáveis de ambiente de teste.

## Notificações

```zumbra
app.notify({
    "title": "Zumbra",
    "body": "Processamento concluído",
    "urgency": "normal"
});
```

No Linux, notificações utilizam `notify-send`.

## System tray

```zumbra
var tray << app.tray({
    "tooltip": "Zumbra",
    "icon": "assets/icon.bmp"
});

tray.add("open", "Abrir", fct(event) {
    event;
    window.show();
    window.focus();
});

tray.add("quit", "Sair", fct(event) {
    event;
    app.quit();
});
```

O suporte efetivo a tray depende do ambiente desktop e das APIs SDL3 disponíveis.

## Caminhos do sistema

```zumbra
var paths << app.paths();
show(paths["home"]);
show(paths["config"]);
show(paths["data"]);
show(paths["cache"]);
show(paths["runtime"]);
show(paths["executable"]);
show(paths["cwd"]);
show(paths["temp"]);
```

No Linux, os caminhos respeitam as variáveis XDG quando disponíveis.

## Abrir recursos externos

```zumbra
app.openExternal("https://zumbra.dev");
app.openExternal("/tmp/report.pdf");
```

O backend Linux usa `xdg-open`.

## Processos

```zumbra
var process << desktop.startProcess("/bin/sh", {
    "args": ["-c", "exit 7"],
    "cwd": "/tmp",
    "env": {"MODE": "desktop"}
});

show(process.id());
show(process.running());
show(process.wait());
```

Métodos:

```text
wait
kill
running
id
```

Um código de saída diferente de zero é retornado normalmente por `wait()`. Falhas para iniciar, sinalizar ou aguardar o processo são erros do runtime.

## Backend headless

```zumbra
var app << desktop.app({"backend": "headless"});
```

Também pode ser ativado com:

```bash
ZUMBRA_DESKTOP_HEADLESS=1 zumbra run app.zum
```

O headless mantém janelas virtuais, clipboard, filas de eventos, seletores determinísticos, notificações, trays e processos reais. Ele é usado pela suíte automatizada para assegurar paridade entre evaluator, VM, Clang e GCC.

## Exemplo gráfico real

O arquivo `code_examples/core/desktop_window.zum` abre uma janela SDL3 real, registra `Ctrl+Q`, recebe arquivos arrastados e encerra pelo fechamento da janela. Ele pode ser executado com:

```bash
go run . run code_examples/core/desktop_window.zum
```

Também pode ser compilado:

```bash
go run . build --release --compiler clang -o build/desktop-window code_examples/core/desktop_window.zum
./build/desktop-window
```

## Dependências Linux

Para janelas reais:

```bash
sudo apt install libsdl3-0
```

Para desenvolvimento e utilitários de integração:

```bash
sudo apt install libsdl3-dev zenity libnotify-bin xdg-utils
```

Os nomes exatos dos pacotes podem variar de acordo com a versão da distribuição. O backend nativo carrega SDL3 dinamicamente e liga `libdl` somente quando o programa usa APIs desktop.

## Validação

```bash
go test ./...
scripts/test-desktop.sh
go run . version
```

A versão esperada é `0.7.0`.

## Limites do Z13

O Z13 fornece infraestrutura de aplicação e integração com o sistema. Componentes visuais — botões, inputs, tabelas, layouts, estado reativo e acessibilidade de widgets — pertencem ao Z14. Empacotamento AppImage, `.deb`, instalador Windows e assets incorporados pertencem ao Z15.
