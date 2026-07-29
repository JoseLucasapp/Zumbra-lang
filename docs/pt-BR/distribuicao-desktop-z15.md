# Z15 — Distribuição desktop multiplataforma

O Z15 transforma um projeto Zumbra em executáveis e pacotes para Linux, Windows e macOS. O fluxo público permanece pequeno: um `zumbra.toml`, um entrypoint e os comandos `app doctor`, `app build` e `app package`.

## Estrutura mínima

```text
meu-app/
├── zumbra.toml
├── src/
│   └── main.zum
└── assets/
```

## Manifesto

```toml
[app]
name = "Meu Aplicativo"
version = "1.0.0"
identifier = "dev.zumbra.meuapp"
entry = "src/main.zum"
icon = "assets/icon.png"
icon_linux = "assets/icon.png"
icon_windows = "assets/icon.ico"
icon_macos = "assets/icon.icns"

[build]
output = "dist/meu-app"
compiler = "auto"
release = true

[package]
description = "Aplicativo desktop escrito em Zumbra."
publisher = "Minha Empresa"
homepage = "https://seu-dominio.example"
license = "Apache-2.0"
category = "Utility"

[linux]
dependencies = ["libc6"]
recommends = ["libsdl3-0", "libsdl3-ttf0"]
# runtime_files = ["runtime/libSDL3.so.0", "runtime/libSDL3_ttf.so.0"]

[windows]
console = false
installer = "nsis"
# runtime_files = ["runtime/SDL3.dll", "runtime/SDL3_ttf.dll"]

[macos]
minimum_version = "12.0"
category = "public.app-category.utilities"
# runtime_files = ["runtime/libSDL3.dylib", "runtime/libSDL3_ttf.dylib"]

[updates]
# url = "https://seu-dominio.example/releases"
channel = "stable"

[assets]
include = ["assets/**"]
exclude = ["**/*.tmp"]
```

`icon` é o fallback. Os ícones específicos substituem esse valor apenas no alvo correspondente.

`package.homepage` é obrigatório quando o formato AppImage é solicitado. O valor é incorporado à metadata AppStream e evita que a validação oficial do `appimagetool` rejeite o pacote por falta de homepage.

`runtime_files` inclui bibliotecas dinâmicas não pertencentes ao sistema operacional. Os arquivos precisam estar dentro do projeto, não podem ser symlinks e não podem gerar nomes de destino duplicados.

## Comandos

```bash
zumbra app inspect --manifest zumbra.toml
zumbra app run --manifest zumbra.toml
zumbra app doctor --manifest zumbra.toml
zumbra app build --manifest zumbra.toml
zumbra app package --manifest zumbra.toml
```

O alvo padrão é o sistema atual:

```bash
zumbra app package --target linux --arch amd64
zumbra app package --target windows --arch amd64
zumbra app package --target macos --arch arm64
```

## Doctor

O `app doctor` verifica antes do build:

- manifesto e entrypoint;
- suporte do runtime desktop no alvo;
- compilador e resource compiler;
- formato e arquitetura de um binário fornecido;
- `appimagetool`, runtime AppImage, `makensis` e ferramentas de assinatura quando necessários.

```bash
zumbra app doctor \
  --manifest zumbra.toml \
  --target linux \
  --arch amd64 \
  --format all
```

A saída JSON pode ser usada em CI:

```bash
zumbra app doctor --manifest zumbra.toml --target windows --format all --json
```

Falhas operacionais não imprimem o manual inteiro. O bloco global de uso é reservado a comandos ou opções inválidas.

## Linux

```bash
zumbra app package --target linux --format bundle
zumbra app package --target linux --format deb
zumbra app package --target linux --format appdir
zumbra app package --target linux --format appimage
zumbra app package --target linux --format all
```

Artefatos:

- bundle `.tar.gz`;
- pacote `.deb` determinístico;
- AppDir;
- AppImage real.

O AppDir inclui arquivo `.desktop`, ícone e metadata AppStream completa em `usr/share/metainfo/<identificador>.appdata.xml`. A descrição AppStream é expandida para um parágrafo informativo quando a descrição curta do manifesto não é suficiente para o validador.

Quando `app package` precisa compilar o executável automaticamente, o código C intermediário e o binário de trabalho são criados em um diretório temporário. Apenas os artefatos finais permanecem no diretório de saída.

### Descoberta do appimagetool

O Zumbra pesquisa, na ordem:

1. `--appimagetool`;
2. `APPIMAGETOOL`;
3. `tools/` ao lado do manifesto;
4. `tools/` no diretório atual;
5. cache do usuário;
6. `PATH`.

Instalação leve no cache do usuário:

```bash
scripts/setup-appimage-tools.sh
```

Exemplo explícito:

```bash
zumbra app package \
  --target linux \
  --format all \
  --appimagetool "$HOME/.cache/zumbra/tools/appimagetool-x86_64.AppImage"
```

Um caminho fornecido por `--appimagetool` ou `APPIMAGETOOL` é autoritativo. Se estiver incorreto, o comando falha em vez de usar silenciosamente outra ferramenta encontrada.

