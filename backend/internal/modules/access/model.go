package access

type User struct {
	ID              string
	Role            string
	Login           string
	PasswordHash    string
	Enabled         bool
	ClientAccountID *string
	ClientStatus    *string
}
