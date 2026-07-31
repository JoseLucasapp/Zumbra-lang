# Fechamento do Z15 — 0.10.4

A versão 0.10.4 corrige os últimos bloqueios encontrados na validação real do Z15 e reforça a higiene do repositório.

## Mudanças principais

- metadados AppStream válidos para o `appimagetool` oficial;
- `package.homepage` obrigatório quando o formato AppImage é solicitado;
- versões públicas e testes de regressão sincronizados;
- build intermediário de empacotamento em diretório temporário;
- testes sem caminhos absolutos presumidamente inexistentes;
- limpeza recursiva de saídas geradas;
- verificação automática de diretórios gerados e arquivos maiores que 5 MB.

O comando `zumbra app package` continua produzindo os artefatos finais no diretório configurado, normalmente `dist/packages`. Apenas o código C intermediário e o binário temporário usados pelo empacotador deixam de poluir a árvore do projeto.
