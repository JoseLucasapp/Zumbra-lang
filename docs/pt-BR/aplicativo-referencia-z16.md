# Z16 — Primeiro aplicativo desktop oficial

O aplicativo de referência do Z16 é **Minha Coleção de Jogos**, mantido em um repositório separado do compilador. Ele valida o ecossistema desktop completo com um produto local e utilizável.

## Escopo concluído

- janela redimensionável e interface retida;
- formulários com validação e binding bidirecional;
- cadastro, edição, confirmação de exclusão e filtros;
- SQLite com migrations, busca, estatísticas e integridade;
- importação e exportação JSON/CSV;
- backup e restauração com backup de segurança;
- preferências persistentes;
- temas claro e escuro;
- file picker de abertura e salvamento;
- notificações do sistema;
- modais com foco, acessibilidade e bloqueio do fundo;
- scroll interno e listas dinâmicas;
- execução pela VM/evaluator e pelo backend C11;
- empacotamento Linux, Windows e macOS pelo Z15.

## Primitivas adicionadas na linguagem

O fechamento do Z16 exigiu APIs recuperáveis para arquivos, CSV, backup SQLite, picker em modo salvar, isolamento modal e mutação incremental de arrays no runtime nativo. Essas APIs pertencem à linguagem e podem ser reutilizadas por qualquer aplicativo desktop.

## Critério de conclusão

O Z16 é considerado encerrado quando a suíte da linguagem e a suíte do aplicativo passam, o aplicativo executa headless na VM e no binário nativo, e ao menos um pacote desktop real é validado. Linux/Debian foi validado com AppImage e `.deb`; Windows e macOS permanecem alvos suportados pelo sistema de distribuição e devem ser revalidados em cada release nas plataformas correspondentes.
