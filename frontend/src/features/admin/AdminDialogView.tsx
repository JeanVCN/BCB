import type { FormEvent } from 'react'
import { plans } from '../../domain'
import type { AdminDialog } from '../../types/ui'
import { formatMoney, planLabel, transactionLabel } from '../../utils/format'

export function AdminDialogView({
  dialog,
  busy,
  onChange,
  onClose,
  onSubmit,
}: {
  dialog: AdminDialog | null
  busy: boolean
  onChange: (dialog: AdminDialog) => void
  onClose: () => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  if (!dialog) return null

  if (dialog.type === 'transactions') {
    return (
      <div className="modal-backdrop" role="presentation">
        <section className="modal" role="dialog" aria-modal="true" aria-labelledby="transactions-title">
          <div className="modal-heading">
            <div>
              <span className="eyebrow">Histórico financeiro</span>
              <h2 id="transactions-title">{dialog.client.name}</h2>
            </div>
            <button type="button" onClick={onClose} aria-label="Fechar">x</button>
          </div>
          <div className="transaction-list">
            {dialog.items.length === 0 ? (
              <p className="muted">Nenhuma movimentação financeira registrada.</p>
            ) : dialog.items.slice(0, 10).map(item => (
              <div key={item.id} className="history-row">
                <span>{transactionLabel(item.type)}</span>
                <strong>{formatMoney(item.amountCents)}</strong>
                <small>{new Date(item.createdAt).toLocaleString('pt-BR')}</small>
              </div>
            ))}
          </div>
        </section>
      </div>
    )
  }

  const activeDialog = dialog
  const title = dialogTitle(activeDialog)
  const description = dialogDescription(activeDialog)
  const amountLabel = dialogAmountLabel(activeDialog)
  const amountValue = 'value' in activeDialog ? activeDialog.value : 'amount' in activeDialog ? activeDialog.amount : ''
  const reasonValue = 'reason' in activeDialog ? activeDialog.reason : ''

  function updateAmount(value: string) {
    if (activeDialog.type === 'activate' || activeDialog.type === 'approvePlan') onChange({ ...activeDialog, value })
    if (activeDialog.type === 'credit' || activeDialog.type === 'limit') onChange({ ...activeDialog, amount: value })
  }

  function updateReason(reason: string) {
    if ('reason' in activeDialog) onChange({ ...activeDialog, reason })
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <form className="modal" role="dialog" aria-modal="true" aria-labelledby="admin-dialog-title" onSubmit={onSubmit}>
        <div className="modal-heading">
          <div>
            <span className="eyebrow">Ação administrativa</span>
            <h2 id="admin-dialog-title">{title}</h2>
          </div>
          <button type="button" onClick={onClose} aria-label="Fechar">x</button>
        </div>
        <p className="muted">{description}</p>
        {amountLabel && (
          <label>{amountLabel}
            <input value={amountValue} inputMode="numeric" onChange={event => updateAmount(event.currentTarget.value)} required />
          </label>
        )}
        {'reason' in activeDialog && (
          <label>Motivo
            <textarea rows={3} value={reasonValue} onChange={event => updateReason(event.currentTarget.value)} required={activeDialog.type === 'rejectClient' || activeDialog.type === 'rejectPlan'} />
          </label>
        )}
        {activeDialog.type === 'zero' && (
          <div className="warning-box">
            <strong>{formatMoney(activeDialog.currentAmount)}</strong>
            <span>Valor atual de {activeDialog.actionLabel} que será compensado.</span>
          </div>
        )}
        <div className="modal-actions">
          <button type="button" onClick={onClose}>Cancelar</button>
          <button className="primary" disabled={busy}>{busy ? 'Processando...' : 'Confirmar'}</button>
        </div>
      </form>
    </div>
  )
}

function dialogTitle(dialog: Exclude<AdminDialog, { type: 'transactions' }>) {
  if (dialog.type === 'activate') return `Ativar ${dialog.client.name}`
  if (dialog.type === 'rejectClient') return `Rejeitar ${dialog.client.name}`
  if (dialog.type === 'credit') return `Adicionar crédito para ${dialog.client.name}`
  if (dialog.type === 'limit') return `Ajustar limite de ${dialog.client.name}`
  if (dialog.type === 'zero') return `Zerar ${dialog.actionLabel}`
  if (dialog.type === 'approvePlan') return 'Aprovar mudança de plano'
  return 'Rejeitar mudança de plano'
}

function dialogDescription(dialog: Exclude<AdminDialog, { type: 'transactions' }>) {
  if (dialog.type === 'activate') return `Confirme o plano ${planLabel(dialog.client.requestedPlan)} e informe o valor inicial em centavos.`
  if (dialog.type === 'rejectClient') return 'A rejeição preserva o cadastro para auditoria e impede duplicidade do documento.'
  if (dialog.type === 'credit') return 'O crédito entra no histórico financeiro com idempotência e auditoria.'
  if (dialog.type === 'limit') return 'O limite total será atualizado sem apagar o consumo já registrado.'
  if (dialog.type === 'zero') return 'Use esta ação para liberar a aprovação de mudança de plano quando fizer sentido operacional.'
  if (dialog.type === 'approvePlan') return `Destino: ${planLabel(dialog.request.toPlan)}. Informe ${dialog.request.toPlan === plans.prepaid ? 'saldo inicial' : 'limite total'} em centavos.`
  return 'Informe um motivo claro para registrar a decisão administrativa.'
}

function dialogAmountLabel(dialog: Exclude<AdminDialog, { type: 'transactions' }>) {
  if (dialog.type === 'activate') return dialog.client.requestedPlan === plans.prepaid ? 'Saldo inicial em centavos' : 'Limite total em centavos'
  if (dialog.type === 'credit') return 'Valor do crédito em centavos'
  if (dialog.type === 'limit') return 'Novo limite total em centavos'
  if (dialog.type === 'approvePlan') return dialog.request.toPlan === plans.prepaid ? 'Saldo inicial em centavos' : 'Limite total em centavos'
  return ''
}
