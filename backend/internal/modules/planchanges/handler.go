package planchanges

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

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) create(ctx *gin.Context) {
	var request struct {
		TargetPlan string `json:"targetPlan" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe o plano desejado.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	planChange, err := handler.service.create(ctx, *claims.ClientID, claims.Subject, request.TargetPlan)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, planChange)
}

func (handler *Handler) current(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	planChange, err := handler.service.current(ctx, *claims.ClientID)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, planChange)
}

func (handler *Handler) cancel(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.service.cancel(ctx, *claims.ClientID, claims.Subject, ctx.Param("requestId"))
	handler.writeMutation(ctx, err)
}

func (handler *Handler) summary(ctx *gin.Context) {
	summary, err := handler.service.summary(ctx)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, summary)
}

func (handler *Handler) adminList(ctx *gin.Context) {
	requests, err := handler.service.list(ctx, ctx.Query("status"))
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": requests})
}

func (handler *Handler) approve(ctx *gin.Context) {
	var request struct {
		InitialBalanceCents int64 `json:"initialBalanceCents"`
		TotalLimitCents     int64 `json:"totalLimitCents"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Condição financeira inválida.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.service.approve(ctx, claims.Subject, ctx.Param("requestId"), request.InitialBalanceCents, request.TotalLimitCents)
	handler.writeMutation(ctx, err)
}

func (handler *Handler) reject(ctx *gin.Context) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe o motivo.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.service.reject(ctx, claims.Subject, ctx.Param("requestId"), request.Reason)
	handler.writeMutation(ctx, err)
}

func (handler *Handler) writeMutation(ctx *gin.Context, err error) {
	if err == nil {
		ctx.Status(http.StatusNoContent)
		return
	}
	handler.writeError(ctx, err)
}

func (handler *Handler) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidTargetPlan):
		response.Error(ctx, http.StatusBadRequest, "invalid_target_plan", "Plano desejado inválido.", nil)
	case errors.Is(err, ErrInvalidInitialBalance), errors.Is(err, ErrInvalidPostpaidLimit), errors.Is(err, ErrApprovalPayload):
		response.Error(ctx, http.StatusBadRequest, "invalid_billing_condition", "Condição financeira incompatível com o plano.", nil)
	case errors.Is(err, ErrMissingRejectionReason):
		response.Error(ctx, http.StatusBadRequest, "missing_rejection_reason", "Informe o motivo da rejeição.", nil)
	case errors.Is(err, ErrClientNotActive):
		response.Error(ctx, http.StatusForbidden, "client_not_active", "Empresa cliente inativa ou indisponível.", nil)
	case errors.Is(err, ErrRequestNotFound):
		response.Error(ctx, http.StatusNotFound, "plan_change_request_not_found", "Solicitação de mudança de plano não encontrada.", nil)
	case errors.Is(err, ErrSamePlan):
		response.Error(ctx, http.StatusUnprocessableEntity, "same_plan", "A empresa já está neste plano.", nil)
	case errors.Is(err, ErrFinancialStateBlocked):
		response.Error(ctx, http.StatusUnprocessableEntity, "financial_state_blocks_plan_change", "Quite o saldo ou consumo pendente antes de mudar de plano.", nil)
	case errors.Is(err, ErrPendingRequestExists):
		response.Error(ctx, http.StatusConflict, "pending_plan_change_exists", "Já existe uma solicitação de mudança de plano pendente.", nil)
	case errors.Is(err, ErrInvalidTransition):
		response.Error(ctx, http.StatusConflict, "invalid_plan_change_transition", "Transição de solicitação inválida.", nil)
	default:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a solicitação de mudança de plano.", nil)
	}
}
