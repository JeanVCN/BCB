import { APIRequestError } from '../api'

export function errorMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : 'Erro inesperado.'
}

export function nullWhenNotFound(reason: unknown) {
  if (reason instanceof APIRequestError && reason.status === 404) return null
  throw reason
}
