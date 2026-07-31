# Correção de scrollbars — Zumbra 0.11.3

A versão 0.11.3 corrige uma regressão visual introduzida junto com a rolagem vertical do toolkit.

## Problema

O renderer desenhava uma barra sempre que `contentHeight` fosse maior que a altura interna do componente. Isso também atingia labels, botões, cards e containers comuns quando a medição de texto excedia a área por poucos pixels.

## Correção

A barra agora só é desenhada quando o componente habilita rolagem vertical por uma destas propriedades:

- `overflowY: "auto"`;
- `overflowY: "scroll"`;
- `scrollY: true`.

Também é aplicada uma tolerância de meio pixel para evitar artefatos causados por arredondamento. A regra vale para o backend SDL3 Go/cgo e para o runtime nativo C11.
