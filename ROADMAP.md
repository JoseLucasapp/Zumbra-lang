# Roadmap do Zumbra para sistemas, games e emulação

## Direção e processo

- [x] Definir sistemas, games e emulação como direção principal do rebuild
- [x] Registrar simplicidade como principal pilar da linguagem
- [x] Definir que toda feature exige implementação, testes, documentação e exemplo
- [ ] Estabilizar toda a suíte atual sem dependências de ambiente externo
- [x] Criar o primeiro teste automático de conformidade entre evaluator e VM
- [ ] Expandir conformidade para toda a linguagem
- [ ] Criar CI oficial do rebuild

## Primitivas de sistemas

- [x] Literais hexadecimais `0x`
- [x] Literais binários `0b`
- [x] Literais octais `0o`
- [x] Separadores numéricos `_`
- [x] Operadores `band`, `bor`, `bxor`, `bnot`, `shl`, `shr`
- [x] Type checker para operações bit a bit
- [x] Opcodes e execução na VM
- [x] Execução equivalente no evaluator
- [x] Documentação e exemplo executável
- [x] Tipos `u8`, `u16`, `u32`, `u64`
- [x] Tipos `i8`, `i16`, `i32`, `i64`
- [x] Conversões explícitas e seguras
- [x] Aritmética wrapping, checked e saturating

## Memória e dados binários

- [x] Arrays tipados
- [x] Mutação por índice
- [x] `ByteArray`
- [x] Slices
- [ ] Leitura e escrita de arquivos binários
- [ ] Leitura little-endian e big-endian

## Estruturas e compilação

- [ ] Constantes
- [ ] Structs
- [ ] Enums
- [ ] Métodos
- [ ] Módulos nativos reais
- [ ] IR tipada
- [ ] Bytecode v2
- [ ] Representação compacta de valores na VM

## Runtime de games

- [ ] Relógio monotônico
- [ ] Janela
- [ ] Framebuffer
- [ ] Teclado
- [ ] Gamepads e hotplug
- [ ] Áudio
- [ ] Assets
- [ ] Loop de jogo
- [ ] Pacote `.zcart`
- [ ] Sandbox para jogos de terceiros

## Emulação e produto

- [ ] Core CHIP-8 escrito em Zumbra
- [ ] Benchmarks da VM
- [ ] Infraestrutura comum de cores
- [ ] Core NES/Famicom
- [ ] Runtime de conquistas
- [ ] Conta e assinatura
- [ ] SDK para criar jogos em Zumbra
- [ ] Primeiro jogo completo em `.zcart`
- [ ] Backend futuro para gerar ROM real

## Próximo passo recomendado

**Arquivos binários e endianness**, usando as estruturas compactas concluídas no Z3 como base.
