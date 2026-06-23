package messages

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/modules/billing"
)

const (
	normalCostCents = int64(25)
	urgentCostCents = int64(50)
	maxAttempts     = 4
)

func parseChannel(value string) (domain.Channel, bool) {
	channel := domain.Channel(strings.TrimSpace(strings.ToLower(value)))
	switch channel {
	case domain.ChannelSMS, domain.ChannelWhatsApp:
		return channel, true
	default:
		return "", false
	}
}

func parsePriority(value string) (domain.Priority, bool) {
	priority := domain.Priority(strings.TrimSpace(strings.ToLower(value)))
	switch priority {
	case domain.PriorityNormal, domain.PriorityUrgent:
		return priority, true
	default:
		return "", false
	}
}

func messageCost(priority domain.Priority) int64 {
	if priority == domain.PriorityUrgent {
		return urgentCostCents
	}
	return normalCostCents
}

func priorityRank(priority domain.Priority) int {
	if priority == domain.PriorityUrgent {
		return 0
	}
	return 1
}

func requestHash(content string, channel domain.Channel, priority domain.Priority) string {
	return billing.RequestHash("message.send", struct {
		Content  string          `json:"content"`
		Channel  domain.Channel  `json:"channel"`
		Priority domain.Priority `json:"priority"`
	}{Content: content, Channel: channel, Priority: priority})
}

func retryDelay(attempt int) time.Duration {
	base := 4 * time.Second
	switch attempt {
	case 1:
		base = time.Second
	case 2:
		base = 2 * time.Second
	}
	return base + retryJitter()
}

func retryJitter() time.Duration {
	value, err := rand.Int(rand.Reader, big.NewInt(250))
	if err != nil {
		return 0
	}
	return time.Duration(value.Int64()) * time.Millisecond
}
