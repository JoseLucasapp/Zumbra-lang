# Inferência contextual de callbacks

O Z8.1 torna callbacks concisos sem perder tipagem estática. Quando uma função recebe um parâmetro do tipo `callback(...) -> ...`, essa assinatura passa a ser o contexto usado para analisar a função enviada.

## Exemplo

```zumbra
extern "C" {
    fct apply(value: i32, transform: callback(i32) -> i32) -> i32;
}

var double << fct(value) {
    value * 2i32;
};

unsafe {
    show(apply(21i32, double));
}
```

Antes do Z8.1, `double` aparecia na IR como:

```text
fct(unknown) -> unknown
```

Agora o type checker usa a assinatura esperada antes de analisar o corpo:

```text
fct(i32) -> i32
```

## Fluxo

```text
assinatura esperada do callback
    ↓
tipos dos parâmetros
    ↓
análise do corpo
    ↓
inferência do retorno
    ↓
validação de compatibilidade
    ↓
Type Analysis, HIR, MIR e backend nativo
```

## Callbacks nomeados

Uma função guardada em variável é vinculada ao seu nó de função. Quando ela é passada para um parâmetro callback, o símbolo e o nó original são refinados juntos.

```zumbra
var triple << fct(value) {
    value * 3i32;
};

unsafe {
    apply(14i32, triple);
}
```

Depois da chamada, `triple` possui o tipo concreto `fct(i32) -> i32` em toda a análise.

## Callbacks inline

```zumbra
unsafe {
    apply(21i32, fct(value) {
        value * 2i32;
    });
}
```

A função inline recebe a mesma inferência contextual.

## Erros detectados

### Retorno incompatível

```zumbra
var wrong << fct(value) {
    "texto";
};

unsafe {
    apply(21i32, wrong);
}
```

Erro esperado:

```text
callback return expects i32, got string
```

### Aridade incompatível

```zumbra
var wrong << fct(left, right) {
    left + right;
};
```

Uma assinatura `callback(i32) -> i32` exige exatamente um parâmetro.

### Reutilização incompatível

Depois que uma função é concretizada como `fct(i32) -> i32`, ela não pode ser reutilizada silenciosamente como `fct(string) -> string`. O compilador reporta a incompatibilidade em vez de apagar a informação anterior.

## Escopo léxico

O checker guarda o escopo em que a função foi declarada. Quando a função precisa ser reanalisada com tipos contextuais, o corpo usa o escopo léxico original, não um escopo artificial criado no local da chamada.

Isso preserva corretamente capturas e símbolos externos.

## Impacto na performance

A inferência ocorre durante compilação. Não adiciona verificações por chamada no executável. O backend nativo recebe a MIR já concretizada.

## Limites do Z8.1

- a inferência contextual é aplicada a parâmetros de função já tipados;
- callbacks C síncronos continuam sendo o caso principal do Z8;
- callbacks persistentes, assíncronos ou chamados por outras threads continuam reservados para concorrência e gerenciamento de vida útil futuros;
- não foi adicionada sintaxe obrigatória de anotação aos parâmetros de funções comuns.
