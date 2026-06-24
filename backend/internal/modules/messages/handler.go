package messages

import (
	"errors"
	"net/http"

	"bcb/backend/internal/httpserver/response"
	"bcb/backend/internal/modules/access"
	"bcb/backend/internal/modules/billing"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) send(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	var request struct {
		Content  string `json:"content" binding:"required"`
		Channel  string `json:"channel" binding:"required"`
		Priority string `json:"priority" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe conteúdo, canal e prioridade.", nil)
		return
	}

	result, err := handler.service.send(ctx, *claims.ClientID, claims.Subject, ctx.Param("conversationId"), request.Content, request.Channel, request.Priority, ctx.GetHeader("Idempotency-Key"))
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func (handler *Handler) list(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	messages, err := handler.service.list(ctx, *claims.ClientID, ctx.Param("conversationId"))
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": messages})
}

func (handler *Handler) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, billing.ErrMissingIdempotencyKey):
		response.Error(ctx, http.StatusBadRequest, "missing_idempotency_key", "Informe a chave de idempotência.", nil)
	case errors.Is(err, ErrInvalidMessage):
		response.Error(ctx, http.StatusUnprocessableEntity, "message_invalid", "Informe uma mensagem válida com canal e prioridade suportados.", nil)
	case errors.Is(err, ErrClientInactive), errors.Is(err, billing.ErrClientNotActive):
		response.Error(ctx, http.StatusForbidden, "client_not_active", "Empresa cliente inativa ou indisponível.", nil)
	case errors.Is(err, ErrConversationNotFound):
		response.Error(ctx, http.StatusNotFound, "conversation_not_found", "Conversa não encontrada.", nil)
	case errors.Is(err, billing.ErrInsufficientBalance):
		response.Error(ctx, http.StatusUnprocessableEntity, "insufficient_balance", "Saldo insuficiente para enviar a mensagem.", nil)
	case errors.Is(err, billing.ErrLimitExceeded):
		response.Error(ctx, http.StatusUnprocessableEntity, "limit_exceeded", "Limite pós-pago insuficiente para enviar a mensagem.", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		response.Error(ctx, http.StatusConflict, "idempotency_conflict", "A chave de idempotência já foi usada com outro conteúdo.", nil)
	case errors.Is(err, billing.ErrLockUnavailable):
		response.Error(ctx, http.StatusServiceUnavailable, "billing_lock_unavailable", "Não foi possível obter segurança concorrente para a cobrança.", nil)
	default:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível enviar a mensagem.", nil)
	}
}
