# Z14 — Toolkit de Interface Gráfica

A versão 0.8.1 adiciona ao Zumbra um toolkit de interface gráfica retido e portátil, construído sobre o runtime desktop do Z13. O código Zumbra trabalha com componentes, estado, layout e eventos; SDL3 permanece como detalhe do backend gráfico Linux.

## Objetivos do marco

O Z14 fornece:

- layouts em linhas, colunas e containers;
- texto, botões, inputs, textareas e selects;
- checkboxes e radios;
- tabelas, listas, árvores, tabs e menus;
- modais, tooltips e barras de progresso;
- imagens PNG, JPEG e BMP, além de canvas 2D;
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

Imagens usam `fit: "contain"` para preservar a proporção. O backend SDL3 tenta SDL3_image e, no Linux, usa GdkPixbuf como fallback para PNG e JPEG; BMP permanece disponível pelo carregador básico do SDL3.

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

Valores aceitos para `overflowY` são `"auto"` e `"scroll"`. A forma booleana equivalente é `scrollY: true`. A roda do mouse procura o ancestral rolável sob o ponteiro, limita o deslocamento ao conteúdo e recorta os descendentes ao viewport. Para apenas recortar os filhos, sem habilitar rolagem, use `overflow: "hidden"` ou `clipChildren: true`.

O snapshot headless inclui `contentX`, `contentY`, `contentWidth`, `contentHeight` e `scrollOffsetY`, permitindo testar overflow sem abrir uma janela gráfica.

## Selects com dropdown

`ui.select` abre uma lista real sobre o layout, sem alterar o valor apenas por abrir o controle. A opção muda somente após seleção por mouse ou teclado.

```zumbra
var platform << ui.stringState("Todas");
var picker << ui.select({
    "value": "Todas",
    "options": ["Todas", "PlayStation", "Nintendo"],
    "maxVisibleOptions": 6,
    "optionHeight": 36
});
ui.bind(picker, "value", platform);
```

A lista fecha ao escolher uma opção, pressionar `Escape`, avançar com `Tab` ou clicar fora. `ArrowUp` e `ArrowDown` navegam pelas opções. Quando a lista excede `maxVisibleOptions`, a roda do mouse controla `popupOffset`.

## Modais reais

`ui.modal` é um overlay e não participa do fluxo de linhas, colunas ou containers. Quando um modal visível existe:

- foco e `focusNext` permanecem dentro do modal mais alto;
- a interface de fundo não recebe mouse ou teclado;
- somente a árvore modal ativa é exposta pela acessibilidade;
- filhos de modais ocultos não podem receber foco;
- bindings das propriedades `visible` e `enabled` são aplicados imediatamente;
- `width` e `height` representam o card, que é centralizado e limitado ao viewport;
- `backdropBlur` desfoca a interface já renderizada atrás do card;
- alpha blending mantém overlay e sombra translúcidos;
- `overflow: "hidden"` impede que filhos ultrapassem o conteúdo do diálogo.

```zumbra
var visible << ui.boolState(false);
var dialog << ui.modal({
    "id": "confirm",
    "visible": false,
    "width": 480,
    "height": 280,
    "backdropBlur": 6,
    "overflow": "hidden"
}, [
    ui.buttonWith({"id": "confirm-action", "text": "Confirmar"})
]);
ui.bind(dialog, "visible", visible);
```

## Alinhamento e overflow de texto em controles

Botões centralizam o texto horizontal e verticalmente por padrão. `input` e `select` mantêm alinhamento à esquerda. O renderer limita o texto à área interna do controle e usa reticências no backend SDL3 Go quando necessário.

Propriedades disponíveis:

```zum
ui.buttonWith({
    "text": "Ação com nome longo",
    "textAlign": "center",
    "textOverflow": "ellipsis",
    "width": 160
});
```

`textAlign` aceita `left`, `center` e `right`. `textOverflow` aceita `ellipsis` e `visible`. O backend C11 sempre recorta o texto ao limite do controle para impedir vazamento visual.

Os selects desenham um chevron gráfico no lado direito, em vez de usar a letra `v` como indicador.

## Regiões de navegação

`ui.navigation` é um alias semântico de `ui.menu` para barras laterais, headers e barras inferiores. O programador define a posição e os tamanhos expandido e recolhido:

```zum
var collapsed << ui.boolState(false);
var navigation << ui.navigation({
    "placement": "left",
    "expandedSize": 300,
    "collapsedSize": 56,
    "collapsed": false,
    "border": true
}, children);
ui.bind(navigation, "collapsed", collapsed);
```

