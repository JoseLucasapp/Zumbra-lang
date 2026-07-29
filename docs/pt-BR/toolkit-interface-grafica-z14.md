# Z14 — Toolkit de Interface Gráfica

A versão 0.8.1 adiciona ao Zumbra um toolkit de interface gráfica retido e portátil, construído sobre o runtime desktop do Z13. O código Zumbra trabalha com componentes, estado, layout e eventos; SDL3 permanece como detalhe do backend gráfico Linux.

## Objetivos do marco

O Z14 fornece:

- layouts em linhas, colunas e containers;
- texto, botões, inputs, textareas e selects;
- checkboxes e radios;
- tabelas, listas, árvores, tabs e menus;
- modais, tooltips e barras de progresso;
- imagens BMP e canvas 2D;
- temas light, dark e customizados;
- estado reativo e data binding bidirecional;
- componentes personalizados;
- foco e navegação por teclado;
- árvore de acessibilidade;
- backend headless determinístico para testes;
- renderização SDL3 na VM, evaluator e executáveis C11 nativos.

## Primeiro aplicativo

```zumbra
import "../../std/desktop.zum" as desktop;
import "../../std/ui.zum" as ui;

var app << desktop.app({
    "name": "Minha aplicação",
    "version": "0.8.1",
    "identifier": "dev.zumbra.example"
});

var window << app.window({
    "title": "Zumbra GUI",
    "width": 900,
    "height": 650,
    "resizable": true,
    "highDPI": true
});

var message << ui.state("Ready");
var messageText << ui.text("");
ui.bind(messageText, "text", message);

var root << ui.columnWith({"padding": 20, "gap": 12}, [
    ui.textWith("Zumbra GUI Toolkit", {"fontSize": 20}),
    ui.input({"placeholder": "Project name", "grow": 1}),
    ui.button("Save", fct(event) {
        event;
        ui.update(message, "Saved");
    }),
    messageText
]);

var context << ui.mount(app, window, root, {
    "theme": ui.theme("light")
});

window.show();
app.run();
context.close();
app.close();
```

## Modelo retido

Cada componente é um `UINode`. A árvore permanece em memória e só é reconstruída quando o programa altera suas propriedades ou filhos. O layout gera um frame lógico independente do renderer.

Objetos centrais:

```text
UINode
UIState
UITheme
UIContext
```

`UIContext` conecta uma árvore de componentes a uma janela do Z13.

## Layout

Containers aceitam:

```text
direction
width / height
minWidth / minHeight
maxWidth / maxHeight
grow
gap
padding / paddingLeft / paddingTop / paddingRight / paddingBottom
overflowY / scrollY
scrollStep / scrollbarWidth / scrollbarGutter
scrollbarTrack / scrollbarThumb
margin / marginLeft / marginTop / marginRight / marginBottom
align: start | center | end | stretch
justify: start | center | end | space-between
```

Atalhos no módulo `ui`:

```zumbra
ui.row([...]);
ui.rowWith({"gap": 8}, [...]);
ui.column([...]);
ui.columnWith({"padding": 12}, [...]);
ui.container([...]);
ui.containerWith({"border": true}, [...]);
```

## Componentes

```text
ui.text / ui.textWith
ui.button / ui.buttonWith
ui.input
ui.textarea
ui.select
ui.checkbox
ui.radio
ui.table
ui.list
ui.tree
ui.tabs
ui.menu
ui.modal
ui.tooltip
ui.progress
ui.image
ui.canvas
ui.spacer
ui.custom
```

Todos aceitam um dicionário de propriedades. Componentes compostos também recebem filhos.

## Estado e data binding

```zumbra
var name << ui.state("Lucas");
var field << ui.input({"value": ""});
ui.bind(field, "value", name);

ui.update(name, "José Lucas");
show(ui.value(name));
```

A ligação é bidirecional para propriedades editáveis. Alterações do estado atualizam componentes vinculados; entrada de texto, checkbox, radio, select e tabs atualizam o estado associado.

Assinaturas reativas:

```zumbra
name.subscribe(fct(value) {
    show(value);
});
```

## Eventos

Propriedades comuns:

```text
onClick
onChange
onInput
onFocus
onHover
```

O toolkit recebe eventos normalizados do Z13. Mouse, teclado, texto e resize usam o mesmo caminho na VM e no backend nativo.

## Foco e teclado

```zumbra
context.focus(node);
context.focusNext(false);
context.focusNext(true);
```

`Tab` avança o foco e `Shift+Tab` retorna. Botões, inputs, textareas, selects, checkboxes, radios, tabs e menus são focalizáveis por padrão. `Enter` e `Space` ativam controles; `Backspace` edita inputs e textareas.

## Temas

```zumbra
ui.theme("light");
ui.theme("dark");
ui.theme({
    "background": "#10131a",
    "surface": "#202633",
    "text": "#ffffff",
    "primary": "#3867e8"
});
```

