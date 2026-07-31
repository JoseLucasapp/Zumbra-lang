# Primitivas de sistemas — literais e operações bit a bit

Esta entrega adiciona os primeiros recursos de baixo nível ao Zumbra sem alterar a sintaxe de atribuição existente.

## Literais inteiros

### Decimal

```zumbra
var value << 255;
```

### Hexadecimal

Use o prefixo `0x`:

```zumbra
var mask << 0xFF;
var address << 0x8000;
```

### Binário

Use o prefixo `0b`:

```zumbra
var flags << 0b1010_0001;
```

### Octal

Use o prefixo `0o`:

```zumbra
var permissions << 0o755;
```

### Separador `_`

O caractere `_` pode separar grupos de dígitos para melhorar a leitura:

```zumbra
var cycles << 1_000_000;
var byteMask << 0b1111_0000;
var color << 0xFF_00_FF;
```

O `_` não altera o valor.

## Operadores bit a bit

O Zumbra utiliza palavras para manter a leitura simples e preservar `<<` como atribuição durante o rebuild.

| Operador | Função | Exemplo |
|---|---|---|
| `band` | AND bit a bit | `value band 0xFF` |
| `bor` | OR bit a bit | `left bor right` |
| `bxor` | XOR bit a bit | `flags bxor mask` |
| `bnot` | NOT bit a bit | `bnot value` |
| `shl` | deslocamento para esquerda | `value shl 8` |
| `shr` | deslocamento para direita | `value shr 4` |

Exemplo completo:

```zumbra
var packed << 0xABCD;
var high << (packed shr 8) band 0xFF;
var low << packed band 0xFF;
var rebuilt << (high shl 8) bor low;
```

## Tipos aceitos

Nesta versão, os operadores trabalham apenas com o tipo `int`, atualmente representado como inteiro assinado de 64 bits.

```zumbra
var valid << 10 band 3;
var invalid << 10.5 band 3; // erro de tipo
```

Os tipos `u8`, `u16`, `u32` e outros inteiros de tamanho fixo serão implementados em uma etapa posterior.

## Regras de deslocamento

O número de bits deve estar entre `0` e `63`.

```zumbra
var valid << 1 shl 8;
var invalid << 1 shl 64; // erro
```

Como `int` ainda é assinado, `shr` é um deslocamento aritmético para valores negativos. Quando os tipos sem sinal forem adicionados, `shr` respeitará o tipo do operando.

## Precedência

Da menor para a maior precedência relevante:

1. `or`
2. `and`
3. comparações (`==`, `!=`, `<`, `<=`, `>`, `>=`)
4. `bor`
5. `bxor`
6. `band`
7. `shl`, `shr`
8. `+`, `-`
9. `*`, `/`, `%`
10. `**`
11. prefixos `!`, `-`, `bnot`

Exemplos:

```zumbra
var a << mask band 1 == 1;
// interpretado como: (mask band 1) == 1

var b << 1 + 2 shl 3;
// interpretado como: (1 + 2) shl 3
```

Use parênteses quando quiser deixar a intenção explícita.

## Implementação interna

O recurso foi implementado em todas as camadas aplicáveis:

- tokens e keywords;
- lexer;
- parser e precedência;
- type checker;
- evaluator;
- compiler;
- bytecode;
- VM;
- exemplos e testes.

O exemplo executável está em:

```text
code_examples/core/system_primitives.zum
```
