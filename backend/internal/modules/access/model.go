package access

import "bcb/backend/internal/domain"

type User struct {
	ID              string
	Role            domain.Role
	Login           string
	PasswordHash    string
	Enabled         bool
	ClientAccountID *string
	ClientStatus    *string
}
