# Z10 — Rede, streams e TLS

O Z10 adiciona comunicação de rede real ao Zumbra sem transformar o núcleo da linguagem em um framework de rede. A API é composta por builtins tipados e pelo módulo de conveniência `std/net.zum`.

## Princípios

- TCP e TLS usam a mesma abstração `NetStream`.
- UDP usa `UDPSocket`, pois preserva fronteiras de datagramas.
- Todos os tamanhos são limitados antes de alocar memória.
- Portas válidas ficam entre `0` e `65535`; a porta `0` pede uma porta efêmera ao sistema.
- Timeouts são explícitos e medidos em milissegundos.
- Cancelamento continua cooperativo: fechar listener, stream ou socket interrompe operações bloqueadas; cancelar uma task não mata uma syscall arbitrariamente.
- A VM/evaluator usam a biblioteca `net` do Go.
- O backend nativo usa sockets POSIX e `getaddrinfo`.
- OpenSSL só é compilado e linkado quando a MIR usa TLS.

## Módulo padrão

Enquanto imports por nome de pacote não existem, o módulo é importado por caminho:

```zumbra
import "../../std/net.zum" as net;
```

Em um projeto instalado, o caminho pode ser ajustado conforme a organização do projeto. O futuro package manager permitirá `import std.net` sem alterar a API das funções.

## Servidor TCP

```zumbra
import "../../std/net.zum" as net;

fct handle(listener) {
    var connection << net.accept(listener);
    var request << net.read(connection, 4096);
    net.writeText(connection, "ok");
    net.closeStream(connection);
    return;
}

var listener << net.listen("127.0.0.1", 8080);

while true {
    spawn handle(listener);
}
```

Para testes e serviços dinâmicos, use porta `0`:

```zumbra
var listener << net.listen("127.0.0.1", 0);
show(net.listenerPortNumber(listener));
```

## Cliente TCP e timeout

```zumbra
var connection << net.connectTimeout(
    "127.0.0.1",
    8080,
    1000
);

net.writeText(connection, "ping");
var response << net.readExact(connection, 4);
net.closeStream(connection);
```

`streamReadExact` falha quando a conexão termina antes da quantidade pedida. Para protocolos baseados em frames, isso evita aceitar silenciosamente mensagens truncadas.

## Leitura parcial e timeout

```zumbra
var result << net.readTimeout(connection, 4096, 250);

var data << result[0];
var received << result[1];
var eof << result[2];
```

Contrato:

- `received = false` e `eof = false`: timeout sem dados;
- `received = true`: ao menos um byte foi lido;
- `eof = true`: o peer encerrou o lado de escrita.

## Escrita

```zumbra
streamWrite(connection, data);
streamWriteAll(connection, data);
```

- `streamWrite` executa uma escrita e retorna quantos bytes foram aceitos;
- `streamWriteAll` continua até transmitir todos os bytes ou encontrar erro.

`data` pode ser `string`, `ByteArray`, `arrayOf("u8")`, `arrayOf("i8")` ou slice de bytes.

## Half-close

```zumbra
streamShutdownWrite(connection);
streamShutdownRead(connection);
```

Half-close é útil em protocolos nos quais um lado termina de enviar, mas ainda precisa receber uma resposta.

## Endereços

```zumbra
show(streamLocalAddress(connection));
show(streamLocalPort(connection));
show(streamRemoteAddress(connection));
show(streamRemotePort(connection));
```

Os endereços retornados são numéricos e funcionam com IPv4 ou IPv6.

## Keepalive

```zumbra
tcpSetKeepAlive(connection, true, 30000);
```

O terceiro argumento é o tempo ocioso em milissegundos. Sistemas operacionais podem arredondar ou limitar o valor.

## DNS

```zumbra
var addresses << net.lookup("localhost");
```

Resultado: `Array<string>` sem endereços duplicados.

Com timeout:

```zumbra
var result << net.lookupTimeout("localhost", 500);
var addresses << result[0];
var completed << result[1];
```

O backend nativo executa `getaddrinfo` em uma thread curta e aguarda com condition variable. O runtime espera workers de DNS pendentes antes do shutdown.

## UDP

```zumbra
var receiver << net.bindUdp("127.0.0.1", 0);
var sender << net.bindUdp("127.0.0.1", 0);

net.sendUdpText(
    sender,
    "127.0.0.1",
    net.udpPortNumber(receiver),
    "hello"
);

var packet << net.receiveUdpTimeout(receiver, 2048, 1000);
var data << packet[0];
var host << packet[1];
var port << packet[2];
var received << packet[3];
```

UDP preserva cada datagrama. Se o buffer for menor que o datagrama, o sistema operacional pode truncar a mensagem; escolha um limite adequado ao protocolo.

## TLS

Servidor:

```zumbra
var listener << net.listenTls(
    "127.0.0.1",
    8443,
    "certificate.pem",
    "private-key.pem"
);
```

Cliente verificado:

```zumbra
var connection << net.connectTlsTimeout(
    "example.com",
    443,
    "example.com",
    false,
    3000
);
```

O argumento `insecure` deve permanecer `false` em produção. Defini-lo como `true` desativa a verificação do certificado e só é adequado para testes locais com certificado autoassinado.

Requisitos do build nativo com TLS:

```bash
sudo apt install libssl-dev
```

O builder adiciona `-lssl -lcrypto` somente quando TLS é usado. Programas TCP/UDP e programas sem rede não recebem essa dependência.

## Segurança e limites

- Cada leitura permite no máximo 64 MiB por chamada.
- Não use leitura sem limite para protocolos controlados por terceiros.
- Use TLS com verificação para tráfego externo.
- Defina timeouts em conexões de longa duração.
- Feche streams, listeners e sockets quando não forem mais necessários.
- Compartilhe conexões entre tasks somente quando o protocolo permitir; use mutex ou uma task proprietária para serializar escrita.

## Backends

| Recurso | Evaluator | VM | Nativo |
|---|---:|---:|---:|
| TCP IPv4/IPv6 | Sim | Sim | Sim |
| UDP IPv4/IPv6 | Sim | Sim | Sim |
| DNS | Sim | Sim | Sim |
| TLS 1.2+ | Sim | Sim | Sim, OpenSSL |
| Timeouts | Sim | Sim | Sim |
| Half-close | Sim | Sim | Sim |
| Keepalive | Sim | Sim | Sim |

## Próxima etapa

O Z11 construirá HTTP, WebSockets e APIs sobre `NetStream`, TLS, tasks e channels. A camada HTTP não deverá reimplementar sockets nem concorrência.
