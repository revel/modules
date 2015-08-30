# JWT Auth Module for Revel Go Framework

Pluggable and easy to use JWT auth module.

### Module Configuration
```ini
auth.jwt.realm.name = "REVEL-JWT-AUTH"                  // default is REVEL-JWT-AUTH
auth.jwt.issuer = "REVEL-JWT-AUTH" 				        // use appropriate values (string, URL), default is REVEL-JWT-AUTH
auth.jwt.expiration = 30						        // In minutes, default is 60 minutes
auth.jwt.key.private = "/Users/jeeva/private.rsa"
auth.jwt.key.public = "/Users/jeeva/public.rsa.pub"
auth.jwt.anonymous = "/token, /freepass/.*"  				// Valid regexp allowed for path
```

### Enabling Auth Module

Add `module.jwtauth = github.com/jeevatkm/jwtauth` into `conf/app.conf`

### Registering Auth Routes

Add `module:jwtauth` into `conf/routes`. Auth modules enables following routes
```sh
# JWT Auth Routes
POST	/token									Auth.Token
GET		/refresh-token							Auth.RefreshToken
GET		/logout									Auth.Logout
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
// and can be accessed using c.Args[jwt.TOKEN_CLAIMS_KEY]
```

### Register Auth Handler

Auth handler is responsible for validate user and returning `Subject (aka sub)` value and success/failure boolean. It should comply [AuthHandler](https://github.com/jeevatkm/jwtauth/blob/master/app/jwt/jwt.go#L31) interface or use raw func via [jwt.AuthHandlerFunc](https://github.com/jeevatkm/jwtauth/blob/master/app/jwt/jwt.go#L37).
```go
revel.OnAppStart(func() {
	jwt.Init(&MyAuth{})
	//          OR
	jwt.Init(jwt.AuthHandlerFunc(func(username, password string) (string, bool) {
		revel.INFO.Printf("Username: %v, Password: %v", username, password)
		return "This is my subject value from function", true
	}))
})
```
