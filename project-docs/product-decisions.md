# Escopo e decisões do projeto BCB

Este documento registra as decisões autorais do projeto. Ele complementa os
requisitos fornecidos pela empresa, sem modificar os arquivos oficiais em
`docs/`, e evoluirá durante o desenvolvimento.

## Estado do projeto

- Data da consolidação: 2026-06-22.
- Fase: planejamento.
- Desenvolvimento da aplicação: ainda não iniciado.

## Escopo confirmado

| Tema | Decisão | Motivação |
|---|---|---|
| Backend | Go e Gin | Tecnologia definida para o desafio |
| Frontend | React e TypeScript | Tecnologia definida para o desafio |
| Banco | PostgreSQL desde o início | Persistência real e base para regras financeiras |
| Execução | Docker obrigatório | Reprodutibilidade da avaliação |
| Planos | Pré-pago e pós-pago desde o início | Evitar modelo provisório e demonstrar regras centrais |
| Limite pós-pago | Limite total e consumo em campos conceitualmente separados | Rastrear alterações durante o período |
| Fila | Fila simples com worker | Atender ao core com complexidade controlada |
| Evolução da fila | Prever substituição por serviço dedicado/RabbitMQ | Permitir evolução sem reescrever o domínio |
| Mensageria externa | Não integrar inicialmente com SMS/WhatsApp reais | O FAQ permite simulação e provedores gerariam custo externo |
| Status inicial | Processar até `sent` ou `failed` | Corresponde ao ciclo de negócio explicitamente descrito |
| Conversa | Cadastro básico com nome e telefone do destinatário | Viabilizar o fluxo completo sem depender de dados artificiais |
| Cadastro de empresa | Incluir cadastro inicial | Viabilizar autenticação e demonstração ponta a ponta |
| Autenticação | CPF/CNPJ mais senha | Documento isolado não oferece segurança mínima |
| Cliente inativo | Não pode autenticar nem enviar mensagens | Aplicar o bloqueio de forma consistente |
| Financeiro administrativo | Implementar | É relevante para o domínio e para a experiência do candidato |
| Concorrência financeira | Bloqueio distribuído com Redis e transações persistentes | Evitar corrida em escala horizontal sem fazer do lock a única garantia |
| Falhas | Retry e estorno no escopo essencial | Evitar cobrança definitiva de operação não concluída |
| Canal | SMS ou WhatsApp selecionável desde o início | Representar o domínio mesmo sem integração externa |
| Mensagens inbound | Fora da primeira versão | Não existe contrato oficial de entrada |
| Evolução de status | Preparar `delivered` e `read` | Evolução confirmada, sem implementação inicial |
| Mudança de plano | Solicitação do cliente com decisão administrativa | Demonstrar workflow, RBAC e rastreabilidade |
| Notificação administrativa | Contador e lista internos no painel | Entregar valor sem adicionar infraestrutura de tempo real |
| Onboarding | Autocadastro inativo com ativação administrativa | Preservar o fluxo completo e aplicar governança B2B |
| RBAC inicial | Papéis `admin` e `client` | Manter autorização simples e suficiente |
| Primeiro admin | Bootstrap idempotente por comando e variáveis de ambiente | Evitar endpoint público privilegiado |
| Plano no cadastro | Cliente solicita; admin confirma e define condição inicial | Combinar preferência do cliente e controle administrativo |
| Cadastro rejeitado | Preservado e reconsiderável, sem duplicar documento | Manter auditoria e unicidade |
| Sessão | Token de uma hora, sem renovação automática | Manter segurança e escopo inicial simples |
| Ordem da fila | Urgente antes de normal; FIFO dentro da prioridade | Cumprir prioridade sem perder previsibilidade |

## Conclusão sobre envio e recebimento reais

O sistema deve implementar o ato de enviar dentro do produto: validação,
cálculo do custo, movimentação financeira, persistência, fila, processamento
e atualização de status.

Não é necessária integração com um disparador real. O FAQ oficial informa
explicitamente que o envio pode ser simulado por log, delay e mudança de status.
Da mesma forma, não há contrato ou exigência operacional para receber eventos de
um provedor externo. A expressão "enviar/receber" do fluxo Fullstack é atendida
pela comunicação bidirecional entre interface e backend e pela visualização do
histórico; qualquer simulação de mensagens recebidas será uma extensão local.

SMS e WhatsApp continuam como tipos selecionáveis do domínio, mas sem
integração externa no primeiro escopo. A primeira versão também não receberá
mensagens externas. `queued` e `processing` são estados intermediários e os
resultados terminais serão `sent` ou `failed`.

## Regras financeiras consolidadas

### Pré-pago

- O saldo deve cobrir integralmente o custo.
- O débito e o registro financeiro acontecem antes do enfileiramento.
- Falhas definitivas deverão produzir estorno rastreável depois de esgotados os
  retries.

### Pós-pago

- O limite total e o consumo acumulado são informações diferentes.
- A disponibilidade resulta da diferença entre limite total e consumo.
- Alterar o limite total não apaga o consumo anterior.
- Falhas definitivas deverão reverter o consumo correspondente de forma
  rastreável depois de esgotados os retries.

### Administração

O escopo inclui:

