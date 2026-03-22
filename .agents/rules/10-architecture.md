# Rule: Architecture

## Objetivo

Preservar a coerencia do modular monolith e evitar erosao arquitetural na evolucao da Phase 3.

## Arquitetura adotada

- **Modular Monolith**
- **DDD**
- **Clean Architecture**
- **Ports and Adapters**
- **Internal Event-Driven Communication**

## Modulos de referencia

### Ativos

- `customers`
- `invoices`
- `billing`
- `payments`

## Regra de ouro

> Se um modulo depender da implementacao interna de outro modulo, a arquitetura esta quebrada.

## Direcao de dependencias

```text
interface -> application -> domain
```

### Permitido

- `infrastructure` depende de `application` e `domain`
- `application` depende de `domain`
- `domain` nao depende de infraestrutura
- modulo A consome apenas contrato publico de modulo B
- comunicacao entre modulos via eventos internos e portas publicas pequenas

### Proibido

- importar `infrastructure` de outro modulo
- acessar tabela de outro modulo sem contrato explicito
- colocar regra de negocio em handler HTTP, adapter, middleware ou repositorio
- transformar `internal/shared` em dominio escondido

## Comunicacao entre modulos

### Preferencia

- eventos internos in-process como mecanismo principal

### Excecoes aceitas na Phase 3

- verificacao de existencia de customer para emissao de invoice
- port publico de `billing` para retry manual em `POST /payments`

### Contratos vivos da referencia atual

- `InvoiceCreated -> BillingRequested`
- `BillingRequested -> PaymentApproved | PaymentFailed`
- `PaymentApproved -> InvoicePaid`

## Estrutura padrao sugerida por modulo

```text
internal/<module>/
├── domain/
│   ├── entities/
│   ├── events/
│   └── repositories/
├── application/
│   ├── dto/
│   ├── handlers/
│   ├── ports/
│   └── usecases/
├── infrastructure/
│   ├── http/
│   ├── mappers/
│   └── persistence/
└── module.go
```

Aceitar pequenas variacoes de nomenclatura apenas quando ja existirem no codigo e nao aumentarem ambiguidade.

## Criterios para evolucao estrutural

Introduza nova abstracao apenas quando pelo menos um destes sinais existir:

- duplicacao relevante de regra ou fluxo
- necessidade real de troca de adapter
- fronteira de modulo ficando difusa
- necessidade de teste ou isolamento mais claro

## ADR e diagramas

Exigem atualizacao quando houver:

- novo modulo ativo
- nova fronteira de integracao
- mudanca na comunicacao entre modulos
- decisao estrutural dificil de reverter
- alteracao importante no fluxo ponta a ponta
