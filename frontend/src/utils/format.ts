import type { Client, FinancialTransaction, Message, PlanChangeRequest } from '../api'
import { clientStatuses, financialTransactionTypes, messageStatuses, planRequestStatuses, plans } from '../domain'
import type { Plan } from '../domain'

export function formatMoney(cents: number) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(cents / 100)
}

export function transactionLabel(type: FinancialTransaction['type']) {
  const labels: Record<FinancialTransaction['type'], string> = {
    [financialTransactionTypes.credit]: 'Crédito',
    [financialTransactionTypes.debit]: 'Débito',
    [financialTransactionTypes.consumption]: 'Consumo',
    [financialTransactionTypes.refund]: 'Estorno',
    [financialTransactionTypes.consumptionReversal]: 'Reversão de consumo',
  }
  return labels[type]
}

export function planLabel(plan: Plan) {
  return plan === plans.prepaid ? 'Pré-pago' : 'Pós-pago'
}

export function planRequestStatusLabel(status: PlanChangeRequest['status']) {
  const labels: Record<PlanChangeRequest['status'], string> = {
    [planRequestStatuses.pending]: 'Pendente',
    [planRequestStatuses.approved]: 'Aprovada',
    [planRequestStatuses.rejected]: 'Rejeitada',
    [planRequestStatuses.cancelled]: 'Cancelada',
  }
  return labels[status]
}

export function messageStatusLabel(status: Message['status']) {
  const labels: Record<Message['status'], string> = {
    [messageStatuses.queued]: 'Na fila',
    [messageStatuses.processing]: 'Processando',
    [messageStatuses.sent]: 'Enviada',
    [messageStatuses.failed]: 'Falhou',
  }
  return labels[status]
}

export function clientStatusLabel(status: Client['status']) {
  const labels: Record<Client['status'], string> = {
    [clientStatuses.pending]: 'Pendente',
    [clientStatuses.active]: 'Ativo',
    [clientStatuses.inactive]: 'Inativo',
    [clientStatuses.rejected]: 'Rejeitado',
  }
  return labels[status]
}
