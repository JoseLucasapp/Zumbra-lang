# Z5 — Constantes, structs, enums e match

O Z5 adiciona tipos estruturados sem transformar o Zumbra em uma linguagem cerimonial. A sintaxe continua curta e usa os mesmos elementos já conhecidos: `<<`, blocos e `fct`.

## Constantes

Use `const` para declarar um nome que não poderá receber outro valor:

```zumbra
const MAX_LIVES << 3;
const RESET_VECTOR << 0xFFFCu16;
```

Uma nova atribuição é rejeitada pelo analisador semântico, compilador, evaluator e VM:

```zumbra
MAX_LIVES << 5; // erro: constante imutável
```

Constantes podem guardar qualquer valor aceito pela linguagem, inclusive enums, buffers e instâncias de structs.

## Type aliases

Um alias dá um nome mais claro a um tipo existente:

```zumbra
type Byte << u8;
type Address << u16;
```

O alias não cria um tipo incompatível novo. Ele apenas melhora a legibilidade:

```zumbra
struct Cpu {
    opcode: Byte;
    pc: Address;
}
```

## Structs

Uma struct agrupa campos relacionados:

```zumbra
struct Player {
    x: int;
    y: int;
    energy: u8;
}
```

### Construção posicional

Os valores seguem a ordem declarada:

```zumbra
var player << Player(10, 20, 100u8);
```

### Construção nomeada

Para código mais explícito, passe um dicionário com todos os campos:

```zumbra
var player << Player({
    "x": 10,
    "y": 20,
    "energy": 100u8
});
```

Campos ausentes, desconhecidos ou com tipo incorreto geram erro.

## Leitura e alteração de campos

```zumbra
show(player.x);
player.x << player.x + 5;
player.energy << 90u8;
```

Somente campos declarados podem ser usados.

## Métodos

Métodos são funções declaradas dentro da struct:

```zumbra
struct Player {
    x: int;
    y: int;

    fct move(dx, dy) {
        self.x << self.x + dx;
        self.y << self.y + dy;
    }
}
```

O parâmetro `self` é inserido automaticamente. O código de uso continua simples:

```zumbra
var player << Player(10, 20);
player.move(5, -2);
```

Internamente, métodos continuam usando a infraestrutura normal de funções e closures. Isso evita criar um segundo modelo de execução na linguagem.

## Enums

Enums representam um conjunto fechado de valores nomeados:

```zumbra
enum Direction {
    Up;
    Down;
    Left;
    Right;
}
```

Use os membros com acesso por ponto:

```zumbra
var direction << Direction.Right;
```

Membros de enums diferentes não são considerados iguais, mesmo quando possuem o mesmo nome.

## Match

`match` escolhe o primeiro `case` igual ao valor analisado:

```zumbra
var label << match(direction) {
    case Direction.Up {
        "up";
    }
    case Direction.Down {
        "down";
    }
    case Direction.Left {
        "left";
    }
    case Direction.Right {
        "right";
    }
    else {
        "unknown";
    }
};
```

`match` é uma expressão, portanto pode:

- inicializar uma variável;
- ser retornado por uma função;
- ser usado como argumento;
- produzir structs, enums, números, strings ou outros valores.

Os blocos devem produzir tipos compatíveis. Sem `else` e sem caso correspondente, o resultado é `null`.

## Exemplo para emulação

```zumbra
type Byte << u8;
type Address << u16;

const RESET_VECTOR << 0xFFFCu16;

struct Cpu6502 {
    a: Byte;
    x: Byte;
    y: Byte;
    pc: Address;
    cycles: u64;

    fct advance(amount) {
        self.pc << self.pc + amount;
        self.cycles << self.cycles + 1u64;
    }
}

enum AddressingMode {
    Immediate;
    ZeroPage;
    Absolute;
}
```

## Exemplo para games

```zumbra
struct Transform {
    x: int;
    y: int;

    fct move(dx, dy) {
        self.x << self.x + dx;
        self.y << self.y + dy;
    }
}

struct Sprite {
    name: string;
    transform: Transform;
}
```

Structs podem guardar outras structs porque instâncias são valores normais do Zumbra.

## O que não entrou no Z5

Visibilidade `public`/`private`, namespaces e exportação entre módulos serão definidos junto do sistema modular. Implementá-los antes do modelo de módulos criaria regras provisórias e complexidade desnecessária.

Enums com dados e generics também ficam para etapas posteriores. O Z5 entrega a base menor que já atende APIs, games, emuladores e ferramentas.

## Erros protegidos

O Z5 valida:

- reatribuição de constantes;
- tipos desconhecidos;
- campos duplicados;
- métodos duplicados;
- membros duplicados de enum;
- campo inexistente;
- valor incompatível com o campo;
- quantidade incorreta de valores no construtor;
- campos ausentes ou desconhecidos na construção nomeada;
- chamada de método com aridade incorreta;
- comparação incompatível em `match`.
