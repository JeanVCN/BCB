# Escopo e decisões do projeto BCB

Este documento registra as decisões autorais do projeto. Ele complementa os
requisitos fornecidos pela empresa, sem modificar os arquivos oficiais em
`docs/`, e evoluirá durante o desenvolvimento.

## Estado do projeto

- Data da consolidação: 2026-06-23.
- Fase: fluxo principal de conversa, envio, cobrança, fila simples, worker,
  retry e estorno implementado localmente.
- Próxima fase recomendada: validação ponta a ponta, testes específicos,
  polimento de UX e, se houver tempo, solicitação de mudança de plano.

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
| Telefone de destinatário | Normalizar separadores, mas exigir resultado E.164 com `+` e 8 a 15 dígitos | Manter previsibilidade sem aceitar número local ambíguo |
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
| README evolutivo | Atualizar em cada commit relevante e distinguir planejado de entregue | Manter a apresentação confiável durante toda a construção |
| Organização inicial | Monorepo com backend, frontend e infraestrutura na raiz | Facilitar integração e execução do desafio |
| Runtime HTTP | Timeouts, logs estruturados e shutdown gracioso | Partir de um servidor previsível e operável |
| Health checks | Liveness do processo e readiness de PostgreSQL/Redis | Separar processo vivo de dependências essenciais disponíveis |
| Migrations | SQL embutido no backend e aplicado pela API/bootstrap | Simplificar execução Docker e instalação limpa |
| Acesso PostgreSQL | `pgx`/`pgxpool` | Driver idiomático, explícito e sem ORM prematuro |
| Acesso Redis | `go-redis` | Cliente mantido e suficiente para rate limit e locks futuros |
| Token de sessão | JWT HS256 com segredo mínimo de 32 caracteres | Stateless simples para o desafio, sem refresh inicial |
| Hash de senha | Argon2id com salt aleatório | Resistência adequada para senhas sem armazenar segredo em claro |
| Rate limit de login | Redis, três falhas antes de bloqueio temporário | Demonstra proteção básica sem persistir senha ou revelar motivo |
| Módulos backend | `internal/modules`, com cada módulo agrupando repository, service, handler e rotas | Manter coesão por domínio e evitar duplicação em camadas paralelas |
| Persistência interna | Usar `Repository` em vez de `Store` | Comunicar melhor a abstração de acesso aos dados/agregados do domínio |
| Composição da aplicação | `main` abre conexões; `modules` monta dependências internas de cada módulo | Manter lifecycle centralizado sem poluir a entrada da aplicação |
| Camada HTTP | Roteador, middlewares e respostas compartilhadas em `httpserver` | Manter transporte HTTP comum separado dos handlers de domínio |
| Migration automática | `RUN_MIGRATIONS=true` por padrão, configurável | Facilitar avaliação local sem impedir operação controlada por pipeline |
| Idempotência financeira | Tabela genérica de registros idempotentes para mutações administrativas | Reaproveitar a proteção em crédito e ajuste de limite sem confundir limite com movimentação financeira |
| Ajuste de limite | Registrar em auditoria, não como transação financeira | Limite não movimenta dinheiro; transações ficam reservadas a crédito, débito, consumo e estorno |
| Worker inicial | Rodar junto da API, consumindo jobs persistentes no PostgreSQL | Entregar o core com simplicidade e preservar caminho para extrair worker dedicado |
| Simulação de envio | Sucesso por padrão; `[fail]` força falha permanente; `[retry]` força falha transitória até estorno | Permitir demonstrar sucesso, retry e estorno sem custo de provedor externo |

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

Na implementação atual, crédito pré-pago e ajuste de limite pós-pago já estão
disponíveis para administradores e exigem `Idempotency-Key`. Créditos geram
registros imutáveis em `financial_transactions`; ajustes de limite são
auditados em `audit_events`, porque não representam entrada, saída ou reversão
de dinheiro. A conversão de plano permanece como etapa posterior.

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

Na implementação atual, o worker simples roda no mesmo processo da API e
consome `dispatch_jobs` persistentes no PostgreSQL usando bloqueio transacional.
Mensagens comuns simulam sucesso. Para demonstração controlada, `[fail]` no
conteúdo gera falha permanente e `[retry]` gera falhas transitórias até esgotar
as quatro tentativas totais, quando ocorre a falha definitiva e o estorno
idempotente.

