# Concorrência e tarefas — Z9

O Z9 adiciona concorrência real ao Zumbra sem transformar o código comum em um sistema complexo. A mesma sintaxe funciona na VM e no backend nativo:

- a VM e o evaluator usam goroutines;
- o executável nativo usa `pthread` e atomics C11;
- programas que não usam concorrência continuam sem precisar dessas APIs no código-fonte.

## Tarefas com `spawn`

`spawn` inicia uma chamada em paralelo e devolve uma `Task<T>`:

```zumbra
fct calculate(value) {
    sleepMs(5);
    value * value;
}

var task << spawn calculate(6);
show(await task);
```

Saída:

```text
36
```

`spawn` aceita uma chamada. Isso mantém claro qual função será executada e quais argumentos serão copiados para a tarefa:

```zumbra
var task << spawn calculate(10);
```

Uma expressão arbitrária sem chamada não é aceita depois de `spawn`.

## Funções `async`

Uma função `async` sempre devolve uma tarefa quando é chamada:

```zumbra
var download << async fct() {
    sleepMs(10);
    "complete";
};

var task << download();
show(await task);
```

`spawn` e `async` resolvem necessidades diferentes:

- `spawn function()` inicia uma chamada comum em paralelo naquele ponto;
- uma função `async` define que todas as suas chamadas são assíncronas.

## `await` e `join`

Estas formas esperam a conclusão da tarefa e devolvem seu resultado:

```zumbra
var resultA << await task;
var resultB << join(task);
```

`await` é a forma principal. `join` existe para APIs que precisam passar a operação como função ou manter uma sequência explícita de chamadas.

Quando a tarefa termina com erro, o erro é propagado para quem aguarda.

## Estado de uma tarefa

```zumbra
show(taskDone(task));
show(taskCancelled(task));
```

- `taskDone(task)` informa se a tarefa já liberou seus aguardadores;
- `taskCancelled(task)` informa se ela foi cancelada.

## Cancelamento cooperativo

```zumbra
var task << spawn slowOperation();
show(cancel(task));
show(await task);
```

`cancel` marca a tarefa como cancelada e libera `await`/`join`. Ele não encerra à força uma função que já está executando.

Essa decisão evita corromper:

- mutexes bloqueados;
- arquivos sendo gravados;
- memória compartilhada;
- bibliotecas C;
- invariantes internas do runtime.

Uma API futura de contexto de cancelamento permitirá que tarefas longas consultem voluntariamente o sinal e encerrem seu trabalho de forma limpa.

## Timeout de tarefa

```zumbra
var outcome << joinTimeout(task, 50);
var value << outcome[0];
var completed << outcome[1];
```

O retorno é:

```text
[value, completed]
```

- `completed == true`: `value` contém o resultado;
- `completed == false`: o tempo acabou e `value` é `null`.

Um timeout não cancela automaticamente a tarefa.

## Pausa em milissegundos

```zumbra
sleepMs(25);
```

Valores negativos são rejeitados.

# Channels

Um channel transporta valores entre tarefas:

```zumbra
var messages << channel(8);

fct produce(output) {
    send(output, "ready");
    closeChannel(output);
    return;
}

var producer << spawn produce(messages);
show(receive(messages));
await producer;
```

## Channels tipados por inferência

O tipo do elemento é inferido no primeiro envio conhecido:

```zumbra
var messages << channel(4);
send(messages, "ready");
```

A partir desse ponto, o channel é tratado como `Channel<string>`. Isto é rejeitado:

```zumbra
send(messages, 42);
```

A inferência mantém a API curta: não é necessário repetir o tipo quando ele já está claro pelo uso.

## Buffer

```zumbra
var unbuffered << channel();
var alsoUnbuffered << channel(0);
var buffered << channel(32);
```

- capacidade `0`: o envio espera um receptor;
- capacidade maior que `0`: mensagens podem aguardar no buffer;
- capacidade negativa é rejeitada.

## Enviar e receber

```zumbra
send(messages, "hello");
var message << receive(messages);
```

`receive` devolve `null` quando o channel está fechado e não possui mais valores.

## Receber com estado de fechamento

```zumbra
var outcome << receiveOk(messages);
var value << outcome[0];
var open << outcome[1];
```

