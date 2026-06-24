package accounts

import (
	"errors"
	"net/http"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/httpserver/response"
	"bcb/backend/internal/modules/access"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service    *Service
	repository *Repository
}

func newHandler(service *Service, repository *Repository) *Handler {
	return &Handler{service: service, repository: repository}
}

func (handler *Handler) register(ctx *gin.Context) {
	var request struct {
		Name          string `json:"name" binding:"required"`
		DocumentType  string `json:"documentType" binding:"required"`
		DocumentID    string `json:"documentId" binding:"required"`
		Password      string `json:"password" binding:"required"`
		RequestedPlan string `json:"requestedPlan" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Revise os dados informados.", nil)
		return
	}
	clientID, err := handler.service.register(ctx, request.Name, request.DocumentType, request.DocumentID, request.Password, request.RequestedPlan)
	if errors.Is(err, ErrConflict) {
		response.Error(ctx, http.StatusConflict, "document_already_registered", "CPF ou CNPJ já cadastrado.", nil)
		return
	}
	if err != nil {
		response.Error(ctx, http.StatusUnprocessableEntity, "registration_invalid", err.Error(), nil)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"clientId": clientID,
		"status":   domain.ClientStatusPending,
		"message":  "Cadastro recebido e aguardando aprovação.",
	})
}

func (handler *Handler) listClients(ctx *gin.Context) {
	clients, err := handler.repository.clients(ctx, ctx.Query("status"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível listar os clientes.", nil)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": clients})
}

func (handler *Handler) activate(ctx *gin.Context) {
	var request struct {
		PlanType            string `json:"planType" binding:"required"`
		InitialBalanceCents int64  `json:"initialBalanceCents"`
		TotalLimitCents     int64  `json:"totalLimitCents"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.InitialBalanceCents < 0 || request.TotalLimitCents < 0 {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Condição financeira inválida.", nil)
		return
	}
	if (request.PlanType == string(domain.PlanPrepaid) && request.TotalLimitCents != 0) ||
		(request.PlanType == string(domain.PlanPostpaid) && request.InitialBalanceCents != 0) {
		response.Error(ctx, http.StatusUnprocessableEntity, "billing_profile_invalid", "Informe apenas os valores aplicáveis ao plano solicitado.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.repository.activate(ctx, claims.Subject, ctx.Param("clientId"), Activation{
		PlanType: request.PlanType, InitialBalanceCents: request.InitialBalanceCents,
		PostpaidTotalLimitCents: request.TotalLimitCents,
	})
	handler.writeAdminResult(ctx, err)
}

func (handler *Handler) reject(ctx *gin.Context) {
	handler.changeStatus(ctx, string(domain.ClientStatusRejected))
}

func (handler *Handler) deactivate(ctx *gin.Context) {
	handler.changeStatus(ctx, string(domain.ClientStatusInactive))
}

func (handler *Handler) changeStatus(ctx *gin.Context, status string) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe o motivo.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	handler.writeAdminResult(ctx, handler.repository.changeStatus(ctx, claims.Subject, ctx.Param("clientId"), status, request.Reason))
}

func (handler *Handler) writeAdminResult(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(ctx, http.StatusNotFound, "client_not_found", "Cliente não encontrado.", nil)
	case errors.Is(err, ErrConflict):
		response.Error(ctx, http.StatusConflict, "invalid_transition", "Transição de estado inválida.", nil)
	case err != nil:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação.", nil)
	default:
		ctx.Status(http.StatusNoContent)
	}
}
