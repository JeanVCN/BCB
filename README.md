<div align="center">

# Big Chat Brasil

### Comunicação empresarial com controle financeiro e processamento confiável

![Status](https://img.shields.io/badge/status-planejamento%20conclu%C3%ADdo-2563eb)
![Backend](https://img.shields.io/badge/backend-Go%20%2B%20Gin-00ADD8)
![Frontend](https://img.shields.io/badge/frontend-React%20%2B%20TypeScript-3178C6)
![Database](https://img.shields.io/badge/database-PostgreSQL-4169E1)

</div>

## Sobre o projeto

O **Big Chat Brasil (BCB)** é uma aplicação Fullstack de chat para empresas se
comunicarem com seus clientes finais. Além da experiência de conversa, o produto
combina planos pré e pó-pagos, mensagens normais e urgentes, processamento em
fila e rastreabilidade financeira.

Este repositório faz parte de um desafio técnico e foi planejado para demonstrar
não apenas o funcionamento do produto, mas também decisões conscientes sobre
consistência, concorrência, integração e evolução arquitetural.

> **Estado atual:** planejamento funcional e técnico concluído. O código da
> aplicação ainda não foi iniciado.

## Experiência planejada

- Autocadastro de empresas por CPF ou CNPJ, com ativação administrativa.
- Autenticação segura e controle de acesso por papéis.
- Planos pré-pago e pós-pago com histórico financeiro auditável.
- Cadastro de destinatários e organização por conversas.
- Mensagens SMS ou WhatsApp simuladas, normais ou urgentes.
- Fila persistente com worker, prioridade, retry e estorno idempotente.
- Solicitação e aprovação administrativa de mudança de plano.
- Interface responsiva com estados de carregamento, erro e sucesso.

O desafio não exige integração com provedores reais de SMS ou WhatsApp. O
disparo será simulado, preservando todo o fluxo interno de validação, cobrança,
enfileiramento, processamento e atualização de status.

## Tecnologias definidas

| Camada | Tecnologia | Papel no projeto |
|---|---|---|
| Backend | Go + Gin | API, regras de negócio e worker |
| Frontend | React + TypeScript | Interface responsiva e integração |
| Persistência | PostgreSQL | Dados transacionais, fila e auditoria |
| Coordenação | Redis | Bloqueio distribuído de operações financeiras |
| Ambiente | Docker Compose | Execução reproduzível dos serviços |
| Evolução opcional | RabbitMQ | Mensageria dedicada, caso o core esteja concluído |

As tecnologias acima estão **definidas**, mas ainda serão incorporadas ao
repositório durante as próximas etapas.

## Decisões que orientam a solução

- Valores monetários serão representados em centavos, sem ponto flutuante.
- PostgreSQL será a garantia final de consistência transacional.
- Redis complementará o banco na coordenação de instâncias concorrentes.
- Operações financeiras, retries e estornos serão idempotentes.
- Mensagens urgentes terão precedência, preservando FIFO na mesma prioridade.
- A fila inicial manterá uma fronteira preparada para futura adoção do
  RabbitMQ.
- O fluxo completo e integrado tem prioridade sobre funcionalidades opcionais.

## Documentação do projeto

| Documento | Conteúdo |
|---|---|
| [Decisões de produto](project-docs/product-decisions.md) | Escopo, justificativas e histórico de decisões |
| [Critérios de aceite](project-docs/acceptance-criteria.md) | Comportamentos verificáveis e definição de pronto |
| [Modelo conceitual](project-docs/domain-model.md) | Entidades, invariantes, estados e autorização |
| [Contratos HTTP](project-docs/api-contracts.md) | API v1 e integração entre frontend e backend |

## Roadmap

- [x] Levantamento e consolidação dos requisitos.
- [x] Decisões de produto e regras de negócio.
- [x] Critérios de aceite.
- [x] Modelo conceitual e contratos HTTP v1.
- [ ] Ambiente Docker com PostgreSQL e Redis.
- [ ] Cadastro, ativação, RBAC e autenticação.
- [ ] Planos e administração financeira.
- [ ] Conversas e histórico.
- [ ] Envio, fila, worker, retry e estorno.
- [ ] Frontend responsivo e fluxo integrado.
- [ ] Testes, documentação final e preparação da entrega.

## Execução

O ambiente executável ainda não está disponível. Esta seção será atualizada
assim que o primeiro setup com Docker Compose for implementado e validado em um
ambiente limpo.

## Princípios de entrega

- Qualidade e clareza antes de quantidade de funcionalidades.
- Regras de negócio protegidas por testes proporcionais ao risco.
- Commits pequenos, coerentes e explicáveis durante a apresentação.
- Documentação atualizada junto com o comportamento real da aplicação.
- Limitações e trabalhos futuros descritos de maneira transparente.
