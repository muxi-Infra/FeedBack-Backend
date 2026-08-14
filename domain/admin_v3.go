package domain

type AdminLoginInputV3 struct {
	Username string
	Password string
}

type AdminLoginResultV3 struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}
