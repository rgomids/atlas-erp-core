# Skill: Documentation and Governance

## Quando usar

Use esta skill sempre que a mudança afetar entendimento, operação ou evolução do sistema.

## Contexto mínimo a carregar

- `.agents/rules/40-documentation.md`
- `.agents/rules/60-delivery.md`
- `.agents/rules/70-phase-governance.md`

## Documentos que normalmente precisam de revisão

- `README.md`
- `CHANGELOG.md`
- `docs/commands.md`
- `docs/adr/`
- `docs/diagrams/`
- `AGENTS.md` e arquivos correlatos em `.agents`

## Perguntas de fechamento

- a fase atual continua corretamente refletida?
- a mudança alterou comando, config, stack ou fluxo operacional?
- houve decisão estrutural que merece ADR?
- o diagrama ficou desatualizado?
- handoff ou checklist de review precisam ser atualizados?

## Critério de saída

- documentação crítica sincronizada
- pendências registradas
- impacto operacional não ficou implícito
