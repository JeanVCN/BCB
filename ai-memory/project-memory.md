# Memória operacional do projeto BCB

> Documento destinado a modelos de inteligência artificial. Deve refletir o
> entendimento vigente do projeto e ser atualizado junto com as decisões do
> produto. Não substitui a documentação oficial nem o README.

## Metadados

- Projeto: Big Chat Brasil (BCB), desafio técnico Fullstack.
- Última consolidação: 2026-06-24.
- Estado atual: cadastro, ativação, RBAC, autenticação, destinatários,
  conversas, financeiro administrativo básico, zeramento administrativo de
  saldo/consumo para troca de plano e envio de mensagens com cobrança, fila
  persistente, worker, retry, estorno e solicitação de mudança de plano
  implementados localmente.
- Fonte inicial: documentos oficiais locais em `docs/` e definições do
  responsável pelo projeto.

## Protocolo de manutenção

Ao surgir uma nova definição:

1. atualizar este arquivo e `project-docs/product-decisions.md` na mesma
   alteração;
2. distinguir decisão confirmada, hipótese e pendência;
3. registrar a data e a justificativa;
4. nunca converter exemplos da documentação oficial em obrigações sem uma
   decisão explícita;
5. preservar decisões substituídas no histórico da documentação do projeto.

## Objetivo vigente

Construir uma aplicação Fullstack de chat empresarial que permita cadastrar e
autenticar empresas clientes, manter conversas com seus clientes finais, enviar
mensagens, controlar custos pré e pós-pagos, processar mensagens em fila e
acompanhar seu estado. A entrega deve favorecer correção, rastreabilidade,
integração completa e capacidade de explicação durante apresentação e live
coding.

## Vocabulário adotado

- **Empresa cliente**: titular da conta no BCB, identificada por CPF ou CNPJ.
- **Cliente final/destinatário**: pessoa com quem a empresa mantém uma
  conversa; inicialmente possui nome e telefone.
- **Mensagem enviada**: mensagem solicitada pela empresa cliente para um
  destinatário.

Essa nomenclatura deve ser preservada para evitar a ambiguidade do termo
"cliente" encontrada nos documentos oficiais.

## Decisões confirmadas

### Tecnologias e execução

- Backend: Go com Gin.
- Frontend: React com TypeScript.
- Persistência: PostgreSQL desde a primeira versão.
- Docker deve ser implementado para a execução do ambiente.
- Redis integrará o controle de concorrência das operações financeiras por
  meio de bloqueio distribuído, visando segurança no escalonamento horizontal.
- RabbitMQ não pertence ao escopo inicial. É uma evolução condicionada a
  tempo disponível.
- O repositório é um monorepo com `backend/`, `frontend/`, infraestrutura e
  documentação na raiz.
- A fundação usa Go 1.25+, Gin 1.12, React 19.2, TypeScript 6, Vite 8,
  PostgreSQL 17 e Redis 8.
- O servidor HTTP possui timeouts, logs estruturados e shutdown gracioso.
- `/health/live` representa liveness do processo HTTP.
- `/health/ready` verifica conectividade com PostgreSQL e Redis.
- Migrations SQL ficam embutidas no binário do backend e são aplicadas na
  inicialização da API e no comando de bootstrap administrativo.
- O acesso ao PostgreSQL usa `pgx`/`pgxpool`; o acesso ao Redis usa
  `go-redis`.
- Tokens de sessão usam JWT assinado com HS256 e segredo obrigatório de pelo
  menos 32 caracteres.
- Senhas são armazenadas com Argon2id, salt aleatório e parâmetros fixos no
  código para o escopo inicial.
- A limitação de tentativas de login usa Redis com janela de falhas e bloqueio
  temporário depois de três falhas.
- O ambiente Docker completo foi integrado ao backend: API, frontend,
  PostgreSQL e Redis se comunicam pelo Compose.
- A API pode aplicar migrations automaticamente na inicialização quando
  `RUN_MIGRATIONS=true`, padrão do ambiente de desafio. Em ambientes onde a
  migração for controlada por pipeline ou comando operacional separado, esse
  comportamento pode ser desligado.
