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

## Regras do pipeline a partir do Z6

8. **Uma única análise para todos os backends**
   Arquivos devem passar pelo pipeline canônico antes de compiler, evaluator, transpiler ou futuros backends nativos.

9. **Otimização não pode mudar semântica**
   Toda transformação de HIR ou MIR deve ser validada por testes de conformidade e pelo verifier.

10. **Performance deve ser mensurável**
    Novos passes devem incluir benchmarks quando afetarem compilação, alocação ou execução. Simplicidade não pode esconder regressões.

11. **IR é interna, mas inspecionável**
    O usuário comum não precisa conhecer HIR ou MIR. Desenvolvedores da linguagem devem poder examiná-las com comandos oficiais e saída determinística.


## Regras do backend nativo a partir do Z7

12. **A MIR é a fonte de verdade do código nativo**
    O backend C não pode voltar à AST nem reutilizar o transpiler Go para contornar limitações.

13. **Standalone significa sem toolchain em runtime**
    O executável produzido não deve exigir Go, a VM do Zumbra ou arquivos do projeto para iniciar.

14. **Recursos incompletos falham durante o build**
    O backend deve rejeitar operações não implementadas com diagnóstico claro. Nunca deve alterar a semântica silenciosamente.

15. **Debug e release são contratos distintos**
    Debug prioriza símbolos e diagnóstico. Release prioriza otimização, mas ambos devem produzir o mesmo resultado observável.

16. **Performance nativa deve ser medida por camada**
    Tempo de pipeline, geração C, compilação C, inicialização e execução devem ser registrados separadamente.

## Regras de módulos e FFI a partir do Z8

17. **Privado por padrão**
    Um arquivo não deve expor acidentalmente sua implementação. Somente declarações `pub` atravessam um import com alias.

18. **Imports não podem depender do diretório atual do terminal**
    Caminhos de módulos e arquivos declarados em `extern from` são resolvidos a partir do arquivo que os declara.

19. **Fronteiras nativas são tipadas e explícitas**
    Toda assinatura C deve declarar tipos compatíveis. Chamadas externas exigem `unsafe`, mas continuam passando por semantic resolver e type checker.

20. **O backend nunca deve adivinhar uma ABI**
    Tipos, declarações ou headers não suportados devem produzir diagnóstico. O gerador de bindings não pode inferir silenciosamente layouts de structs, unions, variádicos ou callbacks complexos.

21. **Ponteiros são opacos antes de serem poderosos**
    Handles C podem atravessar a FFI como `ptr`. Dereference, aritmética e memória manual só entram quando existirem contratos claros de segurança e vida útil.

22. **Callbacks têm contrato de vida útil**
    O suporte inicial é apenas síncrono, não retido e não reentrante. Uma biblioteca que guarda ou chama o callback em outra thread precisa de uma API futura própria.

23. **Dependências nativas são observáveis**
    O comando de grafo deve mostrar módulos, exportações e links. Builds não podem esconder quais arquivos ou bibliotecas externas são incorporados.

24. **Módulos são carregados por alcance**
    Somente arquivos alcançáveis a partir da entrada entram no pipeline. Otimização por símbolo pode ser adicionada depois sem mudar a sintaxe.

## Contrato de inferência contextual de callbacks

Quando uma assinatura de callback já é conhecida, o compilador deve usá-la antes de analisar o corpo da função enviada. A sintaxe do usuário permanece curta, mas Type Analysis, HIR, MIR e backends não podem conservar `unknown` onde o contexto fornece um tipo concreto.

A inferência nunca deve mascarar incompatibilidades: aridade, parâmetros, retorno ou reutilização com assinaturas diferentes devem produzir diagnóstico explícito.

## Contrato de concorrência a partir do Z9

25. **Concorrência é opt-in**
    Objetos sequenciais não pagam automaticamente o custo de mutexes. Compartilhamento seguro é feito por channels, atomics ou regiões explicitamente protegidas.

26. **Cancelamento é cooperativo**
    O runtime pode liberar quem aguarda e marcar uma tarefa, mas não pode interromper arbitrariamente código que mantém recursos ou invariantes.

27. **`spawn` recebe uma chamada explícita**
    A sintaxe deixa visíveis a função e os argumentos enviados à tarefa. O compilador não deve capturar expressões arbitrárias de forma implícita.

