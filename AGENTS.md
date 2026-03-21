# AGENTS.md

## Nota de contexto

Este repositório já possui a **Fase 0 — Foundation** implementada e deve evoluir preservando a arquitetura-alvo definida neste documento.

Quando houver divergência entre o estado do código e práticas antigas, trate este arquivo como o contrato arquitetural vigente e mantenha-o sincronizado com README, CHANGELOG, ADRs, diagramas e comandos operacionais.

---

## Propósito

Este repositório existe para demonstrar como projetar e construir um **modular monolith** com disciplina arquitetural, usando:

- Domain-Driven Design (DDD)
- Clean Architecture
- Event-Driven patterns internos
- Caminho claro de evolução para microservices

O objetivo é preservar limites modulares, linguagem de domínio e qualidade estrutural enquanto o sistema cresce.

---

## Visão geral da arquitetura

### Princípios obrigatórios

1. **Modular Monolith**
   O sistema é um único artefato de deploy, porém dividido internamente em módulos com fronteiras claras.

2. **DDD**
   O código deve refletir bounded contexts, aggregates, entities, value objects, repositories e domain events quando esses elementos forem introduzidos.

3. **Clean Architecture**
   As dependências sempre apontam para dentro:

   ```text
   interface -> application -> domain
   ```

4. **Event-Driven interno**
   A comunicação entre módulos deve privilegiar eventos de domínio internos, evitando chamadas diretas entre implementações.

5. **Evolução segura**
   O monólito modular deve permitir extração futura de módulos para serviços independentes sem reescrever o domínio.

### Regra de ouro

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.

---

## Estado atual vs arquitetura-alvo

### Fato atual

- O repositório possui a foundation operacional da aplicação.
- Existe bootstrap HTTP com `chi`, logger estruturado, correlation ID, conexão com PostgreSQL e migrations vazias.
- Os módulos `customers`, `billing`, `invoices` e `payments` existem apenas como scaffold estrutural.
- Ainda não existem regras de negócio, entidades, aggregates, handlers de domínio, integrações externas reais ou eventos internos implementados.

### Convenção mandatória para evolução

- Toda implementação futura deve seguir a arquitetura descrita neste documento.
- Nenhuma decisão estrutural deve contradizer os limites de módulo, as camadas arquiteturais ou as regras de teste descritas aqui.

---

## Bounded Contexts de referência

Os bounded contexts iniciais do projeto são:

- `customers`
- `billing`
- `invoices`
- `payments`

### Responsabilidades-alvo por módulo

#### `customers`

- Domínio de cadastro, identificação e ciclo de vida de clientes.
- Services esperados: `CreateCustomer`, `UpdateCustomerProfile`, `DeactivateCustomer`.
- Jobs esperados: `RebuildCustomerProjections`, `SyncCustomerReadModel`.
- Models esperados: `Customer`, `CustomerDocument`, `CustomerStatus`, `CustomerCreated`.

#### `billing`

- Domínio de cobrança, cálculo de valores e políticas de vencimento.
- Services esperados: `GenerateCharge`, `ApplyBillingPolicy`, `CloseBillingCycle`.
- Jobs esperados: `CloseOverdueCharges`, `RecalculateBillingCycle`.
- Models esperados: `Charge`, `BillingPolicy`, `BillingCycle`, `ChargeGenerated`.

#### `invoices`

- Domínio de emissão, consolidação e acompanhamento de invoices.
- Services esperados: `GenerateInvoice`, `IssueInvoice`, `CancelInvoice`.
- Jobs esperados: `ReconcileInvoices`, `RetryInvoiceDispatch`.
- Models esperados: `Invoice`, `InvoiceLine`, `InvoiceStatus`, `InvoiceGenerated`.

#### `payments`

- Domínio de processamento, confirmação e estorno de pagamentos.
- Services esperados: `ProcessPayment`, `ConfirmPayment`, `RefundPayment`.
- Jobs esperados: `RetryPaymentSettlement`, `ExpirePendingPayments`.
- Models esperados: `Payment`, `PaymentAttempt`, `PaymentStatus`, `PaymentProcessed`.

---

## Estrutura padrão do repositório

Estrutura vigente e esperada para evolução:

