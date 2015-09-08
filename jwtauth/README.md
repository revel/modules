# JWT Auth Module for Revel Framework

Pluggable and easy to use JWT auth module in Revel Framework. 

Planning to bring following enhancement to this moudle:
* Choosing Signing Method (`HS*, RS*, ES*`) via config, currently module does `RS512`
* Module error messages via Revel messages `/messages/<filename>.en, etc`

### Module Configuration
```ini
# default is REVEL-JWT-AUTH
auth.jwt.realm.name = "REVEL-JWT-AUTH"

# use appropriate values (string, URL), default is REVEL-JWT-AUTH
auth.jwt.issuer = "REVEL-JWT-AUTH"

# In minutes, default is 60 minutes
auth.jwt.expiration = 30

# Secured Key
auth.jwt.key.private = "/Users/jeeva/private.rsa"
auth.jwt.key.public = "/Users/jeeva/public.rsa.pub"

# Valid regexp allowed for path
auth.jwt.anonymous = "/token, /register, /(forgot|validate-reset|reset)-password, /freepass/.*"
```

### Enabling Auth Module

Add following into `conf/app.conf` revel app configuration
```ini
# Enabling JWT Auth module 
module.jwtauth = github.com/jeevatkm/jwtauth
```

### Registering Auth Routes

Add following into `conf/routes`. 
```sh
# Adding JWT Auth routes into application
module:jwtauth
```
JWT Auth modules enables following routes-
```sh
# JWT Auth Routes
POST	/token									JwtAuth.Token
GET		/refresh-token							JwtAuth.RefreshToken
GET		/logout									JwtAuth.Logout
```

### Registering Auth Filter

Revel Filter for JWT Auth Token verification. Register it in the `revel.Filters` in `<APP_PATH>/app/init.go`

```go
// Add jwt.AuthFilter anywhere deemed appropriate, it must be register after revel.PanicFilter
revel.Filters = []revel.Filter{
  revel.PanicFilter,
	...
	jwt.AuthFilter,		// JWT Auth Token verification for Request Paths
	...
}
// Note: If everything looks good then Claims map made available via c.Args
// and can be accessed using c.Args[jwt.TokenClaimsKey]
```

### Register Auth Handler

Auth handler is responsible for validate user and returning `Subject (aka sub)` value and success/failure boolean. It should comply [AuthHandler](https://github.com/jeevatkm/jwtauth/blob/master/app/jwt/jwt.go#L31) interface or use raw func via [jwt.AuthHandlerFunc](https://github.com/jeevatkm/jwtauth/blob/master/app/jwt/jwt.go#L37).
```go
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
		
		userId := 100001

		return fmt.Sprintf("%d", userId), authenticated
	}))
})
```
