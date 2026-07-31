# Processo obrigatório para alterações na linguagem

Cada feature do Zumbra deve ser entregue como um corte vertical completo.

## Estrutura de uma entrega

1. **Especificação curta**  
   Define sintaxe, comportamento, erros e compatibilidade.

2. **Implementação real**  
   Altera todas as camadas necessárias. Não é aceitável implementar somente no evaluator ou somente na VM.

3. **Testes por camada**  
   Cada camada alterada deve receber seu próprio teste.

4. **Teste ponta a ponta**  
   O programa deve compilar e executar na VM.

5. **Documentação de uso**  
   Deve explicar o que o recurso faz, como usar, limitações e exemplos.

6. **Exemplo executável**  
   Deve existir em `code_examples/`.

7. **Registro da alteração**  
   `ALTERACOES.md` deve listar arquivos, comandos de teste e instruções de aplicação.

## Padrão de testes

Para uma feature que altera operadores, por exemplo:

```text
lexer/<feature>_test.go
parser/<feature>_test.go
types/<feature>_test.go
evaluator/<feature>_test.go
compiler/<feature>_test.go
vm/<feature>_test.go
```

Nem toda feature toca todas as pastas. Só devem ser criados testes nas camadas realmente envolvidas.

## Critério de conclusão

Uma feature não deve ser marcada como concluída enquanto qualquer item abaixo estiver faltando:

- [ ] implementação;
- [ ] mensagens de erro;
- [ ] testes positivos;
- [ ] testes negativos;
- [ ] documentação;
- [ ] exemplo;
- [ ] registro da mudança;
- [ ] suíte completa passando.
