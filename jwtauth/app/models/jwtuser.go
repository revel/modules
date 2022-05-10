package models

type JwtUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
