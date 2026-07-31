# Zumbra 0.10.2 — fechamento do Z15

A 0.10.2 fecha a implementação do Z15 sem ampliar a superfície pública da linguagem. O fluxo continua baseado em `zumbra.toml`, `app doctor`, `app build` e `app package`.

## Itens fechados

- descoberta de `appimagetool` no projeto e no diretório atual;
- runtime AppImage explícito, cacheado e reutilizável offline;
- metadata AppStream no AppDir;
- artefatos falsos de teste isolados de `dist/packages`;
- erros operacionais sem impressão do uso global;
- bibliotecas de runtime declarativas por sistema;
- launcher Linux com `LD_LIBRARY_PATH` local;
- DLLs no pacote portátil e instalador Windows;
- dylibs em `Contents/Frameworks` no macOS;
- backend desktop/GUI SDL3 dinâmico para Linux, Windows e macOS;
- diálogos, paths, notificações, abertura externa e processos por sistema;
- smoke test gráfico finito e multiplataforma;
- workflow de aceitação Linux, Windows e macOS.

## Compatibilidade

Os manifestos 0.10.0 e 0.10.1 continuam válidos. `runtime_files` é opcional. Aplicações sem bibliotecas externas mantêm o mesmo formato mínimo.

## Gate de release

A implementação do Z15 fica encerrada na árvore da 0.10.2. A promoção de release candidate para baseline estável depende da execução verde da matriz real de CI nos três sistemas operacionais.
