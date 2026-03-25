# Phase Status

## Fase corrente

- Nome: Phase 6 - Architectural Evolution & Distribution Readiness
- Status: active

## Objetivo da fase

Preparar o sistema para futura distribuicao sem abandonar o monolito modular atual, consolidando contratos publicos, eventos padronizados e prontidao arquitetural.

## Escopo permitido

- explicitar contratos publicos entre `customers`, `invoices`, `billing` e `payments`
- padronizar eventos internos com envelope e catalogo publico por modulo
- evoluir `outbox_events` para refletir `pending`, `processed` e `failed`
- reforcar fronteiras internas com validacao automatizada de imports
- documentar estrategia de extracao futura e trade-offs de permanecer no monolito modular

## Entregaveis esperados

- contratos publicos por modulo em `internal/<module>/public`
- catalogo de eventos publicos e envelope padronizado com metadados minimos
- `outbox_events` com campos de aggregate, correlation e lifecycle de processamento
- documentacao de arquitetura e ADR atualizadas para Phase 6
- README, AGENTS, commands, diagrams, ADR e changelog atualizados para Phase 6

## Criterios de conclusao

- o fluxo principal continua funcionando apos a segregacao de contratos
- eventos internos possuem envelope padronizado e catalogo publico
- o outbox reflete `pending`, `processed` e `failed` no dispatch sincronico atual
- existe documentacao objetiva sobre candidatos a extracao e bloqueadores atuais
- documentacao critica reflete a arquitetura e a operacao da Phase 6

## Restricoes

- nao introduzir microservices
- nao adicionar mensageria externa
- nao alterar regras de negocio nem contratos HTTP funcionais
- nao adicionar dispatcher assincrono real nesta fase

## Riscos aceitos

- o monolito continua em um unico deployable
- o outbox continua sincronico e sem dispatcher assíncrono real
- contratos publicos ainda nao equivalem a contratos de rede

## Proximos marcos

- decidir quando `payments` ou `billing` justificarem extracao operacional independente
- avaliar dispatcher assincrono apenas quando backlog, throughput ou integracoes externas exigirem
- revisitar ownership de banco e bootstrap quando houver razao real para multiplos deployables
