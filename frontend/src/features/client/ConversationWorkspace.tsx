import type { FormEvent, RefObject } from 'react'
import type { Conversation, Message } from '../../api'
import { SubmitButton } from '../../components/ui/SubmitButton'
import { channels, priorities } from '../../domain'
import { formatMoney, messageStatusLabel } from '../../utils/format'
import { maskPhoneInput } from '../../utils/forms'
import type { MessageStats } from '../../utils/stats'

export function ConversationWorkspace({
  conversations,
  selected,
  messages,
  messageStats,
  loading,
  loadingConversations,
  messagesEndRef,
  onCreateConversation,
  onOpenConversation,
  onCloseConversation,
  onSendMessage,
}: {
  conversations: Conversation[]
  selected: Conversation | null
  messages: Message[]
  messageStats: MessageStats
  loading: boolean
  loadingConversations: boolean
  messagesEndRef: RefObject<HTMLDivElement | null>
  onCreateConversation: (event: FormEvent<HTMLFormElement>) => void
  onOpenConversation: (conversation: Conversation) => void
  onCloseConversation: () => void
  onSendMessage: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <section className={`conversation-layout chat-workspace ${selected ? 'has-selection' : ''}`}>
      <aside className="conversation-sidebar">
        <form onSubmit={onCreateConversation} className="compact-form">
          <h2>Novo destinatário</h2>
          <label>Nome<input name="name" required /></label>
          <label>Telefone<input name="phone" placeholder="+55 (55) 5555-5555" required onChange={maskPhoneInput} /></label>
          <SubmitButton loading={loading} label="Criar conversa" />
        </form>
        <ConversationList
          conversations={conversations}
          selected={selected}
          loading={loadingConversations}
          onOpenConversation={onOpenConversation}
        />
      </aside>
      <section className="conversation-panel">
        {!selected ? (
          <div className="empty-state"><h2>Selecione uma conversa</h2><p>O histórico aparecerá aqui.</p></div>
        ) : (
          <>
            <ConversationHeading conversation={selected} stats={messageStats} onBack={onCloseConversation} />
            <MessageList messages={messages} messagesEndRef={messagesEndRef} />
            <MessageComposer loading={loading} onSendMessage={onSendMessage} />
          </>
        )}
      </section>
    </section>
  )
}

function ConversationList({
  conversations,
  selected,
  loading,
  onOpenConversation,
}: {
  conversations: Conversation[]
  selected: Conversation | null
  loading: boolean
  onOpenConversation: (conversation: Conversation) => void
}) {
  if (loading) {
    return <div className="empty-state"><h2>Carregando conversas</h2><p>Buscando o estado salvo no backend...</p></div>
  }
  if (conversations.length === 0) {
    return <div className="empty-state"><h2>Nenhuma conversa</h2><p>Cadastre um destinatário para iniciar o fluxo.</p></div>
  }
  return (
    <div className="conversation-list">
      {conversations.map(conversation => (
        <button
          key={conversation.id}
          className={`conversation-item ${selected?.id === conversation.id ? 'active' : ''}`}
          onClick={() => onOpenConversation(conversation)}
        >
          <strong>{conversation.recipient.name}</strong>
          <span>{conversation.recipient.phone}</span>
          <small>{conversation.lastActivityAt ? new Date(conversation.lastActivityAt).toLocaleString('pt-BR') : 'Sem atividade recente'}</small>
        </button>
      ))}
    </div>
  )
}

function ConversationHeading({ conversation, stats, onBack }: { conversation: Conversation; stats: MessageStats; onBack: () => void }) {
  return (
    <div className="conversation-heading">
      <button type="button" className="mobile-back" onClick={onBack} aria-label="Voltar para conversas" title="Voltar para conversas">
        <span className="mobile-back-icon" aria-hidden="true" />
      </button>
      <div>
        <h2>{conversation.recipient.name}</h2>
        <p>{conversation.recipient.phone}</p>
      </div>
      <div className="status-strip" aria-label="Resumo da conversa">
        <span>{stats.sent} enviadas</span>
        <span>{stats.processing} em andamento</span>
        <span>{stats.failed} falhas</span>
      </div>
    </div>
  )
}

function MessageList({
  messages,
  messagesEndRef,
}: {
  messages: Message[]
  messagesEndRef: RefObject<HTMLDivElement | null>
}) {
  if (messages.length === 0) {
    return <div className="empty-state"><h2>Sem mensagens ainda</h2><p>O envio entra na próxima etapa.</p></div>
  }

  return (
    <div className="message-list" aria-live="polite">
      {messages.map(message => (
        <article key={message.id} className={`message-card outbound ${message.status}`}>
          <p>{message.content}</p>
          <footer>
            <span>{message.channel.toUpperCase()} - {message.priority === priorities.urgent ? 'Urgente' : 'Normal'} - {formatMoney(message.costCents)}</span>
            <strong>{messageStatusLabel(message.status)}</strong>
          </footer>
          {message.failureCode && <small>Falha: {message.failureCode}</small>}
        </article>
      ))}
      <div ref={messagesEndRef} />
    </div>
  )
}

function MessageComposer({
  loading,
  onSendMessage,
}: {
  loading: boolean
  onSendMessage: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <form onSubmit={onSendMessage} className="message-form">
      <label>Mensagem<textarea name="content" required rows={3} placeholder="Digite a mensagem. Use [fail] ou [retry] para simular falhas." /></label>
      <div className="field-row">
        <label>Canal<select name="channel" defaultValue={channels.whatsapp}><option value={channels.whatsapp}>WhatsApp</option><option value={channels.sms}>SMS</option></select></label>
        <label>Prioridade<select name="priority" defaultValue={priorities.normal}><option value={priorities.normal}>Normal - R$ 0,25</option><option value={priorities.urgent}>Urgente - R$ 0,50</option></select></label>
      </div>
      <SubmitButton loading={loading} label="Enviar mensagem" />
    </form>
  )
}