O tema pode ser trocado durante a execução:

```zumbra
context.setTheme(ui.theme("dark"));
```

## Canvas

```zumbra
var drawing << ui.canvas({"height": 120});
ui.draw(drawing, "fillRect", {
    "x": 10,
    "y": 10,
    "width": 80,
    "height": 40,
    "color": "#3867e8"
});
ui.draw(drawing, "line", {"x": 10, "y": 70, "x2": 200, "y2": 70});
ui.draw(drawing, "text", {"x": 110, "y": 24, "text": "Canvas"});
```

Comandos suportados:

```text
fillRect
rect
line
text
image
```

Imagens usam BMP no backend SDL3 atual.

## Componentes personalizados

```zumbra
fct statusCard(title, value) {
    ui.custom("status-card", {"border": true, "padding": 12}, [
        ui.text(title),
        ui.text(value)
    ]);
}
```

Componentes personalizados reutilizam a mesma árvore, layout, tema, eventos e acessibilidade.

## Acessibilidade

```zumbra
var tree << context.accessibility();
```

Cada entrada contém:

```text
id
role
label
enabled
focusable
```

Use `accessibilityLabel` e `role` para fornecer semântica explícita.

## Backend headless

```zumbra
var app << desktop.app({"backend": "headless"});
```

O backend headless executa layout, eventos, callbacks, estado e acessibilidade sem abrir janela. Ele é usado pela suíte de conformidade e pelo race detector.

```zumbra
var snapshot << context.snapshot();
show(snapshot["width"]);
show(snapshot["items"]);
```

## Backend gráfico Linux

O backend gráfico carrega SDL3 dinamicamente. Programas GUI não precisam importar SDL diretamente. A renderização usa coordenadas lógicas e respeita janelas High DPI.

```bash
go run . run code_examples/core/gui_window.zum
```

Build nativo:

```bash
go run . build --release --compiler clang \
  -o build/gui-window code_examples/core/gui_window.zum
./build/gui-window
```

## Validação

```bash
go test ./...
scripts/test-gui.sh
go run . version
```

Versão esperada:

```text
0.8.1
```

A ABI nativa do Z14 é a versão 6.

## Hardening 0.8.1: fontes, UTF-8 e temas dinâmicos

A versão 0.8.1 substitui a fonte bitmap de depuração por renderização de texto via SDL3_ttf quando a biblioteca está disponível. O toolkit mede o texto com a mesma fonte usada para renderizar, preserva strings UTF-8 e mantém um cache de fontes por caminho, tamanho e estilo.

Propriedades de texto suportadas:

```text
fontFamily
fontPath
fontSize
fontWeight
fontStyle
lineHeight
wrapWidth
```

Exemplo:

```zumbra
ui.textWith("Ação, café e São João", {
    "fontFamily": "sans",
    "fontSize": 20,
    "fontWeight": "bold",
    "lineHeight": 1.3
});
```

`fontPath` aceita um arquivo de fonte do sistema. A variável `ZUMBRA_UI_FONT` pode definir a fonte padrão sem alterar o programa:

```bash
ZUMBRA_UI_FONT=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf \
  go run . run code_examples/core/gui_window.zum
```

Na ausência de SDL3_ttf ou de uma fonte válida, o backend mantém um fallback de depuração para que o aplicativo continue executando. O backend headless usa medição determinística baseada em pontos de código Unicode.

Temas podem ser trocados durante a execução:

```zumbra
var darkMode << ui.boolState(false);
var toggle << ui.checkbox({"text": "Dark mode", "checked": false});
ui.bind(toggle, "checked", darkMode);

var context << ui.mount(app, window, root, {"theme": ui.theme("light")});

ui.subscribe(darkMode, fct(enabled) {
    if (enabled) {
        context.setTheme(ui.theme("dark"));
    } else {
        context.setTheme(ui.theme("light"));
    }
});
```

O exemplo `gui_window.zum` usa essa ligação e serve como validação visual do modo claro e escuro.
## Rolagem vertical interna

`row`, `column` e `container` não possuem padding implícito. Declare `padding` quando o conteúdo precisar de margem interna.

Para limitar uma lista ao espaço disponível e manter a rolagem dentro do componente, use um container vertical com overflow:

```zum
var list << ui.columnWith({
    "grow": 1,
    "gap": 8,
    "overflowY": "auto",
    "scrollStep": 88,
    "scrollbarWidth": 8,
    "scrollbarGutter": 4
}, items);
```

Valores aceitos para `overflowY` são `"auto"` e `"scroll"`. A forma booleana equivalente é `scrollY: true`. A roda do mouse procura o ancestral rolável sob o ponteiro, limita o deslocamento ao conteúdo e recorta os descendentes ao viewport.

O snapshot headless inclui `contentX`, `contentY`, `contentWidth`, `contentHeight` e `scrollOffsetY`, permitindo testar overflow sem abrir uma janela gráfica.