### Runtime AppImage fixado

Para não depender de download durante o empacotamento:

```bash
zumbra app package \
  --target linux \
  --format appimage \
  --appimage-runtime "$HOME/.cache/zumbra/tools/runtime-x86_64"
```

Também são aceitos `APPIMAGE_RUNTIME`, uma ferramenta local em `tools/runtime-x86_64` e o cache do Zumbra. Quando o primeiro build online gera um AppImage válido, o runtime é extraído e armazenado para os próximos builds.

## Windows

```bash
zumbra app package --target windows --format portable
zumbra app package --target windows --format installer
zumbra app package --target windows --format all
```

Artefatos:

- ZIP portátil;
- script NSIS auditável;
- instalador `.exe`.

O backend desktop usa SDL3 carregado dinamicamente e integra diálogos, processos, notificações, abertura externa e paths com APIs do Windows. DLLs declaradas em `windows.runtime_files` são copiadas para a raiz portátil e para o instalador.

Um binário já compilado também pode ser empacotado:

```bash
zumbra app package \
  --manifest zumbra.toml \
  --target windows \
  --arch amd64 \
  --binary dist/meu-app.exe \
  --format all
```

## macOS

```bash
zumbra app package --target macos --format app
zumbra app package --target macos --format zip
zumbra app package --target macos --format all
```

Artefatos:

- bundle `.app` com `Info.plist`;
- ZIP preservando estrutura e permissões.

O backend desktop usa SDL3 carregado dinamicamente e integra seletores, notificações, paths, processos e abertura externa com recursos do macOS. Bibliotecas declaradas em `macos.runtime_files` são copiadas para `Contents/Frameworks`.

## Runtime desktop e GUI

O backend nativo do Z13/Z14 possui caminhos para:

- Linux: SDL3/SDL3_ttf dinâmicos e integrações POSIX;
- Windows: SDL3/SDL3_ttf dinâmicos e APIs Win32;
- macOS: SDL3/SDL3_ttf dinâmicos e integrações nativas por sistema.

O modo `headless` continua disponível nos três alvos para testes determinísticos. O exemplo finito `code_examples/core/desktop_gui_smoke.zum` valida inicialização SDL3, janela oculta, tema, UTF-8 e árvore de acessibilidade sem manter o processo aberto.

## Cross-build

Variáveis de compilador:

```text
ZUMBRA_CC_LINUX_AMD64
ZUMBRA_CC_LINUX_ARM64
ZUMBRA_CC_WINDOWS_AMD64
ZUMBRA_CC_WINDOWS_ARM64
ZUMBRA_CC_MACOS_AMD64
ZUMBRA_CC_MACOS_ARM64
ZUMBRA_WINDRES_WINDOWS_AMD64
ZUMBRA_WINDRES_WINDOWS_ARM64
```

Configuração adicional por alvo:

```text
ZUMBRA_SYSROOT_<ALVO>
ZUMBRA_CFLAGS_<ALVO>
ZUMBRA_LDFLAGS_<ALVO>
```

Exemplo:

```text
ZUMBRA_SYSROOT_MACOS_ARM64
ZUMBRA_CFLAGS_WINDOWS_AMD64
ZUMBRA_LDFLAGS_LINUX_ARM64
```

O Zumbra valida cabeçalhos ELF, PE e Mach-O, além da arquitetura `amd64` ou `arm64`, antes de criar o pacote.

## Reprodutibilidade

```bash
SOURCE_DATE_EPOCH=1700000000 zumbra app package --manifest zumbra.toml
```

A entrega usa ordenação, timestamps e proprietários normalizados. Cada execução gera:

- `SHA256SUMS.txt`;
- relatório JSON;
- build ID;
- descritor de atualização, quando configurado;
- auditoria de dependências, quando a ferramenta do alvo está disponível.

No AppImage, o runtime fixado ou cacheado evita que um download variável faça parte do caminho normal de reprodução.

## Assinatura e símbolos

```bash
zumbra app package --sign IDENTIDADE
zumbra app package --symbols
```

Backends de assinatura:

- Linux: GPG;
- Windows: `signtool`;
- macOS: `codesign`.

Símbolos separados usam `objcopy`, `llvm-objcopy` ou `dsymutil`.

## `--format all`

`all` é estrito. Todos os formatos do alvo precisam ser produzidos. Para uma entrega parcial, liste os formatos explicitamente:

```bash
zumbra app package --target linux --format bundle,deb,appdir
zumbra app package --target windows --format portable
```

## Matriz de aceitação

O workflow `.github/workflows/z15-multiplatform.yml` executa:

- Linux amd64: suíte completa e distribuição;
- Windows amd64: build PE, backend desktop/GUI, ZIP portátil e NSIS;
- macOS arm64: build Mach-O, backend desktop/GUI, `.app` e ZIP.

O workflow usa os mesmos exemplos e comandos públicos, sem um empacotador paralelo exclusivo de CI.

## Checklist final

O checklist fechado está em `docs/pt-BR/checklist-z15.md`.
