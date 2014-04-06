package auth

type UserAuth interface {
	UserId() string
	Secret() string
}

type SecurityDriver interface {
	Register()
	Authenticate()
}

type StorageDriver interface {
	AddUser(userid, secret string)
	RemoveUser(userid string)
	GetSecret(userid string) string
	SetSecret(userid, secret string)
}