```text
.
├── AGENTS.md
├── CHANGELOG.md
├── README.md
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── cmd/
│   ├── api/
│   └── migrate/
├── configs/
│   ├── app/
│   └── observability/
├── docs/
│   ├── adr/
│   ├── commands.md
│   └── diagrams/
├── internal/
│   ├── shared/
│   ├── customers/
│   ├── billing/
│   ├── invoices/
│   └── payments/
├── migrations/
└── test/
    ├── integration/
    ├── functional/
    └── support/
```

### Estrutura padrão de módulo

Cada módulo deve viver em `internal/<module-name>`:

```text
internal/customers/
├── domain/
├── application/
│   ├── usecase/
│   └── dto/
├── infrastructure/
│   ├── repository/
│   ├── http/
│   └── persistence/
└── module.go
```

### Estrutura do diretório de conteúdo e documentação

- `docs/adr/`: Architectural Decision Records.
- `docs/diagrams/`: diagramas Mermaid e artefatos C4.
- `docs/commands.md`: referência operacional dos principais comandos.
- `configs/`: configuração por ambiente e observabilidade.
- `test/integration/`: testes de integração com infraestrutura real.
- `test/functional/`: testes funcionais e fluxos críticos.
- `test/support/`: helpers de teste e bootstrap compartilhado para ambientes de teste.

---

## Stack tecnológico completo

### Linguagem e runtime

- Go

### HTTP e composição da aplicação

- `chi` para roteamento HTTP
- `cmd/api` como ponto de entrada da API
- `cmd/migrate` como ponto de entrada das migrations
- `Makefile` como interface preferencial de automação

### Configuração e logging

- `.env` com `godotenv` para bootstrap local
- `log/slog` para logs estruturados em JSON
- Correlation ID propagado desde a borda HTTP

### Persistência

- PostgreSQL como banco transacional principal
- `pgx/v5` para acesso ao PostgreSQL
- `golang-migrate` para migrations

### Cache, coordenação e suporte a eventos

- Redis segue como baseline arquitetural futura
- Event bus interno segue como baseline futura, preparado para estratégia outbox

### Observabilidade

- Logs estruturados
- Correlation ID obrigatório
- Código preparado para expansão futura com OpenTelemetry

### Containerização e ambiente local

- Docker
- Docker Compose

### CI e qualidade

- GitHub Actions
- Testes unitários
- Testes de integração
- Testes funcionais
- `testcontainers-go` para testes de integração com dependências reais

---

## Variáveis de ambiente

### Contrato canônico de runtime da Fase 0

| Variável | Obrigatória | Descrição |
| --- | --- | --- |
| `APP_PORT` | Sim | Porta HTTP da aplicação. |
| `DB_HOST` | Sim | Host do PostgreSQL. |
| `DB_PORT` | Sim | Porta do PostgreSQL. |
| `DB_USER` | Sim | Usuário do PostgreSQL. |
| `DB_PASSWORD` | Sim | Senha do PostgreSQL. |
| `DB_NAME` | Sim | Nome do banco PostgreSQL. |
| `APP_NAME` | Não | Nome lógico da aplicação. Padrão: `atlas-erp-core`. |
| `APP_ENV` | Não | Ambiente atual. Padrão: `local`. |
| `LOG_LEVEL` | Não | Nível de log. Padrão: `info`. |
| `CORRELATION_ID_HEADER` | Não | Header HTTP de correlação. Padrão: `X-Correlation-ID`. |

### Baseline documentada para fases futuras

As variáveis abaixo permanecem registradas como baseline arquitetural e devem ser documentadas no README sempre que forem ativadas em runtime:

| Variável | Obrigatória hoje | Descrição |
| --- | --- | --- |
| `DATABASE_URL` | Não | String de conexão consolidada para PostgreSQL quando a aplicação migrar para esse contrato. |
| `DATABASE_MAX_OPEN_CONNS` | Não | Limite de conexões abertas com o banco. |
| `DATABASE_MAX_IDLE_CONNS` | Não | Limite de conexões ociosas com o banco. |
| `DATABASE_CONN_MAX_LIFETIME` | Não | Tempo máximo de vida de conexão. |
| `REDIS_URL` | Não | String de conexão do Redis. |
| `HTTP_READ_TIMEOUT` | Não | Timeout de leitura do servidor HTTP. |
| `HTTP_WRITE_TIMEOUT` | Não | Timeout de escrita do servidor HTTP. |
| `HTTP_IDLE_TIMEOUT` | Não | Timeout idle do servidor HTTP. |
| `OTEL_SERVICE_NAME` | Não | Nome do serviço reportado ao pipeline de observabilidade. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Não | Endpoint OTLP para traces e métricas. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Não | Headers para autenticação OTLP. |
| `OTEL_TRACES_SAMPLER` | Não | Estratégia de sampling de traces. |

