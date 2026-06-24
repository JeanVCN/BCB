import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import type { Client, PlanChangeRequest, Session } from '../../api'
import { APIRequestError, api } from '../../api'
import { AppHeader } from '../../components/layout/AppHeader'
import { ToastMessage } from '../../components/ui/ToastMessage'
import { clientStatuses, plans } from '../../domain'
import type { Plan } from '../../domain'
import type { AdminDialog, Toast } from '../../types/ui'
import { errorMessage } from '../../utils/errors'
import { idempotencyKey } from '../../utils/forms'
import { clientStatusLabel, planLabel, planRequestStatusLabel } from '../../utils/format'
import { AdminDialogView } from './AdminDialogView'

type AdminView = 'dashboard' | 'requests' | 'clients'

export function AdminDashboard({ session, onLogout }: { session: Session; onLogout: () => void }) {
  const [clients, setClients] = useState<Client[]>([])
  const [planRequests, setPlanRequests] = useState<PlanChangeRequest[]>([])
  const [summary, setSummary] = useState<{ pendingClientActivations: number; pendingPlanChanges: number } | null>(null)
  const [error, setError] = useState('')
  const [toast, setToast] = useState<Toast | null>(null)
  const [dialog, setDialog] = useState<AdminDialog | null>(null)
  const [activeView, setActiveView] = useState<AdminView>('dashboard')
  const [busy, setBusy] = useState(false)
  const token = session.accessToken

  const adminStats = useMemo(() => {
    const active = clients.filter(client => client.status === clientStatuses.active).length
    const pending = clients.filter(client => client.status === clientStatuses.pending).length
    const blocked = clients.filter(client => client.status === clientStatuses.inactive || client.status === clientStatuses.rejected).length
    return { active, pending, blocked, total: clients.length }
  }, [clients])

  async function refresh() {
    try {
      const [clientResponse, summaryResponse, requestResponse] = await Promise.all([
        api.clients(token),
        api.adminSummary(token),
        api.adminPlanChangeRequests(token),
      ])
      setClients(clientResponse.items)
      setSummary(summaryResponse)
      setPlanRequests(requestResponse.items)
      setError('')
    } catch (reason) {
      setError(errorMessage(reason))
    }
  }

  useEffect(() => {
    void Promise.all([
      api.clients(token),
      api.adminSummary(token),
      api.adminPlanChangeRequests(token),
    ])
      .then(([clientResponse, summaryResponse, requestResponse]) => {
        setClients(clientResponse.items)
        setSummary(summaryResponse)
        setPlanRequests(requestResponse.items)
        setError('')
      })
      .catch(reason => setError(errorMessage(reason)))
  }, [token])

  async function submitDialog(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!dialog) return
    setBusy(true)
    setError('')
    try {
      if (dialog.type === 'activate') await activate(dialog.client, dialog.value)
      if (dialog.type === 'rejectClient') await reject(dialog.client, dialog.reason)
      if (dialog.type === 'credit') await addCredit(dialog.client, dialog.amount, dialog.reason)
      if (dialog.type === 'limit') await setPostpaidLimit(dialog.client, dialog.amount, dialog.reason)
      if (dialog.type === 'zero') await zeroCurrentBalanceSubmit(dialog.client, dialog.reason)
      if (dialog.type === 'approvePlan') await approvePlanChange(dialog.request, dialog.value)
      if (dialog.type === 'rejectPlan') await rejectPlanChange(dialog.request, dialog.reason)
      setDialog(null)
      await refresh()
    } catch (failure) {
      if (dialog.type === 'approvePlan' && failure instanceof APIRequestError && failure.code === 'financial_state_blocks_plan_change') {
        await openZeroBalanceForPlanRequest(dialog.request)
        return
      }
      setError(errorMessage(failure))
    } finally {
      setBusy(false)
    }
  }

  async function activate(client: Client, value: string) {
    const amount = Number(value)
    if (!Number.isSafeInteger(amount) || amount < 0) throw new Error('Informe um valor inteiro em centavos.')
    const body = client.requestedPlan === plans.prepaid
      ? { planType: plans.prepaid as Plan, initialBalanceCents: amount }
      : { planType: plans.postpaid as Plan, totalLimitCents: amount }
    await api.activate(token, client.id, body)
    setToast({ type: 'success', text: `${client.name} ativado com sucesso.` })
  }

  async function reject(client: Client, reasonInput: string) {
    const reason = reasonInput.trim()
    if (!reason) throw new Error('Informe o motivo da rejeição.')
    await api.reject(token, client.id, reason)
    setToast({ type: 'success', text: 'Cadastro rejeitado e auditado.' })
  }

  async function prepareCredit(client: Client) {
    const billing = await loadAdminBilling(client)
    if (!billing) return
    if (billing.planType !== plans.prepaid) {
      setError('Crédito administrativo está disponível apenas para clientes pré-pagos.')
      return
    }
    setDialog({ type: 'credit', client, amount: '1000', reason: 'Recarga administrativa' })
  }

  async function addCredit(client: Client, value: string, reason: string) {
    const amount = Number(value)
    if (!Number.isSafeInteger(amount) || amount <= 0) throw new Error('Informe um crédito maior que zero.')
    await api.addCredit(token, client.id, { amountCents: amount, reason }, idempotencyKey())
    setToast({ type: 'success', text: 'Crédito registrado.' })
  }

  async function preparePostpaidLimit(client: Client) {
    const billing = await loadAdminBilling(client)
    if (!billing) return
    if (billing.planType !== plans.postpaid) {
      setError('Ajuste de limite está disponível apenas para clientes pós-pagos.')
      return
    }
    setDialog({ type: 'limit', client, amount: String(billing.postpaidTotalLimitCents || 5000), reason: 'Revisão de limite' })
  }

  async function setPostpaidLimit(client: Client, value: string, reason: string) {
    const amount = Number(value)
    if (!Number.isSafeInteger(amount) || amount <= 0) throw new Error('Informe um limite maior que zero.')
    await api.setPostpaidLimit(token, client.id, { totalLimitCents: amount, reason }, idempotencyKey())
    setToast({ type: 'success', text: 'Limite pós-pago atualizado.' })
  }

  async function loadAdminBilling(client: Client) {
    try {
      return await api.adminBilling(token, client.id)
    } catch (failure) {
      setError(errorMessage(failure))
      return null
    }
  }

  async function zeroCurrentBalance(client: Client) {
    const billing = await loadAdminBilling(client)
    if (!billing) return false
    const currentAmount = billing.planType === plans.prepaid ? billing.prepaidBalanceCents : billing.postpaidConsumedCents
    const actionLabel = billing.planType === plans.prepaid ? 'saldo pré-pago' : 'consumo pós-pago'
    setDialog({ type: 'zero', client, currentAmount, actionLabel, reason: 'Preparação para mudança de plano' })
    return true
  }

  async function openZeroBalanceForPlanRequest(request: PlanChangeRequest) {
    const client = clients.find(item => item.id === request.clientId)
    if (!client) {
      setError('Cliente da solicitação não foi encontrado na lista administrativa.')
      return
    }
    const opened = await zeroCurrentBalance(client)
    if (!opened) return
    setToast({ type: 'error', text: 'Antes de aprovar a mudança de plano, zere o saldo ou consumo pendente deste cliente.' })
  }

  async function zeroCurrentBalanceSubmit(client: Client, reason: string) {
    await api.zeroCurrentBalance(token, client.id, { reason }, idempotencyKey())
    setToast({ type: 'success', text: 'Saldo/consumo zerado com auditoria.' })
  }

  async function showTransactions(client: Client) {
    try {
      const response = await api.adminFinancialTransactions(token, client.id)
      setDialog({ type: 'transactions', client, items: response.items })
    } catch (failure) {
      setError(errorMessage(failure))
    }
  }

  async function approvePlanChange(request: PlanChangeRequest, value: string) {
    const amount = Number(value)
    if (!Number.isSafeInteger(amount) || amount < 0) throw new Error('Informe um valor inteiro em centavos.')
    const body = request.toPlan === plans.prepaid
      ? { initialBalanceCents: amount }
      : { totalLimitCents: amount }
    await api.approvePlanChangeRequest(token, request.id, body)
    setToast({ type: 'success', text: 'Mudança de plano aprovada.' })
  }

  async function rejectPlanChange(request: PlanChangeRequest, reasonInput: string) {
    const reason = reasonInput.trim()
    if (!reason) throw new Error('Informe o motivo da rejeição.')
    await api.rejectPlanChangeRequest(token, request.id, reason)
    setToast({ type: 'success', text: 'Mudança de plano rejeitada.' })
  }

  return (
    <main className={`dashboard app-shell view-${activeView}`}>
      <ToastMessage toast={toast} onClose={() => setToast(null)} />
      <AdminDialogView dialog={dialog} busy={busy} onChange={setDialog} onClose={() => setDialog(null)} onSubmit={submitDialog} />
      <AppHeader eyebrow="Administração" title={adminViewTitle(activeView)}>
        <button className={activeView === 'dashboard' ? 'active' : ''} onClick={() => setActiveView('dashboard')}>Dashboard</button>
        <button className={activeView === 'requests' ? 'active' : ''} onClick={() => setActiveView('requests')}>Solicitações</button>
        <button className={activeView === 'clients' ? 'active' : ''} onClick={() => setActiveView('clients')}>Clientes</button>
        <button onClick={onLogout}>Sair</button>
      </AppHeader>
      {error && <p className="error">{error}</p>}
      {activeView === 'dashboard' && summary && (
        <section className="page-view overview-view">
          <AdminMetrics summary={summary} stats={adminStats} />
          <div className="operations-grid">
            <article className="operation-panel primary-panel">
              <span className="eyebrow">Fila</span>
              <h2>{adminStats.pending + summary.pendingPlanChanges}</h2>
              <p>Itens aguardando decisão administrativa.</p>
              <button className="primary" onClick={() => setActiveView(summary.pendingPlanChanges > 0 ? 'requests' : 'clients')}>Abrir fila</button>
            </article>
            <article className="operation-panel">
              <span className="eyebrow">Clientes</span>
              <h2>{adminStats.active} ativos</h2>
              <p>{adminStats.blocked} bloqueados ou rejeitados.</p>
              <button onClick={() => setActiveView('clients')}>Gerenciar clientes</button>
            </article>
            <article className="operation-panel">
              <span className="eyebrow">Mudanças de plano</span>
              <h2>{summary.pendingPlanChanges}</h2>
              <p>Solicitações pendentes de aprovação ou rejeição.</p>
              <button onClick={() => setActiveView('requests')}>Ver solicitações</button>
            </article>
          </div>
        </section>
      )}
      {activeView === 'requests' && <section className="admin-section page-view">
        <h2>Solicitações de mudança de plano</h2>
        {planRequests.length === 0 ? (
          <p className="muted">Nenhuma solicitação pendente.</p>
        ) : planRequests.map(request => (
          <article key={request.id} className="client-card">
            <div>
              <span className={`pill ${request.status}`}>{planRequestStatusLabel(request.status)}</span>
              <h2>{request.clientName || request.clientId}</h2>
              <p>{planLabel(request.fromPlan)} para {planLabel(request.toPlan)}</p>
              <small>Solicitada em {new Date(request.createdAt).toLocaleString('pt-BR')}</small>
            </div>
            <div className="actions">
              <button className="primary" onClick={() => setDialog({ type: 'approvePlan', request, value: '0' })}>Aprovar</button>
              <button onClick={() => setDialog({ type: 'rejectPlan', request, reason: '' })}>Rejeitar</button>
            </div>
          </article>
        ))}
      </section>}
      {activeView === 'clients' && <section className="client-list page-view">
        {clients.length === 0 ? (
          <div className="empty-state"><h2>Nenhum cadastro</h2><p>Novas solicitações aparecerão aqui.</p></div>
        ) : clients.map(client => (
          <article key={client.id} className="client-card">
            <div>
              <span className={`pill ${client.status}`}>{clientStatusLabel(client.status)}</span>
              <h2>{client.name}</h2>
              <p>{client.documentType.toUpperCase()} - {client.documentId}</p>
              <small>Plano solicitado: {client.requestedPlan === plans.prepaid ? 'Pré-pago' : 'Pós-pago'}</small>
            </div>
            <div className="actions">
              {client.status !== clientStatuses.active ? (
                <>
                  <button className="primary" onClick={() => setDialog({ type: 'activate', client, value: '0' })}>Ativar</button>
                  {client.status === clientStatuses.pending && <button onClick={() => setDialog({ type: 'rejectClient', client, reason: '' })}>Rejeitar</button>}
                </>
              ) : (
                <>
                  <button className="primary" onClick={() => void prepareCredit(client)}>Adicionar crédito</button>
                  <button onClick={() => void preparePostpaidLimit(client)}>Ajustar limite</button>
                  <button onClick={() => void zeroCurrentBalance(client)}>Zerar saldo/consumo</button>
                  <button onClick={() => void showTransactions(client)}>Histórico</button>
                </>
              )}
            </div>
          </article>
        ))}
      </section>}
    </main>
  )
}

function AdminMetrics({
  summary,
  stats,
}: {
  summary: { pendingClientActivations: number; pendingPlanChanges: number }
  stats: { active: number; pending: number; blocked: number; total: number }
}) {
  return (
    <section className="summary-grid admin-metrics">
      <article className="metric-card"><span>Cadastros pendentes</span><strong>{summary.pendingClientActivations}</strong></article>
      <article className="metric-card"><span>Mudanças de plano</span><strong>{summary.pendingPlanChanges}</strong></article>
      <article className="metric-card"><span>Clientes ativos</span><strong>{stats.active}</strong></article>
      <article className="metric-card"><span>Total de clientes</span><strong>{stats.total}</strong></article>
      <article className="metric-card"><span>Bloqueados/rejeitados</span><strong>{stats.blocked}</strong></article>
      <article className="metric-card"><span>Fila de aprovação</span><strong>{stats.pending + summary.pendingPlanChanges}</strong></article>
    </section>
  )
}

function adminViewTitle(view: AdminView) {
  if (view === 'requests') return 'Solicitações'
  if (view === 'clients') return 'Clientes'
  return 'Painel operacional'
}
