import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from './api'
import type { BillingProfile, Client, Conversation, FinancialTransaction, Message, Session } from './api'
import { clientStatuses, channels, documentTypes, financialTransactionTypes, messageStatuses, plans, priorities, roles } from './domain'
import type { Plan } from './domain'
import './App.css'

type View = 'login' | 'register' | 'pending'

function App() {
  const [view, setView] = useState<View>('login')
  const [session, setSession] = useState<Session | null>(() => {
    const stored = sessionStorage.getItem('bcb-session')
    return stored ? (JSON.parse(stored) as Session) : null
  })
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function saveSession(next: Session | null) {
    setSession(next)
    if (next) sessionStorage.setItem('bcb-session', JSON.stringify(next))
    else sessionStorage.removeItem('bcb-session')
  }

  async function submitLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    await perform(async () => {
      const next = await api.login({ login: data.get('login'), password: data.get('password') })
      saveSession(next)
    })
  }

  async function submitRegistration(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    await perform(async () => {
      const response = await api.register({
        name: data.get('name'), documentType: data.get('documentType'),
        documentId: data.get('documentId'), password: data.get('password'),
        requestedPlan: data.get('requestedPlan'),
      })
      setMessage(response.message)
      setView('pending')
    })
  }

  async function perform(action: () => Promise<void>) {
    setLoading(true)
    setError('')
    try { await action() } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Erro inesperado.')
    } finally { setLoading(false) }
  }

  if (session?.user.role === roles.admin) {
    return <AdminDashboard session={session} onLogout={() => saveSession(null)} />
  }
  if (session?.user.role === roles.client) {
    return <ClientHome session={session} onLogout={() => saveSession(null)} />
  }

  return (
    <main className="shell">
      <section className="brand-panel">
        <span className="eyebrow">Big Chat Brasil</span>
        <h1>Conversas que respeitam cada centavo.</h1>
        <p>Mensagens empresariais com prioridade, controle financeiro e rastreabilidade.</p>
      </section>

      <section className="form-panel">
        {view === 'pending' ? (
          <div className="notice">
            <span className="notice-icon">✓</span>
            <h2>Cadastro recebido</h2>
            <p>{message}</p>
            <button type="button" onClick={() => setView('login')}>Voltar ao acesso</button>
          </div>
        ) : (
          <>
            <div className="tabs" aria-label="Acesso">
              <button className={view === 'login' ? 'active' : ''} onClick={() => setView('login')}>Entrar</button>
              <button className={view === 'register' ? 'active' : ''} onClick={() => setView('register')}>Criar conta</button>
            </div>
            {view === 'login' ? (
              <form onSubmit={submitLogin}>
                <h2>Boas-vindas</h2>
                <label>CPF, CNPJ ou login administrativo<input name="login" required autoComplete="username" /></label>
                <label>Senha<input name="password" type="password" required autoComplete="current-password" /></label>
                <Submit loading={loading} label="Entrar" />
              </form>
            ) : (
              <form onSubmit={submitRegistration}>
                <h2>Cadastre sua empresa</h2>
                <label>Nome ou razão social<input name="name" required /></label>
                <div className="field-row">
                  <label>Documento<select name="documentType"><option value={documentTypes.cpf}>CPF</option><option value={documentTypes.cnpj}>CNPJ</option></select></label>
                  <label>Número<input name="documentId" inputMode="numeric" required /></label>
                </div>
                <label>Plano desejado<select name="requestedPlan"><option value={plans.prepaid}>Pré-pago</option><option value={plans.postpaid}>Pós-pago</option></select></label>
                <label>Senha<input name="password" type="password" minLength={9} maxLength={128} required autoComplete="new-password" /><small>9+ caracteres, com letras, números e caractere especial.</small></label>
                <Submit loading={loading} label="Solicitar cadastro" />
              </form>
            )}
          </>
        )}
        {error && <p className="error" role="alert">{error}</p>}
      </section>
    </main>
  )
}

function Submit({ loading, label }: { loading: boolean; label: string }) {
  return <button className="primary" disabled={loading}>{loading ? 'Processando…' : label}</button>
}

