package messages

import (
	"fmt"
	"testing"
	"time"

	"bcb/backend/internal/domain"
)

func TestParseChannel(t *testing.T) {
	tests := []struct {
		value  string
		want   domain.Channel
		wantOK bool
	}{
		{value: "sms", want: domain.ChannelSMS, wantOK: true},
		{value: " WhatsApp ", want: domain.ChannelWhatsApp, wantOK: true},
		{value: "email", wantOK: false},
		{value: "", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			got, ok := parseChannel(test.value)
			if got != test.want || ok != test.wantOK {
				t.Fatalf("parseChannel(%q) = %q, %v; want %q, %v", test.value, got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestParsePriorityCostAndRank(t *testing.T) {
	normal, ok := parsePriority(" normal ")
	if !ok || normal != domain.PriorityNormal {
		t.Fatalf("normal priority = %q, %v", normal, ok)
	}
	if cost := messageCost(normal); cost != normalCostCents {
		t.Fatalf("normal cost = %d, want %d", cost, normalCostCents)
	}
	if rank := priorityRank(normal); rank != 1 {
		t.Fatalf("normal rank = %d, want 1", rank)
	}

	urgent, ok := parsePriority("URGENT")
	if !ok || urgent != domain.PriorityUrgent {
		t.Fatalf("urgent priority = %q, %v", urgent, ok)
	}
	if cost := messageCost(urgent); cost != urgentCostCents {
		t.Fatalf("urgent cost = %d, want %d", cost, urgentCostCents)
	}
	if rank := priorityRank(urgent); rank != 0 {
		t.Fatalf("urgent rank = %d, want 0", rank)
	}

	if _, ok := parsePriority("later"); ok {
		t.Fatalf("invalid priority should not parse")
	}
}

func TestRequestHashChangesWithMessageContent(t *testing.T) {
	first := requestHash("Oi", domain.ChannelSMS, domain.PriorityNormal)
	second := requestHash("Oi", domain.ChannelSMS, domain.PriorityNormal)
	if first == "" || first != second {
		t.Fatalf("request hash should be stable and not empty")
	}

	changed := requestHash("Outra mensagem", domain.ChannelSMS, domain.PriorityNormal)
	if first == changed {
		t.Fatalf("request hash should include content")
	}
}

func TestRetryDelayUsesExpectedBackoffWindow(t *testing.T) {
	tests := []struct {
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{attempt: 1, min: time.Second, max: time.Second + 249*time.Millisecond},
		{attempt: 2, min: 2 * time.Second, max: 2*time.Second + 249*time.Millisecond},
		{attempt: 3, min: 4 * time.Second, max: 4*time.Second + 249*time.Millisecond},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("attempt_%d", test.attempt), func(t *testing.T) {
			got := retryDelay(test.attempt)
			if got < test.min || got > test.max {
				t.Fatalf("retryDelay(%d) = %s, want between %s and %s", test.attempt, got, test.min, test.max)
			}
		})
	}
}

func TestSimulateDispatch(t *testing.T) {
	tests := []struct {
		content string
		outcome domain.DeliveryAttemptOutcome
		errCode string
	}{
		{content: "mensagem comum", outcome: domain.DeliveryAttemptSent},
		{content: "forcar [fail]", outcome: domain.DeliveryAttemptPermanentFailure, errCode: "simulated_permanent_failure"},
		{content: "forcar [retry]", outcome: domain.DeliveryAttemptTransientFailure, errCode: "simulated_transient_failure"},
	}

	for _, test := range tests {
		t.Run(test.content, func(t *testing.T) {
			outcome, errCode := simulateDispatch(dispatchJob{Content: test.content})
			if outcome != test.outcome || errCode != test.errCode {
				t.Fatalf("simulateDispatch(%q) = %q, %q; want %q, %q", test.content, outcome, errCode, test.outcome, test.errCode)
			}
		})
	}
}
