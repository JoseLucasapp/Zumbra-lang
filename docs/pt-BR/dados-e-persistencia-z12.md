# Z12 — dados, persistência, configuração e observabilidade

O Z12 completa a base generalista de dados do Zumbra antes do início do runtime desktop. A versão de referência deste marco é a **0.6.0**.

A implementação mantém o pipeline oficial:

```text
source → lexer → parser → semantic/types → compiler → bytecode → VM
```

O evaluator acompanha a semântica oficial. O backend nativo consome a MIR otimizada e gera C11 para Clang ou GCC.

## Dependências nativas no Debian

```bash
sudo apt update
sudo apt install libsqlite3-dev libpq-dev libhiredis-dev pkg-config
```

As bibliotecas são vinculadas somente quando o programa usa o recurso correspondente:

- SQLite: `-lsqlite3`;
- PostgreSQL: `-lpq`;
- Redis: `-lhiredis`.

Programas que não usam persistência não recebem essas dependências.

## Módulos públicos

```zumbra
import "../../std/database.zum" as database;
import "../../std/data.zum" as data;
import "../../std/config.zum" as configuration;
import "../../std/observability.zum" as observability;
import "../../std/sessions.zum" as sessions;
```

## SQLite embutido

SQLite funciona em arquivo ou memória:

```zumbra
var fileDb << database.sqlite("app.db");
var memoryDb << database.memory();
```

### Parâmetros seguros

Valores nunca precisam ser concatenados ao SQL:

```zumbra
database.exec(
    fileDb,
    "INSERT INTO users(name, score) VALUES (:name, :score)",
    {"name": "Lucas", "score": 42}
);
```

Também são aceitos placeholders posicionais:

```zumbra
database.exec(fileDb, "INSERT INTO users(name) VALUES (?)", ["Zumbra"]);
```

Arrays heterogêneos são permitidos contextualmente somente como parâmetros SQL. Arrays comuns continuam homogêneos.

### Prepared statements

```zumbra
var statement << database.prepare(
    fileDb,
    "INSERT INTO users(name, score) VALUES (?, ?)"
);

database.statementExec(statement, ["Lucas", 42]);
show(database.statementParameterCount(statement));
database.closeStatement(statement);
```

### Transactions e savepoints

```zumbra
var transaction << database.begin(fileDb);
database.transactionExec(transaction, "UPDATE users SET score = score + 1", {});

database.savepoint(transaction, "before_bonus");
database.transactionExec(transaction, "UPDATE users SET score = score + 1000", {});
database.rollbackTo(transaction, "before_bonus");
database.releaseSavepoint(transaction, "before_bonus");

database.commit(transaction);
```

Nomes de savepoints são validados e não são inseridos livremente no SQL.

### Migrations

```zumbra
var applied << database.migrate(fileDb, [
    {
        "version": 1,
        "name": "create_users",
        "sql": "CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT NOT NULL)"
    },
    {
        "version": 2,
        "name": "add_score",
        "sql": "ALTER TABLE users ADD COLUMN score INTEGER NOT NULL DEFAULT 0"
    }
]);

show(applied);
show(database.schemaVersion(fileDb));
```

As migrations são ordenadas por versão, executadas atomicamente e registradas em uma tabela interna. Uma versão já aplicada não é executada novamente.

### Row mapping e streaming

Queries comuns retornam arrays de dicionários. Para conjuntos maiores, use um cursor incremental:

```zumbra
var rows << database.stream(fileDb, "SELECT id, name FROM users ORDER BY id", {});
var current << database.next(rows);

while (current[1]) {
    show(current[0]["name"]);
    current << database.next(rows);
}

database.closeRows(rows);
```

O mapping preserva `null`, inteiros, floats, textos e BLOBs como `ByteArray`.

## PostgreSQL

```zumbra
var postgres << database.postgres(
    "postgres://zumbra:zumbra@127.0.0.1:5432/zumbra?sslmode=disable",
    {
        "maxOpen": 8,
        "maxIdle": 4,
        "maxLifetimeMs": 300000,
        "maxIdleTimeMs": 60000,
        "timeoutMs": 5000
    }
);
```

