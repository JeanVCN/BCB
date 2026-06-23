import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { api } from './api'
import type { Client, Plan, Session } from './api'
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

  if (session?.user.role === 'admin') {
    return <AdminDashboard session={session} onLogout={() => saveSession(null)} />
  }
  if (session?.user.role === 'client') {
    return <ClientHome onLogout={() => saveSession(null)} />
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
                  <label>Documento<select name="documentType"><option value="cpf">CPF</option><option value="cnpj">CNPJ</option></select></label>
                  <label>Número<input name="documentId" inputMode="numeric" required /></label>
                </div>
                <label>Plano desejado<select name="requestedPlan"><option value="prepaid">Pré-pago</option><option value="postpaid">Pós-pago</option></select></label>
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

function ClientHome({ onLogout }: { onLogout: () => void }) {
  return <main className="dashboard"><header><div><span className="eyebrow">BCB</span><h1>Conta ativa</h1></div><button onClick={onLogout}>Sair</button></header><section className="empty-state"><h2>Próximo passo: conversas</h2><p>Seu acesso está funcionando. O financeiro e o chat chegam nos próximos incrementos.</p></section></main>
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
    const amount = Number(prompt(client.requestedPlan === 'prepaid' ? 'Saldo inicial em centavos' : 'Limite total em centavos', '0'))
    if (!Number.isSafeInteger(amount) || amount < 0) return
    const body = client.requestedPlan === 'prepaid'
      ? { planType: 'prepaid' as Plan, initialBalanceCents: amount }
      : { planType: 'postpaid' as Plan, totalLimitCents: amount }
    try { await api.activate(token, client.id, body); await refresh() } catch (reason) { setError(reason instanceof Error ? reason.message : 'Erro inesperado.') }
  }

  async function reject(client: Client) {
    const reason = prompt('Motivo da rejeição')?.trim()
    if (!reason) return
    try { await api.reject(token, client.id, reason); await refresh() } catch (failure) { setError(failure instanceof Error ? failure.message : 'Erro inesperado.') }
  }

  return <main className="dashboard"><header><div><span className="eyebrow">Administração</span><h1>Clientes</h1></div><button onClick={onLogout}>Sair</button></header>{error && <p className="error">{error}</p>}<section className="client-list">{clients.length === 0 ? <div className="empty-state"><h2>Nenhum cadastro</h2><p>Novas solicitações aparecerão aqui.</p></div> : clients.map(client => <article key={client.id} className="client-card"><div><span className={`pill ${client.status}`}>{client.status}</span><h2>{client.name}</h2><p>{client.documentType.toUpperCase()} · {client.documentId}</p><small>Plano solicitado: {client.requestedPlan === 'prepaid' ? 'Pré-pago' : 'Pós-pago'}</small></div>{client.status !== 'active' && <div className="actions"><button className="primary" onClick={() => void activate(client)}>Ativar</button>{client.status === 'pending' && <button onClick={() => void reject(client)}>Rejeitar</button>}</div>}</article>)}</section></main>
}

export default App
