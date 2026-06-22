# Orientações para agentes de IA

Antes de analisar, planejar ou alterar este projeto, leia integralmente:

1. `ai-memory/project-memory.md`
2. `project-docs/product-decisions.md`
3. `project-docs/acceptance-criteria.md`
4. `project-docs/domain-model.md`
5. `project-docs/api-contracts.md`

## Regra de continuidade

Sempre que o responsável pelo projeto definir ou alterar uma regra de negócio,
premissa, prioridade, escopo ou decisão técnica:

1. atualize a memória em `ai-memory/project-memory.md`;
2. atualize a documentação correspondente em
   `project-docs/product-decisions.md`;
3. registre definições ainda incompletas como pendências, sem inventar
   comportamentos;
4. mantenha a origem e a data da decisão;
5. sinalize conflitos com decisões anteriores antes de implementar.

## Separação documental

- `docs/` contém material oficial fornecido pela empresa e não deve ser
  alterado nem versionado.
- `ai-memory/` é contexto operacional para modelos de IA.
- `project-docs/` é a documentação autoral e versionada do projeto.
- O README final deve conter apenas apresentação, execução, funcionalidades,
  limitações e links relevantes; ele não substitui os arquivos acima.

Nenhuma decisão marcada como pendente pode ser tratada como requisito fechado.