- O backend é organizado em `internal/modules`. Cada módulo agrupa seu próprio
  repository, service, handler e registro de rotas.
- O módulo `access` concentra autenticação, usuários, sessão, senha, token,
  rate limit, contexto de claims e bootstrap administrativo.
- O módulo `accounts` concentra empresa cliente, autocadastro, ativação,
  rejeição, inativação e auditoria desse ciclo.
- O módulo `billing` concentra consulta financeira, crédito pré-pago, ajuste
  de limite pós-pago, histórico financeiro, cobrança de mensagem, estorno,
  idempotência e lock Redis.
- O módulo `messages` concentra envio, acionamento da cobrança, histórico da
  conversa, dispatch jobs persistentes, tentativas, worker simples, retry e
  acionamento do estorno.
- O worker de mensagens roda como processo/serviço independente da API. A API
  continua responsável por validar, cobrar, registrar mensagem e criar job; o
  worker consome `dispatch_jobs` em outro ciclo de vida, reduzindo risco de
  interromper disparos durante atualização do serviço HTTP.
- Constantes de domínio como plano, canal, prioridade, status e tipos
  financeiros devem ficar centralizadas no backend em `internal/domain` e
  espelhadas no frontend em `src/domain.ts`. Um gerador/schema comum pode ser
  adotado futuramente se o custo compensar.
- A arquitetura do frontend deve permanecer simples e justificável para uma
  aplicação React de pequeno/médio porte: `App.tsx` fica como composição por
  sessão/papel; telas e fluxos ficam em `src/features`; componentes
  reutilizáveis em `src/components`; tipos de UI em `src/types`; helpers puros
  em `src/utils`; integração HTTP permanece centralizada em `src/api.ts`.
- Modais devem priorizar legibilidade e acessibilidade visual com superfície
  opaca, sem estética glass/translúcida que prejudique leitura sobre o
  conteúdo da página.
- A persistência dos contextos usa o nome `Repository`, não `Store`, por
  representar a abstração de acesso ao agregado/dados do domínio.
- A camada `httpserver` contém apenas roteador, middlewares e helpers de
  resposta compartilhados.
- Cada módulo expõe seu próprio `RegisterRoutes`, permitindo que futuros
  módulos, como conversas, mensagens e financeiro, adicionem rotas sem criar
  handlers híbridos nem duplicar domínios em `httpserver/handlers`.
- O pacote `modules` agrega os módulos e monta suas dependências internas a
  partir das conexões já abertas pela `main`.
- A `main` não deve montar manualmente todos os repositories/services e também
  não deve delegar abertura de PostgreSQL/Redis aos módulos. Ela permanece
  responsável por configuração, conexões e ciclo de vida do processo.

### Documentação

- `docs/` é material oficial da empresa, foi reduzido aos documentos da vaga
  Fullstack, não deve ser corrigido e não será commitado.
- As referências ausentes a perfis Backend e Frontend não são lacunas do
  escopo deste projeto.
- A memória para IA e a documentação do projeto devem permanecer separadas do
  README e dos documentos oficiais.
- O README principal deve funcionar como documento público de entrega do
  desafio: descrição, premissas, tecnologias, execução, testes, funcionalidades,
  limitações e links relevantes.
- A documentação OpenAPI 3.0 da API deve ficar em
  `project-docs/openapi.yaml` e ser referenciada no README como contrato
  navegável complementar ao `project-docs/api-contracts.md`.
- O backend deve servir a documentação interativa em `/swagger.html` e o YAML
  bruto em `/openapi.yaml`. O frontend/Nginx deve proxyar essas mesmas rotas
  para facilitar o acesso local pela porta do frontend.
- `project-docs/openapi.yaml` é a fonte autoral do contrato; o backend embute
  uma cópia em `internal/httpserver/openapi.yaml`, sincronizada com
  `go generate ./internal/httpserver` quando o contrato muda.
- O histórico de fases, correções e evolução incremental deve ficar separado em
  `HISTORICO_IMPLEMENTACAO.md`, preservando o README antigo como registro
  consultável sem poluir a apresentação principal.

### Contas, cadastro e segurança

- Deve existir cadastro inicial da empresa cliente para que o fluxo completo
  possa ser demonstrado.
