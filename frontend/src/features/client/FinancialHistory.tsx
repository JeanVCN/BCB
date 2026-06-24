import type { FinancialTransaction } from '../../api'
import { formatMoney, transactionLabel } from '../../utils/format'

export function FinancialHistory({ transactions }: { transactions: FinancialTransaction[] }) {
  return (
    <section className="history-panel" id="financeiro">
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
  )
}
