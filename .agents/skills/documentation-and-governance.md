# Skill: Documentation and Governance

## Objetivo

Garantir que a evolução técnica do projeto continue documentada, rastreável e coerente com a arquitetura vigente.

## Documentos obrigatórios

- `AGENTS.md`
- `README.md`
- `CHANGELOG.md`
- `docs/commands.md`
- `docs/adr/`
- `docs/diagrams/`

## Quando atualizar

Atualize a documentação sempre que houver mudança em:

- arquitetura ou limites modulares
- stack tecnológica
- variáveis de ambiente
- estrutura de diretórios
- comandos operacionais
- observabilidade e logging
- estratégia de testes
- novos módulos, services, jobs ou models de referência

## CHANGELOG

Regras:

- registrar toda evolução relevante
- agrupar por versão ou marco
- separar `Added`, `Changed`, `Fixed` e `Removed`
- atualizar no mesmo conjunto de mudanças do código

## README

Deve refletir:

- fase atual
- visão geral do sistema
- stack
- setup local
- variáveis de ambiente
- comandos
- estratégia de testes
- links de documentação relevante

## ADR e diagramas

- decisão estrutural relevante exige ADR
- mudança arquitetural relevante exige atualização dos diagramas Mermaid/C4

## Checklist pós-implementação

- limites modulares preservados
- domínio sem dependência de infraestrutura
- testes atualizados
- logs e correlation ID preservados
- `AGENTS.md` revisado
- skill/role afetado revisado
- `README.md` atualizado
- `CHANGELOG.md` atualizado
- `docs/commands.md` atualizado se houve impacto operacional
- ADR e diagramas atualizados quando necessário