- A empresa será identificada por CPF ou CNPJ, mas o documento sozinho não
  autentica o acesso.
- O acesso exigirá senha.
- A senha terá entre 9 e 128 caracteres e não poderá ser truncada
  silenciosamente. O mínimo foi reduzido de 15 para 9 caracteres em
  2026-06-23 por se tratar de um projeto de desafio técnico, mantendo a regra
  de ser maior que oito caracteres.
- Por decisão do projeto, a senha deverá combinar letras, números e caractere
  especial. Essa composição é mais restritiva que as recomendações atuais de
  NIST/OWASP, que priorizam comprimento e bloqueio de senhas comprometidas; o
  trade-off deve ser explicado na apresentação.
- Senhas devem ser armazenadas somente como hash resistente e com salt.
- A autenticação deverá ter limitação de tentativas, e senhas comuns ou
  conhecidas como comprometidas deverão ser rejeitadas.
- Empresa cliente inativa não pode autenticar nem enviar mensagens.
- O onboarding adotado é o autocadastro: toda nova empresa cliente entra
  inativa e depende de ativação por um usuário administrador.
- O RBAC inicial possui os papéis `admin` e `client`.
- O administrador controla ativação/inativação, créditos, limites, planos e
  solicitações de mudança.
- Toda ação administrativa deve registrar ator, instante, operação e valores
  anteriores/novos quando aplicável.
- O primeiro administrador será criado por comando idempotente de bootstrap,
  com credenciais fornecidas por variáveis de ambiente e sem endpoint público.
- No autocadastro, o cliente informa o plano desejado. Na ativação, o
  administrador confirma o mesmo plano solicitado e define saldo ou limite
  inicial.
- Cadastro rejeitado permanece registrado para auditoria, não libera novo
  cadastro com o mesmo CPF/CNPJ e pode ser reconsiderado por administrador.
- Depois do autocadastro, a resposta informa que o cadastro foi recebido e
  aguarda aprovação, sem prometer prazo.
- A sessão inicial usa token com uma hora de validade, sem renovação automática;
  após expirar, é necessário autenticar novamente.
- CPF/CNPJ duplicado nunca cria uma segunda conta, inclusive após rejeição.
- Telefone de destinatário é único dentro de cada empresa; tentar abrir nova
  conversa para o mesmo telefone recupera a conversa existente.
- Telefones de destinatários são normalizados aceitando separadores visuais,
  mas devem resultar em E.164 com `+` e 8 a 15 dígitos.

### Conversas e mensagens

- Deve ser possível cadastrar uma nova conversa com, no mínimo, nome e telefone
  do cliente final.
- Não haverá integração inicial com provedor real de SMS ou WhatsApp.
- O envio obrigatório é interno e simulado: validar, cobrar, registrar,
  enfileirar, processar e atualizar o status.
- Não existe requisito oficial para recebimento por provedor externo. Mensagens
  recebidas usadas na demonstração podem vir de dados iniciais ou mecanismo
  interno a ser definido posteriormente.
- O canal SMS/WhatsApp pertence ao domínio documentado, embora a integração
  externa não pertença ao escopo inicial.
- O canal será selecionável desde a primeira versão.
- O ciclo inicial termina em `sent` ou `failed`.
- A estrutura deve permitir evolução posterior para `delivered` e `read`.
- Não haverá mensagem recebida/inbound nem status correspondente na primeira
  versão. `queued` e `processing` permanecem estados internos intermediários;
  os resultados terminais visíveis são `sent` e `failed`.

### Fila e processamento

- A primeira versão terá fila simples e worker.
- Mensagens urgentes têm precedência sobre normais, preservando FIFO dentro de
  cada prioridade.
- As responsabilidades da fila devem permanecer desacopladas o suficiente para
  futura substituição por um serviço dedicado/RabbitMQ.
- Retry faz parte do escopo essencial.
- Cada mensagem terá uma tentativa inicial e até três retries.
- Retries ocorrerão apenas para falhas transitórias, com backoff exponencial e
  jitter. As bases iniciais serão 1, 2 e 4 segundos, configuráveis e com teto de
  30 segundos.
