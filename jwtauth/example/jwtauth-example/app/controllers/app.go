package controllers

import (
	"fmt"

	"github.com/jeevatkm/jwtauth-example/app/models"
	"github.com/jeevatkm/jwtauth/app/jwt"
	"github.com/revel/revel"
)

// For demo purpose
var appUsers map[string]*models.User

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c *App) Register() revel.Result {
	return c.RenderJson(map[string]string{
		"message": "You have reached REGISTER route via POST method",
	})
}

func (c *App) ForgotPassword() revel.Result {
	return c.RenderJson(map[string]string{
		"message": "You have reached FORGOT PASSWORD route via POST method",
	})
}

func (c *App) ResetPassword() revel.Result {
	return c.RenderJson(map[string]string{
		"message": "You have reached RESET PASSWORD route via POST method",
	})
}

func init() {
	// Creating couple of users for example application
	appUsers = make(map[string]*models.User)

	appUsers["100001"] = &models.User{
		Id:        100001,
		Email:     "jeeva@myjeeva.com",
		Password:  "sample1",
		FirstName: "Jeeva",
		LastName:  "M",
	}

	appUsers["100002"] = &models.User{
		Id:        100001,
		Email:     "user1@myjeeva.com",
		Password:  "user1",
		FirstName: "User",
		LastName:  "1",
	}

	appUsers["100003"] = &models.User{
		Id:        100001,
		Email:     "user2@myjeeva.com",
		Password:  "user2",
		FirstName: "User",
		LastName:  "2",
	}

	revel.OnAppStart(func() {
		jwt.Init(jwt.AuthHandlerFunc(func(username, password string) (string, bool) {

			// This method will be invoked by JwtAuth module for authentication
			// Call your implementation to authenticate user
			revel.INFO.Printf("Username: %v, Password: %v", username, password)

			// ....
			// ....

			// after successful authentication
			// create User subject value, which you want to inculde in signed string
			// such as User Id, user email address, etc.

			// Note: using plain password for demo purpose
			var userId int64
			var authenticated bool
			for _, v := range appUsers {
				if v.Email == username && v.Password == password {
					userId = v.Id
					authenticated = true
					break
				}
			}

			return fmt.Sprintf("%d", userId), authenticated
		}))
	})
}
