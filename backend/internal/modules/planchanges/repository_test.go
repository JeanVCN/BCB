package planchanges

import (
	"errors"
	"testing"

	"bcb/backend/internal/domain"
)

func TestBillingSnapshotCanChange(t *testing.T) {
	tests := []struct {
		name     string
		snapshot billingSnapshot
		want     bool
	}{
		{
			name: "prepaid with zero balance can change",
			snapshot: billingSnapshot{
				planType:       string(domain.PlanPrepaid),
				prepaidBalance: 0,
			},
			want: true,
		},
		{
			name: "prepaid with balance cannot change",
			snapshot: billingSnapshot{
				planType:       string(domain.PlanPrepaid),
				prepaidBalance: 1,
			},
			want: false,
		},
		{
			name: "postpaid without consumption can change",
			snapshot: billingSnapshot{
				planType:         string(domain.PlanPostpaid),
				postpaidConsumed: 0,
			},
			want: true,
		},
		{
			name: "postpaid with consumption cannot change",
			snapshot: billingSnapshot{
				planType:         string(domain.PlanPostpaid),
				postpaidConsumed: 1,
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.snapshot.canChange(); got != test.want {
				t.Fatalf("canChange() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestApprovalBillingState(t *testing.T) {
	tests := []struct {
		name                string
		targetPlan          string
		initialBalanceCents int64
		totalLimitCents     int64
		wantPrepaidBalance  int64
		wantPostpaidLimit   int64
		wantErr             error
	}{
		{
			name:                "prepaid accepts initial balance",
			targetPlan:          string(domain.PlanPrepaid),
			initialBalanceCents: 1000,
			wantPrepaidBalance:  1000,
		},
		{
			name:              "postpaid accepts total limit",
			targetPlan:        string(domain.PlanPostpaid),
			totalLimitCents:   5000,
			wantPostpaidLimit: 5000,
		},
		{
			name:                "prepaid rejects postpaid limit",
			targetPlan:          string(domain.PlanPrepaid),
			initialBalanceCents: 1000,
			totalLimitCents:     5000,
			wantErr:             ErrApprovalPayload,
		},
		{
			name:                "postpaid rejects prepaid balance",
			targetPlan:          string(domain.PlanPostpaid),
			initialBalanceCents: 1000,
			totalLimitCents:     5000,
			wantErr:             ErrApprovalPayload,
		},
		{
			name:       "invalid target plan",
			targetPlan: "enterprise",
			wantErr:    ErrInvalidTargetPlan,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepaidBalance, postpaidLimit, err := approvalBillingState(test.targetPlan, test.initialBalanceCents, test.totalLimitCents)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if prepaidBalance != test.wantPrepaidBalance {
				t.Fatalf("prepaid balance = %d, want %d", prepaidBalance, test.wantPrepaidBalance)
			}
			if postpaidLimit != test.wantPostpaidLimit {
				t.Fatalf("postpaid limit = %d, want %d", postpaidLimit, test.wantPostpaidLimit)
			}
		})
	}
}
