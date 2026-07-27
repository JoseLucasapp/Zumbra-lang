# Zumbra 0.5.1 — inventário da correção da VM

## Status

- Baseline de entrada: Zumbra 0.5.0 / Z12.1.
- Versão desta entrega: Zumbra 0.5.1.
- Z12.1 permanece concluído; esta entrega corrige uma regressão da VM descoberta na validação do usuário.
- Próximo incremento após a validação: Z12.2 — migrations, savepoints, versionamento de schema e streaming de rows.

## Falha corrigida

A inclusão dos builtins SQLite elevou a tabela global para 263 entradas. `OpGetBuiltin` ainda codificava o índice em um operando de 8 bits, limitado a 255. Os índices 256–262 eram truncados e a VM carregava builtins incorretos.

Sintomas observados:

```text
bytesEqual(...): ERROR: wrong number of arguments. got=2, want=1
readU32LE(...): resultado incorreto 0
round-trip binário: builtin incorreto chamado pela VM
```

## Correção técnica

- `OpGetBuiltin` passou de operando `uint8` para `uint16`.
- A VM passou a ler dois bytes e avançar o instruction pointer em dois bytes.
- O limite do índice de builtins passou de 256 para 65.536 posições.
- Foram adicionados testes explícitos usando o índice 261 e execução real de builtin acima de 255.
- A versão foi atualizada para 0.5.1.
- O header demonstrativo do exemplo HTTP foi atualizado para 0.5.1.

# Arquivos criados

## `docs/pt-BR/`

- `docs/pt-BR/correcao-indice-builtins-0.5.1.md`
  - Documenta causa, correção, compatibilidade e validação.

## `vm/`

- `vm/builtin_index_test.go`
  - Teste de regressão que executa `bytesEqual` quando seu índice está acima de 255.

# Arquivos alterados

## Raiz

- `README.MD`
  - Adiciona a nota da correção 0.5.1 e referência à documentação.
- `main.go`
  - Atualiza a versão de 0.5.0 para 0.5.1.

## `code/`

- `code/code.go`
  - Altera `OpGetBuiltin` de um operando de 1 byte para 2 bytes.
- `code/code_test.go`
  - Adiciona testes de geração, leitura e disassembly com índice 261.

## `code_examples/core/`

- `code_examples/core/http_api.zum`
  - Atualiza o header demonstrativo `X-Zumbra` para 0.5.1.

## `docs/pt-BR/`

- `docs/pt-BR/http-websockets-e-apis.md`
  - Atualiza o valor demonstrativo do header para 0.5.1.

## `nativec/`

- `nativec/http_z11_test.go`
  - Atualiza a saída esperada do exemplo HTTP para 0.5.1.

## `object/builtins/`

- `object/builtins/http_z11_test.go`
  - Atualiza o valor esperado do header HTTP para 0.5.1.

## `vm/`

- `vm/vm.go`
  - Lê o índice de builtin com `ReadUint16` e avança dois bytes.

# Arquivos removidos

Nenhum arquivo.

# Totais

```text
Criados:    2
Alterados:  9
Removidos:  0
Total:     11
```

# Validação executada

## Executada neste ambiente

- `go test ./code` em módulo isolado: passou.
- `go test ./code ./compiler ./vm ./conformance` em clone de validação: passou.
- `go test ./...` em clone de validação: passou.
- `scripts/test-sqlite.sh` em clone de validação: passou.
- SQLite VM: passou.
- SQLite nativo com Clang: passou.
- SQLite nativo com GCC: passou.
- Race detector dos pacotes SQLite: passou.

As dependências externas de MySQL, PostgreSQL, Redis, JWT e Twilio foram substituídas por stubs somente no clone temporário de validação porque este sandbox não possui rede. Os stubs não fazem parte dos ZIPs entregues e nenhum arquivo de integração externo foi modificado.

## Comandos para validação oficial

```bash
go test ./...
scripts/test-sqlite.sh
go run . version
```

Saída esperada da versão:

```text
0.5.1
```
