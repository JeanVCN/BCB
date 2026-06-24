# Contratos HTTP v1 do BCB

Este documento define o contrato externo inicial entre frontend e backend. Os
nomes são independentes das estruturas internas em Go e TypeScript.

## 1. Convenções gerais

- Prefixo: `/api/v1`.
- Conteúdo: `application/json`.
- Autenticação: `Authorization: Bearer <token>`.
- Token: validade de uma hora e sem refresh no primeiro escopo.
- Datas: ISO 8601 em UTC.
- Dinheiro: inteiro em centavos, com sufixo `Cents`.
- IDs: strings opacas.
- Enumerações: minúsculas no JSON.
- Campos desconhecidos em comandos devem ser rejeitados.
- Respostas nunca expõem hash de senha, locks, segredos ou erros internos.

### Idempotência

Envio de mensagem e mutações financeiras administrativas exigem o header:

```http
Idempotency-Key: <valor-unico-do-cliente>
```

Repetir a mesma chave com o mesmo corpo retorna o resultado original. Repetir a
chave com corpo diferente retorna conflito.

### Envelope de erro

```json
{
  "error": {
    "code": "stable_machine_code",
    "message": "Mensagem segura para apresentação",
    "fields": {
      "fieldName": "motivo opcional"
    },
    "requestId": "opaque-id"
  }
}
```

Status utilizados:

| HTTP | Uso |
|---:|---|
| 400 | JSON ou parâmetro inválido |
| 401 | Credenciais/sessão inválidas |
| 403 | Papel, estado ou propriedade incompatível |
| 404 | Recurso inexistente ou não visível ao ator |
| 409 | Duplicidade, transição ou idempotência conflitante |
| 422 | Regra de negócio não satisfeita |
| 429 | Limite de tentativas excedido |
| 500 | Falha interna sem detalhes sensíveis |
| 503 | Dependência essencial temporariamente indisponível |

## 2. Tipos compartilhados

```text
DocumentType       = "cpf" | "cnpj"
ClientStatus       = "pending" | "active" | "inactive" | "rejected"
Role               = "admin" | "client"
PlanType           = "prepaid" | "postpaid"
Channel            = "sms" | "whatsapp"
Priority           = "normal" | "urgent"
MessageStatus      = "queued" | "processing" | "sent" | "failed"
PlanRequestStatus  = "pending" | "approved" | "rejected" | "cancelled"
```

## 3. Cadastro e autenticação

### POST `/auth/register`

Público. Cria empresa e usuário cliente em estado pendente/inativo.

Requisição:

```json
{
  "name": "Empresa Exemplo",
  "documentType": "cnpj",
  "documentId": "11222333000181",
  "password": "frase-segura-com-requisitos",
  "requestedPlan": "prepaid"
}
```

Resposta `201`:

```json
{
  "clientId": "client-id",
  "status": "pending",
  "message": "Cadastro recebido e aguardando aprovação."
}
```

CPF/CNPJ existente retorna `409`, inclusive se o cadastro anterior foi
rejeitado.

### POST `/auth/login`

Público. Serve administradores e clientes.

```json
{
  "login": "11222333000181",
  "password": "senha-informada"
}
```

Resposta `200`:

```json
{
  "accessToken": "opaque-or-signed-token",
  "tokenType": "Bearer",
  "expiresInSeconds": 3600,
  "user": {
    "id": "user-id",
    "role": "client",
    "clientId": "client-id"
  }
}
```

Credenciais incorretas e conta não autorizada usam mensagem genérica. O código
de erro interno pode diferenciar motivos para observabilidade sem expô-los.

### GET `/me`

Autenticado. Retorna identidade e, para `client`, resumo da conta e financeiro.

```json
{
  "user": {
    "id": "user-id",
    "role": "client"
  },
  "client": {
    "id": "client-id",
    "name": "Empresa Exemplo",
    "documentType": "cnpj",
    "documentId": "11222333000181",
    "status": "active",
    "billing": {
      "planType": "prepaid",
      "balanceCents": 1000
    }
  }
}
```

## 4. Administração de clientes

Todos os contratos desta seção exigem papel `admin`.

### GET `/admin/clients?status=pending`

Lista clientes, opcionalmente filtrados por status. Paginação fica fora do
primeiro escopo.

### GET `/admin/clients/{clientId}`

