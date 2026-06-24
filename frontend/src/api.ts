import type { Channel, ClientStatus, DocumentType, FinancialTransactionType, MessageStatus, Plan, PlanRequestStatus, Priority, Role } from './domain'

export interface Session {
  accessToken: string
  user: { id: string; role: Role; clientId?: string | null }
}

export interface Client {
  id: string
  name: string
  documentType: DocumentType
  documentId: string
  status: ClientStatus
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
  conversationId: string
  content: string
  channel: Channel
  priority: Priority
  costCents: number
  status: MessageStatus
  failureCode?: string
  createdAt: string
  queuedAt: string
  processingAt?: string
  sentAt?: string
  failedAt?: string
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
  type: FinancialTransactionType
  amountCents: number
  messageId?: string
  reversesTransactionId?: string
  actorUserId?: string
  idempotencyKey: string
  reason?: string
  createdAt: string
}

export interface PlanChangeRequest {
  id: string
  clientId: string
  clientName?: string
  fromPlan: Plan
  toPlan: Plan
  status: PlanRequestStatus
  rejectionReason?: string
  createdAt: string
  cancelledAt?: string
  decidedAt?: string
}

export interface AdminSummary {
  pendingClientActivations: number
  pendingPlanChanges: number
}

interface APIError {
  error?: { code?: string; message?: string }
}

export class APIRequestError extends Error {
  status: number
  code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'APIRequestError'
    this.status = status
    this.code = code
  }
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
    throw new APIRequestError(response.status, body.error?.message ?? 'Não foi possível concluir a operação.', body.error?.code)
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
  adminBilling: (token: string, clientId: string) => request<BillingProfile>(
    `/admin/clients/${clientId}/billing`, {}, token,
  ),
  setPostpaidLimit: (token: string, clientId: string, body: object, idempotencyKey: string) => request<void>(
    `/admin/clients/${clientId}/postpaid-limit`, {
      method: 'PUT', body: JSON.stringify(body), headers: { 'Idempotency-Key': idempotencyKey },
    }, token,
  ),
  zeroCurrentBalance: (token: string, clientId: string, body: object, idempotencyKey: string) => request<void>(
    `/admin/clients/${clientId}/zero-balance`, {
      method: 'POST', body: JSON.stringify(body), headers: { 'Idempotency-Key': idempotencyKey },
    }, token,
  ),
  adminFinancialTransactions: (token: string, clientId: string) => request<{ items: FinancialTransaction[] }>(
    `/admin/clients/${clientId}/financial-transactions`, {}, token,
  ),
  billing: (token: string) => request<BillingProfile>('/billing', {}, token),
  billingTransactions: (token: string) => request<{ items: FinancialTransaction[] }>('/billing/transactions', {}, token),
  currentPlanChangeRequest: (token: string) => request<PlanChangeRequest>('/plan-change-requests/current', {}, token),
  createPlanChangeRequest: (token: string, body: object) => request<PlanChangeRequest>('/plan-change-requests', {
    method: 'POST', body: JSON.stringify(body),
  }, token),
  cancelPlanChangeRequest: (token: string, requestId: string) => request<void>(
    `/plan-change-requests/${requestId}/cancel`, { method: 'POST' }, token,
  ),
  adminSummary: (token: string) => request<AdminSummary>('/admin/notifications/summary', {}, token),
  adminPlanChangeRequests: (token: string) => request<{ items: PlanChangeRequest[] }>('/admin/plan-change-requests?status=pending', {}, token),
  approvePlanChangeRequest: (token: string, requestId: string, body: object) => request<void>(
    `/admin/plan-change-requests/${requestId}/approve`, { method: 'POST', body: JSON.stringify(body) }, token,
  ),
  rejectPlanChangeRequest: (token: string, requestId: string, reason: string) => request<void>(
    `/admin/plan-change-requests/${requestId}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }, token,
  ),
  conversations: (token: string) => request<{ items: Conversation[] }>('/conversations', {}, token),
  createConversation: (token: string, body: object) => request<Conversation>('/conversations', {
    method: 'POST', body: JSON.stringify(body),
  }, token),
  messages: (token: string, conversationId: string) => request<{ items: Message[] }>(
    `/conversations/${conversationId}/messages`, {}, token,
  ),
  sendMessage: (token: string, conversationId: string, body: object, idempotencyKey: string) => request<Message & { billing: BillingProfile }>(
    `/conversations/${conversationId}/messages`, {
      method: 'POST', body: JSON.stringify(body), headers: { 'Idempotency-Key': idempotencyKey },
    }, token,
  ),
}
