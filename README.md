<div align="center">

# Big Chat Brasil

### Aplicação fullstack de chat empresarial com cobrança por mensagem

![Go](https://img.shields.io/badge/backend-Go%20%2B%20Gin-00ADD8)
![React](https://img.shields.io/badge/frontend-React%20%2B%20TypeScript-3178C6)
![PostgreSQL](https://img.shields.io/badge/database-PostgreSQL-4169E1)
![Docker](https://img.shields.io/badge/run-Docker%20Compose-2496ED)

</div>

## Sobre

O **Big Chat Brasil (BCB)** é a solução para o desafio técnico Fullstack: uma
plataforma de chat onde empresas conversam com clientes finais, enviam mensagens
SMS ou WhatsApp simuladas, acompanham status de processamento e controlam custos
em planos pré-pago ou pós-pago.

A entrega implementa backend real, frontend integrado e execução completa com
Docker Compose. O foco principal está no fluxo fullstack solicitado:

```text
cadastro/login -> conversas -> envio de mensagem -> fila/worker -> status -> financeiro
```

## Funcionalidades

- Autocadastro de empresa cliente por CPF ou CNPJ.
- Aprovação administrativa antes do primeiro acesso do cliente.
- Autenticação com senha, JWT e limitação de tentativas de login.
- Controle de acesso com papéis `admin` e `client`.
- Cadastro/listagem de conversas com destinatário e telefone E.164.
- Envio de mensagens SMS ou WhatsApp com prioridade normal ou urgente.
- Custos por mensagem:
  - Normal: R$ 0,25.
  - Urgente: R$ 0,50.
- Planos:
  - Pré-pago: exige saldo antes de enviar.
  - Pós-pago: exige limite disponível.
- Fila persistente em PostgreSQL com worker separado.
- Priorização: mensagens urgentes antes de normais, preservando FIFO por
  prioridade.
- Status de mensagem: `queued`, `processing`, `sent` e `failed`.
- Retry de falhas transitórias e estorno idempotente em falhas definitivas.
- Histórico financeiro para cliente e administrador.
- Crédito pré-pago e ajuste de limite pós-pago por administrador.
- Zeramento administrativo de saldo pré-pago ou consumo pós-pago para liberar
  solicitação de mudança de plano.
- Solicitação de mudança de plano pelo cliente e aprovação/rejeição pelo admin.
- Interface responsiva com estados de carregamento, vazio, erro e sucesso.

## Tecnologias

| Camada | Escolha |
|---|---|
| Backend | Go 1.25+ com Gin |
| Frontend | React 19, TypeScript 6 e Vite 8 |
| Banco de dados | PostgreSQL 17 |
| Coordenação | Redis 8 para rate limit e lock financeiro |
| Infraestrutura local | Docker Compose |
| Servidor frontend | Nginx servindo build estático e proxy da API |

## Premissas

- O envio real por SMS/WhatsApp não é necessário no desafio. O sistema simula o
  disparo e atualiza status.
- O cliente final/destinatário possui nome e telefone.
- O cliente da plataforma é a empresa contratante, identificada por CPF ou CNPJ.
- Valores financeiros são armazenados em centavos, sem ponto flutuante.
- O primeiro administrador é criado por comando local, sem endpoint público.
- A API aplica migrations automaticamente no ambiente Docker por padrão.
- `docs/` contém os documentos oficiais recebidos para o desafio e não é
  alterado pela implementação.

## Como executar

### 1. Requisitos

- Docker.
- Docker Compose v2 (`docker compose`).
- Portas livres por padrão:
  - Frontend: `3000`.
  - Backend: `8080`.
  - PostgreSQL: `5432`.
  - Redis: `6379`.

### 2. Configurar ambiente

```bash
cp .env.example .env
```

O arquivo `.env.example` já possui valores de desenvolvimento suficientes para
subir a aplicação. Para uma avaliação local, você pode manter os defaults.

Se alguma porta estiver ocupada, altere no `.env`:

```env
POSTGRES_PORT=15432
REDIS_PORT=16379
BACKEND_PORT=18080
FRONTEND_PORT=13000
```

### 3. Subir os serviços

```bash
docker compose up --build
```

Serviços iniciados:

- `postgres`: banco de dados principal.
- `redis`: rate limit e lock financeiro.
- `backend`: API Go/Gin.
- `message-worker`: processamento assíncrono da fila.
- `frontend`: aplicação React servida por Nginx.

### 4. Acessar

Com portas padrão:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`
- Liveness: `http://localhost:8080/health/live`
- Readiness: `http://localhost:8080/health/ready`

Com as portas alternativas do exemplo:

- Frontend: `http://localhost:13000`
- Backend: `http://localhost:18080`
- Liveness: `http://localhost:18080/health/live`
- Readiness: `http://localhost:18080/health/ready`

### 5. Criar o primeiro administrador

Em outro terminal, execute:

```bash
docker compose run --rm backend /app/admin-bootstrap
```

Credenciais padrão do `.env.example`:

```text
login: admin
senha: ChangeThisAdminPassword123!
```

O comando é idempotente. Rodar novamente não duplica o administrador.

### 6. Parar ou limpar o ambiente

Parar containers preservando dados:

```bash
docker compose down
```

Parar containers e apagar volumes do PostgreSQL/Redis:

```bash
docker compose down -v
```

## Roteiro de teste pela interface

1. Acesse o frontend.
2. Crie uma conta de empresa cliente com CPF ou CNPJ válido.
3. Faça login como `admin`.
4. Ative o cliente recém-cadastrado:
   - Para pré-pago, informe saldo inicial em centavos. Exemplo: `1000`.
   - Para pós-pago, informe limite em centavos. Exemplo: `5000`.
5. Saia e faça login como cliente usando o documento cadastrado e a senha.
6. Crie uma conversa com telefone em formato E.164. Exemplo:

```text
+5511999999999
```

7. Envie uma mensagem normal ou urgente.
8. Acompanhe a troca de status até `sent` ou `failed`.
9. Consulte o resumo e o histórico financeiro.
10. Para simular cenários do worker:
    - Mensagem comum tende a sucesso.
    - Conteúdo com `[fail]` força falha permanente.
    - Conteúdo com `[retry]` força falhas transitórias até esgotar retries e
      acionar estorno.
11. Para testar mudança de plano:
    - Pré-pago só solicita mudança com saldo igual a zero.
    - Pós-pago só solicita mudança com consumo igual a zero.
    - Se necessário, o admin pode zerar saldo/consumo pelo painel.
    - O admin aprova ou rejeita pelo painel administrativo.

## Verificações por terminal

### Health checks

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

Resposta esperada:

```json
{"status":"ok"}
```

```json
{"status":"ready"}
```

### Testes e build locais

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
npm install
npm run lint
npm run build
```

## Estrutura do projeto

```text
.
├── backend/                  # API Go, migrations e worker
│   ├── cmd/api               # Entrada HTTP
│   ├── cmd/admin-bootstrap   # Criação do primeiro admin
│   ├── cmd/message-worker    # Worker da fila
│   └── internal/modules      # Domínios da aplicação
├── frontend/                 # React + TypeScript + Vite
├── project-docs/             # Decisões, contratos e modelo autoral
├── ai-memory/                # Memória operacional para IA
├── docs/                     # Documentação oficial recebida do desafio
├── compose.yaml              # Orquestração Docker Compose
└── .env.example              # Variáveis de ambiente de desenvolvimento
```

## Endpoints principais

| Método | Rota | Uso |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Autocadastro da empresa |
| `POST` | `/api/v1/auth/login` | Login admin ou cliente |
| `GET` | `/api/v1/me` | Sessão atual |
| `GET` | `/api/v1/conversations` | Listar conversas do cliente |
| `POST` | `/api/v1/conversations` | Criar conversa/destinatário |
| `GET` | `/api/v1/conversations/{id}/messages` | Histórico da conversa |
| `POST` | `/api/v1/conversations/{id}/messages` | Enviar mensagem |
| `GET` | `/api/v1/billing` | Resumo financeiro do cliente |
| `GET` | `/api/v1/billing/transactions` | Histórico financeiro |
| `POST` | `/api/v1/plan-change-requests` | Solicitar mudança de plano |
| `GET` | `/api/v1/admin/clients` | Listar clientes para admin |
| `POST` | `/api/v1/admin/clients/{id}/activate` | Ativar cliente |
| `GET` | `/api/v1/admin/clients/{id}/billing` | Consultar financeiro do cliente |
| `POST` | `/api/v1/admin/clients/{id}/credits` | Adicionar crédito |
| `PUT` | `/api/v1/admin/clients/{id}/postpaid-limit` | Ajustar limite pós-pago |
| `POST` | `/api/v1/admin/clients/{id}/zero-balance` | Zerar saldo/consumo |
| `GET` | `/api/v1/admin/plan-change-requests` | Listar mudanças de plano |

Envio de mensagem e mutações financeiras administrativas usam
`Idempotency-Key`.

## Decisões técnicas

- **Go + Gin**: stack simples, rápida e adequada para APIs HTTP.
- **PostgreSQL**: garante consistência transacional para cobrança, fila,
  auditoria e histórico financeiro.
- **Redis**: protege operações sensíveis contra concorrência entre instâncias.
- **Worker separado**: evita acoplar processamento de mensagens ao ciclo de
  vida do servidor HTTP.
- **Fila em banco**: mantém persistência e permite evolução futura para
  RabbitMQ sem mudar o domínio.
- **JWT com expiração curta**: autenticação stateless suficiente para o desafio.
- **Argon2id**: armazenamento seguro de senhas com hash resistente e salt.

## Limitações conhecidas

- Não há integração real com provedores de SMS ou WhatsApp.
- Não há mensagens inbound de provedor externo.
- Estados `delivered` e `read` ficaram fora do escopo inicial.
- Não há reset mensal automático do consumo pós-pago.
- Não há paginação avançada ou busca textual nas listas.
- Não há Swagger/OpenAPI nesta versão.

## Documentação complementar

Este README foi escrito a partir dos documentos oficiais recebidos para o
desafio, mantidos localmente em `docs/`. Essa pasta não faz parte da
documentação autoral do projeto.

- [Decisões de produto](project-docs/product-decisions.md)
- [Critérios de aceite](project-docs/acceptance-criteria.md)
- [Modelo conceitual](project-docs/domain-model.md)
- [Contratos HTTP](project-docs/api-contracts.md)
- [Fundação técnica](project-docs/technical-foundation.md)
- [Histórico de implementação](HISTORICO_IMPLEMENTACAO.md)