## Segurança de acesso

O cadastro da empresa deve solicitar CPF ou CNPJ e senha. A senha deverá ter de
9 a 128 caracteres, sem truncamento silencioso, e nunca ser armazenada em texto
puro. O mínimo foi reduzido de 15 para 9 caracteres em 2026-06-23 por decisão
do responsável do projeto, considerando o contexto de desafio técnico e a
intenção de manter a senha apenas acima de oito caracteres. O projeto mantém a
exigência previamente definida de letras, números e caractere especial, embora
as recomendações atuais privilegiem comprimento e bloqueio de senhas
comprometidas em vez de composição obrigatória.

Também serão necessários hash resistente com salt, bloqueio de senhas comuns ou
comprometidas e limitação progressiva das tentativas de autenticação.

Na implementação inicial, o hash resistente é Argon2id, o bloqueio de senhas
comuns usa uma lista local mínima e a limitação de tentativas usa Redis. Essa
lista local não substitui uma base real de senhas vazadas, mas atende ao escopo
do desafio sem adicionar dependência externa.

Recuperação de senha fica fora do escopo inicial.

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
- Definida senha com limite máximo de 128 caracteres, sem truncamento
  silencioso, e preservada a composição exigida pelo projeto.
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
- Definido que o README evolui com os commits relevantes e documenta apenas o
  estado efetivamente verificável do projeto.
- Implementada e validada a fundação em monorepo com Go/Gin,
  React/TypeScript, PostgreSQL, Redis e Docker Compose.

### 2026-06-23

- Implementadas migrations iniciais para contas, usuários, perfis financeiros
  e auditoria.
- Integrados PostgreSQL e Redis ao backend, com readiness real em
  `/health/ready`.
- Implementado bootstrap administrativo idempotente por comando.
- Implementados autocadastro, login, JWT, Argon2id, validação de CPF/CNPJ,
  restrição de cliente inativo e rate limit de login em Redis.
- Implementadas rotas administrativas para listar, ativar, rejeitar e inativar
  clientes, com auditoria.
- Implementada interface inicial para cadastro, login, painel administrativo e
  estado de espera de aprovação.
- Reduzido o mínimo de senha de 15 para 9 caracteres por se tratar de um
  desafio técnico, mantendo exigência de letras, números e caractere especial.
- Reorganizado o backend em `internal/modules`, inicialmente com `access` e
  `accounts`, separando autenticação/usuários do ciclo da empresa cliente.
- Renomeada a abstração de persistência de `Store` para `Repository`.
- Definido que cada módulo agrupa seu repository, service, handler e
  `RegisterRoutes`.
- Definido que `httpserver` mantém apenas roteador, middlewares e respostas
  compartilhadas.
- Removida a composição intermediária em `internal/app`; o registry de módulos
  monta as dependências internas recebendo conexões já abertas pela `main`.
- Tornada a execução automática de migrations configurável por
  `RUN_MIGRATIONS`.
- Implementado o módulo `conversations`, com migrations de destinatários e
  conversas.
- Implementados cadastro/listagem de conversas, unicidade de telefone por
  empresa e retorno da conversa existente para telefone duplicado.
- Implementado endpoint de histórico de mensagens da conversa retornando lista
  vazia até o módulo de mensagens existir.
- Implementada interface inicial do cliente para criar destinatário, listar e
  selecionar conversas.
- Implementado o módulo `billing`, com resumo financeiro do cliente, histórico
  de transações, crédito pré-pago, ajuste de limite pós-pago, idempotência e
  lock Redis por empresa.
- Definido que ajuste de limite pós-pago é auditoria administrativa e não uma
  transação financeira, preservando `financial_transactions` para movimentos
  de dinheiro, consumo e estorno.
- Implementado o módulo `messages`, com envio idempotente, custo por prioridade,
  cobrança pré/pós-paga, persistência de mensagem, job, tentativas, worker
  simples, retry e estorno.
- Definida a simulação local de disparo por conteúdo: sucesso padrão, `[fail]`
  para falha permanente e `[retry]` para falha transitória até esgotar retries.
