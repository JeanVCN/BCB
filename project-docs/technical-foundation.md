# Fundação técnica

## Decisões iniciais

- Monorepo com `backend/` e `frontend/`, mantendo documentação e ambiente na
  raiz.
- Backend em Go 1.25+ e Gin 1.12, com entrada em `cmd/api` e código interno não
  exportado em `internal/`.
- Frontend em React 19.2, TypeScript 6 e Vite 8, usando o scaffold oficial.
- PostgreSQL 17 e Redis 8 em imagens oficiais com patches fixados.
- Docker Compose como caminho principal de execução completa.
- Servidor HTTP com timeouts, logs estruturados e shutdown gracioso desde o
  primeiro incremento.
- `/health/live` verifica apenas que o processo HTTP está vivo.
- `/health/ready` verifica PostgreSQL e Redis.
- Migrations SQL são embutidas no backend e aplicadas pela API e pelo comando
  de bootstrap administrativo.
- A aplicação automática de migrations é controlada por `RUN_MIGRATIONS`; o
  padrão do ambiente local/Docker é `true`.
- O backend usa `pgx`/`pgxpool` para PostgreSQL e `go-redis` para Redis.
- O primeiro administrador é criado por binário dedicado e idempotente,
  configurado por variáveis de ambiente.
- O backend agrupa contextos em `internal/modules`, atualmente com `access`,
  `accounts`, `conversations`, `billing`, `messages` e `planchanges`.
- Cada módulo usa `Repository` como abstração de persistência e mantém seu
  service, handler e registro de rotas.
- `httpserver` mantém o roteador, middlewares e respostas compartilhadas.
- O registry de `modules` monta as dependências internas usando as conexões
  abertas pela `main`.
- Frontend estático servido por Nginx, que também será o proxy da API no
  ambiente Docker.
- A documentação interativa da API é servida pelo backend em `/swagger.html`,
  com especificação OpenAPI em `/openapi.yaml`; o Nginx do frontend proxya as
  mesmas rotas para acesso pela porta do frontend.
- O arquivo autoral da especificação permanece em `project-docs/openapi.yaml`.
  O backend embute uma cópia em `internal/httpserver/openapi.yaml`, regenerada
  por `go generate ./internal/httpserver` antes do build quando o contrato muda.

## Limites desta etapa

- O financeiro administrativo básico foi implementado: consulta de resumo,
  histórico, crédito pré-pago, ajuste de limite pós-pago, zeramento de
  saldo/consumo para troca de plano, idempotência e lock Redis por empresa.
- Cobrança por mensagem, estorno e reversão de consumo foram implementados no
  módulo `billing` e são chamados pelo módulo `messages` durante o caso de uso
  de envio.
- A transação do envio continua única: `messages` coordena conversa, mensagem e
  job, enquanto `billing` aplica cobrança/estorno recebendo a mesma transação.
- A integração entre `messages` e `billing` ocorre por uma interface do service
  financeiro. Assim, o módulo de mensagens não conhece o repository concreto de
  billing e a regra financeira permanece encapsulada no domínio correto.
- O repository de billing mantém o núcleo de cobrança/estorno em
  `repository.go`. Os demais arquivos agrupam responsabilidades auxiliares
  maiores, como perfil/administração e transações/idempotência/auditoria.
- Enumerações de domínio são centralizadas no backend em `internal/domain` e
  espelhadas no frontend em `frontend/src/domain.ts`; comparações diretas com
  strings soltas devem ser evitadas.
- No backend Go, funções e métodos devem permanecer não exportados sempre que
  forem usados apenas dentro do próprio pacote. A letra maiúscula fica reservada
  para fronteiras reais entre pacotes, contratos externos e exigências da
  linguagem, como testes `Test...`.
- O worker de mensagens roda em processo/serviço independente da API e consome
  `dispatch_jobs` no PostgreSQL com `FOR UPDATE SKIP LOCKED`; ele ainda usa a
  lógica do módulo `messages`, preservando coesão de domínio.
- O Docker Compose possui o serviço `message-worker`, que usa o mesmo build do
  backend e executa `/app/message-worker`.
- A simulação local usa sucesso padrão, `[fail]` para falha permanente e
  `[retry]` para falhas transitórias até esgotar tentativas e estornar.
- O frontend cobre onboarding/autenticação/admin básico, fluxo inicial de
  conversas, visualização/ações financeiras básicas e envio/acompanhamento de
  mensagens.
- O módulo `planchanges` cobre solicitação, consulta atual, cancelamento,
  listagem administrativa, aprovação, rejeição, contador administrativo e
  auditoria da mudança de plano.
- A aprovação de mudança de plano revalida conta ativa, plano de origem e
  saldo/consumo zerado dentro da mesma transação que altera
  `billing_profiles`.
- As rotas administrativas financeiras do módulo `billing` são registradas
  dentro do grupo `/admin` sem repetir o prefixo, mantendo o contrato
  `/api/v1/admin/clients/...`.

## Validação realizada

- Testes do backend com `go test ./...`.
- Lint e build de produção do frontend.
- Build de produção do frontend com o fluxo de mudança de plano.
- Validação E2E em Docker com cadastro, ativação, login, conversa, envio
  simulado, falha permanente, retry, estorno, saldo zerado e aprovação de
  mudança de plano.
- Validação de persistência após `docker compose restart`, confirmando login,
  plano vigente, conversa e mensagens previamente criadas.
- Busca em logs do Compose por senha, token, JWT, secret, bearer,
  authorization e hash sem ocorrências sensíveis no fluxo validado.
- Lint do frontend com `npm run lint`.
- Ajustes de responsividade por CSS para evitar overflow em conteúdo longo,
  reduzir raios de cartões/controles e remover escala tipográfica por viewport.
- Validação estrutural do Docker Compose.
- Build das imagens de backend e frontend.
- Inicialização dos quatro serviços com health checks saudáveis.
- Acesso direto aos endpoints `/health/live` e `/health/ready`.
- Acesso a `/health/ready` pelo proxy do frontend.
- Entrega do HTML de produção pelo Nginx.
