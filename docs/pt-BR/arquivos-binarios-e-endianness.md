# Arquivos binários e endianness

O Z4 adiciona as operações necessárias para abrir ROMs, interpretar cabeçalhos binários, copiar regiões de memória e identificar arquivos por SHA-256.

O objetivo é manter a API pequena e direta. Todas as funções trabalham com buffers compactos já introduzidos no Z3.

## Buffers aceitos

As operações binárias aceitam:

- `ByteArray`, criado por `bytes(tamanho)` ou retornado por `readBytes`;
- `arrayOf("u8", tamanho)`;
- `arrayOf("i8", tamanho)`;
- slices criadas sobre qualquer uma dessas estruturas.

Arrays comuns e arrays tipados maiores que 8 bits não são tratados como buffers binários, porque seus índices representam elementos, não posições de bytes.

## Ler um arquivo

```zumbra
var rom << readBytes("game.nes");
show(sizeOf(rom));
```

`readBytes` retorna um `ByteArray`. O arquivo é carregado exatamente como está, sem interpretação de texto.

Erros como arquivo inexistente ou falta de permissão interrompem a operação com uma mensagem clara.

## Escrever um arquivo

```zumbra
var data << bytes(4);
data[0] << 0x4Eu8;
data[1] << 0x45u8;
data[2] << 0x53u8;
data[3] << 0x1Au8;

var written << writeBytes("header.bin", data);
show(written); // 4
```

`writeBytes` retorna a quantidade de bytes gravados. Diretórios não são criados automaticamente.

## Ler inteiros com endianness

```zumbra
var data << readBytes("header.bin");

var little16 << readU16LE(data, 0);
var big16 << readU16BE(data, 0);
var little32 << readU32LE(data, 0);
var big32 << readU32BE(data, 0);
```

Funções disponíveis:

```text
readU16LE  readU16BE
readU32LE  readU32BE
readU64LE  readU64BE
```

O segundo argumento é o offset em bytes. Os retornos são, respectivamente, `u16`, `u32` e `u64`.

## Escrever inteiros com endianness

```zumbra
var data << bytes(16);

writeU16LE(data, 0, 0x1234u16);
writeU32BE(data, 2, 0x89ABCDEFu32);
writeU64LE(data, 6, 0x0102030405060708u64);
```

Funções disponíveis:

```text
writeU16LE  writeU16BE
writeU32LE  writeU32BE
writeU64LE  writeU64BE
```

As funções alteram o buffer recebido e retornam o próprio buffer. O valor precisa caber no tipo de destino; não existe truncamento silencioso.

## Copiar bytes

```zumbra
var source << readBytes("game.nes");
var header << bytes(16);

copyBytes(header, 0, source, 0, 16);
```

Assinatura:

```text
copyBytes(destino, inicioDestino, origem, inicioOrigem, quantidade)
```

A cópia valida os dois intervalos e funciona corretamente mesmo quando origem e destino compartilham a mesma memória e as regiões se sobrepõem.

## Comparar buffers

```zumbra
var first << bytes(4);
var second << bytes(4);

show(bytesEqual(first, second)); // true
```

`bytesEqual` compara comprimento e conteúdo.

## SHA-256

```zumbra
var rom << readBytes("game.nes");
var hash << sha256(rom);

show(hash);
```

O resultado é uma string hexadecimal minúscula com 64 caracteres. Esse hash poderá identificar ROM, revisão e região sem enviar o arquivo ao servidor.

## Slices binárias

```zumbra
var rom << readBytes("game.nes");
var header << slice(rom, 0, 16);

var mapperData << readU16LE(header, 4);
```

A slice continua sendo uma visão mutável sem cópia.

## Erros validados

O runtime rejeita:

- offsets negativos;
- leitura ou escrita além do fim do buffer;
- valor maior que o tipo de destino;
- uso de `arrayOf("u16", ...)` como sequência de bytes;
- caminhos que não sejam strings;
- buffers de tipos incompatíveis;
- intervalos inválidos em `copyBytes`.

## Exemplo executável

```bash
go run . run code_examples/core/binary_io.zum
```
