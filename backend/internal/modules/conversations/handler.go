package conversations

import (
	"errors"
	"net/http"

	"bcb/backend/internal/httpserver/response"
	"bcb/backend/internal/modules/access"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Create(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	var request struct {
		Recipient struct {
			Name  string `json:"name" binding:"required"`
			Phone string `json:"phone" binding:"required"`
		} `json:"recipient" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe nome e telefone do destinatário.", nil)
		return
	}

	conversation, created, err := handler.service.CreateOrGet(ctx, *claims.ClientID, request.Recipient.Name, request.Recipient.Phone)
	handler.writeConversationResult(ctx, conversation, created, err)
}

func (handler *Handler) List(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	conversations, err := handler.service.List(ctx, *claims.ClientID)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": conversations})
}

func (handler *Handler) Messages(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	messages, err := handler.service.Messages(ctx, *claims.ClientID, ctx.Param("conversationId"))
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": messages})
}

func (handler *Handler) writeConversationResult(ctx *gin.Context, conversation Conversation, created bool, err error) {
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	if created {
		ctx.JSON(http.StatusCreated, conversation)
		return
	}
	ctx.JSON(http.StatusOK, conversation)
}

func (handler *Handler) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrClientInactive):
		response.Error(ctx, http.StatusForbidden, "client_not_active", "Empresa cliente inativa ou indisponível.", nil)
	case errors.Is(err, ErrConversationNotFound):
		response.Error(ctx, http.StatusNotFound, "conversation_not_found", "Conversa não encontrada.", nil)
	case errors.Is(err, ErrInvalidConversation):
		response.Error(ctx, http.StatusUnprocessableEntity, "conversation_invalid", err.Error(), nil)
	default:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.", nil)
	}
}
