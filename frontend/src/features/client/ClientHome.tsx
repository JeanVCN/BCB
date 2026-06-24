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
import { conversationMessageStats } from '../../utils/stats'
import { ClientMetrics } from './ClientMetrics'
import { ConversationWorkspace } from './ConversationWorkspace'
import { FinancialHistory } from './FinancialHistory'
import { PlanChangePanel } from './PlanChangePanel'

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

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ block: 'end' })
  }, [messages, selected])

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
    <main className="dashboard">
      <ToastMessage toast={toast} onClose={() => setToast(null)} />
      <AppHeader eyebrow="Área do cliente" title="Operação de mensagens">
        <a href="#conversas">Conversas</a>
        <a href="#financeiro">Financeiro</a>
        <button onClick={onLogout}>Sair</button>
      </AppHeader>
      {error && <p className="error">{error}</p>}
      {billing && <ClientMetrics billing={billing} conversations={conversations} messages={messages} />}
      {billing && (
        <PlanChangePanel
          billing={billing}
          planChange={planChange}
          loading={loading}
          onRequest={() => void requestPlanChange()}
          onCancel={() => void cancelPlanChange()}
        />
      )}
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
        onSendMessage={sendMessage}
      />
      <FinancialHistory transactions={transactions} />
    </main>
  )
}