A API oferece:

- ping e fechamento determinístico;
- parâmetros posicionais seguros (`$1`, `$2`...);
- exec, query, query-one e streaming;
- prepared statements;
- transactions;
- savepoints;
- configuração e estatísticas de pool;
- cancelamento por timeout no runtime Go.

```zumbra
database.postgresExec(
    postgres,
    "INSERT INTO users(name, score) VALUES ($1, $2)",
    ["Lucas", 42]
);
```

Na VM e no evaluator, o pool é fornecido por `database/sql`. No backend C11, o handle `libpq` é sincronizado e mantém as mesmas garantias de parâmetros, transactions, lifecycle e API. A expansão do gerenciador nativo para múltiplos sockets físicos poderá ser feita sem alterar código Zumbra.

## Redis

```zumbra
var redis << database.redis("127.0.0.1", 6379, "", 0, 10);
show(database.redisPingServer(redis));

database.redisSetValue(redis, "user:1", {"name": "Lucas"}, 60000);
show(database.redisGetValue(redis, "user:1")["name"]);
```

Recursos disponíveis:

- valores Zumbra codificados em JSON;
- TTL em milissegundos;
- `GET`, `SET`, `DEL`, `EXISTS`, `EXPIRE`, `TTL` e incremento atômico;
- pipelines;
- estatísticas de pool;
- fechamento determinístico;
- suporte no runtime Go e no runtime nativo com hiredis.

Pub/Sub permanece como extensão futura e não bloqueia o Z12.

## Arquivos JSON

```zumbra
data.writeJson("build/settings.json", {"fullscreen": true}, true);
var settings << data.readJson("build/settings.json");
```

A escrita é atômica: um arquivo temporário é sincronizado e renomeado sobre o destino. Diretórios pais são criados quando necessário.

## Serialização binária versionada

```zumbra
var encoded << data.encodeBinary({"score": 42, "raw": bytes(4)});
var decoded << data.decodeBinary(encoded);
```

O envelope atual começa com:

```text
ZB1\n
```

Isso permite rejeitar formatos incompatíveis ou dados corrompidos. `ByteArray` é preservado sem conversão silenciosa para texto. Arquivos binários são escritos com permissão `0600`.

O formato foi criado para persistência local e intercâmbio controlado. Não deve ser tratado como execução de código.

## Configuração tipada

Configuração pode vir de dicionários, JSON, `.env` e variáveis de ambiente:

```zumbra
var config << configuration.fromValues({
    "port": "8080",
    "debug": "true",
    "password": "secret"
});

show(configuration.integer(config, "port", 3000));
show(configuration.boolean(config, "debug", false));
configuration.secret(config, "password");
show(configuration.redacted(config));
```

Conversões disponíveis:

- string;
- inteiro;
- float;
- boolean;
- valor obrigatório;
- fallback;
- merge de fontes;
- marcação e redação de segredos.

## Logs estruturados

```zumbra
var log << observability.log("server", "info", "build/server.log");
observability.write(log, "info", "request completed", {
    "method": "GET",
    "path": "/health",
    "status": 200
});
observability.closeLogger(log);
```

Cada entrada é JSON e contém timestamp, logger, nível, mensagem e campos contextuais. Níveis abaixo do limite configurado são descartados. Segredos devem ser redigidos pela camada de configuração antes de entrar nos campos.

## Métricas

```zumbra
var registry << observability.registry();
observability.counter(registry, "requests", 1, {"route": "/health"});
observability.gauge(registry, "workers", 4, {});
observability.observe(registry, "latency_ms", 12.5, {"route": "/health"});
show(observability.snapshot(registry));
```

Há counters, gauges e histograms locais. Exportadores externos permanecem uma extensão futura.

## Tracing