O retorno é:

```text
[value, open]
```

Valores enviados antes de `closeChannel` são drenados antes de `open` se tornar `false`.

## Receber com timeout

```zumbra
var outcome << receiveTimeout(messages, 50);
var value << outcome[0];
var open << outcome[1];
var received << outcome[2];
```

O retorno é:

```text
[value, open, received]
```

- `received == true` e `open == true`: uma mensagem foi recebida;
- `received == true` e `open == false`: o channel fechou sem mensagens restantes;
- `received == false`: ocorreu timeout.

## Fechamento e inspeção

```zumbra
show(closeChannel(messages));
show(channelClosed(messages));
show(channelLen(messages));
show(channelCap(messages));
```

`closeChannel` devolve `true` apenas no primeiro fechamento. Enviar para um channel fechado gera erro.

# Mutexes

## Mutex exclusivo

```zumbra
var guard << mutex();

lock(guard);
// alterar estado compartilhado
unlock(guard);
```

O mutex não é reentrante. A mesma tarefa não deve chamar `lock` duas vezes sem liberar.

## Mutex de leitura e escrita

```zumbra
var guard << rwMutex();

rLock(guard);
// leitura compartilhada
rUnlock(guard);

lock(guard);
// escrita exclusiva
unlock(guard);
```

# Wait groups

Wait groups permitem aguardar um conjunto de operações:

```zumbra
var group << waitGroup();
wgAdd(group, 2);

fct worker(group) {
    // trabalho
    wgDone(group);
    return;
}

var first << spawn worker(group);
var second << spawn worker(group);
wgWait(group);
await first;
await second;
```

O contador não pode ficar negativo. Todo `wgAdd` precisa ter os `wgDone` correspondentes.

# Semáforos

Um semáforo limita quantas tarefas entram simultaneamente em uma região:

```zumbra
var slots << semaphore(4);

acquire(slots);
// recurso limitado
release(slots);
```

A capacidade deve ser positiva.

# Inteiros atômicos

Para contadores simples, um `AtomicInt` evita o custo e a cerimônia de um mutex:

```zumbra
var counter << atomicInt(0);

atomicAdd(counter, 1);
show(atomicLoad(counter));
```

APIs disponíveis:

```zumbra
atomicLoad(counter);
atomicStore(counter, 10);
atomicAdd(counter, 5);
atomicSwap(counter, 0);
atomicCompareSwap(counter, 0, 1);
```

- `atomicAdd` devolve o novo valor;
- `atomicSwap` devolve o valor anterior;
- `atomicCompareSwap` devolve `true` quando a troca acontece.

# Estado compartilhado

O Zumbra não transforma automaticamente todas as estruturas mutáveis em objetos sincronizados. Arrays, dicionários, structs e buffers compartilhados devem usar:

- channels para transferência de dados;
- mutexes para regiões críticas;
- atomics para contadores simples.

Isso mantém objetos comuns leves e não impõe custo de sincronização a programas sequenciais.

# Backends

## VM e evaluator

A implementação usa goroutines e objetos concorrentes próprios do runtime Zumbra.

## Backend nativo

O backend gera C11 e usa:

- `pthread_create` para tasks;
- mutexes e condition variables POSIX;
- `pthread_rwlock_t`;
- atomics C11;
- flag de link `-pthread`.

O runtime aguarda as tarefas nativas ativas antes de liberar sua arena ao encerrar o programa.

## Transpiler Go legado

O transpiler textual Go anterior ao pipeline MIR rejeita `spawn`, `await` e funções `async` com diagnóstico explícito. Os backends oficiais do Z9 são:

- VM para desenvolvimento;
- backend nativo para executáveis.

# Limites do Z9

O Z9 não inclui ainda:

- preempção ou encerramento forçado de tarefas;
- contextos de cancelamento propagáveis;
- `select` sobre vários channels;
- detecção automática de data races;
- scheduler de I/O de rede;
- callbacks C persistentes ou chamados por threads externas;
- estruturas mutáveis automaticamente protegidas;
- garantias de ordem entre tarefas além das primitivas usadas.

Esses pontos serão tratados nos marcos de rede, baixo nível e tooling, sem mudar o modelo básico de `Task`, `Channel` e sincronização.
