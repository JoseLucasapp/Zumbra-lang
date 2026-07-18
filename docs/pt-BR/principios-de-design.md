# Princípios de design do Zumbra

O principal pilar do Zumbra é ser simples. A linguagem pode se tornar mais potente e mais próxima de sistemas sem obrigar o programador a lidar com complexidade desnecessária.

## Regras obrigatórias

1. **Uma forma principal de fazer cada coisa**  
   Recursos novos devem ter uma API pequena e previsível. Sinônimos e alternativas só devem existir quando resolvem compatibilidade real.

2. **Sintaxe legível antes de sintaxe compacta**  
   Operadores e APIs devem continuar compreensíveis para quem está aprendendo. O Zumbra não precisa copiar C para ser capaz de programar sistemas.

3. **Complexidade no runtime, simplicidade no programa**  
   Validação, segurança, gerenciamento de recursos e integração nativa devem ser resolvidos pelo compilador, pela VM e pelo runtime sempre que possível.

4. **Comportamento determinístico**  
   O mesmo programa deve produzir o mesmo resultado no evaluator, na VM e nos futuros backends, exceto quando uma API for explicitamente dependente da plataforma.

5. **Erros claros**  
   Mensagens devem informar o operador ou recurso, os tipos recebidos e a restrição violada.

6. **Compatibilidade consciente**  
   Mudanças que quebram código existente exigem justificativa, documentação de migração e testes.

7. **Toda feature vem completa**  
   Uma alteração de linguagem só é considerada concluída quando inclui:
   - implementação nas camadas aplicáveis;
   - testes automatizados;
   - documentação de uso;
   - exemplo executável;
   - registro dos arquivos alterados.

## Checklist para novas features

- [ ] A sintaxe é a menor possível?
- [ ] Existe apenas uma maneira principal de usar o recurso?
- [ ] Lexer atualizado, quando necessário?
- [ ] Parser e AST atualizados, quando necessário?
- [ ] Type checker atualizado?
- [ ] Evaluator atualizado?
- [ ] Compiler e bytecode atualizados?
- [ ] VM atualizada?
- [ ] Runtime ou backend atualizado, quando necessário?
- [ ] Testes positivos adicionados?
- [ ] Testes de erro adicionados?
- [ ] Precedência ou semântica documentada?
- [ ] Exemplo em `code_examples/` adicionado?
- [ ] Registro em `ALTERACOES.md` atualizado?
