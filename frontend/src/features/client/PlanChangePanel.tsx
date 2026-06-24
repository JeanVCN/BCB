import type { BillingProfile, PlanChangeRequest } from '../../api'
import { planRequestStatuses, plans } from '../../domain'
import { planLabel, planRequestStatusLabel } from '../../utils/format'

export function PlanChangePanel({
  billing,
  planChange,
  loading,
  onRequest,
  onCancel,
}: {
  billing: BillingProfile
  planChange: PlanChangeRequest | null
  loading: boolean
  onRequest: () => void
  onCancel: () => void
}) {
  return (
    <section className="plan-change-panel">
      <div>
        <h2>Mudança de plano</h2>
        <p>
          {planChange
            ? `Última solicitação: ${planLabel(planChange.fromPlan)} para ${planLabel(planChange.toPlan)} - ${planRequestStatusLabel(planChange.status)}`
            : `Você pode solicitar troca para ${billing.planType === plans.prepaid ? 'pós-pago' : 'pré-pago'} quando não houver valor financeiro pendente.`}
        </p>
        {planChange?.rejectionReason && <small>Motivo: {planChange.rejectionReason}</small>}
      </div>
      {planChange?.status === planRequestStatuses.pending ? (
        <button onClick={onCancel} disabled={loading}>Cancelar solicitação</button>
      ) : (
        <button className="primary" onClick={onRequest} disabled={loading}>
          Solicitar {billing.planType === plans.prepaid ? 'pós-pago' : 'pré-pago'}
        </button>
      )}
    </section>
  )
}
