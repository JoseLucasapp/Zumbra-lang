# Alterações — primitivas de sistemas v1

## Objetivo

Adicionar os primeiros recursos necessários para emulação e games, preservando o principal pilar do Zumbra: simplicidade.

A atribuição continua usando `<<`. Para evitar quebra de compatibilidade e conflito visual, operações bit a bit usam palavras legíveis.

## O que foi implementado

- literais hexadecimais: `0xFF`;
- literais binários: `0b1010`;
- literais octais: `0o755`;
- separadores numéricos: `1_000_000`;
- separadores em floats decimais: `10_000.25`;
- `band`, `bor`, `bxor`;
- `bnot`;
- `shl`, `shr`;
- validação de tipos no type checker;
- validação de deslocamentos entre 0 e 63;
- opcodes próprios no bytecode;
- execução consistente no evaluator e na VM;
- documentação e exemplo de uso.

## Arquivos alterados

### Implementação

- `token/token.go`
- `lexer/lexer.go`
- `parser/parser.go`
- `types/checker.go`
- `evaluator/evaluator.go`
- `code/code.go`
- `compiler/compiler.go`
- `vm/vm.go`

### Testes adicionados

- `lexer/systems_primitives_test.go`
- `parser/systems_primitives_test.go`
- `types/systems_primitives_test.go`
- `evaluator/systems_primitives_test.go`
- `compiler/systems_primitives_test.go`
- `vm/systems_primitives_test.go`
- `conformance/systems_primitives_test.go`

### Documentação e exemplos

- `docs/pt-BR/principios-de-design.md`
- `docs/pt-BR/primitivas-de-sistemas.md`
- `docs/pt-BR/processo-de-features.md`
- `code_examples/core/system_primitives.zum`
- `scripts/test-systems-primitives.sh`
- `ALTERACOES.md`
- `ROADMAP.md`

## Onde colocar

O ZIP já contém o repositório completo com os arquivos nos locais corretos. Para aplicar em outra cópia manualmente, copie cada arquivo mantendo exatamente o mesmo caminho relativo a partir da raiz do projeto.

É recomendado criar uma branch antes:

```bash
git checkout -b feature/system-primitives-v1
```

Depois, substitua os arquivos de implementação e adicione os novos arquivos de testes e documentação.

## Como testar

A suíte específica desta entrega:

```bash
./scripts/test-systems-primitives.sh
```

A suíte completa:

```bash
go test ./...
```

O projeto declara Go `1.24.2` em `go.mod`; use essa versão ou uma compatível com o toolchain automático do Go.

## Teste manual

```bash
go run . run code_examples/core/system_primitives.zum
```

## Comportamento esperado

- `0xFF` produz `255`;
- `0b1010` produz `10`;
- `0o755` produz `493`;
- `0b1100 band 0b1010` produz `8`;
- `1 shl 8` produz `256`;
- `bnot 0` produz `-1`;
- `1 shl 64` retorna erro de faixa;
- bitwise com float ou string retorna erro de tipo.

## Limitações conscientes

- todos os inteiros ainda são `int64`;
- ainda não existem `u8`, `u16`, `u32` ou aritmética modular tipada;
- `shr` é aritmético para números negativos;
- a sintaxe de atribuição `<<` foi preservada;
- arrays de bytes e mutação indexada ainda não fazem parte desta entrega.

## Validação realizada nesta entrega

Foram executados com sucesso diretamente:

```text
lexer
parser
ast
token
code
types
```

Os testes específicos de `compiler`, `evaluator` e `vm` também passaram em uma cópia de validação com as integrações externas isoladas, pois o ambiente de geração do ZIP não possui acesso à internet para baixar as dependências opcionais de banco, JWT, Redis e Twilio.

A validação final no repositório completo deve ser feita com:

```bash
go test ./...
```

Não considere a suíte completa validada até esse comando passar no seu ambiente.

## Conteúdo dos ZIPs

- O ZIP completo contém o repositório modificado nos caminhos corretos.
- O ZIP de patch contém somente arquivos novos ou alterados, também mantendo os caminhos corretos.
- `.env` foi excluído do ZIP completo para evitar redistribuir credenciais locais.
- `build/zumbra-app` foi excluído porque é um binário gerado e deve ser recompilado no ambiente de destino.
