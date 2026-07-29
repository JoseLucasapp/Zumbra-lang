# Checklist final do Z15

## Z15.1 — manifesto e assets

- [x] `zumbra.toml` validado estritamente.
- [x] Entrypoint e raiz do projeto protegidos.
- [x] Assets incorporados no executável.
- [x] API de assets na VM, evaluator e backend nativo.
- [x] Metadados e hashes dos assets.

## Z15.2 — Linux

- [x] Binário release/debug.
- [x] Bundle `.tar.gz`.
- [x] Pacote `.deb`.
- [x] AppDir.
- [x] AppImage real.
- [x] Arquivo `.desktop` e ícone.
- [x] Metadata AppStream.
- [x] Bibliotecas locais por `runtime_files`.
- [x] Launcher com `LD_LIBRARY_PATH` local.
- [x] Runtime AppImage explícito/cacheado.
- [x] Builds reproduzíveis.

## Z15.3 — Windows

- [x] Build PE target-aware.
- [x] Recursos de versão e ícone.
- [x] Subsistema console/GUI.
- [x] ZIP portátil.
- [x] Script NSIS auditável.
- [x] Instalador NSIS.
- [x] DLLs declarativas por `runtime_files`.
- [x] Backend SDL3/SDL3_ttf dinâmico.
- [x] Diálogos, paths, processos, notificações e abertura externa Win32.
- [x] Smoke test desktop/GUI.
- [x] Job de aceitação em Windows real.

## Z15.3 — macOS

- [x] Build Mach-O target-aware.
- [x] Bundle `.app` e `Info.plist`.
- [x] ZIP do aplicativo.
- [x] Ícone `.icns`.
- [x] Dylibs em `Contents/Frameworks` por `runtime_files`.
- [x] Backend SDL3/SDL3_ttf dinâmico.
- [x] Seletores, paths, processos, notificações e abertura externa.
- [x] Assinatura opcional antes do ZIP.
- [x] Smoke test desktop/GUI.
- [x] Job de aceitação em macOS real.

## Z15.4 — hardening

- [x] `app doctor`.
- [x] Saída JSON para CI.
- [x] Validação ELF, PE e Mach-O.
- [x] Validação `amd64` e `arm64`.
- [x] Rejeição de binário estrangeiro.
- [x] `--format all` estrito.
- [x] Descoberta de ferramentas na raiz do manifesto, diretório atual, cache e PATH.
- [x] Falhas operacionais sem impressão do uso global.
- [x] Checksums SHA-256 e relatório JSON.
- [x] Metadata de atualização.
- [x] Assinatura opcional.
- [x] Símbolos separados.
- [x] Testes determinísticos isolados de `dist/`.
- [x] Artefatos gerados removidos do controle de versão.
- [x] Matriz CI Linux/Windows/macOS.
- [x] Documentação pública e release notes.

## Estado

A implementação do Z15 está encerrada na versão 0.10.2. A matriz real de CI é o gate operacional de promoção da candidata para baseline estável; não há item de implementação do Z15 pendente antes do Z16.
