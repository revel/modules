package secret

import (
	"errors"

	auth "github.com/revel/modules/auth/basic"
	"golang.org/x/crypto/bcrypt"
)

// example implementation of a Revel auth security driver
// This driver should be embedded into your app-level User model
// It expects your User model to have `Password` and `HashedPassword` string fields
//
// Your User model also needs to set itself as the UserContext for the BcryptAuth driver
//
// func NewUser(email, pass string) *User {
// 	u := &User{
// 		email:    email,
// 		password: pass,
// 	}
// 	u.UserContext = u
// }
//
type BcryptAuth struct {
	UserContext auth.UserAuth
}

// Bcrypt Secret() returns the hashed version of the password.
// It expects an argument of type string, which is the plain text password.
func (ba *BcryptAuth) HashSecret(args ...interface{}) (string, error) {
	if auth.Store == nil {
		return "", errors.New("auth module StorageDriver not set")
	}
	argLen := len(args)
	if argLen == 0 {
		// we are getting
		return ba.UserContext.HashedSecret(), nil
	}

	if argLen == 1 {
		// we are setting
		password, ok := args[0].(string)
		if !ok {
			return "", errors.New("wrong argument type provided, expected plaintext password as string")
		}
		hPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}

		ba.UserContext.SetHashedSecret(string(hPass))
		return ba.UserContext.HashedSecret(), nil
	}

	// bad argument count
	return "", errors.New("too many arguments provided, expected one")
}

// Bycrypt Authenticate() expects a single string argument of the plaintext password
// It returns true on success and false if error or password mismatch.
func (ba *BcryptAuth) Authenticate() (bool, error) {
	// check password
	err := bcrypt.CompareHashAndPassword([]byte(ba.UserContext.HashedSecret()), []byte(ba.UserContext.Secret()))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// successfully authenticated
	return true, nil
}