Retorna cadastro, estado, plano, resumo financeiro e histórico administrativo.

### GET `/admin/clients/{clientId}/billing`

Retorna o perfil financeiro vigente do cliente ativo para orientar ações
administrativas.

### POST `/admin/clients/{clientId}/activate`

Confirma o plano e define a condição financeira inicial.
O `planType` deve corresponder ao `requestedPlan` do autocadastro.

Para pré-pago:

```json
{
  "planType": "prepaid",
  "initialBalanceCents": 1000
}
```

Para pós-pago:

```json
{
  "planType": "postpaid",
  "totalLimitCents": 5000
}
```

Somente `pending`, `inactive` ou `rejected` podem ser ativados. Reconsiderar um
rejeitado usa este mesmo comando e registra auditoria.

### POST `/admin/clients/{clientId}/deactivate`

```json
{
  "reason": "Motivo administrativo"
}
```

### POST `/admin/clients/{clientId}/reject`

Válido para cadastro pendente.

```json
{
  "reason": "Motivo da rejeição"
}
```

### POST `/admin/clients/{clientId}/credits`

Exige `Idempotency-Key` e plano pré-pago.

```json
{
  "amountCents": 1000,
  "reason": "Recarga administrativa"
}
```

### PUT `/admin/clients/{clientId}/postpaid-limit`

Exige `Idempotency-Key` e plano pós-pago.

```json
{
  "totalLimitCents": 10000,
  "reason": "Revisão de limite"
}
```

O novo limite não pode ser inferior ao consumo atual.

### POST `/admin/clients/{clientId}/zero-balance`

Exige `Idempotency-Key`. Zera o saldo pré-pago quando o plano vigente for
pré-pago ou o consumo acumulado quando o plano vigente for pós-pago.

```json
{
  "reason": "Preparação para mudança de plano"
}
```

Quando houver valor a compensar, a operação registra transação financeira
compatível com o plano e sempre registra auditoria administrativa.

### GET `/admin/clients/{clientId}/financial-transactions`

Retorna o histórico financeiro em ordem cronológica decrescente.

## 5. Resumo administrativo

### GET `/admin/notifications/summary`

```json
{
  "pendingClientActivations": 3,
  "pendingPlanChanges": 2
}
```

Os valores são derivados do estado atual, sem entrega em tempo real.

## 6. Destinatários e conversas

Exigem cliente ativo. Todo acesso é limitado à empresa autenticada.

### POST `/conversations`

Cria destinatário e conversa em uma operação simples.

```json
{
  "recipient": {
    "name": "Maria da Silva",
    "phone": "+5511999999999"
  }
}
```

Resposta `201`:

```json
{
  "id": "conversation-id",
  "recipient": {
    "id": "recipient-id",
    "name": "Maria da Silva",
    "phone": "+5511999999999"
  },
  "lastActivityAt": null
}
```

Telefone já associado à empresa retorna a conversa existente com `200`, sem
criar duplicata.

### GET `/conversations`

Lista conversas por `lastActivityAt` decrescente; conversas sem mensagens ficam
ao final por data de criação decrescente.

```json
{
  "items": [
    {
      "id": "conversation-id",
      "recipient": {
        "id": "recipient-id",
        "name": "Maria da Silva",
        "phone": "+5511999999999"
      },
      "lastMessage": {
        "content": "Olá",
        "status": "sent",
        "createdAt": "2026-06-22T15:00:00Z"
      },
      "lastActivityAt": "2026-06-22T15:00:00Z"
    }
  ]
}
```

Não há `unreadCount` no primeiro escopo porque não existem mensagens inbound
nem status de leitura.

### GET `/conversations/{conversationId}/messages`

Retorna mensagens em ordem cronológica crescente.

## 7. Mensagens

### POST `/conversations/{conversationId}/messages`

Exige cliente ativo e `Idempotency-Key`.

```json
{
  "content": "Sua solicitação foi processada.",
  "channel": "whatsapp",
  "priority": "urgent"
}
```

Resposta `202`:

```json
{
  "id": "message-id",
  "conversationId": "conversation-id",
  "content": "Sua solicitação foi processada.",
  "channel": "whatsapp",
  "priority": "urgent",
  "costCents": 50,
  "status": "queued",
  "createdAt": "2026-06-22T15:00:00Z",
  "billing": {
    "planType": "prepaid",
    "prepaidBalanceCents": 950,
    "postpaidTotalLimitCents": 0,
    "postpaidConsumedCents": 0,
    "postpaidAvailableCents": 0,
    "currentPlanAvailableCents": 950
  }
}
```

