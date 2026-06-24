import { useState } from 'react'
import type { FormEvent } from 'react'
import { api } from '../../api'
import type { Session } from '../../api'
import { SubmitButton } from '../../components/ui/SubmitButton'
import { documentTypes, plans } from '../../domain'
import { errorMessage } from '../../utils/errors'

type View = 'login' | 'register' | 'pending'

export function AuthScreen({ onSession }: { onSession: (session: Session) => void }) {
  const [view, setView] = useState<View>('login')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submitLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    await perform(async () => {
      const next = await api.login({ login: data.get('login'), password: data.get('password') })
      onSession(next)
    })
  }

  async function submitRegistration(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    await perform(async () => {
      const response = await api.register({
        name: data.get('name'),
        documentType: data.get('documentType'),
        documentId: data.get('documentId'),
        password: data.get('password'),
        requestedPlan: data.get('requestedPlan'),
      })
      setMessage(response.message)
      setView('pending')
    })
  }

  async function perform(action: () => Promise<void>) {
    setLoading(true)
    setError('')
    try {
      await action()
    } catch (reason) {
      setError(errorMessage(reason))
    } finally {
      setLoading(false)
    }
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
            <span className="notice-icon">OK</span>
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
                <SubmitButton loading={loading} label="Entrar" />
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
                <SubmitButton loading={loading} label="Solicitar cadastro" />
              </form>
            )}
          </>
        )}
        {error && <p className="error" role="alert">{error}</p>}
      </section>
    </main>
  )
}
