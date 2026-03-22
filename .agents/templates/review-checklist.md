# Review Checklist

## Arquitetura

- [ ] fronteiras modulares preservadas
- [ ] dependências apontando para dentro
- [ ] nenhuma lógica de negócio escapou para handlers/adapters
- [ ] mudança estrutural relevante foi registrada em ADR/diagrama, se aplicável

## Código e runtime

- [ ] naming e organização aderem ao domínio
- [ ] `internal/shared` não recebeu regra de negócio
- [ ] migrations, config, compose e comandos foram tratados quando necessário
- [ ] logs continuam estruturados e sem segredo

## Testes e validação

- [ ] existe validação proporcional ao risco
- [ ] bug corrigido ganhou teste de regressão, se aplicável
- [ ] TDD foi respeitado em regra nova ou crítica, quando cabível
- [ ] limitações de validação ficaram explícitas

## Documentação e entrega

- [ ] `README.md` foi revisado quando houve impacto
- [ ] `CHANGELOG.md` foi atualizado quando houve mudança relevante
- [ ] `docs/commands.md` foi atualizado quando houve impacto operacional
- [ ] `AGENTS.md` / rules / skills afetados foram revisados
- [ ] pendências e riscos foram registrados