Erros de negócio relevantes:

| Code | HTTP | Situação |
|---|---:|---|
| `insufficient_balance` | 422 | Pré-pago sem saldo |
| `limit_exceeded` | 422 | Pós-pago sem disponibilidade |
| `client_not_active` | 403 | Empresa bloqueada |
| `conversation_not_found` | 404 | Ausente ou pertencente a outra empresa |
| `idempotency_conflict` | 409 | Mesma chave com outro conteúdo |

O processamento é simulado. Mensagens comuns tendem a sucesso; conteúdo com
`[fail]` força falha permanente para demonstração, e conteúdo com `[retry]`
força falhas transitórias até esgotar as tentativas e acionar estorno.

### GET `/messages/{messageId}`

Usado para sincronização por polling e limitado ao proprietário.

```json
{
  "id": "message-id",
  "conversationId": "conversation-id",
  "content": "Sua solicitação foi processada.",
  "channel": "whatsapp",
  "priority": "urgent",
  "costCents": 50,
  "status": "processing",
  "attempts": 2,
  "createdAt": "2026-06-22T15:00:00Z",
  "sentAt": null,
  "failedAt": null,
  "failureCode": null
}
```

O frontend consulta mensagens não terminais em intervalo configurável e encerra
o polling em `sent` ou `failed`.

## 8. Financeiro do cliente

### GET `/billing`

Exige `client` ativo e retorna seu perfil financeiro.

Pré-pago:

```json
{
  "planType": "prepaid",
  "balanceCents": 950
}
```

Pós-pago:

```json
{
  "planType": "postpaid",
  "totalLimitCents": 5000,
  "consumedCents": 1250,
  "availableCents": 3750
}
```

### GET `/billing/transactions`

Retorna somente as transações da empresa autenticada, em ordem cronológica
decrescente.

## 9. Mudança de plano

### POST `/plan-change-requests`

Exige `client` ativo.

```json
{
  "targetPlan": "postpaid"
}
```

Resposta `201` com status `pending`, mesmo quando houver saldo pré-pago ou
consumo pós-pago pendente. Solicitação pendente existente retorna `409`.

### GET `/plan-change-requests/current`

Retorna a solicitação atual/mais recente da empresa autenticada ou `404`.

### POST `/plan-change-requests/{requestId}/cancel`

O proprietário cancela somente uma solicitação `pending`.

### GET `/admin/plan-change-requests?status=pending`

Lista solicitações para decisão administrativa.

### POST `/admin/plan-change-requests/{requestId}/approve`

Para destino pós-pago:

```json
{
  "totalLimitCents": 5000
}
```

Para destino pré-pago:

```json
{
  "initialBalanceCents": 0
}
```

Revalida estado, saldo/consumo zerado e status da solicitação dentro da
operação atômica. Conflito retorna `409` ou regra financeira violada retorna
`422`.

### POST `/admin/plan-change-requests/{requestId}/reject`

```json
{
  "reason": "Motivo da rejeição"
}
```

## 10. Auditoria administrativa

### GET `/admin/audit-events`

Lista eventos mais recentes. Filtros e paginação avançados ficam para uma
evolução posterior. A resposta nunca inclui segredos ou payloads sensíveis.

## 11. Sincronização frontend/backend

1. Login fornece token e identidade.
2. `/me` reconstitui sessão e estado financeiro.
3. Conversas e mensagens são carregadas do backend como fonte oficial.
4. Envio retorna `202` e mensagem `queued` já cobrada.
5. Frontend faz polling apenas das mensagens não terminais.
6. Ao chegar a `sent` ou `failed`, atualiza histórico, conversa e financeiro.
7. `401` encerra a sessão local e direciona para novo login.
8. Falha de comunicação preserva a mensagem como estado desconhecido até
   reconciliação; nunca dispara automaticamente um novo envio com outra chave.

## 12. Bootstrap administrativo

O primeiro administrador não possui contrato HTTP público. Um comando
idempotente recebe login e senha por variáveis de ambiente, cria o admin apenas
quando necessário e nunca registra a senha. Execuções posteriores não duplicam
nem sobrescrevem silenciosamente a identidade existente.
