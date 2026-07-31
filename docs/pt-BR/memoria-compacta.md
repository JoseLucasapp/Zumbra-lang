# Memória compacta no Zumbra

O Z3 adiciona estruturas simples e eficientes para emulação, games, imagens e áudio. Elas usam a mesma leitura e escrita por índice que os arrays comuns.

## `bytes(tamanho)`

Cria uma sequência compacta de bytes inicializados com zero:

```zumbra
var memory << bytes(4096);
memory[0x200] << 0xA9u8;
show(memory[0x200]);
```

Cada posição é um `u8`. Escritas fora de `0..255` retornam erro em vez de truncar silenciosamente.

## `arrayOf(tipo, tamanho)`

Cria um buffer compacto de inteiros de tamanho fixo:

```zumbra
var registers << arrayOf("u16", 16);
registers[0] << 0x8000u16;
```

Tipos aceitos:

```text
u8 u16 u32 u64
i8 i16 i32 i64
```

O primeiro argumento é uma string para manter a sintaxe simples e evitar introduzir tipos como valores de primeira classe neste momento.

## `slice(valor, início, fim)`

Cria uma visão mutável sem copiar os dados. O fim é exclusivo:

```zumbra
var memory << bytes(16);
var page << slice(memory, 4, 8);

page[0] << 0x42u8;
show(memory[4]); // 66
```

Alterações na slice aparecem no buffer original e vice-versa.

Slices podem ser criadas sobre:

- arrays comuns;
- `ByteArray`;
- arrays tipados;
- outras slices.

## `fill(valor, item)`

Preenche toda a coleção ou slice:

```zumbra
var screen << bytes(64 * 32);
fill(screen, 0u8);
```

## `sizeOf(valor)`

Retorna a quantidade de elementos, não a quantidade de bytes físicos:

```zumbra
var pixels << arrayOf("u32", 256 * 240);
show(sizeOf(pixels)); // 61440
```

## Regras de segurança

- tamanhos não podem ser negativos;
- índices e limites precisam ser inteiros;
- acessos fora da faixa retornam erro;
- `arrayOf` aceita somente inteiros fixos;
- conversões verificam a faixa do tipo;
- slices não possuem os dados: são uma visão do objeto original.

## Representação interna

- `ByteArray` usa `[]byte`;
- arrays tipados usam um buffer de bytes com largura de 1, 2, 4 ou 8 bytes por elemento;
- valores multi-byte são armazenados internamente em little-endian;
- slices armazenam somente fonte, início e tamanho.

A ordem de bytes interna não substitui as futuras APIs explícitas de leitura little-endian e big-endian de arquivos binários, previstas para o Z4.
