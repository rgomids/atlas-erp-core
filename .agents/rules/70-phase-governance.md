# Rule: Phase Governance

## Objetivo

Controlar evolucao por fase sem misturar observabilidade operacional, mudancas de negocio e decisoes estruturais ainda nao autorizadas.

## Fase atual

**Phase 6 - Architectural Evolution & Distribution Readiness**

## Escopo permitido nesta fase

- explicitar contratos publicos por modulo em `internal/<module>/public`
- padronizar eventos internos com envelope estavel e catalogo publico
- evoluir `outbox_events` para refletir `pending`, `processed` e `failed`
- reforcar fronteiras internas com validacao automatizada de imports
- documentar candidatos a extracao e criterios para distribuir
- manter o fluxo principal funcional, observavel e com a mesma simplicidade operacional

## Fora do escopo por padrao

Sem decisao explicita adicional, nao iniciar como trabalho principal:

- extracao de microservices
- Kafka, SQS, RabbitMQ ou qualquer mensageria externa
- OpenTelemetry Collector, Tempo, Grafana ou stack mais complexa que o necessario
- alteracao de regra de negocio ou de contratos HTTP funcionais
- CQRS, event sourcing ou dispatcher/outbox assincrono como trabalho principal
- paralelizacao agressiva de agentes sem particao de dominio
- automacao de autoevolucao de rules/skills sem revisao humana

## Criterios de avanco de fase

Uma evolucao de fase deve registrar:

- objetivo da nova fase
- entregaveis esperados
- restricoes
- riscos aceitos
- atualizacao de `README.md`
- atualizacao do artefato de status de fase adotado pelo repositorio

## Registro recomendado

Usar `.agents/templates/phase-status.md` para consolidar:

- fase atual
- objetivo
- entregaveis
- criterios de conclusao
- restricoes
- proximos marcos

## Regra de decisao

Quando uma mudanca parecer “grande demais” para a fase atual, registrar como proposta, ADR ou pendencia. Nao empurrar silenciosamente para dentro do escopo corrente.