function ClientHome({ session, onLogout }: { session: Session; onLogout: () => void }) {
  const token = session.accessToken
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selected, setSelected] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [billing, setBilling] = useState<BillingProfile | null>(null)
  const [transactions, setTransactions] = useState<FinancialTransaction[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingConversations, setLoadingConversations] = useState(true)

  const refreshBilling = useCallback(async () => {
    const [billingResponse, transactionResponse] = await Promise.all([
      api.billing(token),
      api.billingTransactions(token),
    ])
    setBilling(billingResponse)
    setTransactions(transactionResponse.items)
  }, [token])

  const loadMessages = useCallback(async (conversation: Conversation, silent = false) => {
    if (!silent) setMessages([])
    if (!silent) setError('')
    try {
      const response = await api.messages(token, conversation.id)
      setMessages(response.items)
    } catch (reason) {
      if (!silent) setError(reason instanceof Error ? reason.message : 'Erro inesperado.')
    }
  }, [token])

  async function refresh() {
    try {
      const response = await api.conversations(token)
      setConversations(response.items)
      setError('')
      if (selected && !response.items.some(conversation => conversation.id === selected.id)) setSelected(null)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Erro inesperado.')
    }
  }

  useEffect(() => {
    void api.conversations(token)
      .then(response => { setConversations(response.items); setError('') })
      .catch(reason => setError(reason instanceof Error ? reason.message : 'Erro inesperado.'))
      .finally(() => setLoadingConversations(false))
    void Promise.all([api.billing(token), api.billingTransactions(token)])
      .then(([billingResponse, transactionResponse]) => {
        setBilling(billingResponse)
        setTransactions(transactionResponse.items)
      })
      .catch(reason => setError(reason instanceof Error ? reason.message : 'Erro inesperado.'))
  }, [token])

  useEffect(() => {
    if (!selected) return undefined
    const interval = window.setInterval(() => {
      void loadMessages(selected, true)
      void refreshBilling().catch(() => undefined)
    }, 2000)
    return () => window.clearInterval(interval)
  }, [loadMessages, refreshBilling, selected])

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
      await refresh()
      await openConversation(conversation)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Erro inesperado.')
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
      await refresh()
      await refreshBilling()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Erro inesperado.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="dashboard">
      <header>
        <div><span className="eyebrow">BCB</span><h1>Conversas</h1></div>
        <button onClick={onLogout}>Sair</button>
      </header>
      {error && <p className="error">{error}</p>}
      {billing && (
        <section className="summary-grid">
          <article className="metric-card">
            <span>Plano atual</span>
            <strong>{billing.planType === plans.prepaid ? 'Pré-pago' : 'Pós-pago'}</strong>
          </article>
          <article className="metric-card">
            <span>Disponível</span>
            <strong>{formatMoney(billing.currentPlanAvailableCents)}</strong>
          </article>
          <article className="metric-card">
            <span>{billing.planType === plans.prepaid ? 'Saldo' : 'Consumo / limite'}</span>
            <strong>
              {billing.planType === plans.prepaid
                ? formatMoney(billing.prepaidBalanceCents)
                : `${formatMoney(billing.postpaidConsumedCents)} / ${formatMoney(billing.postpaidTotalLimitCents)}`}
            </strong>
          </article>
        </section>
      )}
      <section className="conversation-layout">
        <aside className="conversation-sidebar">
          <form onSubmit={createConversation} className="compact-form">
            <h2>Novo destinatário</h2>
            <label>Nome<input name="name" required /></label>
            <label>Telefone E.164<input name="phone" placeholder="+5511999999999" required /></label>
            <Submit loading={loading} label="Criar conversa" />
          </form>
          <div className="conversation-list">
            {loadingConversations ? (
              <div className="empty-state"><h2>Carregando conversas</h2><p>Buscando o estado salvo no backend…</p></div>
            ) : conversations.length === 0 ? (
              <div className="empty-state"><h2>Nenhuma conversa</h2><p>Cadastre um destinatário para iniciar o fluxo.</p></div>
            ) : conversations.map(conversation => (
              <button
                key={conversation.id}
                className={`conversation-item ${selected?.id === conversation.id ? 'active' : ''}`}
                onClick={() => void openConversation(conversation)}
              >
                <strong>{conversation.recipient.name}</strong>
                <span>{conversation.recipient.phone}</span>
                <small>{conversation.lastActivityAt ? new Date(conversation.lastActivityAt).toLocaleString('pt-BR') : 'Sem atividade recente'}</small>
              </button>
            ))}
          </div>
        </aside>
        <section className="conversation-panel">
          {!selected ? (
            <div className="empty-state"><h2>Selecione uma conversa</h2><p>O histórico aparecerá aqui.</p></div>
          ) : (
            <>
              <div className="conversation-heading">
                <h2>{selected.recipient.name}</h2>
                <p>{selected.recipient.phone}</p>
              </div>
              {messages.length === 0 ? (
                <div className="empty-state"><h2>Sem mensagens ainda</h2><p>O envio entra na próxima etapa.</p></div>
              ) : (
                <div className="message-list">{messages.map(message => (
                  <article key={message.id} className={`message-card ${message.status}`}>
                    <p>{message.content}</p>
                    <footer>
                      <span>{message.channel.toUpperCase()} · {message.priority === priorities.urgent ? 'Urgente' : 'Normal'} · {formatMoney(message.costCents)}</span>
                      <strong>{messageStatusLabel(message.status)}</strong>
                    </footer>
                    {message.failureCode && <small>Falha: {message.failureCode}</small>}
                  </article>
                ))}</div>
              )}
              <form onSubmit={sendMessage} className="message-form">
                <label>Mensagem<textarea name="content" required rows={3} placeholder="Digite a mensagem. Use [fail] ou [retry] para simular falhas." /></label>
                <div className="field-row">
                  <label>Canal<select name="channel" defaultValue={channels.whatsapp}><option value={channels.whatsapp}>WhatsApp</option><option value={channels.sms}>SMS</option></select></label>
                  <label>Prioridade<select name="priority" defaultValue={priorities.normal}><option value={priorities.normal}>Normal · R$ 0,25</option><option value={priorities.urgent}>Urgente · R$ 0,50</option></select></label>
                </div>
                <Submit loading={loading} label="Enviar mensagem" />
              </form>
            </>
          )}
        </section>
      </section>
      <section className="history-panel">
        <h2>Histórico financeiro</h2>
        {transactions.length === 0 ? (
          <p className="muted">Nenhuma movimentação financeira registrada ainda.</p>
        ) : transactions.slice(0, 5).map(transaction => (
          <div key={transaction.id} className="history-row">
            <span>{transactionLabel(transaction.type)}</span>
            <strong>{formatMoney(transaction.amountCents)}</strong>
            <small>{new Date(transaction.createdAt).toLocaleString('pt-BR')}</small>
          </div>
        ))}
      </section>
    </main>
  )
}

