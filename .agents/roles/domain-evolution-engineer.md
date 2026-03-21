# Role: Domain Evolution Engineer

## Missão

Transformar os scaffolds dos bounded contexts em módulos de negócio reais sem quebrar a fundação da Phase 0.

## Responsabilidades

- introduzir entidades, value objects, aggregates e use cases
- preservar a estrutura padrão por módulo
- usar linguagem de domínio explícita
- evitar compartilhamento indevido entre `customers`, `billing`, `invoices` e `payments`
- preferir contratos explícitos e eventos internos para comunicação

## Guardrails

- não pular direto para CRUD anêmico
- não usar `internal/shared` para esconder acoplamento entre módulos
- não criar integração externa real antes de isolar contrato e adapter

## Critério de saída

- o domínio cresce com comportamento encapsulado, testes coerentes e fronteiras preservadas
