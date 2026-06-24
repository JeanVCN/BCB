package billing

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

func (handler *Handler) profile(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	profile, err := handler.service.profile(ctx, *claims.ClientID)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, profile)
}

func (handler *Handler) clientTransactions(ctx *gin.Context) {
	claims, _ := access.ClaimsFromContext(ctx)
	handler.writeTransactions(ctx, *claims.ClientID)
}

func (handler *Handler) adminTransactions(ctx *gin.Context) {
	handler.writeTransactions(ctx, ctx.Param("clientId"))
}

func (handler *Handler) addCredit(ctx *gin.Context) {
	var request struct {
		AmountCents int64  `json:"amountCents" binding:"required"`
		Reason      string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe o valor do crédito.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.service.addCredit(ctx, claims.Subject, ctx.Param("clientId"), request.AmountCents, request.Reason, ctx.GetHeader("Idempotency-Key"))
	handler.writeMutationResult(ctx, err)
}

func (handler *Handler) adjustPostpaidLimit(ctx *gin.Context) {
	var request struct {
		TotalLimitCents int64  `json:"totalLimitCents"`
		Reason          string `json:"reason"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid_request", "Informe o novo limite.", nil)
		return
	}
	claims, _ := access.ClaimsFromContext(ctx)
	err := handler.service.adjustPostpaidLimit(ctx, claims.Subject, ctx.Param("clientId"), request.TotalLimitCents, request.Reason, ctx.GetHeader("Idempotency-Key"))
	handler.writeMutationResult(ctx, err)
}

func (handler *Handler) writeTransactions(ctx *gin.Context, clientID string) {
	transactions, err := handler.service.transactions(ctx, clientID)
	if err != nil {
		handler.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": transactions})
}

func (handler *Handler) writeMutationResult(ctx *gin.Context, err error) {
	if err == nil {
		ctx.Status(http.StatusNoContent)
		return
	}
	handler.writeError(ctx, err)
}

func (handler *Handler) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrMissingIdempotencyKey):
		response.Error(ctx, http.StatusBadRequest, "missing_idempotency_key", "Informe a chave de idempotência.", nil)
	case errors.Is(err, ErrInvalidAmount):
		response.Error(ctx, http.StatusBadRequest, "invalid_amount", "Informe um valor financeiro maior que zero.", nil)
	case errors.Is(err, ErrProfileNotFound):
		response.Error(ctx, http.StatusNotFound, "billing_profile_not_found", "Perfil financeiro não encontrado.", nil)
	case errors.Is(err, ErrClientNotActive):
		response.Error(ctx, http.StatusForbidden, "client_not_active", "Empresa cliente inativa ou indisponível.", nil)
	case errors.Is(err, ErrPlanMismatch):
		response.Error(ctx, http.StatusUnprocessableEntity, "billing_plan_mismatch", "Operação incompatível com o plano atual.", nil)
	case errors.Is(err, ErrLimitBelowConsumed):
		response.Error(ctx, http.StatusUnprocessableEntity, "limit_below_consumed", "O limite não pode ser menor que o consumo atual.", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		response.Error(ctx, http.StatusConflict, "idempotency_conflict", "A chave de idempotência já foi usada com outro conteúdo.", nil)
	case errors.Is(err, ErrLockUnavailable):
		response.Error(ctx, http.StatusServiceUnavailable, "billing_lock_unavailable", "Não foi possível obter segurança concorrente para a operação financeira.", nil)
	default:
		response.Error(ctx, http.StatusInternalServerError, "internal_error", "Não foi possível concluir a operação financeira.", nil)
	}
}
