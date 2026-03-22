# Rule: Documentation

## Objetivo

Manter o entendimento do sistema sincronizado com a implementação real.

## Documentos obrigatórios de referência

- `AGENTS.md`
- `README.md`
- `CHANGELOG.md`
- `docs/commands.md`
- `docs/adr/`
- `docs/diagrams/`

## Quando atualizar

Atualizar documentação no mesmo change set quando houver mudança em:

- fase do projeto
- arquitetura ou fronteira modular
- stack ou runtime
- variáveis de ambiente
- comandos operacionais
- observabilidade ou logging
- estratégia de testes
- módulos, services, jobs ou models de referência

## README

Deve refletir, no mínimo:

- fase atual
- visão geral do sistema
- stack
- setup local
- variáveis de ambiente
- comandos
- estratégia de testes
- links de documentação relevante

## CHANGELOG

- registrar comportamento entregue, não só lista de arquivos
- agrupar por versão, marco ou fase
- usar `Added`, `Changed`, `Fixed` e `Removed`
- não deixar mudança arquitetural relevante sem registro

## ADR e diagramas

- decisão estrutural relevante exige ADR
- mudança arquitetural relevante exige atualização de diagrama
- preferir Mermaid; usar C4 quando a visão arquitetural pedir esse nível de clareza

## Regra de consistência

Se a implementação divergir da documentação, corrigir a divergência explicitamente.  
Não deixar documentação “para depois” quando ela for necessária para operar ou evoluir o sistema.