- Falhas permanentes de validação ou negócio não entram em retry.
- Processamento, retry e efeitos financeiros devem ser idempotentes.
- A fila simples foi implementada com `dispatch_jobs` persistidos em
  PostgreSQL, usando `FOR UPDATE SKIP LOCKED` para evitar processamento
  simultâneo do mesmo job.
- O worker roda em processo separado, com polling curto, e a lógica permanece
  no módulo `messages` para manter coesão de domínio.
- A simulação de disparo considera mensagens normais como sucesso. Conteúdos
  com `[fail]` geram falha permanente e conteúdos com `[retry]` geram falhas
  transitórias até esgotar as tentativas, permitindo demonstrar retry e estorno
  sem provedor externo.

### Financeiro

- Os planos pré-pago e pós-pago existirão desde a primeira versão.
- Pré-pago exige saldo suficiente e débito antes do enfileiramento.
- Pós-pago deve armazenar separadamente o limite total e o limite consumido.
- A separação preserva a rastreabilidade quando o limite total é alterado no
  decorrer do período de cobrança.
- Administração financeira integra o escopo: créditos, limites, consultas,
  conversão de plano e histórico de transações.
- O cliente pode solicitar uma mudança de plano mesmo quando existir valor
  financeiro pendente. A conversão só pode ser aprovada pelo administrador
  quando não existir valor financeiro pendente: saldo pré-pago igual a zero ou
  consumo pós-pago igual a zero, conforme o plano atual.
- Cada cliente pode manter apenas uma solicitação de mudança pendente.
- A solicitação possui os estados `pending`, `approved`, `rejected` e
  `cancelled`.
- O cliente pode cancelar uma solicitação enquanto ela estiver pendente.
- Usuários administradores podem aprovar ou rejeitar solicitações e verão uma
  notificação interna, por contador e lista no painel administrativo.
- Não haverá inicialmente notificação por e-mail, push, WebSocket ou outro
  canal externo.
- A aprovação deve revalidar saldo/consumo zerado e permissão no momento da
  operação, pois a situação pode ter mudado desde a solicitação.
- Solicitação, cancelamento, aprovação e rejeição devem ser auditáveis.
- Estorno em caso de falha faz parte do escopo essencial.
- O estorno ocorre somente depois de esgotadas as tentativas configuradas e a
  mensagem ser declarada definitivamente `failed`.
- O estorno deve ser idempotente, executado no máximo uma vez e registrado no
  histórico financeiro com referência à cobrança original.
- Redis será usado como coordenação distribuída; a consistência financeira
  também deverá ser garantida pela persistência transacional, sem depender
  exclusivamente do lock.
- O financeiro administrativo básico foi implementado antes do envio de
  mensagens: cliente consulta resumo e histórico; administrador adiciona
  crédito pré-pago, ajusta limite pós-pago e zera saldo pré-pago ou consumo
  pós-pago para permitir que o administrador aprove mudança de plano quando
  necessário.
- Mutações financeiras administrativas exigem `Idempotency-Key`. A repetição
  da mesma chave com o mesmo corpo não reaplica o efeito; a mesma chave com
  corpo diferente retorna conflito.
- Créditos pré-pagos geram `financial_transactions` do tipo `credit`.
- Ajustes de limite pós-pago não entram como transação financeira porque não
  movimentam dinheiro; eles são registrados em `audit_events`.
- Zeramentos administrativos exigem `Idempotency-Key`, usam lock Redis e são
  auditados. Quando há valor a compensar, saldo pré-pago zerado gera transação
  `debit` e consumo pós-pago zerado gera `consumption_reversal`.
- Redis bloqueia mutações financeiras por empresa durante a operação, e o
  PostgreSQL mantém transação, constraints e idempotência como garantia final.

### Escopo posterior

Somente depois do essencial estar estável, avaliar:

- reset mensal do plano pós-pago;
- limite de conteúdo;
- ordenação configurável;
- paginação;
- RabbitMQ;
- estados posteriores a `sent`;
- recebimento simulado mais completo.

## Prioridade inicial de desenvolvimento

