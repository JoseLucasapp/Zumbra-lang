# Z15.1 — manifesto e assets incorporados

O Z15.1 inicia a distribuição desktop do Zumbra com um formato de projeto verificável e assets incorporados ao executável nativo. Esse incremento estabeleceu a base posteriormente completada pelo Z15 em `docs/pt-BR/distribuicao-desktop-z15.md`.

## Estrutura de projeto

```text
meu-app/
├── zumbra.toml
├── src/
│   └── main.zum
└── assets/
    └── message.txt
```

## Manifesto

```toml
[app]
name = "Meu Aplicativo"
version = "1.0.0"
identifier = "dev.exemplo.meuapp"
entry = "src/main.zum"
icon = "assets/icon.bmp"

[build]
output = "dist/meu-aplicativo"
compiler = "auto"
release = true

[assets]
include = ["assets/**"]
exclude = ["**/*.tmp"]
max_file_bytes = 16_777_216
max_total_bytes = 67_108_864
```

O parser aceita deliberadamente um subconjunto estável de TOML: seções, strings, booleanos, inteiros e arrays de strings. Campos desconhecidos são rejeitados para evitar configurações ignoradas silenciosamente.

## Comandos

```bash
zumbra app inspect --manifest zumbra.toml
zumbra app run --manifest zumbra.toml
zumbra app build --manifest zumbra.toml
```

`app inspect` valida o manifesto e lista os assets com tamanho e SHA-256. `app run` executa a entrada na VM com a mesma tabela de assets que será incorporada. `app build` gera o executável nativo e um arquivo lateral `<executável>.manifest.json` para auditoria.

## API de assets

```zumbra
import "../../std/assets.zum" as assets;

show(assets.exists("assets/message.txt"));
show(assets.text("assets/message.txt"));
var raw << assets.bytes("assets/image.bmp");
show(assets.list());
```

Builtins correspondentes:

```text
assetExists
assetText
assetBytes
assetList
```

O executável também incorpora `__zumbra__/manifest.json`, com o nome, versão, identificador, entrada e inventário dos assets.

## Segurança e determinismo

- caminhos não podem escapar da raiz do projeto;
- symlinks não são empacotados;
- assets são ordenados pelo nome lógico;
- limites por arquivo e por bundle são obrigatórios;
- assets ausentes geram erro explícito;
- `assetText` exige UTF-8 válido;
- buffers retornados são cópias mutáveis, sem alterar a imagem incorporada;
- `build/`, `dist/` e `.git/` não entram na varredura.

## Escopo restante do Z15

- Z15.2: `.deb`, AppImage e layout Linux instalável;
- Z15.3: executável e instalador Windows;
- Z15.4: builds reproduzíveis, checksums de distribuição e hardening.
