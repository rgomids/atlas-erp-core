# Rule: Phase Governance

## Objetivo

Controlar evolução por fase sem misturar foundation, core domain e decisões ainda não autorizadas.

## Fase atual

**Phase 2 - Quality & Engineering**

## Escopo permitido nesta fase

- reforçar testes de `customers`, `invoices` e `payments`
- manter `billing` como scaffold com baixo acoplamento
- consolidar validação de borda, error handling e observabilidade por request
- reforçar limites modulares do monolito
- atualizar runtime, documentação e governança da fase

## Fora do escopo por padrão

Sem decisão explícita adicional, não iniciar como trabalho principal:

- extração de microservices
- adoção de Redis como dependência mandatória
- OpenTelemetry como runtime obrigatório nesta fase
- integrações externas reais que ainda não tenham contrato claro
- mensageria externa, CQRS ou outbox como trabalho principal
- paralelização agressiva de agentes sem partição de domínio
- automação de autoevolução de regras/skills sem revisão humana

## Critérios de avanço de fase

Uma evolução de fase deve registrar:

- objetivo da nova fase
- entregáveis esperados
- restrições
- riscos aceitos
- atualização de `README.md`
- atualização do artefato de status de fase adotado pelo repositório

## Registro recomendado

Usar `.agents/templates/phase-status.md` para consolidar:

- fase atual
- objetivo
- entregáveis
- critérios de conclusão
- restrições
- próximos marcos

## Regra de decisão

Quando uma mudança parecer “grande demais” para a fase atual, registrar como proposta, ADR ou pendência — não empurrar silenciosamente para dentro do escopo corrente.
