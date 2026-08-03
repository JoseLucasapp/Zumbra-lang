'use strict';

const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function activate(context) {
  const configuration = vscode.workspace.getConfiguration('zumbra');
  const command = configuration.get('server.path', 'zumbra');
  const serverOptions = {
    run: { command, args: ['lsp', '--stdio'], transport: TransportKind.stdio },
    debug: { command, args: ['lsp', '--stdio'], transport: TransportKind.stdio }
  };
  const watcher = vscode.workspace.createFileSystemWatcher('**/*.zum');
  context.subscriptions.push(watcher);
  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'zumbra' }],
    synchronize: {
      fileEvents: watcher
    },
    outputChannelName: 'Zumbra Language Server'
  };
  client = new LanguageClient(
    'zumbraLanguageServer',
    'Zumbra Language Server',
    serverOptions,
    clientOptions
  );
  return client.start();
}

async function deactivate() {
  if (client) {
    await client.stop();
  }
}

module.exports = { activate, deactivate };