```zumbra
var span << observability.trace("request", {"route": "/health"});
var child << observability.child(span, "database", {"driver": "sqlite"});
observability.event(child, "query", {"rows": 1});
observability.finish(child, "ok");
show(observability.finish(span, "ok"));
```

Spans possuem trace ID, span ID, parent ID, atributos, eventos, status e duração. O runtime é local e não depende de um collector externo.

## Sessões HTTP persistentes

O item adiado do Z11 foi fechado no Z12 com stores SQLite e Redis:

```zumbra
var store << sessions.sqlite("sessions.db");
var id << sessions.create(store, {"user": "Lucas"}, 3600000);
show(sessions.get(store, id));

var rotated << sessions.rotate(store, id, 3600000);
sessions.touch(store, rotated, 3600000);
sessions.delete(store, rotated);
sessions.close(store);
```

O store Redis usa prefixo configurável e rotação atômica por script Lua.

## Rate limiting

O rate limiter adiado do Z11 também foi concluído:

```zumbra
var limiter << sessions.limiter(100, 60000);
var result << sessions.allow(limiter, "client-id");
show(result["allowed"]);
show(result["remaining"]);
show(result["retryAfterMs"]);
```

A implementação atual usa janela fixa por chave e é segura para concorrência dentro do processo.

## Lifecycle e segurança

- recursos fechados não podem ser reutilizados;
- `close` é idempotente quando a plataforma permite;
- transactions encerradas não aceitam novas operações;
- parâmetros SQL são enviados pela API do driver;
- savepoint names são validados;
- sessões possuem expiração, touch, rotação e revogação;
- escritas de arquivos são atômicas;
- arquivos binários privados usam `0600`;
- segredos podem ser redigidos antes de logging;
- runtimes nativos fecham recursos restantes no encerramento.

## Exemplos públicos

- `code_examples/core/data_persistence.zum`;
- `code_examples/core/data_serialization.zum`;
- `code_examples/core/config_observability.zum`;
- `code_examples/core/postgres_redis.zum`.

O último exemplo exige servidores externos. Os demais rodam localmente sem infraestrutura adicional além do SQLite do sistema.

## Validação

```bash
go test ./...
scripts/test-data-persistence.sh
go run . version
```

Versão esperada:

```text
0.6.0
```

## Itens futuros que não bloqueiam o Z12

- MySQL;
- Redis Pub/Sub;
- exportadores OpenTelemetry/Prometheus;
- OpenAPI do Z11;
- pool nativo com múltiplas conexões físicas;
- scheduler próprio de I/O, `select` de channels e detector de data races da linguagem;
- `epoll`, `kqueue`, IOCP e raw sockets públicos.

Esses itens continuam no roadmap como extensões ou hardening. O próximo marco funcional é o **Z13 — runtime desktop**.

## Operações recuperáveis para aplicativos desktop

Aplicativos interativos não devem encerrar o processo por um arquivo inválido ou por uma falha de permissão. O módulo `std/data.zum` oferece variantes que retornam resultados explícitos:

```zumbra
var result << data.readJsonResult(path);
if (result["ok"]) {
    show(result["value"]);
} else {
    show(result["error"]);
}

var csv << data.readCsvResult("colecao.csv");
data.writeCsvResult("exportacao.csv", [["name", "platform"], ["Chrono Trigger", "SNES"]]);
```

O formato comum é:

```text
{ok: bool, value: qualquer, error: string}
```

CSV usa o parser padrão RFC 4180 da plataforma, incluindo campos citados, vírgulas e quebras de linha. JSON e CSV são gravados atomicamente.

## Backup, restauração e integridade SQLite

```zumbra
var backup << database.backup(db, "/tmp/colecao.sqlite");
var integrity << database.integrity(db);
var restored << database.restore(db, "/tmp/colecao.sqlite");
```

As três operações retornam resultados recuperáveis. A restauração valida o banco antes de substituir o arquivo ativo e utiliza troca atômica. O backup usa a API de backup do SQLite, preservando consistência mesmo com a conexão aberta.
