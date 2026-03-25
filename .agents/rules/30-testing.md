# Rule: Testing

## Objetivo

Garantir validação proporcional ao risco e preservar o compromisso de TDD da foundation e da Phase 1.

## Regra principal

Toda nova regra de negócio deve nascer orientada por teste.  
Todo bug corrigido deve ganhar um teste que falha antes e passa depois.

## Ciclo obrigatório

1. escrever um teste que falha
2. implementar o mínimo para fazê-lo passar
3. refatorar preservando comportamento

## Tipos obrigatórios de teste

- **unitário** para domínio e regras puras
- **integração** para persistência, migrations e adapters relevantes
- **funcional/E2E** para fluxos críticos expostos na borda

## Mapeamento por camada

- `domain`: invariantes, entidades, value objects e regras críticas
- `application`: orquestração de use cases, cenários de erro e contratos
- `infrastructure`: integração com banco, handlers, migrations e adapters
- fluxo ponta a ponta: cenários funcionais do core domain

## Regras práticas

- teste deve validar comportamento observável
- mock não pode esconder regra de negócio
- fixture não pode virar fonte de acoplamento implícito
- teste frágil deve ser reescrito, não “tolerado”
- `testcontainers-go` continua sendo o padrão para cenários com infraestrutura real local

## Evidência mínima antes de concluir

Escolher o menor conjunto de validação que ainda seja honesto com o risco:

- `rtk make test-unit`
- `rtk make test-integration`
- `rtk make test-functional`
- `rtk make test`

Quando algum teste não puder ser executado, registrar claramente:

- o que não rodou
- por que não rodou
- qual risco permanece aberto