28. **Channels drenam antes de fechar para o receptor**
    Valores armazenados antes de `closeChannel` continuam disponíveis. Somente depois o receptor observa `open == false`.

29. **O comportamento observável deve coincidir entre VM e nativo**
    Bloqueio, timeout, fechamento, cancelamento e valores atômicos devem seguir o mesmo contrato nos runtimes Go e C.

30. **Deadlocks não podem ser escondidos por comportamento mágico**
    Mutexes e channels mantêm semântica previsível. Ferramentas de diagnóstico podem detectar problemas, mas o runtime não deve alterar silenciosamente a ordem do programa.

31. **Concorrência não autoriza tipos vagos**
    Tasks preservam o tipo de retorno. Channels refinam o tipo de elemento pelo uso e rejeitam envios incompatíveis.

32. **Integração nativa usa primitivas maduras da plataforma**
    O backend C usa pthreads e atomics C11, mantendo uma camada Zumbra pequena em vez de implementar um sistema operacional dentro do runtime.

## Contrato de inferência contextual de chamadas a partir do Z9.1

33. **Informação concreta deve voltar para a declaração**
    Quando uma chamada revela o tipo de um parâmetro, o compilador deve atualizar a função, o símbolo, o inicializador original e todos os nós aplicáveis da Type Analysis. HIR e MIR não podem divergir do símbolo refinado.

34. **`spawn` reutiliza a semântica normal de chamada**
    A chamada interna é inferida primeiro; somente depois seu resultado é envolvido em `Task<T>`. Não deve existir um segundo sistema de tipos exclusivo para tarefas.

35. **Refinamento de tipos compostos é recursivo**
    `Channel<unknown>`, `Task<unknown>`, arrays e dicionários devem preservar suas partes conhecidas e preencher somente as desconhecidas.

36. **Especialização implícita é monomórfica**
    A primeira assinatura concreta compatível fixa o contrato da função naquele programa. O compilador não deve gerar versões escondidas para tipos diferentes sem uma futura feature explícita de generics.

37. **Erros de aridade não provocam inferência parcial**
    Uma chamada com quantidade incorreta de argumentos deve produzir apenas o diagnóstico de aridade. Ela não pode alterar a assinatura da função.

38. **Métodos mantêm `self` como detalhe interno**
    A inferência do corpo considera o tipo do proprietário, mas a assinatura visível do método contém apenas os argumentos fornecidos pelo usuário.

## Contrato de HTTP e APIs a partir do Z11

39. **HTTP reutiliza a rede existente**
    Cliente, servidor, HTTPS e WebSocket devem usar `NetStream`, TLS, tasks e channels. Não deve existir uma pilha de sockets paralela.

40. **Aplicações não compartilham router global**
    Rotas, middleware, CORS, arquivos estáticos e limites pertencem a um `HttpApp` específico. Vários apps podem coexistir no mesmo processo.

41. **Requests e responses são tipos próprios**
    Objetos HTTP não devem ser representados por dicionários sem contrato. Campos dinâmicos como JSON, query e headers continuam disponíveis dentro de tipos definidos.

42. **Limites são aplicados antes do parsing**
    JSON, formulários, multipart e frames WebSocket precisam respeitar limites antes de alocar ou interpretar conteúdo controlado pelo cliente.

43. **Middleware não perde alterações**
    Headers e cookies acumulados por middleware devem sobreviver quando a rota retorna uma nova resposta.

44. **Sessões não criam estado global oculto**
    O Z11 fornece cookies e sessões stateless assinadas. Sessões persistentes pertencem à camada de dados do Z12.

45. **Streaming usa channels**
    SSE e respostas chunked reutilizam a semântica de channels do Z9, incluindo fechamento e backpressure previsível.

46. **WebSocket segue o protocolo, não apenas um socket aberto**
    Mascaramento, fragmentação, ping/pong, close handshake, limites de frame e TLS devem seguir RFC 6455.

47. **Graceful shutdown é parte da API**
    O servidor deve parar novos accepts, acompanhar handlers ativos e obedecer ao timeout antes de liberar recursos.

48. **Dependências continuam condicionais**
    Código sem HTTP não recebe o runtime HTTP. OpenSSL só entra para HTTPS/WSS, e bibliotecas auxiliares só são linkadas quando necessárias.
