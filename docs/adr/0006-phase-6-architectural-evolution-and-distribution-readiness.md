# ADR 0006 - Phase 6 Architectural Evolution and Distribution Readiness

## Status

Accepted

## Context

O sistema ja tinha fluxo financeiro funcional, observabilidade ponta a ponta e outbox inicial, mas ainda mantinha dois riscos para evolucao futura:

- contratos entre modulos ainda apareciam por imports em `application/*`, `domain/events` e `domain/entities`
- eventos internos nao tinham envelope padronizado nem lifecycle operacional util no outbox

Sem tratar isso agora, a futura extracao de modulos exigiria reescrever pontos de contato ao mesmo tempo em que a operacao ficaria mais complexa.

## Decision

Adotar as seguintes decisoes na Phase 6:

1. todo contrato entre bounded contexts passa a viver em `internal/<module>/public`
2. eventos publicos passam a usar envelope padronizado com `metadata` e `payload`
3. `outbox_events` passa a registrar `aggregate_id`, `correlation_id`, `processed_at`, `failed_at` e `error_message`
4. o dispatch continua sincronico e in-process, mas o outbox passa a refletir `pending`, `processed` e `failed`
5. um teste arquitetural bloqueante passa a impedir imports cruzados fora de `public`
6. `payments` vira o principal candidato a extracao futura, seguido por `billing`
7. a extracao futura so deve ocorrer quando houver pressao operacional, ownership ou throughput real, nao apenas desejo de distribuicao

## Consequences

### Positive

- ownership entre modulos fica explicito e auditavel
- eventos ficam prontos para futura externalizacao sem mudar o contrato semantico
- o outbox deixa de ser apenas trilha passiva e passa a refletir o lifecycle do dispatch atual
- a base ganha protecao automatica contra erosao arquitetural
- a decisao de extrair modulo passa a depender de criterio observavel e documentado

### Negative

- o monolito continua sem broker externo ou deploy independente
- o sistema ganha mais artefatos de contrato e documentacao para manter sincronizados
- parte do custo de evolucao foi deslocada para disciplina de catalogo e versionamento interno

## Notes

- esta ADR nao introduz microservices, broker externo, worker distribuido ou CQRS
- `processed` e `failed` descrevem apenas o dispatch sincronico atual, nao entrega distribuida
- a decisao de extrair um modulo continua condicionada a pressao operacional real
- a Phase 7 usa esta ADR como base para benchmark, trade-offs e limitacoes conhecidas, sem alterar o modelo de deploy atual
