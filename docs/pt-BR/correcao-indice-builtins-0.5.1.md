# Zumbra 0.5.1 — correção do índice de builtins na VM

A versão 0.5.1 corrige uma regressão introduzida quando o Z12.1 adicionou os builtins de SQLite.

## Causa

O bytecode `OpGetBuiltin` armazenava o índice do builtin em apenas um byte. Esse formato aceita somente valores de `0` a `255`.

Com o Z12.1, a tabela passou a possuir 263 builtins. As funções localizadas nos índices `256` a `262`, incluindo operações binárias como `copyBytes`, `bytesEqual` e `sha256`, eram truncadas para oito bits durante a geração do bytecode. A VM então carregava outra função da tabela e produzia erros de aridade ou resultados incorretos.

Exemplo do erro anterior:

```text
ERROR: wrong number of arguments. got=2, want=1
```

## Correção

`OpGetBuiltin` agora usa um operando unsigned de 16 bits:

```text
antes:  opcode + uint8
agora:  opcode + uint16
```

O compiler já trabalhava com índices inteiros, portanto não foi necessário alterar a resolução de símbolos. A VM passou a ler dois bytes e a avançar o instruction pointer corretamente.

O novo limite teórico do bytecode é de 65.536 posições de builtin, muito acima da tabela atual.

## Testes de regressão

Foram adicionadas verificações para:

- codificação e leitura do operando `OpGetBuiltin` acima de 255;
- formatação das instruções com índice 261;
- execução real de `bytesEqual` na VM quando seu índice está acima do limite de um byte;
- preservação dos testes existentes de I/O binário e conformidade evaluator/VM.

## Compatibilidade

O formato de bytecode ainda é interno ao processo e não existe persistência pública de arquivos de bytecode nesta etapa. Por isso a ampliação do operando não quebra artefatos públicos da linguagem.

## Validação

```bash
go test ./...
scripts/test-sqlite.sh
go run . version
```

Versão esperada:

```text
0.5.1
```
