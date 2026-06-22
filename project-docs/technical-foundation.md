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
- `/health/live` verifica apenas que o processo HTTP está vivo. Readiness de
  PostgreSQL e Redis será adicionada quando as conexões forem implementadas.
- Frontend estático servido por Nginx, que também será o proxy da API no
  ambiente Docker.

## Limites desta etapa

- PostgreSQL e Redis sobem e passam por health check, mas ainda não são
  consumidos pelo backend.
- Não há migration, entidade de domínio, autenticação ou regra financeira.
- A tela inicial apenas confirma que o frontend foi construído e servido.

## Validação realizada

- Testes do backend com `go test ./...`.
- Lint e build de produção do frontend.
- Validação estrutural do Docker Compose.
- Build das imagens de backend e frontend.
- Inicialização dos quatro serviços com health checks saudáveis.
- Acesso direto e via proxy ao endpoint `/health/live`.
- Entrega do HTML de produção pelo Nginx.
