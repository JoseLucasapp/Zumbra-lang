# Z12.1 — SQLite e núcleo de persistência

O Z12.1 inicia o marco de dados e persistência com SQLite executado dentro do processo do programa Zumbra. Não existe servidor externo. A mesma API funciona no evaluator, na VM e nos executáveis nativos gerados com Clang ou GCC.

## Dependência do sistema

No Debian e Ubuntu, instale os headers e a biblioteca do SQLite:

```bash
sudo apt install libsqlite3-dev pkg-config
```

O backend nativo adiciona `-lsqlite3` somente quando a MIR utiliza uma função SQLite. Programas que não usam SQLite não recebem essa dependência.

## Abrindo bancos

```zumbra
import "../../std/database.zum" as database;

var memoryDb << database.memory();
var fileDb << database.open("app.sqlite");
```

`sqliteMemory()` usa `:memory:`. `sqliteOpen(path)` cria o arquivo quando ele ainda não existe.

## Execução e parâmetros seguros

Toda operação recebe os valores separadamente do SQL:

```zumbra
var result << fileDb.exec(
    "insert into users(name, score) values (?, ?)",
    ["Lucas", 42]
);

show(result["lastInsertId"]);
show(result["rowsAffected"]);
```

Os valores nunca são concatenados ao texto SQL. O runtime usa `sqlite3_bind_*`, verifica a quantidade de placeholders e rejeita tipos incompatíveis.

Tipos aceitos como parâmetros:

- `null`;
- `bool`;
- inteiros comuns e inteiros fixos dentro da faixa signed de 64 bits;
- `float`;
- `string`;
- `ByteArray`, arrays tipados e slices compatíveis com bytes, armazenados como BLOB.

Arrays usados na posição de parâmetros SQL recebem o tipo contextual `array<unknown>`. Por isso podem combinar valores como `["Lucas", 42, true]` sem liberar arrays heterogêneos no restante da linguagem.

## Queries e row mapping

```zumbra
var rows << fileDb.query(
    "select id, name, score from users order by id",
    []
);

show(rows[0]["name"]);
```

O retorno é `array<dict<string, unknown>>`. O mapeamento segue os tipos de armazenamento do SQLite:

| SQLite | Zumbra |
|---|---|
| `NULL` | `null` |
| `INTEGER` | `int` |
| `REAL` | `float` |
| `TEXT` | `string` |
| `BLOB` | `ByteArray` |

## Prepared statements

```zumbra
var insertUser << fileDb.prepare(
    "insert into users(name, score) values (?, ?)"
);

insertUser.exec(["Lucas", 42]);
insertUser.exec(["Zumbra", 100]);
insertUser.close();
```

Statements podem ser reutilizados. Cada execução faz `reset`, limpa bindings anteriores e vincula os parâmetros novos.

## Transactions

```zumbra
var transaction << fileDb.begin();

transaction.exec(
    "insert into users(name, score) values (?, ?)",
    ["Lucas", 42]
);

transaction.commit();
```

Também existe `transaction.rollback()`. Datas não possuem um storage class próprio no SQLite; nesta etapa devem ser persistidas explicitamente como texto ISO 8601 ou como epoch inteiro. Enquanto uma transaction está ativa, operações comuns diretamente no banco são rejeitadas; o handle da transaction deve ser usado. Isso evita que tasks diferentes misturem operações na mesma transaction.

É possível preparar statements dentro da transaction:

```zumbra
var transaction << fileDb.begin();
var statement << transaction.prepare("update users set score = ? where id = ?");
statement.exec([100, 1]);
statement.close();
transaction.commit();
```

## API funcional

As mesmas operações existem como builtins:

```text
sqliteOpen(path)
sqliteMemory()
sqliteExec(database, sql, parameters)
sqliteQuery(database, sql, parameters)
sqlitePrepare(database, sql)
sqliteBegin(database)
sqliteClose(database)
sqliteIsOpen(database)
sqlitePath(database)

sqliteStatementExec(statement, parameters)
sqliteStatementQuery(statement, parameters)
sqliteStatementClose(statement)
sqliteStatementOpen(statement)
sqliteStatementSQL(statement)

sqliteTransactionExec(transaction, sql, parameters)
sqliteTransactionQuery(transaction, sql, parameters)
sqliteTransactionPrepare(transaction, sql)
sqliteCommit(transaction)
sqliteRollback(transaction)
sqliteTransactionActive(transaction)
```

## Fechamento e segurança

- `close()` é idempotente.
- Um banco não pode ser fechado com transaction ativa.
- Statements fechados não podem ser reutilizados.
- Statements vinculados a uma transaction deixam de executar após `commit` ou `rollback`.
- O runtime nativo fecha bancos ainda abertos no encerramento do processo e desfaz transactions pendentes.
- O SQLite usa modo `FULLMUTEX` e busy timeout de cinco segundos.

## Escopo do Z12.1

Concluído nesta atualização:

- bancos em arquivo e memória;
- parâmetros seguros;
- row mapping;
- prepared statements reutilizáveis;
- transactions com commit e rollback;
- blobs;
- API por métodos e API funcional;
- evaluator, VM e backend nativo;
- linking nativo condicional;
- testes e exemplo público.

Ainda pertence aos próximos incrementos do Z12:

- savepoints;
- migrations e versionamento de schema;
- streaming incremental de rows;
- PostgreSQL e pools;
- Redis;
- configuração tipada;
- sessões HTTP persistentes;
- logs estruturados, métricas e tracing.

## Validação

```bash
scripts/test-sqlite.sh
```
