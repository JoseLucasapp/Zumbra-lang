# Z15.5 — fechamento operacional multiplataforma

A versão 0.10.1 inicia o fechamento dos gates operacionais encontrados na validação da 0.10.0. O foco é impedir que um pacote estrutural seja confundido com uma aplicação funcional para outro sistema operacional.

## Diagnóstico antes do build

O comando `app doctor` verifica o manifesto, o pipeline, o compilador, o formato do binário fornecido e as ferramentas exigidas pelo formato solicitado.

```bash
zumbra app doctor \
  --manifest zumbra.toml \
  --target linux \
  --arch amd64 \
  --format all
```

Saída em JSON:

```bash
zumbra app doctor --manifest zumbra.toml --target windows --json
```

Quando `--binary` é usado, o doctor valida diretamente o cabeçalho ELF, PE ou Mach-O. Sem `--binary`, ele verifica se o Zumbra consegue compilar a aplicação para o alvo.

## Validação obrigatória do binário

O empacotador lê o cabeçalho do executável e valida:

- ELF para Linux;
- PE para Windows;
- Mach-O ou universal binary para macOS;
- arquitetura `amd64` ou `arm64`.

Um binário Linux não pode mais ser colocado silenciosamente em um ZIP Windows ou em um bundle `.app`:

```text
binary target mismatch: executable is linux/amd64 but package target is windows/amd64
```

Essa regra vale para a CLI e para a API Go de distribuição.

## `all` significa todos os formatos

A ausência de uma ferramenta não resulta mais em pacote parcial quando `--format all` é solicitado.

Linux:

```text
AppImage requested but appimagetool is unavailable
```

Windows:

```text
Windows installer requested but makensis is unavailable
```

Para produzir apenas os formatos independentes dessas ferramentas:

```bash
zumbra app package --target linux --format bundle,deb,appdir
zumbra app package --target windows --format portable
```

## Descoberta simplificada de ferramentas

O `appimagetool` é procurado nesta ordem:

1. opção `--appimagetool`;
2. variável `APPIMAGETOOL`;
3. diretório `tools/` do projeto;
4. cache de ferramentas do Zumbra;
5. `PATH`.

Nomes aceitos no diretório `tools/`:

```text
tools/appimagetool
tools/appimagetool-x86_64.AppImage
tools/appimagetool-aarch64.AppImage
```

O NSIS usa `--makensis`, `MAKENSIS` ou `PATH`.

## Cross-compilers

Além das variáveis `ZUMBRA_CC_<SISTEMA>_<ARQUITETURA>`, o backend aceita configuração de sysroot e flags:

```text
ZUMBRA_SYSROOT_WINDOWS_AMD64
ZUMBRA_CFLAGS_WINDOWS_AMD64
ZUMBRA_LDFLAGS_WINDOWS_AMD64

ZUMBRA_SYSROOT_MACOS_ARM64
ZUMBRA_CFLAGS_MACOS_ARM64
ZUMBRA_LDFLAGS_MACOS_ARM64
```

Isso permite integração com MinGW-w64, osxcross e toolchains corporativos sem adicionar opções especiais à linguagem.

## Estado dos runtimes

A distribuição aceita um binário externo correto para qualquer alvo. A compilação automática de aplicações gráficas continua bloqueada fora do Linux enquanto os backends Z13/Z14 de Windows e macOS não estiverem implementados e validados. O doctor apresenta esse bloqueio antes do build.

A regra evita gerar um pacote aparentemente correto com um runtime que não existe no sistema de destino.
