export type Role = 'admin' | 'client'
export type Plan = 'prepaid' | 'postpaid'

export interface Session {
  accessToken: string
  user: { id: string; role: Role; clientId?: string | null }
}

export interface Client {
  id: string
  name: string
  documentType: 'cpf' | 'cnpj'
  documentId: string
  status: 'pending' | 'active' | 'inactive' | 'rejected'
  requestedPlan: Plan
  statusReason?: string
  createdAt: string
}

export interface Recipient {
  id: string
  name: string
  phone: string
}

export interface Conversation {
  id: string
  recipient: Recipient
  lastActivityAt: string | null
}

export interface Message {
  id: string
}

export interface BillingProfile {
  planType: Plan
  prepaidBalanceCents: number
  postpaidTotalLimitCents: number
  postpaidConsumedCents: number
  postpaidAvailableCents: number
  currentPlanAvailableCents: number
  updatedAt: string
}

export interface FinancialTransaction {
  id: string
  type: 'credit' | 'debit' | 'consumption' | 'refund' | 'consumption_reversal'
  amountCents: number
  messageId?: string
  reversesTransactionId?: string
  actorUserId?: string
  idempotencyKey: string
  reason?: string
  createdAt: string
}

interface APIError {
  error?: { message?: string }
}

async function request<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as APIError
    throw new Error(body.error?.message ?? 'Não foi possível concluir a operação.')
  }

  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  register: (body: object) => request<{ clientId: string; message: string }>('/auth/register', {
    method: 'POST', body: JSON.stringify(body),
  }),
  login: (body: object) => request<Session>('/auth/login', {
    method: 'POST', body: JSON.stringify(body),
  }),
  clients: (token: string) => request<{ items: Client[] }>('/admin/clients', {}, token),
  activate: (token: string, clientId: string, body: object) => request<void>(
    `/admin/clients/${clientId}/activate`, { method: 'POST', body: JSON.stringify(body) }, token,
  ),
  reject: (token: string, clientId: string, reason: string) => request<void>(
    `/admin/clients/${clientId}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }, token,
  ),
  addCredit: (token: string, clientId: string, body: object, idempotencyKey: string) => request<void>(
    `/admin/clients/${clientId}/credits`, {
      method: 'POST', body: JSON.stringify(body), headers: { 'Idempotency-Key': idempotencyKey },
    }, token,
  ),
  setPostpaidLimit: (token: string, clientId: string, body: object, idempotencyKey: string) => request<void>(
    `/admin/clients/${clientId}/postpaid-limit`, {
      method: 'PUT', body: JSON.stringify(body), headers: { 'Idempotency-Key': idempotencyKey },
    }, token,
  ),
  adminFinancialTransactions: (token: string, clientId: string) => request<{ items: FinancialTransaction[] }>(
    `/admin/clients/${clientId}/financial-transactions`, {}, token,
  ),
  billing: (token: string) => request<BillingProfile>('/billing', {}, token),
  billingTransactions: (token: string) => request<{ items: FinancialTransaction[] }>('/billing/transactions', {}, token),
  conversations: (token: string) => request<{ items: Conversation[] }>('/conversations', {}, token),
  createConversation: (token: string, body: object) => request<Conversation>('/conversations', {
    method: 'POST', body: JSON.stringify(body),
  }, token),
  messages: (token: string, conversationId: string) => request<{ items: Message[] }>(
    `/conversations/${conversationId}/messages`, {}, token,
  ),
}
