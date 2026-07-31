# Z11.1 — Correção dos conversores de plataforma

## Problema corrigido

O pacote final do Z11 continha chamadas para duas funções compartilhadas em
`object/builtins/platforms_builtins.go`:

```go
goValueToObject(...)
objectToGoValue(...)
```

As implementações não haviam sido incluídas. Como o pacote `object/builtins` é
compilado como uma unidade, isso impedia todos os comandos do Zumbra, mesmo
quando PostgreSQL, Redis ou Supabase não eram utilizados.

## Solução

O Z11.1 adiciona uma camada de conversão independente do runtime HTTP:

- valores Go e resultados de `database/sql` para objetos Zumbra;
- objetos Zumbra para valores aceitos por `encoding/json` e drivers;
- conversão recursiva de arrays e dicionários;
- suporte a `null`, strings, booleanos, inteiros e floats;
- suporte a inteiros fixos signed e unsigned;
- suporte a `ByteArray`, arrays tipados e slices;
- suporte a datas, structs, enums e records;
- preservação de `u64` quando o valor ultrapassa `int64`.

## Contrato

As integrações de plataforma não dependem de helpers privados de HTTP. Os
conversores vivem em `object/builtins/value_conversion.go` e podem ser usados
por PostgreSQL, Supabase, storage e integrações futuras.

Resultados textuais de drivers SQL retornados como `[]byte` tornam-se strings
Zumbra. `ByteArray` enviado pelo Zumbra continua sendo preservado como bytes no
sentido Zumbra → Go.

## Validação

Os testes cobrem:

- JSON aninhado;
- valores retornados por drivers de banco;
- dicionários e arrays Zumbra;
- `ByteArray`;
- inteiros fixos signed;
- `u64` acima do limite de `int64`;
- regressão integral do Z11.
