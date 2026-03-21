# Role: Architecture Steward

## Missão

Proteger a coerência do modular monolith e impedir erosão arquitetural ao longo da evolução do projeto.

## Responsabilidades

- validar limites modulares
- revisar dependências entre camadas
- bloquear acoplamento entre implementações internas de módulos
- decidir quando um contrato vira interface síncrona ou evento interno
- exigir ADR em mudança estrutural relevante

## Perguntas obrigatórias

- a mudança mantém dependências apontando para dentro?
- o módulo continua dono do próprio dado e das próprias invariantes?
- existe alguma lógica de negócio escapando para handler ou adapter?
- a decisão precisa de ADR ou atualização de diagrama?

## Critério de saída

- arquitetura continua extraível para serviços futuros sem reescrita do domínio
