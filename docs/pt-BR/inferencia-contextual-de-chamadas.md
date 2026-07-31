# Inferência contextual de chamadas e `spawn` — Z9.1

O Z9.1 estende a inferência contextual introduzida para callbacks no Z8.1. Agora chamadas comuns e chamadas iniciadas com `spawn` podem concretizar parâmetros, retornos e tipos compostos de funções declaradas sem anotações.

O objetivo é manter o código simples sem deixar `unknown` chegar à HIR, à MIR ou ao backend nativo quando o programa já fornece informação suficiente.

## Chamada comum

```zumbra
var identity << fct(value) {
    value;
};

var result << identity(42);
```

A primeira chamada válida define:

```text
identity: fct(int) -> int
result: int
```

O corpo é analisado novamente com `value: int`. O retorno implícito passa a ser `int`, e todos os nós correspondentes da Type Analysis, HIR e MIR são atualizados.

## `spawn` e `Task<T>`

```zumbra
fct square(value) {
    sleepMs(5);
    value * value;
}

var task << spawn square(6);
var result << await task;
```

Tipos finais:

```text
square: fct(int) -> int
task: Task<int>
result: int
```

`spawn` não possui um sistema separado de inferência. A chamada interna é concretizada primeiro; depois seu retorno é envolvido em `Task<T>`.

## Channels

```zumbra
fct publish(output, value) {
    send(output, value);
    closeChannel(output);
    return;
}

var messages << channel(1);
var producer << spawn publish(messages, 7);
```

Durante a análise:

1. `value` recebe `int` pelo argumento `7`;
2. `send(output, value)` concretiza `output` como `Channel<int>`;
3. o tipo refinado do parâmetro volta para o argumento `messages`;
4. o inicializador original `channel(1)` também é atualizado na Type Analysis.

Resultado:

```text
publish: fct(Channel<int>, int) -> null
messages: Channel<int>
producer: Task<null>
```

Essa atualização do inicializador é importante porque a HIR reduz a declaração usando o tipo da expressão que criou a variável. Sem ela, o símbolo poderia estar correto enquanto o dump continuaria exibindo `channel<unknown>`.

## Primitivas concorrentes

Os tipos abaixo também são propagados para parâmetros:

```text
AtomicInt
Mutex
RWMutex
WaitGroup
Semaphore
Task<T>
Channel<T>
```

Exemplo:

```zumbra
fct increment(counter, amount) {
    var index << 0;
    while (index < amount) {
        atomicAdd(counter, 1);
        index << index + 1;
    }
    return;
}

var counter << atomicInt(0);
var worker << spawn increment(counter, 1000);
```

Tipos finais:

```text
increment: fct(AtomicInt, int) -> null
worker: Task<null>
```

## Métodos

```zumbra
struct Box {
    value: int;

    fct echo(input) {
        input;
    }
}

var box << Box(1);
var result << box.echo(9);
```

O compilador adiciona `self` internamente, mas a assinatura pública do método permanece:

```text
echo: fct(int) -> int
```

A HIR e a MIR não expõem `self` como argumento obrigatório para quem chama o método.

## Especialização monomórfica

O Z9.1 não introduz generics implícitos nem cria várias versões escondidas da mesma função. A primeira assinatura concreta compatível torna-se o contrato da função naquele programa.

```zumbra
var identity << fct(value) { value; };

identity(42);
identity("zumbra");
```

A segunda chamada é rejeitada porque `identity` já foi inferida como:

```text
fct(int) -> int
```

Essa decisão preserva:

- previsibilidade;
- geração nativa simples;
- mensagens de erro claras;
- ausência de especializações invisíveis;
- espaço para projetar generics reais posteriormente.

## Tipos compostos

O refinamento é recursivo. Ele preenche partes desconhecidas sem descartar informação já conhecida:

```text
Channel<unknown> + Channel<int> → Channel<int>
Task<unknown> + Task<string> → Task<string>
Array<unknown> + Array<u8> → Array<u8>
Dict<string, unknown> + Dict<string, int> → Dict<string, int>
```

Tipos concretos incompatíveis não são unidos:

```text
Channel<int> + Channel<string> → erro
```

## Pipeline

O fluxo é:

```text
chamada
  ↓
tipos dos argumentos
  ↓
assinatura esperada da função
  ↓
nova análise do corpo
  ↓
retorno e parâmetros refinados
  ↓
propagação de volta aos argumentos
  ↓
Type Analysis → HIR → MIR → backend
```

Não houve mudança de sintaxe, bytecode ou runtime. O Z9.1 é uma melhoria de análise estática que torna mais precisas as estruturas já executadas corretamente no Z9.

## Limites atuais

- A especialização é monomórfica.
- Funções chamadas com aridade incorreta não são especializadas.
- Argumentos incompatíveis são diagnosticados pela validação normal de chamada.
- Generics explícitos ou múltiplas especializações não fazem parte deste marco.
- Inferência entre módulos segue os nomes internos produzidos pelo sistema modular do Z8.
