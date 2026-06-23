package conversations

import "time"

type Recipient struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Conversation struct {
	ID             string     `json:"id"`
	Recipient      Recipient  `json:"recipient"`
	LastActivityAt *time.Time `json:"lastActivityAt"`
}

type Message struct {
	ID string `json:"id"`
}
