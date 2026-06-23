export const documentTypes = {
  cpf: 'cpf',
  cnpj: 'cnpj',
} as const

export type DocumentType = typeof documentTypes[keyof typeof documentTypes]

export const roles = {
  admin: 'admin',
  client: 'client',
} as const

export type Role = typeof roles[keyof typeof roles]

export const plans = {
  prepaid: 'prepaid',
  postpaid: 'postpaid',
} as const

export type Plan = typeof plans[keyof typeof plans]

export const channels = {
  sms: 'sms',
  whatsapp: 'whatsapp',
} as const

export type Channel = typeof channels[keyof typeof channels]

export const priorities = {
  normal: 'normal',
  urgent: 'urgent',
} as const

export type Priority = typeof priorities[keyof typeof priorities]

export const messageStatuses = {
  queued: 'queued',
  processing: 'processing',
  sent: 'sent',
  failed: 'failed',
} as const

export type MessageStatus = typeof messageStatuses[keyof typeof messageStatuses]

export const clientStatuses = {
  pending: 'pending',
  active: 'active',
  inactive: 'inactive',
  rejected: 'rejected',
} as const

export type ClientStatus = typeof clientStatuses[keyof typeof clientStatuses]

export const financialTransactionTypes = {
  credit: 'credit',
  debit: 'debit',
  consumption: 'consumption',
  refund: 'refund',
  consumptionReversal: 'consumption_reversal',
} as const

export type FinancialTransactionType = typeof financialTransactionTypes[keyof typeof financialTransactionTypes]
