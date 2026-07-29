# Higiene do repositório

O repositório principal do Zumbra contém somente arquivos necessários para desenvolver, testar e documentar a linguagem.

## O que permanece versionado

- código-fonte Go, C e Zumbra;
- testes e fixtures textuais pequenas;
- documentação;
- exemplos em código-fonte;
- ícones e assets públicos pequenos;
- scripts de build, teste, limpeza e instalação de ferramentas.

## O que não deve ser versionado

- executáveis do Zumbra;
- código C gerado pelo backend nativo;
- diretórios `build`, `dist`, `out` e `release`;
- AppDir, AppImage, DEB, instaladores, ZIPs e tarballs;
- toolchains e utilitários externos baixados;
- bibliotecas SDL locais;
- cobertura, perfis e binários de teste;
- artefatos temporários de CI.

Esses arquivos são recriados por comandos públicos ou armazenados como artefatos de CI e releases.

## Limpeza local

```bash
scripts/clean-generated.sh
```

Para remover também ferramentas baixadas dentro da raiz do projeto:

```bash
scripts/clean-generated.sh --local-tools
```

O cache global em `~/.cache/zumbra` não é removido pelo script de limpeza. Isso permite reutilizar o `appimagetool` e o runtime AppImage sem aumentar o repositório.

## Instalação do appimagetool

```bash
scripts/setup-appimage-tools.sh
```

O script instala uma versão fixada no cache do usuário. O executável não é copiado para o repositório.

## Histórico Git

Remover arquivos do diretório atual reduz clones e ZIPs futuros, mas não remove objetos já existentes no histórico Git. Uma reescrita de histórico deve ser feita separadamente, com backup e coordenação com todos os clones, somente quando o tamanho de `.git` continuar alto.
