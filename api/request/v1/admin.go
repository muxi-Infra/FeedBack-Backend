package v1

type AdminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateAdminReq struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password" binding:"required"`
}

type ChangeAdminPasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type UpdateAdminStatusReq struct {
	Status string `json:"status" binding:"required"`
}