`placement` aceita:

- `left`;
- `right`;
- `top`;
- `bottom`.

Menus laterais usam layout em coluna. Menus superiores e inferiores usam layout em linha. O pai continua controlando a ordem espacial: em uma `row`, coloque o menu antes do conteúdo para `left` e depois para `right`; em uma `column`, coloque-o antes para `top` e depois para `bottom`.

Filhos com `visible: false` deixam de consumir espaço no layout. Isso permite manter uma árvore expandida e outra compacta dentro da mesma navegação.

## Gráficos e tabelas de dados

`ui.chart` cria um canvas portátil voltado a visualizações. Os helpers disponíveis são:

```zum
var chart << ui.chart({"height": 240});

ui.pieChart(chart, {
    "values": [8, 3],
    "labels": ["Físico", "Digital"],
    "colors": ["#3867e8", "#e89b38"],
    "legend": true
});

ui.barChart(chart, {
    "values": [5, 3, 2],
    "labels": ["PS1", "PC", "NES"]
});

ui.lineChart(chart, {
    "values": [1, 4, 3, 7],
    "color": "#3867e8"
});
```

Use `ui.clearChart(chart)` antes de reconstruir uma visualização dinâmica.

`ui.dataTable` é um alias de `ui.table` para tabelas estruturadas. Cada linha pode ser um array de células:

```zum
var table << ui.dataTable({
    "columns": ["Plataforma", "Jogos"],
    "rows": [["PS1", "5"], ["PC", "3"]],
    "rowHeight": 34
});
```

Pizza, barras, linhas e tabelas possuem paridade entre o renderer SDL3 Go/cgo, o backend C11 e o snapshot headless dos comandos.

## Cursores, caret e foco

Controles interativos podem definir o cursor exibido pelo sistema:

```zum
ui.buttonWith({
    "text": "Salvar",
    "cursor": "pointer"
});

ui.input({
    "placeholder": "Nome do jogo",
    "cursor": "text"
});
```

Valores suportados:

- `default`;
- `pointer` ou `hand`;
- `text`, `ibeam` ou `i-beam`.

Botões, selects, checkboxes e radios usam `pointer` por padrão. Inputs e textareas usam `text`, recebem um fundo de foco derivado do tema e exibem um caret real enquanto estão focados. O anel de foco é desenhado dentro do componente, evitando clipping nas bordas de modais e containers.

## Ícones vetoriais em botões

Botões podem usar ícones independentes da fonte:

```zum
ui.buttonWith({
    "icon": "close",
    "text": "",
    "width": 38,
    "height": 36,
    "accessibilityLabel": "Fechar"
});
```

Ícones disponíveis:

- `close`;
- `menu`;
- `chevron-left`;
- `chevron-right`.

O texto acessível deve ser mantido em `accessibilityLabel` quando o botão não possui texto visível.

## Sidebar fora do fluxo

Para ocultar completamente uma navegação e permitir que o conteúdo ocupe todo o espaço, vincule `visible`, `expandedSize` e o `gap` do layout:

```zum
var visible << ui.boolState(true);
var size << ui.intState(300);
var gap << ui.intState(16);

var sidebar << ui.navigation({
    "placement": "left",
    "expandedSize": 300
}, children);
ui.bind(sidebar, "visible", visible);
ui.bind(sidebar, "expandedSize", size);

var layout << ui.rowWith({"gap": 16, "grow": 1}, [sidebar, content]);
ui.bind(layout, "gap", gap);
```

Ao ocultar, use tamanho e gap iguais a zero antes de definir `visible` como `false`. Nós invisíveis recebem bounds zerados e não participam do cálculo do layout.

## Scrollbar sobreposta

`scrollbarOverlay: true` desenha a barra dentro do viewport sem reduzir `contentWidth`:

```zum
ui.columnWith({
    "overflowY": "auto",
    "scrollbarOverlay": true,
    "scrollbarWidth": 8,
    "scrollbarGutter": 4
}, children);
```

Isso mantém margens e alinhamentos simétricos. Use padding interno nos itens quando não quiser que a barra sobreponha conteúdo clicável.

## Cursores, caret e foco

Controles interativos podem definir o cursor exibido pelo sistema:

