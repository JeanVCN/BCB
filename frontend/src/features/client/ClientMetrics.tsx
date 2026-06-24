import type { BillingProfile, Conversation, Message } from '../../api'
import { messageStatuses, plans } from '../../domain'
import { formatMoney } from '../../utils/format'

export function ClientMetrics({
  billing,
  conversations,
  messages,
}: {
  billing: BillingProfile
  conversations: Conversation[]
  messages: Message[]
}) {
  const processing = messages.filter(message => message.status === messageStatuses.processing || message.status === messageStatuses.queued).length

  return (
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
      <article className="metric-card">
        <span>Conversas</span>
        <strong>{conversations.length}</strong>
      </article>
      <article className="metric-card">
        <span>Mensagens nesta conversa</span>
        <strong>{messages.length}</strong>
      </article>
      <article className="metric-card">
        <span>Em processamento</span>
        <strong>{processing}</strong>
      </article>
    </section>
  )
}