Novas variáveis de ambiente só podem ser adicionadas com:

- atualização deste `AGENTS.md`
- atualização do `README.md`
- registro em `CHANGELOG.md`

---

## Regras obrigatórias de engenharia

### SOLID e Object Calisthenics

- Aplicar SOLID em serviços, use cases, adapters e composição da aplicação.
- Aplicar Object Calisthenics para manter objetos coesos, métodos curtos, baixo acoplamento e encapsulamento real.
- Evitar classes ou structs anêmicas quando houver comportamento de domínio relevante.

### Limites modulares

- Um módulo não pode importar outro módulo para acessar sua implementação interna.
- Comunicação síncrona entre módulos só pode acontecer via interface publicada na borda do módulo.
- Comunicação assíncrona entre módulos deve preferir eventos de domínio.
- Acesso a banco é restrito ao módulo dono do dado.

### Camadas

- `domain` não depende de framework, banco, HTTP ou detalhes de infraestrutura.
- `application` orquestra casos de uso, DTOs e portas.
- `infrastructure` implementa adapters, handlers, persistência e integrações.
- `internal/shared` deve conter apenas utilidades realmente transversais e estáveis.

### Regras proibidas

#### Acoplamento direto entre módulos

```go
// PROIBIDO
import "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure"
```

#### Regra de negócio em handler

```go
// PROIBIDO
func handler() {
    // regra de negócio aqui
}
```

#### Modelo mutável compartilhado

- Cada módulo é dono do seu domínio e dos seus invariantes.

---

## Estratégia de testes

Criar e manter, no mínimo:

- testes unitários
- testes de integração
- testes funcionais

### Mapeamento por camada

- `domain`: testes unitários puros para entidades, value objects e regras invariantes.
- `application`: testes unitários e de orquestração para use cases e contratos entre portas.
- `infrastructure`: testes de integração para repositórios, handlers, migrations e integrações externas.
- fluxos críticos: testes funcionais ou E2E cobrindo jornadas de negócio relevantes.

### Cobertura mínima vigente da foundation

- `internal/shared/config`: testes unitários de carregamento e validação.
- `internal/shared/logging`: testes unitários do logger estruturado.
- `internal/shared/correlation`: testes unitários do middleware de correlação.
- `internal/shared/http`: teste do contrato do `GET /health`.
- `test/integration`: bootstrap do PostgreSQL e migrations vazias com `testcontainers-go`.
- `test/functional`: contrato funcional do healthcheck.

### Regras de qualidade

- Toda correção de bug deve vir acompanhada de teste que falha antes e passa depois.
- Toda nova regra de negócio deve nascer orientada por teste.
- Testes frágeis, acoplados a detalhes internos ou altamente dependentes de timing devem ser reescritos.
- Usar `testcontainers-go` para cenários de integração com PostgreSQL e Redis quando Redis entrar em runtime.

### Instruções de TDD

Adotar o ciclo:

1. Escrever um teste que falha.
2. Implementar o mínimo para fazê-lo passar.
3. Refatorar preservando comportamento.

TDD é obrigatório para regras de domínio, use cases e contratos críticos entre módulos.

---

## Observabilidade obrigatória

- Logs estruturados em todos os fluxos relevantes.
- Correlation ID propagado da entrada HTTP até jobs e eventos internos.
- Métricas básicas de latência, erro e throughput devem ser introduzidas em fases futuras.
- Código preparado para tracing distribuído futuro, mesmo que o sistema ainda seja monolítico.

### Convenções de logging

- Logs devem usar mensagens objetivas e contextualizadas.
- Logs devem ser textuais, consistentes e sem emojis.
- Nunca registrar segredo, token, senha ou payload sensível.

Exemplos:

```text
INFO  api starting app_name=atlas-erp-core app_env=local app_port=8080
INFO  http request completed method=GET path=/health status_code=200 correlation_id=abc123
ERROR ping postgres failed correlation_id=abc123 err="timeout"
```

