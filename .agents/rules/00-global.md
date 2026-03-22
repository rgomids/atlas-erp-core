# Rule: Global

## Objetivo

Definir o comportamento padrão de qualquer agente que atue neste repositório.

## Regras universais

- Trabalhar com **objetividade, rastreabilidade e contexto mínimo**.
- Diferenciar explicitamente:
  - **fato**: confirmado em código, documento ou comando
  - **hipótese**: inferência ainda não validada
  - **decisão**: escolha assumida para seguir com segurança
- Preferir mudanças **pequenas, reversíveis e verificáveis**.
- Não inventar contexto ausente nem “preencher” lacunas com detalhes implícitos.
- Não ampliar escopo sem registrar o motivo.
- Não tratar documentação como opcional quando a mudança altera entendimento do sistema.

## Conduta esperada

- Ler primeiro o roteador (`AGENTS.md`) e depois apenas o necessário.
- Explicar riscos, trade-offs e pendências quando existirem.
- Preservar o que já está funcionando antes de introduzir abstrações novas.
- Manter consistência entre código, comandos, runtime e documentação.
- Tratar conflito entre docs e implementação como sinal para revisão explícita.

## Guardrails

- Não mover regra de negócio para handler, adapter, middleware ou utilitário transversal.
- Não usar `internal/shared` para esconder acoplamento entre módulos.
- Não assumir que banco compartilhado permite integração livre entre bounded contexts.
- Não concluir tarefa sem alguma evidência de validação compatível com o impacto.

## Registro de saída

Ao finalizar, deixar claro:

- o que mudou
- o que foi validado
- o que ficou pendente
- o que exige revisão posterior
