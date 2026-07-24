# Z11 — HTTP, WebSockets e APIs

O Z11 adiciona uma camada HTTP/1.1 completa sobre os streams, TLS, tasks e channels já existentes no Zumbra. A API é orientada a objetos tipados, mas continua pequena: uma aplicação possui rotas e middleware; `listen` devolve um handle de servidor; cliente HTTP, JSON, JWT, SSE e WebSocket são funções explícitas.

## Princípios

- HTTP reutiliza `NetStream`; não existe uma segunda implementação de sockets.
- `HttpApp`, `HttpServer`, `HttpRequest`, `HttpResponse`, `HttpClientResponse`, `HttpStream`, `HttpFile` e `WebSocket` são objetos próprios.
- Estado de rotas pertence a cada `HttpApp`; não existe router global.
- O servidor é não bloqueante e devolve um handle para graceful shutdown.
- O backend nativo habilita HTTP somente quando a MIR usa recursos HTTP.
- OpenSSL só é linkado em HTTPS/WSS; zlib só entra quando o runtime HTTP é usado.
- Bodies e frames têm limites explícitos para evitar crescimento ilimitado de memória.

## Importação

Enquanto o package manager ainda não existe, o módulo é importado por caminho:

```zumbra
import "../../std/http.zum" as http;
```

Em um arquivo na raiz do projeto:

```zumbra
import "std/http.zum" as http;
```

## Aplicação e rotas

```zumbra
var api << http.app();

api.get("/health", fct(request, response) {
    request;
    response;
    http.json(200, {
        "status": "ok",
        "runtime": "zumbra"
    });
});

api.post("/users", fct(request, response) {
    response;
    http.json(201, request.body);
});

var server << api.listen("127.0.0.1", 8080);
```

Métodos disponíveis diretamente em `HttpApp`:

```zumbra
api.route(method, path, handler);
api.get(path, handler);
api.post(path, handler);
api.put(path, handler);
api.patch(path, handler);
api.delete(path, handler);
api.use(middleware);
api.static(prefix, directory);
api.bodyLimit(bytes);
api.compression(enabled);
api.cors(origins, methods, headers, credentials, maxAgeSeconds);
api.listen(host, port);
api.listenTls(host, port, certificate, privateKey);
```

## Parâmetros de rota e wildcard

```zumbra
api.get("/users/:id", fct(request, response) {
    response;
    http.json(200, {
        "id": request.params["id"]
    });
});

api.get("/files/*path", fct(request, response) {
    response;
    http.text(200, request.params["path"]);
});
```

## Request

`HttpRequest` expõe:

```text
method
scheme
host
path
remoteAddress
params
query
headers
cookies
form
files
body
rawBody
rawBytes
```

O body é interpretado conforme `Content-Type`:

- `application/json` → objeto JSON Zumbra;
- `application/x-www-form-urlencoded` → `form`;
- `multipart/form-data` → `form` e `files`;
- demais tipos → string e bytes brutos.

Arquivos multipart são representados por `HttpFile` e mantidos em memória somente depois de o limite do app ser aplicado.

## Response

Helpers:

```zumbra
http.text(200, "ok");
http.json(200, {"status": "ok"});
http.html(200, "<h1>Olá</h1>");
http.redirect(302, "/login");
http.file(200, "public/index.html");
```

Uma resposta pode ser alterada antes de ser retornada:

```zumbra
var response << http.json(201, {"created": true});
response.header("X-Request-ID", "abc-123");
response.status(202);
response;
```

Métodos de `HttpResponse`:

```zumbra
response.status(code);
response.header(name, value);
response.json(value);
response.send(text);
response.html(text);
```

## Middleware

```zumbra
api.use(fct(request, response) {
    request;
    response.header("X-Zumbra", "0.4.0");
    true;
});
```

Contrato:

- retornar `true` ou `null` continua a cadeia;
- retornar `false` encerra com a resposta acumulada;
- retornar `HttpResponse` encerra a cadeia com essa resposta;
- headers/cookies colocados pela middleware são preservados quando a rota retorna outra resposta.

## CORS

```zumbra
api.cors(
    ["https://example.com"],
    ["GET", "POST", "OPTIONS"],
    ["Content-Type", "Authorization"],
    true,
    600
);
```

Preflight `OPTIONS` é tratado pelo runtime.

## Compressão

```zumbra
api.compression(true);
```

Quando o cliente anuncia `Accept-Encoding: gzip`, respostas elegíveis podem ser comprimidas. Streaming/SSE e upgrades WebSocket não são comprimidos pelo encoder HTTP comum.

## Limite do body

```zumbra
api.bodyLimit(2 * 1024 * 1024);
```

Faixa permitida: 1 byte a 64 MiB. O padrão é 8 MiB. O limite é aplicado antes da interpretação de JSON, formulário ou multipart.

## Arquivos estáticos

```zumbra
api.static("/assets", "public/assets");
```

O runtime normaliza o caminho e impede escape para fora do diretório registrado.

## Cliente HTTP