- adicionar crédito;
- ajustar limite total;
- consultar saldo, limite e consumo;
- consultar histórico de transações;
- converter planos preservando a rastreabilidade financeira.

### Mudança de plano

- A conversão só pode ser concluída quando não houver valor financeiro
  pendente: saldo pré-pago igual a zero ou consumo pós-pago igual a zero,
  conforme o plano atual.
- O cliente pode solicitar uma mudança quando cumprir essa condição.
- Só pode existir uma solicitação pendente por cliente.
- Os estados da solicitação são `pending`, `approved`, `rejected` e `cancelled`.
- O cliente pode cancelar a solicitação enquanto ela estiver pendente.
- O administrador pode aprovar ou rejeitar a solicitação.
- A aprovação deve revalidar a situação financeira e as permissões no momento
  da alteração.
- Solicitação e todas as transições posteriores devem registrar auditoria.
- Administradores serão notificados por contador e lista no painel; e-mail,
  push e atualização em tempo real ficam fora do escopo inicial.

## Retry e estorno

- Será realizada uma tentativa inicial e, em falhas transitórias, até três
  novas tentativas.
- O intervalo usará backoff exponencial com jitter, inicialmente baseado em 1,
  2 e 4 segundos, configurável e limitado a 30 segundos.
- Falhas permanentes de validação ou negócio não serão repetidas.
- Toda operação repetível deverá ser idempotente.
- Após o esgotamento das tentativas, a mensagem passará para `failed` e o
  estorno será executado uma única vez, ligado à transação original.

## Segurança de acesso

O cadastro da empresa deve solicitar CPF ou CNPJ e senha. A senha deverá ter de
15 a 128 caracteres, sem truncamento silencioso, e nunca ser armazenada em texto
puro. O projeto manterá a exigência previamente definida de letras, números e
caractere especial, embora as recomendações atuais privilegiem comprimento e
bloqueio de senhas comprometidas em vez de composição obrigatória.

Também serão necessários hash resistente com salt, bloqueio de senhas comuns ou
comprometidas e limitação progressiva das tentativas de autenticação.

Detalhes de sessão, recuperação de senha, expiração e proteções contra abuso
serão definidos conforme o escopo de segurança for fechado.

## Entregas posteriores ao essencial

- Reset mensal automatizado.
- Limite de tamanho do conteúdo.
- Paginação e ordenação avançadas.
- RabbitMQ.
- Estados `delivered` e `read`.
- Recebimento simulado mais completo.

## Alternativas de onboarding e RBAC

### Opção 1 — autocadastro inativo e ativação administrativa

Vantagens:

- preserva o fluxo completo de cadastro escolhido para o desafio;
- torna o estado ativo/inativo uma regra visível e testável;
- o cliente cria sua própria senha;
- reduz o trabalho operacional do administrador;
- demonstra RBAC, auditoria e onboarding na apresentação.

Custos:

- exige tela/fila de aprovação;
- precisa controlar cadastros indevidos, enumeração de documentos e ativação
  duplicada;
- a conta ainda não ativada precisa receber resposta segura e clara.

### Opção 2 — cadastro exclusivamente administrativo

Vantagens:

- menor superfície pública e governança B2B mais fechada;
- elimina fila de aprovação e cadastros espontâneos;
- é adequada quando existe contrato comercial prévio.

Custos:

- contradiz parcialmente a decisão de demonstrar cadastro inicial completo;
- aumenta o trabalho administrativo;
- exige convite/definição segura de senha para que o administrador nunca a
  conheça;
- reduz o valor demonstrativo do estado inativo.

### Decisão aprovada

Adotar a opção 1 com dois papéis iniciais: `admin` e `client`. O cliente se
cadastra como inativo e não consegue autenticar nem enviar mensagens até que um
administrador o ative. O administrador também controla crédito, limite, plano e
estado do cliente. Toda ação administrativa deve registrar ator, instante,
operação e valores anteriores/novos.

A opção 1 foi aprovada em 2026-06-22.

## Definições ainda necessárias

Não há definição funcional bloqueante para iniciar a implementação.

## Histórico

### 2026-06-22

- Criado o registro autoral de decisões.
- Docker e PostgreSQL tornados obrigatórios.
- Confirmados os dois planos desde o início.
- Definidos fila simples, worker e evolução opcional para RabbitMQ.
- Incluídos cadastro inicial, senha, novas conversas e administração financeira.
- Incluídos Redis, retry e estorno no escopo.
- Excluída a integração inicial com provedores reais.
- Definida senha entre 15 e 128 caracteres e preservada a composição exigida
  pelo projeto.
- Definidos três retries com backoff exponencial, jitter e estorno posterior.
- Confirmados SMS/WhatsApp selecionáveis e evolução para `delivered`/`read`.
- Excluídas mensagens inbound da primeira versão.
- Registradas as alternativas de onboarding e a recomendação pela opção 1.
- Aprovado o fluxo enxuto de solicitação de mudança de plano, decisão
  administrativa, notificação interna e auditoria.
- Aprovado o autocadastro inativo, com ativação administrativa e papéis
  iniciais `admin` e `client`.
- Aprovados bootstrap administrativo, plano solicitado no cadastro, retenção
  de rejeitados, resposta de espera, sessão de uma hora e prioridade da fila.
- Concluídos o modelo conceitual e os contratos HTTP v1.
