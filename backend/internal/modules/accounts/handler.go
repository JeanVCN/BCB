package accounts

import (
	"errors"
	"net/http"

	"bcb/backend/internal/httpserver/response"
	"bcb/backend/internal/modules/access"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service    *Service
	repository *Repository
}

func NewHandler(service *Service, repository *Repository) *Handler {
	return &Handler{service: service, repository: repository}
}

func (handler *Handler) Register(ctx *gin.Context) {
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
	clientID, err := handler.service.Register(ctx, request.Name, request.DocumentType, request.DocumentID, request.Password, request.RequestedPlan)
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
		"status":   "pending",
		"message":  "Cadastro recebido e aguardando aprovação.",
	})
}

func (handler *Handler) ListClients(ctx *gin.Context) {
	clients, err := handler.repository.Clients(ctx, ctx.Query("status"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível listar os clientes.", nil)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": clients})
}

func (handler *Handler) Activate(ctx *gin.Context) {
	var request struct {
		PlanType            string `json:"planType" binding:"required"`
		InitialBalanceCents int64  `json:"initialBalanceCents"`
		TotalLimitCents     int64  `json:"totalLimitCents"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil || request.InitialBalanceCents < 0 || request.TotalLimitCents < 0 {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Condição financeira inválida.", nil)
		return
	}
	if (request.PlanType == "prepaid" && request.TotalLimitCents != 0) ||
		(request.PlanType == "postpaid" && request.InitialBalanceCents != 0) {
		response.Error(ctx, http.StatusUnprocessableEntity, "billing_profile_invalid", "Informe apenas os valores aplicáveis ao plano solicitado.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.repository.Activate(ctx, claims.Subject, ctx.Param("clientId"), Activation{
		PlanType: request.PlanType, InitialBalanceCents: request.InitialBalanceCents,
		PostpaidTotalLimitCents: request.TotalLimitCents,
	})
	handler.writeAdminResult(ctx, err)
}

func (handler *Handler) Reject(ctx *gin.Context) {
	handler.changeStatus(ctx, "rejected")
}

func (handler *Handler) Deactivate(ctx *gin.Context) {
	handler.changeStatus(ctx, "inactive")
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
	handler.writeAdminResult(ctx, handler.repository.ChangeStatus(ctx, claims.Subject, ctx.Param("clientId"), status, request.Reason))
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
