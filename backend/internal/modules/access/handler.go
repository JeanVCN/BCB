package access

import (
	"errors"
	"net/http"

	"bcb/backend/internal/httpserver/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Login(ctx *gin.Context) {
	var request struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe login e senha.", nil)
		return
	}
	token, user, err := handler.service.Login(ctx, request.Login, request.Password)
	switch {
	case errors.Is(err, ErrRateLimited):
		response.Error(ctx, http.StatusTooManyRequests, "authentication_rate_limited", "Aguarde antes de tentar novamente.", nil)
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrClientInactive):
		response.Error(ctx, http.StatusUnauthorized, "invalid_credentials", "Credenciais inválidas ou conta indisponível.", nil)
	case err != nil:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível autenticar.", nil)
	default:
		ctx.JSON(http.StatusOK, gin.H{
			"accessToken": token, "tokenType": "Bearer", "expiresInSeconds": 3600,
			"user": gin.H{"id": user.ID, "role": user.Role, "clientId": user.ClientAccountID},
		})
	}
}

func (handler *Handler) Me(ctx *gin.Context) {
	claims, _ := ClaimsFromContext(ctx)
	ctx.JSON(http.StatusOK, gin.H{"user": gin.H{"id": claims.Subject, "role": claims.Role, "clientId": claims.ClientID}})
}
