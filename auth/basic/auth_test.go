package auth_test

import (
	"errors"
	"testing"

	auth "github.com/revel/modules/auth/basic"
	"github.com/revel/modules/auth/basic/driver/secret"
)

type User struct {
	email    string
	password string
	hashpass string

	secret.BcryptAuth // SecurityDriver for testing
}

func NewUser(email, pass string) *User {
	u := &User{
		email:    email,
		password: pass,
	}
	u.UserContext = u
	return u
}

func (u *User) UserId() string {
	return u.email
}

func (u *User) Secret() string {
	return u.password
}

func (u *User) HashedSecret() string {
	return u.hashpass
}

func (u *User) SetHashedSecret(hpass string) {
	u.hashpass = hpass
}

// func (u *User) Load() string

type TestStore struct {
	data map[string]string
}

func (ts *TestStore) Save(user interface{}) error {
	u, ok := user.(*User)
	if !ok {
		return errors.New("TestStore.Save() expected arg of type User")
	}

	hPass, err := u.HashSecret(u.Secret())
	if err != nil {
		return err
	}
	ts.data[u.UserId()] = hPass

	return nil
}

func (ts *TestStore) Load(user interface{}) error {
	u, ok := user.(*User)
	if !ok {
		return errors.New("TestStore.Load() expected arg of type User")
	}

	hpass, ok := ts.data[u.UserId()]
	if !ok {
		return errors.New("Record Not Found")
	}
	u.SetHashedSecret(hpass)
	return nil
}

func TestPasswordHash(t *testing.T) {
	auth.Store = &TestStore{
		data: make(map[string]string),
	}
	u := NewUser("demo@domain.com", "demopass")
	fail := NewUser("demo@domain.com", "")

	var err error
	u.hashpass, err = u.HashSecret(u.password)
	if err != nil {
		t.Errorf("Should have hashed password, get error: %v\n", err)
	}
	fail.hashpass, err = fail.HashSecret(fail.password)
	if err == nil {
		t.Errorf("Should have failed hashing\n")
	}
}

func TestAuthenticate(t *testing.T) {
	auth.Store = &TestStore{
		data: make(map[string]string),
	}

	// user registered a long time ago
	u := NewUser("demo@domain.com", "demopass")
	err := auth.Store.Save(u)
	if err != nil {
		t.Errorf("Should have saved user: %v", err)
	}

	// users now logging in
	pass := NewUser("demo@domain.com", "demopass")
	fail := NewUser("demo@domain.com", "invalid")

	// valid user is now trying to login
	// check user in DB
	err = auth.Store.Load(pass)
	if err != nil {
		t.Errorf("Should have loaded pass user: %v\n", err)
	}
	// check credentials
	ok, err := pass.Authenticate()
	if !ok || err != nil {
		t.Errorf("Should have authenticated user")
	}

	// invalid user is now trying to login
	err = auth.Store.Load(fail)
	if err != nil {
		t.Errorf("Should have loaded fail user")
	}
	// this should fail
	ok, err = fail.Authenticate()
	if ok || err != nil {
		t.Errorf("Should have failed to authenticate user: %v\n", err)
	}
}
