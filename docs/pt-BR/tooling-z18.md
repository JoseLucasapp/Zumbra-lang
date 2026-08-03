# Z18 — Tooling oficial e fechamento da linguagem

A Z18 fecha a primeira fase estrutural da Zumbra em uma única versão estável: **Zumbra 0.14.0**. Não existem marcos Z18.1 ou Z18.2. O formatter, o linter, a documentação, o gerenciamento de projetos, o profiler, o protocolo de editor e os diagnósticos estruturados pertencem ao mesmo contrato de release.

## Objetivo

Até a Z17, a linguagem já possuía evaluator, bytecode/VM, pipeline tipado HIR/MIR, backend C11, módulos, FFI, concorrência, rede, persistência, desktop, GUI, distribuição e programação de sistemas. A Z18 acrescenta a camada necessária para trabalhar com a linguagem de forma diária e reproduzível:

- estilo canônico de código;
- análise estática oficial;
- documentação de API extraída do código;
- projetos com manifesto e layout previsível;
- medição do pipeline e arquivos `pprof`;
- servidor LSP embutido na CLI;
- extensão oficial para VS Code;
- diagnósticos estáveis para terminal, JSON e editores;
- gate único de validação da Z18.

## Formatter

```bash
zumbra fmt app.zum
zumbra fmt src tests
zumbra fmt --check .
zumbra fmt --stdout app.zum
zumbra fmt --indent 2 app.zum
```

O formatter:

- preserva comentários `//` e documentação `///`;
- normaliza CRLF para LF;
- usa espaços para indentação;
- normaliza espaços entre operadores, argumentos e delimitadores;
- expande blocos de forma determinística;
- valida o código antes e depois da transformação;
- é idempotente: formatar duas vezes produz exatamente o mesmo arquivo;
- escreve por arquivo temporário e renomeia, evitando arquivos parcialmente gravados.

Diretórios gerados, ocultos, `vendor` e `node_modules` não são percorridos.

## Linter

```bash
zumbra lint app.zum
zumbra lint --deny-warnings src tests
zumbra lint --json app.zum
zumbra lint --no-pipeline app.zum
zumbra lint --no-public-docs code_examples
```

O linter combina regras textuais, AST e o pipeline oficial. Os diagnósticos usam códigos estáveis:

| Código | Regra |
|---|---|
| `ZL0001` | erro de parser |
| `ZL1001` | espaço no final da linha |
| `ZL1002` | tabulação usada para indentar |
| `ZL1003` | linha acima do limite configurado |
| `ZL2001` | import duplicado |
| `ZL2002` | API pública sem comentário `///` |
| `ZL2003` | tipo sem nome PascalCase |
| `ZL2004` | comparação booleana redundante |
| `ZL2005` | instrução inalcançável |

Com o pipeline habilitado, erros de módulos, semântica, tipos, HIR e MIR também são apresentados pelo linter. `--deny-warnings` é destinado a CI.

## Diagnósticos estruturados

`pipeline.Diagnostic` agora contém:

- estágio;
- código estável;
- arquivo;
- intervalo com linha e coluna;
- severidade;
- mensagem;
- indicação de warning.

```bash
zumbra check --json app.zum
zumbra lint --json app.zum
zumbra project check --json
```

As posições humanas são baseadas em 1. O LSP converte os intervalos para a base 0 exigida pelo protocolo.

## Documentação de API

Comentários imediatamente acima de uma declaração pública usam `///`:

```zumbra
/// Soma dois valores inteiros.
pub fct add(left, right) {
    return left + right;
}
```

Geração:

```bash
zumbra doc src -o docs/api.md
zumbra doc --format json src -o docs/api.json
zumbra doc --private src
```

São documentados:

- funções nomeadas;
- variáveis e constantes;
- structs, campos e métodos;
- enums e membros;
- aliases de tipo;
- blocos e funções `extern`.

O JSON possui `schema_version: 1` para permitir consumo por sites, IDEs e geradores externos.

## Projetos

### Inicialização

```bash
zumbra project init "Meu CLI"
zumbra project init --kind library "Minha Biblioteca"
zumbra project init --kind desktop --identifier dev.exemplo.app "Meu App"
```

Kinds suportados:

- `cli`;
- `library`;
- `desktop`.

Projetos CLI e library usam:

```toml
[project]
name = "Meu Projeto"
version = "0.1.0"
kind = "cli"
entry = "src/main.zum"

[tooling]
source_dirs = ["src"]
test_dirs = ["tests"]
docs_output = "docs/api.md"
```

Projetos desktop continuam usando o manifesto `[app]` já aceito por `zumbra app`, e também podem ser encontrados pelos comandos `zumbra project`.

### Comandos

```bash
zumbra project info
zumbra project check
zumbra project test
zumbra project run
zumbra project build
zumbra project fmt
zumbra project lint
zumbra project doc
zumbra project clean
```

O manifesto pode ser localizado a partir de qualquer subdiretório. `clean` remove somente `build`, `dist` e `.zumbra` dentro da raiz do projeto.

## Profiler

```bash
zumbra profile app.zum
zumbra profile --runs 50 --warmup 5 app.zum
zumbra profile --json app.zum
zumbra profile --cpu-profile cpu.prof --heap-profile heap.prof app.zum
```

O relatório contém:

- média, mediana, p95, mínimo e máximo;
- bytes alocados e número de alocações por execução;
- tempo acumulado e médio por estágio;
- percentuais de parser, módulos, semântica, tipos, HIR, MIR e otimização.

Arquivos de CPU e heap usam o formato padrão do Go:

```bash
go tool pprof cpu.prof
go tool pprof heap.prof
```

## Language Server Protocol

```bash
zumbra lsp --stdio
```

O servidor implementa JSON-RPC 2.0 com framing `Content-Length` e suporta:

- `initialize`, `initialized`, `shutdown` e `exit`;
- sincronização completa por `didOpen`, `didChange`, `didSave` e `didClose`;
- publicação de diagnósticos;
- formatação de documento;
- símbolos do documento;
- hover de declarações e builtins;
- completion de keywords e builtins.

O servidor não depende de um processo Node ou de um binário separado: ele faz parte da CLI oficial.

## VS Code

A extensão está em `editors/vscode` e inicia:

```bash
zumbra lsp --stdio
```

Para empacotar:

```bash
cd editors/vscode
npm install
npm run test
npm run package
```

A configuração `zumbra.server.path` permite apontar para outro caminho da CLI.

## Gate da Z18

```bash
scripts/test-z18-tooling.sh
```

O gate valida:

- testes dos pacotes de tooling;
- testes do pipeline e da CLI;
- build da CLI 0.14.0;
- scaffold de projeto;
- formatter e idempotência;
- linter e JSON;
- documentação Markdown e JSON;
- project check/test;
- profiler JSON;
- sintaxe da extensão VS Code;
- higiene do repositório.

A Z18 é considerada concluída apenas quando o gate inteiro passa junto com a regressão global. O analisador `unsafeptr` é desabilitado porque a Z17 oferece conversões explícitas de endereço e FFI por definição; todos os demais analisadores do `go vet` continuam ativos:

```bash
go test ./...
go vet -unsafeptr=false ./...
scripts/check-repository-hygiene.sh
```
