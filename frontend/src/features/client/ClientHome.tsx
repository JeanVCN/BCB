import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import type { BillingProfile, Conversation, FinancialTransaction, Message, PlanChangeRequest, Session } from '../../api'
import { api } from '../../api'
import { AppHeader } from '../../components/layout/AppHeader'
import { ToastMessage } from '../../components/ui/ToastMessage'
import { planRequestStatuses, plans } from '../../domain'
import type { Toast } from '../../types/ui'
import { errorMessage, nullWhenNotFound } from '../../utils/errors'
import { idempotencyKey } from '../../utils/forms'
import { formatMoney } from '../../utils/format'
import { conversationMessageStats } from '../../utils/stats'
import { ClientMetrics } from './ClientMetrics'
import { ConversationWorkspace } from './ConversationWorkspace'
import { FinancialHistory } from './FinancialHistory'
import { PlanChangePanel } from './PlanChangePanel'

type ClientView = 'dashboard' | 'conversations' | 'finance'

export function ClientHome({ session, onLogout }: { session: Session; onLogout: () => void }) {
  const token = session.accessToken
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selected, setSelected] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [billing, setBilling] = useState<BillingProfile | null>(null)
  const [transactions, setTransactions] = useState<FinancialTransaction[]>([])
  const [planChange, setPlanChange] = useState<PlanChangeRequest | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingConversations, setLoadingConversations] = useState(true)
  const [toast, setToast] = useState<Toast | null>(null)
  const [activeView, setActiveView] = useState<ClientView>('conversations')
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  const clientStats = useMemo(() => conversationMessageStats(messages), [messages])

  const refreshBilling = useCallback(async () => {
    const [billingResponse, transactionResponse, planChangeResponse] = await Promise.all([
      api.billing(token),
      api.billingTransactions(token),
      api.currentPlanChangeRequest(token).catch(nullWhenNotFound),
    ])
    setBilling(billingResponse)
    setTransactions(transactionResponse.items)
    setPlanChange(planChangeResponse)
  }, [token])

  const loadMessages = useCallback(async (conversation: Conversation, silent = false) => {
    if (!silent) setMessages([])
    if (!silent) setError('')
    try {
      const response = await api.messages(token, conversation.id)
      setMessages(response.items)
    } catch (reason) {
      if (!silent) setError(errorMessage(reason))
    }
  }, [token])

  async function refreshConversations() {
    try {
      const response = await api.conversations(token)
      setConversations(response.items)
      setError('')
      if (selected && !response.items.some(conversation => conversation.id === selected.id)) setSelected(null)
    } catch (reason) {
      setError(errorMessage(reason))
    }
  }

  useEffect(() => {
    void api.conversations(token)
      .then(response => { setConversations(response.items); setError('') })
      .catch(reason => setError(errorMessage(reason)))
      .finally(() => setLoadingConversations(false))
    void Promise.all([
      api.billing(token),
      api.billingTransactions(token),
      api.currentPlanChangeRequest(token).catch(nullWhenNotFound),
    ])
      .then(([billingResponse, transactionResponse, planChangeResponse]) => {
        setBilling(billingResponse)
        setTransactions(transactionResponse.items)
        setPlanChange(planChangeResponse)
      })
      .catch(reason => setError(errorMessage(reason)))
  }, [token])

  useEffect(() => {
    if (!selected) return undefined
    const interval = window.setInterval(() => {
      void loadMessages(selected, true)
      void refreshBilling().catch(() => undefined)
    }, 2000)
    return () => window.clearInterval(interval)
  }, [loadMessages, refreshBilling, selected])

  function scrollMessagesToEnd() {
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        messagesEndRef.current?.scrollIntoView({ block: 'end' })
      })
    })
  }

  async function createConversation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const data = new FormData(form)
    setLoading(true)
    setError('')
    try {
      const conversation = await api.createConversation(token, {
        recipient: { name: data.get('name'), phone: data.get('phone') },
      })
      form.reset()
      await refreshConversations()
      await openConversation(conversation)
      setActiveView('conversations')
      setToast({ type: 'success', text: 'Conversa pronta para envio.' })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setLoading(false)
    }
  }

  async function openConversation(conversation: Conversation) {
    setSelected(conversation)
    await loadMessages(conversation)
    scrollMessagesToEnd()
  }

  async function sendMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selected) return
    const form = event.currentTarget
    const data = new FormData(form)
    setLoading(true)
    setError('')
    try {
      const response = await api.sendMessage(token, selected.id, {
        content: data.get('content'),
        channel: data.get('channel'),
        priority: data.get('priority'),
      }, idempotencyKey())
      setBilling(response.billing)
      form.reset()
      await loadMessages(selected)
      scrollMessagesToEnd()
      await refreshConversations()
      await refreshBilling()
      setToast({ type: 'success', text: 'Mensagem enfileirada e cobrança registrada.' })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setLoading(false)
    }
  }

  async function requestPlanChange() {
    if (!billing) return
    const targetPlan = billing.planType === plans.prepaid ? plans.postpaid : plans.prepaid
    setLoading(true)
    setError('')
    try {
      const response = await api.createPlanChangeRequest(token, { targetPlan })
      setPlanChange(response)
      await refreshBilling()
      setToast({ type: 'success', text: 'Solicitação enviada para análise administrativa.' })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setLoading(false)
    }
  }

  async function cancelPlanChange() {
    if (!planChange || planChange.status !== planRequestStatuses.pending) return
    setLoading(true)
    setError('')
    try {
      await api.cancelPlanChangeRequest(token, planChange.id)
      const response = await api.currentPlanChangeRequest(token).catch(nullWhenNotFound)
      setPlanChange(response)
      setToast({ type: 'success', text: 'Solicitação cancelada.' })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className={`dashboard app-shell view-${activeView}`}>
      <ToastMessage toast={toast} onClose={() => setToast(null)} />
      <AppHeader
        eyebrow="Área do cliente"
        title={clientViewTitle(activeView)}
        meta={billing ? <span className="header-balance">Disponível: <strong>{formatMoney(billing.currentPlanAvailableCents)}</strong></span> : null}
      >
        <button className={activeView === 'dashboard' ? 'active' : ''} onClick={() => setActiveView('dashboard')}>Dashboard</button>
        <button className={activeView === 'conversations' ? 'active' : ''} onClick={() => setActiveView('conversations')}>Conversas</button>
        <button className={activeView === 'finance' ? 'active' : ''} onClick={() => setActiveView('finance')}>Financeiro</button>
        <button onClick={onLogout}>Sair</button>
      </AppHeader>
      {error && <p className="error">{error}</p>}
      {activeView === 'dashboard' && billing && (
        <ClientDashboardView
          billing={billing}
          conversations={conversations}
          messages={messages}
          transactions={transactions}
          planChange={planChange}
          loading={loading}
          onRequestPlanChange={() => void requestPlanChange()}
          onCancelPlanChange={() => void cancelPlanChange()}
          onOpenConversations={() => setActiveView('conversations')}
          onOpenFinance={() => setActiveView('finance')}
        />
      )}
      {activeView === 'conversations' && (
        <ConversationWorkspace
          conversations={conversations}
          selected={selected}
          messages={messages}
          messageStats={clientStats}
          loading={loading}
          loadingConversations={loadingConversations}
          messagesEndRef={messagesEndRef}
          onCreateConversation={createConversation}
          onOpenConversation={conversation => void openConversation(conversation)}
          onCloseConversation={() => setSelected(null)}
          onSendMessage={sendMessage}
        />
      )}
      {activeView === 'finance' && billing && (
        <section className="page-view finance-view">
          <div className="finance-grid">
            <ClientMetrics billing={billing} conversations={conversations} messages={messages} />
            <PlanChangePanel
              billing={billing}
              planChange={planChange}
              loading={loading}
              onRequest={() => void requestPlanChange()}
              onCancel={() => void cancelPlanChange()}
            />
            <FinancialHistory transactions={transactions} />
          </div>
        </section>
      )}
    </main>
  )
}