1. preparar ambiente Docker com PostgreSQL e Redis;
2. implementar cadastro e autenticação da empresa cliente;
3. implementar cadastro/listagem de conversas;
4. implementar os dois planos e administração financeira;
5. implementar envio, fila simples, worker, retry, estorno e status;
6. integrar a experiência completa no frontend;
7. implementar solicitação de mudança de plano;
8. adicionar testes, refinamentos e evoluções condicionadas ao tempo.

## Estratégia de versionamento

- O desenvolvimento individual ocorrerá diretamente na branch `main`.
- Não criar branches permanentes separadas para backend e frontend, evitando
  integração tardia e complexidade sem benefício para o desafio.
- Usar branches curtas somente para experimentos arriscados ou opcionais que
  possam ser descartados, como `experiment/rabbitmq`.
- Manter commits pequenos, coerentes, explicáveis e preferencialmente com build
  e testes verdes.
- Incluir migrations, testes e ajustes documentais junto da funcionalidade que
  os exige, sem criar commits artificiais por tipo de arquivo.
- Usar os prefixos `docs`, `chore`, `feat`, `fix`, `test` e `refactor` nas
  mensagens de commit.
- Evitar commits genéricos ou intermediários como `wip`, `ajustes` e `fix` sem
  contexto.
- O primeiro commit estabelece o marco documental anterior ao código, com a
  mensagem `docs: define product scope, decisions and contracts`.
- Fases previstas para o histórico: planejamento; fundação; cadastro/RBAC/auth;
  financeiro; conversas; mensagens/fila; mudança de plano; estabilização e
  entrega.
- Tags opcionais de marco: `planning-complete` após o planejamento e `v0.1.0`
  ou `mvp-complete` após o fluxo mínimo integrado.
- O README oficial deve evoluir junto de cada commit relevante, refletindo
  somente funcionalidades realmente disponíveis, instruções verificadas,
  limitações vigentes e o estado atual do roadmap.
- Nunca apresentar no README uma tecnologia planejada como implementada nem
  publicar instruções de execução que ainda não foram validadas.
- A partir de 2026-06-24, detalhes históricos extensos ficam em
  `HISTORICO_IMPLEMENTACAO.md`; o README principal deve responder diretamente
  ao que os documentos oficiais em `docs/` pedem para avaliação.

## Implementação atual

- A API possui autocadastro público em `/api/v1/auth/register`.
- Login público em `/api/v1/auth/login` emite token de uma hora para admin ou
  cliente ativo.
- Cliente pendente, inativo ou rejeitado não autentica.
- Rotas protegidas exigem `Authorization: Bearer <token>`.
- `/api/v1/me` retorna a identidade autenticada.
- Admin lista clientes e pode ativar, rejeitar ou inativar.
- Ativação cria/atualiza o perfil financeiro inicial conforme o plano
  solicitado no cadastro.
- Ativação, rejeição e inativação registram auditoria administrativa.
- O frontend já apresenta telas de login, cadastro, espera de aprovação,
  painel administrativo básico e área inicial do cliente.
- O painel administrativo básico permite listar clientes, ativar e rejeitar.
- O primeiro admin é criado por `admin-bootstrap`, comando idempotente que lê
  credenciais de variáveis de ambiente.
- A criação do admin usa somente o módulo `access`; a administração de
  empresas usa `accounts`. A composição dos repositories, services e handlers
  HTTP fica dentro dos próprios módulos e do registry em `internal/modules`.
- O módulo `conversations` implementa cadastro/listagem de conversas e
  destinatários.
- `POST /api/v1/conversations` cria uma conversa para o cliente autenticado e
  ativo ou retorna a conversa existente quando o telefone já pertence à empresa.
- `GET /api/v1/conversations` lista somente conversas da empresa autenticada.
- `GET /api/v1/conversations/{conversationId}/messages` valida propriedade da
  conversa e retorna histórico vazio até o módulo de mensagens ser implementado.
- O frontend de cliente já permite criar destinatário, listar conversas,
  selecionar conversa e visualizar estado vazio de mensagens.
- O módulo `billing` já expõe resumo financeiro do cliente em
  `/api/v1/billing` e histórico em `/api/v1/billing/transactions`.
- O admin já pode adicionar crédito pré-pago por
  `/api/v1/admin/clients/{clientId}/credits`, com lock Redis e idempotência.
