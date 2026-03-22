# Rule: Phase Governance

## Objetivo

Controlar evolucao por fase sem misturar observabilidade operacional, mudancas de negocio e decisoes estruturais ainda nao autorizadas.

## Fase atual

**Phase 5 - Observability & Operations**

## Escopo permitido nesta fase

- instrumentar tracing e metricas com OpenTelemetry
- expor `/metrics` e manter `GET /health`
- enriquecer logs com `trace_id`, `span_id`, `event_name`, ids de dominio e `error_type`
- padronizar categorias de erro para troubleshooting
- adicionar stack local simples com Jaeger e Prometheus
- reforcar testes e documentacao operacional da observabilidade

## Fora do escopo por padrao

Sem decisao explicita adicional, nao iniciar como trabalho principal:

- extracao de microservices
- Kafka, SQS, RabbitMQ ou qualquer mensageria externa
- OpenTelemetry Collector, Tempo, Grafana ou stack mais complexa que o necessario
- alteracao de regra de negocio, payloads ou contratos funcionais
- CQRS, event sourcing ou outbox assincrono como trabalho principal
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