---

## Comunicação entre módulos

| Tipo | Permitido |
| --- | --- |
| Sync | Apenas via interfaces de borda |
| Async | Preferencial, via eventos de domínio |
| Banco | Apenas dentro do módulo dono do dado |

### Exemplo de evento de domínio

```go
type OrderCreated struct {
    OrderID string
}
```

### Intenção arquitetural

- Eventos internos devem reduzir acoplamento.
- O contrato do evento deve ser explícito e estável.
- O sistema deve estar pronto para estratégia outbox quando a extração de serviços se tornar necessária.

---

## Design patterns do projeto

Os padrões abaixo formam a baseline de implementação:

- **Repository Pattern** para persistência na borda do domínio.
- **Use Case Pattern** para coordenação de comportamento de aplicação.
- **Dependency Inversion** para dependências entre camadas e adapters.
- **Domain Events** para comunicação interna desacoplada.
- **Factory** quando a criação de agregados exigir invariantes ou montagem complexa.
- **Anti-Corruption Layer** para integrações externas e futuras extrações de serviço.
- **Outbox-ready mindset** para publicação confiável de eventos quando houver necessidade de distribuição.

Evitar padrões cerimoniais sem ganho real. Preferir simplicidade, coesão e clareza.

---

## Visualização da arquitetura

Todos os diagramas devem ser mantidos em `docs/diagrams/`.

### Regras

- Usar Mermaid para diagramas versionados no repositório.
- Organizar diagramas conforme o C4 Model:
  - C1: contexto
  - C2: containers
  - C3: componentes
- Atualizar diagramas sempre que houver mudança estrutural relevante.

Os diagramas vigentes da foundation estão em `docs/diagrams/architecture.md`.

---

## ADR e log de decisões

Toda decisão estrutural relevante deve ser registrada em `docs/adr/`.

Exemplos:

- por que modular monolith
- por que Go
- por que PostgreSQL
- por que Redis
- por que não microservices neste estágio

Documentar decisões, não suposições.

---

## Estratégia de evolução

### Fase 0

- Foundation técnica concluída

### Fase 1

- Primeiro fluxo de domínio completo

### Fase 2

- Extração gradual de módulos com fronteiras já estabilizadas

### Fase 3

- Sistema distribuído orientado a eventos

A prioridade atual é manter fronteiras corretas e introduzir domínio com disciplina, não antecipar complexidade operacional.

---

## Trade-offs

### Benefícios

- Deploy mais simples
- Menor custo operacional
- Modelagem de domínio forte
- Evolução incremental mais segura

### Custos

- Requer disciplina para manter fronteiras
- Violações de módulo podem crescer silenciosamente se não forem monitoradas
- Eventos mal definidos podem criar acoplamento indireto

---

## Common hurdles

### 1. Acoplamento entre módulos

- Sintoma: um módulo acessa diretamente struct, repository ou adapter de outro.
- Solução: substituir por interface de borda ou evento de domínio.

### 2. Vazamento de infraestrutura para o domínio

- Sintoma: `domain` conhece SQL, HTTP, Redis ou detalhes de framework.
- Solução: mover dependência para adapter em `infrastructure` e preservar portas no domínio ou aplicação.

### 3. Handlers gordos

- Sintoma: validação de regra, branching de negócio e persistência dentro do endpoint.
- Solução: mover fluxo para use case e deixar o handler apenas como adaptador de entrada.

### 4. Testes frágeis

- Sintoma: testes quebram por detalhes de implementação sem mudança de comportamento.
- Solução: testar contrato e comportamento observável, não detalhes internos acidentais.

### 5. Falta de correlation ID

- Sintoma: não é possível rastrear uma requisição por logs, jobs e eventos.
- Solução: padronizar propagação desde a borda HTTP e exigir enrichment em logs.

### 6. Eventos sem contrato

- Sintoma: produtores e consumidores divergem silenciosamente.
- Solução: definir payload estável, dono do evento e versionamento quando necessário.

### 7. README, CHANGELOG ou comandos desatualizados

- Sintoma: setup, operação e histórico deixam de refletir a realidade do projeto.
- Solução: tratar documentação e `docs/commands.md` como parte da definição de pronto.

### 8. Docker daemon indisponível

