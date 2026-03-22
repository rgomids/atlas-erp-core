# Rule: Architecture

## Objetivo

Preservar a coerência do modular monolith e evitar erosão arquitetural na evolução da Phase 1.

## Arquitetura adotada

- **Modular Monolith**
- **DDD**
- **Clean Architecture**
- **Ports and Adapters**

## Módulos de referência

### Ativos

- `customers`
- `invoices`
- `payments`

### Em scaffold

- `billing`

## Regra de ouro

> Se um módulo depender da implementação interna de outro módulo, a arquitetura está quebrada.

## Direção de dependências

```text
interface -> application -> domain
```

### Permitido

- `infrastructure` depende de `application` e `domain`
- `application` depende de `domain`
- `domain` não depende de infraestrutura
- módulo A consome apenas contrato público de módulo B

### Proibido

- importar `infrastructure` de outro módulo
- acessar tabela de outro módulo sem contrato explícito
- colocar regra de negócio em handler HTTP, mapper, middleware ou repositório
- transformar `internal/shared` em domínio escondido

## Comunicação entre módulos

### Preferência

- eventos internos in-process quando a modelagem justificar desacoplamento

### Exceção aceita na Phase 1

- contratos síncronos explícitos e pequenos, quando necessários para fechar o fluxo já ativo

### Contratos já conhecidos na referência atual

- verificação de existência de customer para emissão de invoice
- contrato de pagamento para atualização de invoice

## Estrutura padrão sugerida por módulo

```text
internal/<module>/
├── domain/
│   ├── entities/
│   ├── valueobjects/
│   └── repositories/
├── application/
│   ├── dto/
│   ├── ports/
│   └── usecases/
├── infrastructure/
│   ├── http/
│   ├── mappers/
│   └── persistence/
└── module.go
```

Aceitar pequenas variações de nomenclatura apenas quando já existirem no código e não aumentarem ambiguidade.

## Critérios para evolução estrutural

Introduza nova abstração apenas quando pelo menos um destes sinais existir:

- duplicação relevante de regra ou fluxo
- necessidade real de troca de adapter
- fronteira de módulo ficando difusa
- necessidade de teste ou isolamento mais claro

## ADR e diagramas

Exigem atualização quando houver:

- novo módulo
- nova fronteira de integração
- mudança na comunicação entre módulos
- decisão estrutural difícil de reverter
- alteração importante no fluxo ponta a ponta
