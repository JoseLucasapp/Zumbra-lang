# Distribuição multiplataforma da CLI Zumbra

A release oficial da CLI usa runners nativos do GitHub Actions e gera:

- `zumbra-<versão>-linux-amd64.tar.gz`;
- `zumbra-<versão>-windows-amd64.zip`;
- `zumbra-<versão>-macos-arm64.tar.gz`;
- `zumbra-<versão>-macos-amd64.tar.gz`;
- `zumbra-<versão>-source.zip`;
- `SHA256SUMS`.

## Estratégia de testes

Linux é o host canônico da suíte integral (`go test ./...`), pois o runtime nativo atual,
o backend SDL e parte dos testes de distribuição ainda contêm contratos específicos de
Linux. Windows e macOS executam a suíte de pacotes portáveis, compilam a CLI com CGO e
validam a versão do executável produzido.

Essa separação não afirma que todos os programas nativos gerados pela Zumbra sejam
idênticos em todas as plataformas. Ela garante que a CLI distribuída inicia e que o
compilador, parser, pipeline, VM, evaluator, tipos, diagnósticos e tooling portáveis
funcionam no host da release. A expansão da suíte nativa completa para Windows e macOS
deve ocorrer junto com a portabilidade dos respectivos runtimes.

## Dependências do runner Linux

O gate integral requer:

```text
build-essential
clang
libhiredis-dev
libpq-dev
libsqlite3-dev
libssl-dev
zlib1g-dev
```

## Dependências do runner Windows

O job usa MSYS2/UCRT64 com:

```text
mingw-w64-ucrt-x86_64-binutils
mingw-w64-ucrt-x86_64-gcc
mingw-w64-ucrt-x86_64-sqlite
```

As DLLs não pertencentes ao Windows e efetivamente usadas pelo executável são copiadas
para o ZIP com base na saída de `ldd`.

## Dependências do runner macOS

O job instala SQLite pelo Homebrew e configura `CGO_CFLAGS` e `CGO_LDFLAGS` para o
prefixo detectado. São produzidos artefatos separados para Apple Silicon e Intel.

## Execução local do gate de release

```bash
scripts/test-release-platform.sh
```

Para também validar a versão esperada:

```bash
EXPECTED_VERSION=0.14.2 scripts/test-release-platform.sh
```

## Publicação

O workflow é disparado por tags com prefixo `v`:

```bash
git tag -a v0.14.2 -m "Zumbra 0.14.2"
git push origin v0.14.2
```

A release só é publicada depois que Linux, Windows e os dois jobs de macOS geram seus
artefatos com sucesso. O job final também gera o pacote de código-fonte e os checksums.