- Sintoma: `make up` ou testes com `testcontainers-go` falham com `Cannot connect to the Docker daemon`.
- Solução: iniciar o Docker Desktop ou garantir acesso ao socket do daemon antes de rodar compose, integração ou testes funcionais dependentes de container.

---

## Política de atualização do AGENTS.md

O `AGENTS.md` deve evoluir junto com o projeto e nunca pode ficar desatualizado em relação à arquitetura, stack, módulos, comandos, práticas de teste e convenções operacionais.

Atualize este arquivo sempre que houver mudança em qualquer um dos pontos abaixo:

- arquitetura ou limites entre módulos
- stack tecnológica ou ferramentas oficiais do projeto
- variáveis de ambiente
- estrutura de diretórios
- novos módulos, apps, jobs, services ou models de referência
- estratégia de testes
- observabilidade, logging ou convenções operacionais
- comandos do `Makefile`
- processo de documentação, ADR, `README.md`, `CHANGELOG.md` ou `docs/commands.md`

Regras obrigatórias:

- nenhuma mudança estrutural relevante pode ser entregue sem revisão do `AGENTS.md`
- se a mudança não exigir edição no arquivo, isso deve ser uma decisão consciente e verificável na revisão
- o `AGENTS.md` deve refletir o estado vigente e a arquitetura-alvo, sem contradizer o repositório
- o checklist pós-implementação deve tratar a revisão deste arquivo como item obrigatório

---

## README.md, CHANGELOG.md e comandos

### README.md

O `README.md` deve ser criado e mantido como documentação operacional do projeto, contendo no mínimo:

- visão geral do sistema
- arquitetura resumida
- stack tecnológica
- instruções de setup local
- variáveis de ambiente
- comandos principais
- estrutura de módulos
- estratégia de testes
- status atual de fase

Toda mudança relevante de setup, arquitetura, módulos, comandos ou observabilidade exige atualização do `README.md`.

### CHANGELOG.md

O `CHANGELOG.md` deve ser criado e mantido continuamente.

Regras:

- registrar toda evolução relevante do sistema
- agrupar mudanças por versão ou marco
- separar claramente o que foi adicionado, alterado, corrigido e removido
- atualizar o changelog no mesmo conjunto de mudanças do código

Categorias sugeridas:

- `Added`
- `Changed`
- `Fixed`
- `Removed`

Nenhuma feature estrutural, alteração de contrato, novo módulo ou mudança operacional relevante deve ser entregue sem atualização do `CHANGELOG.md`.

### docs/commands.md

Deve existir um markdown com os principais comandos operacionais do projeto e ele deve ser mantido atualizado sempre que um fluxo recorrente for criado ou alterado.

---

## Comandos principais

O projeto deve centralizar automações no `Makefile` sempre que possível.

Comandos baseline vigentes:

```makefile
make setup
make up
make down
make run
make build
make fmt
make lint
make test
make test-unit
make test-integration
make test-functional
make migrate-up
make migrate-down
```

Regras:

- Preferir `make <target>` a comandos longos de ferramenta.
- Ao adicionar um novo fluxo operacional recorrente, expor esse fluxo no `Makefile`.
- Atualizar `README.md`, `CHANGELOG.md` e `docs/commands.md` quando novos comandos forem introduzidos ou alterados.

---

## Checklist pós-implementação

Toda mudança relevante deve verificar:

- limites modulares preservados
- domínio sem dependência de infraestrutura
- use cases cobrindo a regra de negócio
- testes unitários criados ou atualizados
- testes de integração criados ou atualizados
- testes funcionais criados ou atualizados quando o fluxo for crítico
- logs estruturados e correlation ID presentes
- `AGENTS.md` revisado e atualizado quando necessário
- `README.md` atualizado
- `CHANGELOG.md` atualizado
- `docs/commands.md` atualizado se houver mudança operacional
- ADR criada ou revisada se houver decisão estrutural
- diagramas Mermaid e C4 atualizados quando houver impacto arquitetural

---

## Notas para mantenedores

- Preferir simplicidade a abstrações prematuras.
- Não transformar arquitetura em burocracia vazia.
- Modelar o domínio antes de modelar endpoints.
- Decisões importantes devem ser explícitas, registradas e rastreáveis.
- Este projeto não é sobre CRUD isolado. É sobre modelar domínio, proteger fronteiras e escalar complexidade com disciplina.