function ClientDashboardView({
  billing,
  conversations,
  messages,
  transactions,
  planChange,
  loading,
  onRequestPlanChange,
  onCancelPlanChange,
  onOpenConversations,
  onOpenFinance,
}: {
  billing: BillingProfile
  conversations: Conversation[]
  messages: Message[]
  transactions: FinancialTransaction[]
  planChange: PlanChangeRequest | null
  loading: boolean
  onRequestPlanChange: () => void
  onCancelPlanChange: () => void
  onOpenConversations: () => void
  onOpenFinance: () => void
}) {
  const lastConversation = conversations[0]
  const lastTransaction = transactions[0]

  return (
    <section className="page-view overview-view">
      <ClientMetrics billing={billing} conversations={conversations} messages={messages} />
      <div className="operations-grid">
        <article className="operation-panel primary-panel">
          <span className="eyebrow">Conta</span>
          <h2>{billing.planType === plans.prepaid ? 'Pré-pago' : 'Pós-pago'}</h2>
          <p>Disponível atual: <strong>{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(billing.currentPlanAvailableCents / 100)}</strong></p>
          <button className="primary" onClick={onOpenFinance}>Ver financeiro</button>
        </article>
        <article className="operation-panel">
          <span className="eyebrow">Último atendimento</span>
          <h2>{lastConversation ? lastConversation.recipient.name : 'Sem conversas'}</h2>
          <p>{lastConversation?.lastActivityAt ? new Date(lastConversation.lastActivityAt).toLocaleString('pt-BR') : 'Nenhuma atividade recente.'}</p>
          <button onClick={onOpenConversations}>Abrir conversas</button>
        </article>
        <article className="operation-panel">
          <span className="eyebrow">Última movimentação</span>
          <h2>{lastTransaction ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(lastTransaction.amountCents / 100) : 'Sem histórico'}</h2>
          <p>{lastTransaction ? new Date(lastTransaction.createdAt).toLocaleString('pt-BR') : 'Movimentações aparecerão aqui.'}</p>
          <button onClick={onOpenFinance}>Abrir histórico</button>
        </article>
      </div>
      <PlanChangePanel
        billing={billing}
        planChange={planChange}
        loading={loading}
        onRequest={onRequestPlanChange}
        onCancel={onCancelPlanChange}
      />
    </section>
  )
}

function clientViewTitle(view: ClientView) {
  if (view === 'conversations') return 'Conversas'
  if (view === 'finance') return 'Financeiro'
  return 'Dashboard'
}