- O admin já pode ajustar limite pós-pago por
  `/api/v1/admin/clients/{clientId}/postpaid-limit`, impedindo limite menor
  que o consumo atual.
- O admin já pode consultar o perfil financeiro vigente de um cliente ativo por
  `/api/v1/admin/clients/{clientId}/billing` e zerar o saldo pré-pago ou o
  consumo pós-pago por `/api/v1/admin/clients/{clientId}/zero-balance`, com
  idempotência, auditoria e registro financeiro compensatório quando aplicável.
- O admin já consulta histórico financeiro por
  `/api/v1/admin/clients/{clientId}/financial-transactions`.
- O frontend já apresenta resumo financeiro para o cliente, histórico
  financeiro inicial e ações administrativas simples de crédito/limite.
- O módulo `messages` já expõe `POST /api/v1/conversations/{conversationId}/messages`
  com `Idempotency-Key`, canal, prioridade, custo calculado, cobrança e job de
  processamento na mesma transação.
- `GET /api/v1/conversations/{conversationId}/messages` retorna o histórico
  real de mensagens da conversa em ordem cronológica.
- Mensagens normais custam 25 centavos e urgentes custam 50 centavos.
- Pré-pago sem saldo suficiente e pós-pago sem disponibilidade não criam
  mensagem, cobrança ou job.
- O worker persiste tentativas em `delivery_attempts`, respeita até quatro
  tentativas totais, agenda retry com backoff 1/2/4 segundos e estorna apenas
  depois da falha definitiva.
- O frontend de cliente já permite enviar mensagem, escolher SMS/WhatsApp,
  escolher prioridade, acompanhar `queued`, `processing`, `sent` e `failed`, e
  atualizar resumo/histórico financeiro.
- A API não inicia mais o worker automaticamente. O ambiente Docker possui um
  serviço `message-worker`, baseado no mesmo build do backend e executando
  `/app/message-worker`.
- A regra financeira de cobrança/estorno pertence ao módulo `billing`. O módulo
  `messages` abre a transação do caso de uso e chama `billing` passando a mesma
  transação, preservando atomicidade entre cobrança, mensagem e job sem mover
  regra financeira para o domínio de mensagens.
- A fronteira entre `messages` e `billing` é o service de billing, exposto por
  uma interface consumida por `messages`. O módulo de mensagens não instancia
  nem depende diretamente do repository concreto de billing.
- O repository de billing mantém o núcleo financeiro transacional em
  `repository.go`, especialmente cobrança e estorno de mensagens. Arquivos
  auxiliares agrupam perfil/administração e transações/idempotência/auditoria.
  A divisão deve evitar tanto arquivos excessivamente longos quanto
  microarquivos difíceis de navegar.
- Comparações de enumerações do domínio não devem usar strings soltas. Backend
  deve centralizar esses valores em `internal/domain`; frontend deve espelhá-los
  em `src/domain.ts`.
- Funções e métodos em Go só devem ser exportados quando forem fronteiras reais
  entre pacotes, contratos externos ou exigências da linguagem, como testes
  `Test...`. Handlers, services e repositories usados apenas dentro do próprio
  pacote devem permanecer não exportados.
- O módulo `planchanges` implementa o workflow de mudança de plano com
  `plan_change_requests`, estados `pending`, `approved`, `rejected` e
  `cancelled`, regra de uma solicitação pendente por empresa, auditoria em
  cada transição e aprovação administrativa atômica com alteração do perfil
  financeiro.
- O cliente acessa `/api/v1/plan-change-requests`, consulta a solicitação mais
  recente e cancela pendências próprias. O admin acessa
  `/api/v1/admin/plan-change-requests`, aprova/rejeita pendências e consulta
  `/api/v1/admin/notifications/summary` com contador de cadastros e mudanças
  pendentes.
- Em 2026-06-24 foi corrigido o registro das rotas financeiras administrativas
  do módulo `billing`: como o router já monta o grupo `/admin`, as rotas agora
  são registradas como `/clients/...`, preservando o contrato público
  `/api/v1/admin/clients/...`.