function AdminDashboard({ session, onLogout }: { session: Session; onLogout: () => void }) {
  const [clients, setClients] = useState<Client[]>([])
  const [error, setError] = useState('')
  const token = session.accessToken

  async function refresh() {
    try { setClients((await api.clients(token)).items); setError('') }
    catch (reason) { setError(reason instanceof Error ? reason.message : 'Erro inesperado.') }
  }
  useEffect(() => {
    void api.clients(token)
      .then(response => { setClients(response.items); setError('') })
      .catch(reason => setError(reason instanceof Error ? reason.message : 'Erro inesperado.'))
  }, [token])

  async function activate(client: Client) {
    const amount = Number(prompt(client.requestedPlan === plans.prepaid ? 'Saldo inicial em centavos' : 'Limite total em centavos', '0'))
    if (!Number.isSafeInteger(amount) || amount < 0) return
    const body = client.requestedPlan === plans.prepaid
      ? { planType: plans.prepaid as Plan, initialBalanceCents: amount }
      : { planType: plans.postpaid as Plan, totalLimitCents: amount }
    try { await api.activate(token, client.id, body); await refresh() } catch (reason) { setError(reason instanceof Error ? reason.message : 'Erro inesperado.') }
  }

  async function reject(client: Client) {
    const reason = prompt('Motivo da rejeição')?.trim()
    if (!reason) return
    try { await api.reject(token, client.id, reason); await refresh() } catch (failure) { setError(failure instanceof Error ? failure.message : 'Erro inesperado.') }
  }

  async function addCredit(client: Client) {
    const amount = Number(prompt('Valor do crédito em centavos', '1000'))
    if (!Number.isSafeInteger(amount) || amount <= 0) return
    const reason = prompt('Motivo da recarga')?.trim() ?? ''
    try {
      await api.addCredit(token, client.id, { amountCents: amount, reason }, idempotencyKey())
      await refresh()
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : 'Erro inesperado.')
    }
  }

  async function setPostpaidLimit(client: Client) {
    const amount = Number(prompt('Novo limite total em centavos', '5000'))
    if (!Number.isSafeInteger(amount) || amount <= 0) return
    const reason = prompt('Motivo do ajuste')?.trim() ?? ''
    try {
      await api.setPostpaidLimit(token, client.id, { totalLimitCents: amount, reason }, idempotencyKey())
      await refresh()
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : 'Erro inesperado.')
    }
  }

  async function showTransactions(client: Client) {
    try {
      const response = await api.adminFinancialTransactions(token, client.id)
      const summary = response.items.length === 0
        ? 'Nenhuma movimentação financeira registrada.'
        : response.items
          .slice(0, 10)
          .map(item => `${transactionLabel(item.type)} · ${formatMoney(item.amountCents)} · ${new Date(item.createdAt).toLocaleString('pt-BR')}`)
          .join('\n')
      alert(summary)
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : 'Erro inesperado.')
    }
  }

  return <main className="dashboard"><header><div><span className="eyebrow">Administração</span><h1>Clientes</h1></div><button onClick={onLogout}>Sair</button></header>{error && <p className="error">{error}</p>}<section className="client-list">{clients.length === 0 ? <div className="empty-state"><h2>Nenhum cadastro</h2><p>Novas solicitações aparecerão aqui.</p></div> : clients.map(client => <article key={client.id} className="client-card"><div><span className={`pill ${client.status}`}>{client.status}</span><h2>{client.name}</h2><p>{client.documentType.toUpperCase()} · {client.documentId}</p><small>Plano solicitado: {client.requestedPlan === plans.prepaid ? 'Pré-pago' : 'Pós-pago'}</small></div><div className="actions">{client.status !== clientStatuses.active ? <><button className="primary" onClick={() => void activate(client)}>Ativar</button>{client.status === clientStatuses.pending && <button onClick={() => void reject(client)}>Rejeitar</button>}</> : <><button className="primary" onClick={() => client.requestedPlan === plans.prepaid ? void addCredit(client) : void setPostpaidLimit(client)}>{client.requestedPlan === plans.prepaid ? 'Adicionar crédito' : 'Ajustar limite'}</button><button onClick={() => void showTransactions(client)}>Histórico</button></>}</div></article>)}</section></main>
}

function formatMoney(cents: number) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(cents / 100)
}

function transactionLabel(type: FinancialTransaction['type']) {
  const labels: Record<FinancialTransaction['type'], string> = {
    [financialTransactionTypes.credit]: 'Crédito',
    [financialTransactionTypes.debit]: 'Débito',
    [financialTransactionTypes.consumption]: 'Consumo',
    [financialTransactionTypes.refund]: 'Estorno',
    [financialTransactionTypes.consumptionReversal]: 'Reversão de consumo',
  }
  return labels[type]
}

function messageStatusLabel(status: Message['status']) {
  const labels: Record<Message['status'], string> = {
    [messageStatuses.queued]: 'Na fila',
    [messageStatuses.processing]: 'Processando',
    [messageStatuses.sent]: 'Enviada',
    [messageStatuses.failed]: 'Falhou',
  }
  return labels[status]
}

function idempotencyKey() {
  return crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export default App