```zum
ui.buttonWith({
    "text": "Salvar",
    "cursor": "pointer"
});

ui.input({
    "placeholder": "Nome do jogo",
    "cursor": "text"
});
```

Valores suportados:

- `default`;
- `pointer` ou `hand`;
- `text`, `ibeam` ou `i-beam`.

Botões, selects, checkboxes e radios usam `pointer` por padrão. Inputs e textareas usam `text`, recebem um fundo de foco derivado do tema e exibem um caret real enquanto estão focados. O anel de foco é desenhado dentro do componente, evitando clipping nas bordas de modais e containers.

## Ícones vetoriais em botões

Botões podem usar ícones independentes da fonte:

```zum
ui.buttonWith({
    "icon": "close",
    "text": "",
    "width": 38,
    "height": 36,
    "accessibilityLabel": "Fechar"
});
```

Ícones disponíveis:

- `close`;
- `menu`;
- `chevron-left`;
- `chevron-right`.

O texto acessível deve ser mantido em `accessibilityLabel` quando o botão não possui texto visível.

## Sidebar fora do fluxo

Para ocultar completamente uma navegação e permitir que o conteúdo ocupe todo o espaço, vincule `visible`, `expandedSize` e o `gap` do layout:

```zum
var visible << ui.boolState(true);
var size << ui.intState(300);
var gap << ui.intState(16);

var sidebar << ui.navigation({
    "placement": "left",
    "expandedSize": 300
}, children);
ui.bind(sidebar, "visible", visible);
ui.bind(sidebar, "expandedSize", size);

var layout << ui.rowWith({"gap": 16, "grow": 1}, [sidebar, content]);
ui.bind(layout, "gap", gap);
```

Ao ocultar, use tamanho e gap iguais a zero antes de definir `visible` como `false`. Nós invisíveis recebem bounds zerados e não participam do cálculo do layout.

## Scrollbar sobreposta

`scrollbarOverlay: true` desenha a barra dentro do viewport sem reduzir `contentWidth`:

```zum
ui.columnWith({
    "overflowY": "auto",
    "scrollbarOverlay": true,
    "scrollbarWidth": 8,
    "scrollbarGutter": 4
}, children);
```

Isso mantém margens e alinhamentos simétricos. Use padding interno nos itens quando não quiser que a barra sobreponha conteúdo clicável.


## Correções da versão 0.12.4

Navegações laterais usam toda a altura disponível. Gráficos de linha exibem valores sobre os pontos e rótulos categóricos abaixo deles. Use `showValues: false` para ocultar os valores.


## Edição de texto completa e gutter de scrollbar — 0.12.5

Inputs e textareas mantêm posição do caret e intervalo de seleção em índices UTF-8. A edição funciona de forma equivalente na VM/evaluator e no backend C11.

Controles de teclado suportados:

- `Left` e `Right` movem o caret por caractere;
- `Home` e `End` movem para o início e o fim;
- `Ctrl+Left` e `Ctrl+Right` ou equivalentes com `Super` movem por palavra;
- `Shift` combinado com os movimentos amplia ou reduz a seleção;
- `Backspace` e `Delete` removem a seleção ou o caractere adjacente;
- `Ctrl+A`, `Ctrl+C`, `Ctrl+X` e `Ctrl+V` selecionam, copiam, recortam e colam.

O mouse permite posicionar o caret, arrastar uma seleção e selecionar uma palavra com clique duplo. O texto selecionado e o caret são renderizados de acordo com o tema, e campos longos deslocam horizontalmente o viewport para manter o caret visível.

Para scroll interno sem a barra atravessar cards, inputs ou botões, use um gutter dedicado:

```zum
ui.columnWith({
    "overflowY": "auto",
    "scrollbarOverlay": true,
    "scrollbarAvoidContent": true,
    "scrollbarTrack": "transparent",
    "scrollbarWidth": 8,
    "scrollbarGutter": 4
}, children);
```

`scrollbarAvoidContent: true` mantém a barra sobreposta ao viewport, mas reduz a largura disponível aos filhos pela soma de `scrollbarWidth` e `scrollbarGutter`. Assim a margem visual permanece controlável sem a barra ser desenhada por cima do conteúdo interativo.

Imagens usadas como miniaturas também aceitam recorte proporcional:

```zum
ui.image({
    "path": coverPath,
    "fit": "cover",
    "width": 72,
    "height": 72
});
```

Com `fit: "cover"`, a imagem preserva a proporção, preenche o retângulo e recorta apenas o excesso centralizado.