- Em 2026-06-24 foi validado o fluxo E2E em Docker com portas locais
  alternativas em `.env` ignorado pelo Git: frontend `13000`, backend `18080`,
  PostgreSQL `15432` e Redis `16379`. O teste cobriu cadastro, ativação, login,
  conversa, envio com sucesso, `[fail]`, `[retry]`, estornos, saldo zerado,
  solicitação/aprovação de mudança de plano, restart do Compose e persistência
  de login, billing, conversa e mensagens.
- A busca nos logs do Compose por senha, token, bearer, authorization, JWT,
  secret e hash não encontrou exposição sensível no fluxo validado. Os logs do
  backend ainda registram método, rota e IDs na URL pelo `gin.Logger`, o que é
  aceitável para o escopo atual por não incluir corpo, senha ou token.
- O frontend passou por ajuste responsivo em CSS, removendo escala
  tipográfica por viewport, letter-spacing negativo e riscos de overflow em
  IDs, telefones, mensagens e ações administrativas.
- Em 2026-06-24, por solicitação do responsável do projeto, o frontend foi
  reorganizado para reduzir concentração em `App.tsx`: autenticação,
  dashboard administrativo, área do cliente, chat, métricas, histórico,
  modais, notificações, layout, tipos de UI e utilitários passaram a ficar em
  módulos próprios sem criar uma arquitetura excessivamente complexa.
- Em 2026-06-24, por solicitação do responsável do projeto, a navegação do
  frontend deixou de usar atalhos por rolagem entre seções empilhadas e passou
  a separar áreas por estado de tela: dashboard, conversas e financeiro para
  cliente; dashboard, solicitações e clientes para administrador. O chat deve
  funcionar como workspace central, com lista de conversas, histórico
  scrollável próprio e compositor sempre acessível, inclusive em mobile.
- Em 2026-06-24, por solicitação do responsável do projeto, usuários `client`
  passam a entrar inicialmente na área de conversas, reforçando o chat como
  funcionalidade principal. A topbar do cliente deve exibir o valor disponível
  do plano vigente, e o cadastro de destinatário deve usar máscara visual de
  telefone brasileiro no formato fictício `+55 (55) 5555-5555`, mantendo a normalização
  E.164 no backend.
- Em 2026-06-24, por solicitação do responsável do projeto, a regra de mudança
  de plano foi ajustada: o cliente pode solicitar troca mesmo com saldo
  pré-pago ou consumo pós-pago pendente; o administrador só pode aprovar depois
  de zerar a pendência financeira correspondente.
- Em 2026-06-24, por solicitação do responsável do projeto, foi adicionada
  documentação OpenAPI 3.0 em `project-docs/openapi.yaml`, o README passou a
  apontar para essa especificação e o guia `ENTREGA_APRESENTACAO_LOCAL.md` foi
  reconstruído como material de preparação para apresentação e live coding.
- Em 2026-06-25, por solicitação do responsável do projeto, a OpenAPI deixou
  de ser apenas um arquivo versionado e passou a ser servida localmente pelo
  backend em `/swagger.html` e `/openapi.yaml`, com proxy equivalente no
  frontend/Nginx.
- Em 2026-06-25, por solicitação do responsável do projeto, os modais do
  frontend passaram a usar superfície opaca, removendo a estética glass para
  melhorar legibilidade e usabilidade.

## Pendências abertas

Não há pendência funcional bloqueante para finalizar o fluxo mínimo integrado.
As próximas melhorias naturais são polimento do fluxo ponta a ponta, testes
mais específicos, revisão de erros internos e preparação final de entrega.

## Onboarding e RBAC aprovados

- Adotar autocadastro da empresa em estado inativo.
- Permitir autenticação somente depois da ativação por usuário administrador.
- Usar inicialmente apenas os papéis `admin` e `client`.
- Reservar ao `admin` a ativação/inativação de contas, crédito, alteração de
  limite e conversão de plano.
- Registrar em auditoria o administrador, instante, operação e valores
  anteriores/novos.

Essa opção preserva o cadastro ponta a ponta, torna o estado ativo/inativo
demonstrável e evita que o administrador conheça ou distribua a senha do
cliente. A decisão foi confirmada em 2026-06-22.
