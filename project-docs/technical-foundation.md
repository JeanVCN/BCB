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
  `accounts`, `conversations`, `billing` e `messages`.
- Cada módulo usa `Repository` como abstração de persistência e mantém seu
  service, handler e registro de rotas.
- `httpserver` mantém o roteador, middlewares e respostas compartilhadas.
- O registry de `modules` monta as dependências internas usando as conexões
  abertas pela `main`.
- Frontend estático servido por Nginx, que também será o proxy da API no
  ambiente Docker.

## Limites desta etapa

- O financeiro administrativo básico foi implementado: consulta de resumo,
  histórico, crédito pré-pago, ajuste de limite pós-pago, idempotência e lock
  Redis por empresa.
- Cobrança por mensagem, estorno e reversão de consumo foram implementados no
  módulo `messages`, junto com jobs persistentes e worker simples.
- O worker inicial roda no processo da API e consome `dispatch_jobs` no
  PostgreSQL com `FOR UPDATE SKIP LOCKED`; ele pode ser extraído depois para
  processo dedicado/RabbitMQ.
- A simulação local usa sucesso padrão, `[fail]` para falha permanente e
  `[retry]` para falhas transitórias até esgotar tentativas e estornar.
- O frontend cobre onboarding/autenticação/admin básico, fluxo inicial de
  conversas, visualização/ações financeiras básicas e envio/acompanhamento de
  mensagens.

## Validação realizada

- Testes do backend com `go test ./...`.
- Lint e build de produção do frontend.
- Validação estrutural do Docker Compose.
- Build das imagens de backend e frontend.
- Inicialização dos quatro serviços com health checks saudáveis.
- Acesso direto aos endpoints `/health/live` e `/health/ready`.
- Acesso a `/health/ready` pelo proxy do frontend.
- Entrega do HTML de produção pelo Nginx.