```zumbra
var response << http.request(
    "POST",
    "https://api.example.com/items",
    {
        "Authorization": "Bearer token",
        "Content-Type": "application/json"
    },
    {"name": "Zumbra"},
    3000
);

show(http.status(response));
show(http.body(response));
show(http.bodyJson(response)["id"]);
show(http.headers(response)["content-type"]);
```

O cliente segue no máximo cinco redirects e limita a resposta a 64 MiB.

Atalhos:

```zumbra
http.getUrl(url, timeoutMs);
http.postJson(url, headers, body, timeoutMs);
```

## Cookies e sessões stateless

Cookie:

```zumbra
var response << http.json(200, {"authenticated": true});
http.cookie(response, "session", "token", {
    "httpOnly": true,
    "secure": true,
    "sameSite": "Strict",
    "path": "/",
    "maxAge": 3600
});
```

O Z11 suporta sessões stateless usando JWT HS256 dentro de um cookie:

```zumbra
var token << http.signJwt(
    {"sub": "user-123", "role": "admin"},
    environmentSecret,
    3600
);

http.cookie(response, "session", token, {
    "httpOnly": true,
    "secure": true,
    "sameSite": "Strict"
});
```

Validação:

```zumbra
var verification << http.verifyJwt(
    request.cookies["session"],
    environmentSecret
);

var valid << verification[0];
var claims << verification[1];
```

Armazenamento de sessões no servidor será construído sobre SQLite/Redis no Z12. O Z11 não mantém uma tabela global oculta de sessões.

## JSON

```zumbra
var encoded << http.stringify({
    "name": "Zumbra",
    "version": 11,
    "stable": true
});

var decoded << http.parse(encoded);
show(decoded["name"]);
```

Dicionários JSON podem conter valores heterogêneos e são tipados como `Dict<string, unknown>`.

## Server-Sent Events

```zumbra
api.get("/events", fct(request, response) {
    request;
    response;

    var events << channel(8);
    send(events, http.event("ready", "status", "1", 1000));
    send(events, http.event("complete", "status", "2", 0));
    closeChannel(events);

    http.sse(200, events);
});
```

Cada valor enviado ao channel é escrito como um chunk. O channel deve ser fechado quando o stream terminar.

## Streaming genérico

```zumbra
var chunks << channel(4);
send(chunks, "primeiro\n");
send(chunks, "segundo\n");
closeChannel(chunks);
http.stream(200, "text/plain; charset=utf-8", chunks);
```

## WebSocket

Servidor:

```zumbra
api.get("/socket", fct(request, response) {
    response;
    var socket << http.upgradeWebSocket(request);
    var frame << http.readWebSocket(socket);
    http.writeWebSocketText(socket, "echo:" + frame[1]);
    http.closeWebSocket(socket, 1000, "complete");
    return;
});
```

Cliente:

```zumbra
var socket << http.connectWebSocket(
    "ws://127.0.0.1:8080/socket",
    {},
    2000
);

http.writeWebSocketText(socket, "ping");
var frame << http.readWebSocket(socket);
show(frame[0]); // text, binary, ping, pong, close ou timeout
show(frame[1]);
```

APIs:

```zumbra
http.readWebSocket(socket);
http.readWebSocketTimeout(socket, timeoutMs);
http.writeWebSocketText(socket, text);
http.writeWebSocketBytes(socket, bytes);
http.pingWebSocket(socket, bytes);
http.closeWebSocket(socket, code, reason);
http.webSocketClosed(socket);
```

O runtime implementa RFC 6455, mascaramento do cliente, fragmentação, ping/pong, close handshake e limite de 16 MiB por mensagem. `wss://` usa a camada TLS do Z10.

## HTTPS

```zumbra
var server << api.listenTls(
    "127.0.0.1",
    8443,
    "certificate.pem",
    "private-key.pem"
);
```

No backend nativo, OpenSSL só é linkado quando HTTPS ou WSS aparece na MIR.

## Graceful shutdown

```zumbra
show(server.running());
show(server.port());
show(server.address());
show(server.shutdown(5000));
```

O runtime deixa de aceitar conexões novas e aguarda handlers ativos até o timeout.

## Segurança

- Não use segredo JWT fixo no código de produção.
- Cookies de autenticação devem usar `httpOnly`, `secure` e `sameSite` adequados.
- O modo TLS inseguro do Z10 existe apenas para testes locais.
- Defina body limits menores que o máximo quando a API não recebe arquivos grandes.
- Valide tipos e campos do JSON antes de executar regras de negócio.
- Não exponha arquivos estáticos fora de um diretório dedicado.
- WebSockets devem validar origem, autenticação e tamanho de mensagens conforme a aplicação.

## Backends

| Recurso | Evaluator | VM | Nativo C |
|---|---:|---:|---:|
| HTTP client/server | Sim | Sim | Sim |
| HTTPS | Sim | Sim | Sim |
| Router/middleware | Sim | Sim | Sim |
| JSON/JWT/cookies | Sim | Sim | Sim |
| Multipart/static/gzip | Sim | Sim | Sim |
| SSE/streaming | Sim | Sim | Sim |
| WebSocket/WSS | Sim | Sim | Sim |

O transpiler Go textual legado rejeita o Z11 com diagnóstico explícito. Os backends oficiais são a VM e o backend nativo baseado na MIR.
