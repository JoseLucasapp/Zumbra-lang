# Zumbra Language for VS Code

Official editor support for Zumbra 0.14.1.

The extension launches `zumbra lsp --stdio` and provides:

- live parser, semantic, type and lint diagnostics;
- canonical document formatting;
- completion for keywords and builtins;
- hover for local declarations and builtins;
- document symbols;
- syntax highlighting and indentation.

Set `zumbra.server.path` when the CLI is not available as `zumbra` on `PATH`.

To package locally:

```bash
npm install
npm run test
npm run package
```
