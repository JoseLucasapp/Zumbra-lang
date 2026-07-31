# Mutação por índice

A mutação por índice permite alterar diretamente um elemento de array ou uma entrada de dicionário sem adicionar uma sintaxe nova e complexa.

## Arrays

```zumbra
var memory << [0u8, 0u8, 0u8, 0u8];

memory[2] << 0xA9u8;
show(memory[2]); // 169
```

O índice precisa ser um inteiro comum ou um inteiro de tamanho fixo.

```zumbra
var index << 1u16;
memory[index] << 0x10u8;
```

A escrita precisa permanecer dentro dos limites do array. Um índice negativo ou maior que o último índice produz erro:

```zumbra
var values << [1, 2];
values[5] << 10; // erro: array index out of bounds
```

O type checker também preserva o tipo dos elementos quando ele é conhecido:

```zumbra
var bytes << [0u8, 0u8];
bytes[0] << 255u8; // válido
bytes[0] << 300;   // erro de tipo: o array contém u8
```

## Dicionários

Uma atribuição pode atualizar uma chave existente:

```zumbra
var player << {
    "score": 10,
};

player["score"] << 25;
```

Também pode criar uma nova chave, desde que o tipo da chave e do valor seja compatível com o dicionário:

```zumbra
player["lives"] << 3;
```

## Arrays aninhados

A mutação funciona em estruturas aninhadas:

```zumbra
var matrix << [
    [0, 0],
    [0, 0],
];

matrix[1][0] << 7;
show(matrix[1][0]);
```

## Strings

Strings continuam imutáveis:

```zumbra
var name << "zumbra";
name[0] << "Z"; // erro
```

Isso evita alterações parciais ambíguas em texto. Operações de strings deverão continuar sendo feitas por funções específicas.

## Comportamento nas camadas do Zumbra

A feature é processada por todo o pipeline:

```text
lexer -> parser -> AST -> semantic -> type checker -> compiler -> VM
                                      \-> evaluator
                                      \-> transpiler
```

A AST representa a operação com `IndexAssignStatement`. O compiler gera `OpSetIndex`, e a VM altera o objeto já existente em memória. Arrays e dicionários são objetos mutáveis, portanto referências à mesma estrutura observam a alteração.

## Limites desta etapa

Esta implementação ainda usa arrays genéricos de objetos. Ela prepara a sintaxe e a semântica necessárias para o próximo recurso: `ByteArray`, que armazenará bytes em um buffer compacto e eficiente para ROMs, RAM emulada, pixels e áudio.
