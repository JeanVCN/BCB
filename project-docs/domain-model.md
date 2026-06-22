# Modelo conceitual do BCB

Este documento descreve conceitos, relacionamentos, estados e invariantes do
produto. Nomes de tabelas, structs e detalhes de migrations serão definidos na
implementação.

## 1. Convenções

- Identificadores são opacos e não carregam significado de negócio.
- Datas e horas são armazenadas em UTC e formatadas em ISO 8601 nos contratos.
- Valores monetários são inteiros em centavos de real; ponto flutuante é
  proibido para cálculo financeiro.
- CPF/CNPJ são normalizados para dígitos e validados antes da persistência.
- Telefones são normalizados para formato internacional E.164.
- Registros financeiros e de auditoria são imutáveis; correções geram novos
  eventos compensatórios.
- Exclusão física de dados auditáveis fica fora do fluxo normal do produto.

## 2. Visão dos relacionamentos

```text
User (admin) ---------------------------> AuditEvent

ClientAccount 1 ---- 1 User (client)
      | 1
      +---- 1 BillingProfile ---- * FinancialTransaction
      |
      +---- * Recipient ---- 1 Conversation ---- * Message
      |                                             |
      |                                             +---- * DeliveryAttempt
      |                                             +---- 1 DispatchJob
      |
      +---- * PlanChangeRequest
      +---- * AuditEvent
```

No primeiro escopo existe exatamente um usuário `client` por empresa. O modelo
não deve impedir uma futura evolução para múltiplos usuários por empresa.

## 3. Agregados e entidades

### 3.1 User

Representa uma identidade autenticável.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Role | `admin` ou `client` |
| Login | Único entre usuários |
| Password hash | Nunca armazena senha em texto puro |
| Enabled | Controla acesso da identidade |
| Client account | Obrigatório para `client`; ausente para admin global |
| Created/updated at | Auditoria temporal |

O login do cliente deriva de seu CPF/CNPJ normalizado. O login administrativo é
definido no bootstrap.

### 3.2 ClientAccount

Representa a empresa titular da conta BCB.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Name/legal name | Obrigatório |
| Document type | `CPF` ou `CNPJ` |
| Document | Normalizado, válido e globalmente único |
| Status | `pending`, `active`, `inactive` ou `rejected` |
| Requested plan | `prepaid` ou `postpaid` |
| Status reason | Obrigatório para rejeição/inativação administrativa |
| Created/updated at | Auditoria temporal |

Estados `pending`, `inactive` e `rejected` não podem autenticar nem enviar
mensagens. Um cadastro rejeitado preserva documento e histórico; somente admin
pode reconsiderá-lo. Na ativação inicial, o plano confirmado deve corresponder
ao solicitado no autocadastro.

### 3.3 BillingProfile

Mantém o estado financeiro atual da empresa.

| Atributo conceitual | Regra |
|---|---|
| Client account | Relação exclusiva um-para-um |
| Plan type | `prepaid` ou `postpaid` |
| Prepaid balance | Inteiro não negativo em centavos |
| Postpaid total limit | Inteiro não negativo em centavos |
| Postpaid consumed | Inteiro não negativo em centavos |
| Version | Apoia controle de concorrência |
| Updated at | Instante da última mudança |

Invariantes:

- disponibilidade pós-paga = limite total − consumo;
- consumo nunca pode superar o limite total durante nova cobrança;
- alterar limite não altera consumo;
- apenas os campos do plano vigente podem financiar uma mensagem;
- toda mutação deve possuir transação financeira correspondente.

### 3.4 FinancialTransaction

Razão imutável das movimentações financeiras.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Client account | Titular da movimentação |
| Type | `credit`, `debit`, `consumption`, `refund` ou `consumption_reversal` |
| Amount | Inteiro positivo em centavos |
| Message | Presente para cobrança/estorno de mensagem |
| Reverses transaction | Obrigatório em estorno/reversão |
| Actor | Usuário ou processo que originou a operação |
| Idempotency key | Impede repetição do mesmo efeito |
| Created at | Imutável |

Ajustes de limite, mudanças de plano e ativações pertencem à auditoria,
mesmo quando não movimentam dinheiro.

### 3.5 Recipient

Cliente final/destinatário pertencente a uma empresa.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Client account | Define propriedade e isolamento |
| Name | Obrigatório |
| Phone | E.164 e único dentro da empresa no escopo inicial |
| Created/updated at | Auditoria temporal |

O mesmo telefone pode existir em empresas diferentes.

### 3.6 Conversation

Canal lógico entre uma empresa e um destinatário.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Client account | Proprietário |
| Recipient | Destinatário da conversa |
| Created at | Instante de criação |
| Last activity at | Atualizado por nova mensagem |

