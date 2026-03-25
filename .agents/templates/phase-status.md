# Phase Status

## Fase corrente

- Nome: Phase 7 - Portfolio Differentiation & Advanced Engineering
- Status: active

## Objetivo da fase

Transformar o projeto em uma vitrine tecnica clara, mensuravel e defendivel, preservando o monolito modular atual e adicionando evidencia objetiva de comportamento, falhas controladas e decisoes arquiteturais explicitas.

## Escopo permitido

- manter contratos publicos, envelope de eventos e lifecycle do outbox consistentes com a implementacao
- adicionar benchmark reproduzivel dos fluxos principais
- adicionar perfis de falha controlados no runtime local
- reforcar testes de duplicidade, timeout, retry e falha de consumo
- documentar trade-offs, limitacoes, criterios de extracao e evidencias de engenharia

## Entregaveis esperados

- benchmark package em `test/benchmark` com export opcional para `docs/benchmarks/`
- perfis de falha controlados via `ATLAS_FAULT_PROFILE`
- cobertura adicional para duplicidade, timeout, retry e falha de outbox/consumo
- ADRs, diagrams, trade-offs e failure scenarios sincronizados com o codigo real
- README, AGENTS, commands, diagrams, ADR e changelog atualizados para Phase 7

## Criterios de conclusao

- existe benchmark local reproduzivel com latencia media, p95, throughput e taxa de erro
- existem cenarios de falha controlados e documentados sem alterar contratos HTTP
- os testes explicam as garantias de idempotencia, retry e previsibilidade operacional
- README e docs apresentam arquitetura, trade-offs, limites conhecidos e evidencias tecnicas
- documentacao critica reflete a arquitetura e a operacao da Phase 7

## Restricoes

- nao introduzir microservices
- nao adicionar mensageria externa
- nao alterar regras de negocio nem contratos HTTP funcionais
- nao adicionar dispatcher assincrono real nesta fase

## Riscos aceitos

- o monolito continua em um unico deployable
- o outbox continua sincronico e sem dispatcher assincrono real
- benchmarks continuam locais e dependentes do ambiente de execucao
- falhas controladas servem a avaliacao local e nao sao modo operacional padrao

## Proximos marcos

- revisar periodicamente se `payments` ou `billing` ja possuem pressao suficiente para extracao
- avaliar dispatcher assincrono apenas quando backlog, throughput ou integracoes externas exigirem
- transformar benchmark local em historico comparavel quando houver regressao real a acompanhar
