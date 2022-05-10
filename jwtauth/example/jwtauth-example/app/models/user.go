package models

type User struct {
	Id        int64
	Email     string
	Password  string
	FirstName string
	LastName  string

	// so on...
}
