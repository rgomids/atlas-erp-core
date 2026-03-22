# Rule: Phase Governance

## Objetivo

Controlar evolucao por fase sem misturar endurecimento tecnico, comunicacao event-driven e decisoes ainda nao autorizadas.

## Fase atual

**Phase 4 - Resilience & Maturity**

## Escopo permitido nesta fase

- endurecer idempotencia em fluxos financeiros
- controlar retry manual e tecnico com `attempt_number`
- isolar timeout e falhas de gateway sem quebrar o fluxo principal
- preparar persistencia de eventos via `outbox_events`
- reforcar logs de dominio, testes de resiliencia e documentacao da fase

## Fora do escopo por padrao

Sem decisao explicita adicional, nao iniciar como trabalho principal:

- extracao de microservices
- adocao de Redis como dependencia mandatoria
- OpenTelemetry como runtime obrigatorio nesta fase
- integracoes externas reais que ainda nao tenham contrato claro
- mensageria externa, CQRS ou outbox assincrono como trabalho principal
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

Quando uma mudanca parecer “grande demais” para a fase atual, registrar como proposta, ADR ou pendencia — nao empurrar silenciosamente para dentro do escopo corrente.
