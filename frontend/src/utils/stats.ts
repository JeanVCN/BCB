import type { Message } from '../api'
import { messageStatuses } from '../domain'

export type MessageStats = { sent: number; failed: number; processing: number }

export function conversationMessageStats(messages: Message[]): MessageStats {
  return {
    sent: messages.filter(message => message.status === messageStatuses.sent).length,
    failed: messages.filter(message => message.status === messageStatuses.failed).length,
    processing: messages.filter(message => message.status === messageStatuses.processing || message.status === messageStatuses.queued).length,
  }
}
