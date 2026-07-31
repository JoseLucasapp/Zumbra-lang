# Inteiros de tamanho fixo

Esta feature adiciona os tipos necessários para registradores, endereços, pixels, áudio e memória de máquinas emuladas sem tornar a sintaxe do Zumbra complexa.

## Tipos disponíveis

```text
u8   0 até 255
u16  0 até 65.535
u32  0 até 4.294.967.295
u64  0 até 18.446.744.073.709.551.615

i8   -128 até 127
i16  -32.768 até 32.767
i32  -2.147.483.648 até 2.147.483.647
i64  -9.223.372.036.854.775.808 até 9.223.372.036.854.775.807
```

O tipo `int` continua existindo e continua sendo o inteiro padrão da linguagem.

## Literais tipados

Acrescente o tipo ao final do número:

```zumbra
var opcode << 0xA9u8;
var address << 0x8000u16;
var color << 0xFF00FF00u32;
var cycles << 1_000_000u64;
var offset << 10i8;
```

Os sufixos funcionam com números decimais, hexadecimais, binários e octais:

```zumbra
var decimal << 255u8;
var hexadecimal << 0xFFu8;
var binary << 0b11111111u8;
var octal << 0o377u16;
```

Um literal fora da faixa é rejeitado pelo parser:

```zumbra
var invalid << 256u8; // erro
```

Para o menor valor de um tipo assinado, use a conversão explícita:

```zumbra
var minimum << i8(-128);
```

## Conversões explícitas

Os nomes dos tipos também são funções de conversão:

```zumbra
var source << 200;
var byte << u8(source);
var signed << i16(-500);
var larger << u32(byte);
```

A conversão é segura. Valores fora da faixa retornam erro em vez de serem truncados silenciosamente:

```zumbra
var invalid << u8(300); // erro de faixa
```

## Aritmética padrão

A aritmética de inteiros de tamanho fixo imita registradores de hardware e faz wrap na largura do tipo:

```zumbra
var value << 255u8;
value << value + 1;
show(value); // 0
```

Outros exemplos:

```zumbra
var underflow << 0u8 - 1;     // 255
var product << 200u8 * 2;     // 144
var inverted << bnot 0u8;     // 255
var shifted << 255u8 shl 1;   // 254
```

O resultado mantém o tipo do operando fixo.

Um `int` comum pode ser usado junto de um inteiro fixo quando cabe na faixa:

```zumbra
var opcode << 0xA0u8;
var next << opcode + 1; // u8
```

Dois tipos fixos diferentes exigem conversão explícita:

```zumbra
var small << 1u8;
var large << 1u16;
var result << u16(small) + large;
```

## Modos explícitos de overflow

### Wrapping

O modo wrapping é o mesmo comportamento da aritmética padrão, mas pode ser indicado explicitamente:

```zumbra
var result << wrapAdd(255u8, 1u8); // 0
var result2 << wrapSub(0u8, 1u8);  // 255
var result3 << wrapMul(200u8, 2u8); // 144
```

### Checked

Retorna erro quando o resultado não cabe no tipo:

```zumbra
var result << checkedAdd(255u8, 1u8); // erro de overflow
```

Funções disponíveis:

```text
checkedAdd
checkedSub
checkedMul
```

### Saturating

Limita o resultado ao menor ou maior valor do tipo:

```zumbra
var maximum << satAdd(255u8, 1u8); // 255
var minimum << satSub(0u8, 1u8);   // 0
```

Funções disponíveis:

```text
satAdd
satSub
satMul
```

## Operações bit a bit

Os operadores adicionados na etapa anterior preservam o tipo fixo:

```zumbra
var flags << 0b11110000u8;
var low << flags band 0x0Fu8;
var combined << flags bor 0x03u8;
var changed << flags bxor 0xFFu8;
var inverted << bnot flags;
```

O shift preserva o tipo do operando esquerdo. A quantidade deve ser menor que a largura do tipo:

```zumbra
var value << 1u8 shl 7; // válido
var invalid << 1u8 shl 8; // erro
```

## Arrays e índices

Inteiros fixos já podem ser usados para ler índices de arrays existentes:

```zumbra
var values << [10, 20, 30];
var index << 1u8;
show(values[index]);
```

Arrays tipados, bytes e mutação por índice pertencem à próxima etapa do roadmap.

## Execução

A feature funciona no:

- lexer;
- parser e AST;
- type checker;
- evaluator;
- compiler;
- bytecode e VM;
- runtime Go;
- transpiler experimental.

## Decisões de simplicidade

- `int` permanece sendo o padrão;
- não é obrigatório declarar o tipo da variável;
- o tipo fica visível somente onde importa: no literal ou na conversão;
- não foram adicionadas palavras-chave novas;
- os operadores bit a bit continuam usando palavras legíveis;
- aritmética de hardware usa wrap por padrão;
- checked e saturating são funções explícitas.
