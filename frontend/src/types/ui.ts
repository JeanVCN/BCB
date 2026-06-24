import type { Client, FinancialTransaction, PlanChangeRequest } from '../api'

export type Toast = { type: 'success' | 'error'; text: string }

export type AdminDialog =
  | { type: 'activate'; client: Client; value: string }
  | { type: 'rejectClient'; client: Client; reason: string }
  | { type: 'credit'; client: Client; amount: string; reason: string }
  | { type: 'limit'; client: Client; amount: string; reason: string }
  | { type: 'zero'; client: Client; currentAmount: number; actionLabel: string; reason: string }
  | { type: 'transactions'; client: Client; items: FinancialTransaction[] }
  | { type: 'approvePlan'; request: PlanChangeRequest; value: string }
  | { type: 'rejectPlan'; request: PlanChangeRequest; reason: string }
