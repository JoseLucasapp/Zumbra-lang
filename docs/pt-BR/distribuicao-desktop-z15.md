# Z15 — Distribuição desktop

O Z15 transforma um projeto Zumbra com `zumbra.toml` em artefatos instaláveis. O fluxo continua simples: um manifesto, um comando e formatos específicos por sistema operacional.

## Estrutura mínima

```text
meu-app/
├── zumbra.toml
├── src/main.zum
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
homepage = "https://example.com"
license = "Apache-2.0"
category = "Utility"

[linux]
dependencies = ["libc6"]
recommends = ["libsdl3-0", "libsdl3-ttf0"]

[windows]
console = false
installer = "nsis"

[macos]
minimum_version = "12.0"
category = "public.app-category.utilities"

[updates]
url = "https://example.com/releases"
channel = "stable"

[assets]
include = ["assets/**"]
exclude = ["**/*.tmp"]
```

`icon` é o fallback. Os campos `icon_linux`, `icon_windows` e `icon_macos` substituem o fallback apenas no alvo correspondente.

## Comandos

```bash
zumbra app inspect --manifest zumbra.toml
zumbra app run --manifest zumbra.toml
zumbra app doctor --manifest zumbra.toml
zumbra app build --manifest zumbra.toml
zumbra app package --manifest zumbra.toml
```

O alvo padrão é o sistema atual. Para escolher explicitamente:

```bash
zumbra app package --target linux --arch amd64
zumbra app package --target windows --arch amd64
zumbra app package --target macos --arch arm64
```

## Formatos

### Linux

```bash
zumbra app package --target linux --format bundle
zumbra app package --target linux --format deb
zumbra app package --target linux --format appimage
zumbra app package --target linux --format all
```

Artefatos:

- bundle `.tar.gz` standalone;
- pacote `.deb` determinístico;
- diretório `.AppDir`;
- AppImage, com `appimagetool` obrigatório quando o formato é solicitado.

O `.deb` é criado diretamente pelo Zumbra, sem depender de `dpkg-deb`.

### Windows

```bash
zumbra app package --target windows --format portable
zumbra app package --target windows --format installer
zumbra app package --target windows --format all
```

Artefatos:

- ZIP portátil;
- script NSIS auditável;
- instalador `.exe`, com `makensis` obrigatório quando o formato é solicitado.

Um build Windows feito pelo Zumbra usa MinGW, adiciona metadados de versão, ícone `.ico` e pode usar o subsistema gráfico sem console.

### macOS

```bash
zumbra app package --target macos --format app
zumbra app package --target macos --format zip
zumbra app package --target macos --format all
```

Artefatos:

- bundle `.app` com `Info.plist`;
- ZIP preservando permissões e estrutura do aplicativo;
- ícone `.icns`, quando configurado.

## Cross-build

O compilador é detectado pelo alvo. Variáveis opcionais:

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

Quando já existe um binário para o alvo, ele pode ser empacotado sem recompilar:

```bash
zumbra app package \
  --target windows \
  --binary dist/meu-app.exe \
  --format all
```

O binário fornecido precisa ter sido realmente compilado para o sistema e a arquitetura escolhidos. O Zumbra valida diretamente os cabeçalhos ELF, PE e Mach-O antes de empacotar.

## Builds reproduzíveis

Os arquivos, tarballs, ZIPs e pacotes Debian usam ordenação determinística, proprietários normalizados e timestamp controlado por:

```bash
SOURCE_DATE_EPOCH=1700000000 zumbra app package
```

Cada entrega gera:

- `SHA256SUMS.txt`;
- relatório JSON;
- build ID derivado do manifesto, binário, alvo e arquitetura;
- auditoria de dependências quando `ldd`, `otool` ou `objdump` estiver disponível.

## Atualizações

Quando `[updates].url` está configurado, o pacote gera um descritor `update.json` com versão, canal, target, arquitetura, URLs e hashes. O Z15 fornece a metadata de distribuição; o mecanismo de atualização automática dentro do aplicativo continua opcional.

## Assinatura

```bash
zumbra app package --sign IDENTIDADE
```

- Linux: assinatura ASCII destacada com GPG;
- Windows: `signtool`;
- macOS: `codesign` antes da criação do ZIP.

As credenciais nunca entram no manifesto.

## Símbolos e crash reports

```bash
zumbra app package --symbols
```

O comando extrai símbolos com `objcopy`/`llvm-objcopy` ou `dsymutil`, criando artefatos separados para depuração e integração futura com serviços de crash reporting.

## Ferramentas externas opcionais

```text
AppImage: appimagetool
Windows installer: makensis (NSIS)
Windows cross-build: MinGW GCC + windres
Assinatura Linux: gpg
Assinatura Windows: signtool
Assinatura macOS: codesign
Símbolos: objcopy, llvm-objcopy ou dsymutil
```

`--format all` significa que todos os formatos precisam ser produzidos. A ausência de `appimagetool` ou `makensis` é um erro acionável. Para um pacote parcial, os formatos devem ser listados explicitamente, por exemplo `bundle,deb,appdir` ou `portable`. Antes de empacotar, use `zumbra app doctor` para verificar todos os requisitos.

## Limite de portabilidade

O sistema de distribuição é multi-plataforma. A aplicação precisa usar recursos de runtime disponíveis no alvo. O backend gráfico SDL3 implementado em Z13/Z14 continua Linux-first; empacotar para Windows ou macOS não transforma automaticamente um backend ainda não implementado nesses sistemas.


## Hardening 0.10.1

Consulte `docs/pt-BR/fechamento-multiplataforma-z15-0.10.1.md` para o doctor, validação de binários e configuração de toolchains por alvo.
