package httpserver

import (
	"errors"
	"net/http"

	"bcb/backend/internal/identity"
	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	service *identity.Service
	store   *identity.Store
}

func NewIdentityHandler(service *identity.Service, store *identity.Store) *IdentityHandler {
	return &IdentityHandler{service: service, store: store}
}

func (handler *IdentityHandler) Register(ctx *gin.Context) {
	var request struct {
		Name          string `json:"name" binding:"required"`
		DocumentType  string `json:"documentType" binding:"required"`
		DocumentID    string `json:"documentId" binding:"required"`
		Password      string `json:"password" binding:"required"`
		RequestedPlan string `json:"requestedPlan" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_request", "Revise os dados informados.", nil)
		return
	}
	clientID, err := handler.service.Register(ctx, request.Name, request.DocumentType, request.DocumentID, request.Password, request.RequestedPlan)
	if errors.Is(err, identity.ErrConflict) {
		writeError(ctx, http.StatusConflict, "document_already_registered", "CPF ou CNPJ já cadastrado.", nil)
		return
	}
	if err != nil {
		writeError(ctx, http.StatusUnprocessableEntity, "registration_invalid", err.Error(), nil)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"clientId": clientID,
		"status":   "pending",
		"message":  "Cadastro recebido e aguardando aprovação.",
	})
}

func (handler *IdentityHandler) Login(ctx *gin.Context) {
	var request struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_request", "Informe login e senha.", nil)
		return
	}
	token, user, err := handler.service.Login(ctx, request.Login, request.Password)
	switch {
	case errors.Is(err, identity.ErrRateLimited):
		writeError(ctx, http.StatusTooManyRequests, "authentication_rate_limited", "Aguarde antes de tentar novamente.", nil)
	case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrClientInactive):
		writeError(ctx, http.StatusUnauthorized, "invalid_credentials", "Credenciais inválidas ou conta indisponível.", nil)
	case err != nil:
		writeError(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível autenticar.", nil)
	default:
		ctx.JSON(http.StatusOK, gin.H{
			"accessToken": token, "tokenType": "Bearer", "expiresInSeconds": 3600,
			"user": gin.H{"id": user.ID, "role": user.Role, "clientId": user.ClientAccountID},
		})
	}
}

func (handler *IdentityHandler) Me(ctx *gin.Context) {
	claims, _ := currentClaims(ctx)
	ctx.JSON(http.StatusOK, gin.H{"user": gin.H{"id": claims.Subject, "role": claims.Role, "clientId": claims.ClientID}})
}

func (handler *IdentityHandler) ListClients(ctx *gin.Context) {
	clients, err := handler.store.Clients(ctx, ctx.Query("status"))
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível listar os clientes.", nil)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": clients})
}

func (handler *IdentityHandler) Activate(ctx *gin.Context) {
	var request struct {
		PlanType            string `json:"planType" binding:"required"`
		InitialBalanceCents int64  `json:"initialBalanceCents"`
		TotalLimitCents     int64  `json:"totalLimitCents"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.InitialBalanceCents < 0 || request.TotalLimitCents < 0 {
		writeError(ctx, http.StatusBadRequest, "invalid_request", "Condição financeira inválida.", nil)
		return
	}
	if (request.PlanType == "prepaid" && request.TotalLimitCents != 0) ||
		(request.PlanType == "postpaid" && request.InitialBalanceCents != 0) {
		writeError(ctx, http.StatusUnprocessableEntity, "billing_profile_invalid", "Informe apenas os valores aplicáveis ao plano solicitado.", nil)
		return
	}
	claims, _ := currentClaims(ctx)
	err := handler.store.Activate(ctx, claims.Subject, ctx.Param("clientId"), identity.Activation{
		PlanType: request.PlanType, InitialBalanceCents: request.InitialBalanceCents,
		PostpaidTotalLimitCents: request.TotalLimitCents,
	})
	handler.writeAdminResult(ctx, err)
}

func (handler *IdentityHandler) Reject(ctx *gin.Context) {
	handler.changeStatus(ctx, "rejected")
}

func (handler *IdentityHandler) Deactivate(ctx *gin.Context) {
	handler.changeStatus(ctx, "inactive")
}

func (handler *IdentityHandler) changeStatus(ctx *gin.Context, status string) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_request", "Informe o motivo.", nil)
		return
	}
	claims, _ := currentClaims(ctx)
	handler.writeAdminResult(ctx, handler.store.ChangeStatus(ctx, claims.Subject, ctx.Param("clientId"), status, request.Reason))
}

func (handler *IdentityHandler) writeAdminResult(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, identity.ErrNotFound):
		writeError(ctx, http.StatusNotFound, "client_not_found", "Cliente não encontrado.", nil)
	case errors.Is(err, identity.ErrConflict):
		writeError(ctx, http.StatusConflict, "invalid_transition", "Transição de estado inválida.", nil)
	case err != nil:
		writeError(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.", nil)
	default:
		ctx.Status(http.StatusNoContent)
	}
}
