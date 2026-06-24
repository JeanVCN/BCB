import type { ChangeEvent } from 'react'

export function idempotencyKey() {
  return crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function maskPhoneInput(event: ChangeEvent<HTMLInputElement>) {
  const raw = event.currentTarget.value.replace(/[^\d+]/g, '')
  const hasPlus = raw.startsWith('+')
  const digits = raw.replace(/\D/g, '').slice(0, 15)
  if (!digits) {
    event.currentTarget.value = hasPlus ? '+' : ''
    return
  }
  if (digits.startsWith('55') && digits.length > 4) {
    const country = digits.slice(0, 2)
    const area = digits.slice(2, 4)
    const first = digits.length > 10 ? digits.slice(4, 9) : digits.slice(4, 8)
    const second = digits.length > 10 ? digits.slice(9, 13) : digits.slice(8, 12)
    event.currentTarget.value = `+${country} ${area}${first ? ` ${first}` : ''}${second ? `-${second}` : ''}`
    return
  }
  event.currentTarget.value = `${hasPlus ? '+' : ''}${digits}`
}
