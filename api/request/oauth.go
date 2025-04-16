package request

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type GenerateTokenReq struct {
	Token string `json:"token" binding:"required"`
}