Há no máximo uma conversa ativa por empresa e destinatário no primeiro
escopo.

### 3.7 Message

Solicitação de envio realizada pela empresa.

| Atributo conceitual | Regra |
|---|---|
| ID | Identificador opaco |
| Client account/conversation/recipient | Devem pertencer ao mesmo agregado |
| Content | Obrigatório; limite máximo fica para evolução posterior |
| Channel | `sms` ou `whatsapp` |
| Priority | `normal` ou `urgent` |
| Cost | 25 centavos normal; 50 centavos urgente |
| Status | `queued`, `processing`, `sent` ou `failed` |
| Requested by | Usuário cliente autenticado |
| Idempotency key | Única por empresa |
| Queued/sent/failed at | Preenchidos conforme o ciclo |
| Failure code | Presente em falha definitiva |

Transições válidas:

```text
queued -> processing -> sent
                    \-> failed
processing -> processing  (nova tentativa registrada, sem duplicar mensagem)
```

`sent` e `failed` são terminais. `delivered` e `read` ficam reservados para
evolução posterior.

### 3.8 DeliveryAttempt

Registra cada tentativa do worker.

| Atributo conceitual | Regra |
|---|---|
| Message | Mensagem processada |
| Attempt number | De 1 a 4 |
| Outcome | `sent`, `transient_failure` ou `permanent_failure` |
| Error code | Código seguro e estável |
| Started/finished at | Duração da tentativa |
| Next retry at | Somente quando houver nova tentativa |

### 3.9 DispatchJob

Representa o trabalho persistente consumido pelo worker e constitui a fronteira
para futura troca por RabbitMQ.

| Atributo conceitual | Regra |
|---|---|
| Message | Exatamente um job por mensagem |
| Priority rank | Urgente precede normal |
| State | `pending`, `processing`, `completed` ou `failed` |
| Available at | Controla backoff |
| Claimed by/at | Evita consumo simultâneo |
| Attempt count | Máximo de quatro tentativas totais |

Ordenação: prioridade urgente primeiro; dentro da mesma prioridade, instante
de enfileiramento e ID como desempate determinístico.

### 3.10 PlanChangeRequest

Workflow de mudança de plano.

| Atributo conceitual | Regra |
|---|---|
| Client account | Solicitante |
| From/to plan | Devem ser diferentes |
| Status | `pending`, `approved`, `rejected` ou `cancelled` |
| Requested/cancelled/decided by | Atores de cada transição |
| Reason | Obrigatório na rejeição |
| Created/decided at | Auditoria temporal |

Invariantes:

- no máximo uma solicitação pendente por empresa;
- pré-pago solicita apenas com saldo zero;
- pós-pago solicita apenas com consumo zero;
- aprovação revalida saldo/consumo e estado do cliente;
- cliente cancela apenas a própria solicitação pendente;
- somente admin aprova ou rejeita.

### 3.11 AuditEvent

Registro imutável das operações sensíveis.

Contém ator, ação, tipo/ID do alvo, instante, motivo e valores anteriores/novos
sanitizados. Senhas, tokens e segredos nunca entram na auditoria.

## 4. Limites transacionais

As seguintes mudanças devem ser atômicas no PostgreSQL:

- ativação do cliente, confirmação do plano e condição financeira inicial;
- cobrança, transação financeira, mensagem e job de processamento;
- estorno/reversão, transação compensatória e estado terminal da mensagem;
- aprovação da mudança de plano e auditoria correspondente.

Redis coordena operações financeiras concorrentes por empresa. O lock é
complementar: constraints, transações e idempotência no PostgreSQL continuam
sendo a garantia final de consistência.

## 5. Autorização

| Operação | Público | Client | Admin |
|---|:---:|:---:|:---:|
| Autocadastro | Sim | — | — |
| Login | Sim | Sim | Sim |
| Consultar dados próprios | Não | Sim | Não |
| Conversas e mensagens próprias | Não | Sim | Não |
| Solicitar/cancelar mudança | Não | Sim | Não |
| Consultar/ativar/inativar clientes | Não | Não | Sim |
| Administrar financeiro | Não | Não | Sim |
| Decidir mudança de plano | Não | Não | Sim |
| Consultar auditoria administrativa | Não | Não | Sim |

## 6. Notificações internas

O painel administrativo calcula seus contadores a partir de cadastros
`pending` e solicitações de plano `pending`. Não é necessária uma entidade de
notificação na primeira versão.

## 7. Fora do modelo inicial

- Provedores reais de SMS/WhatsApp.
- Mensagens inbound.
- Entrega/leitura pelo destinatário.
- Reset mensal automatizado.
- Múltiplos usuários por empresa.
- Recuperação de senha e MFA.
- Exclusão definitiva e retenção regulatória.
