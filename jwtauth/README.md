# JWT Auth Module for Revel Framework

Pluggable and easy to use JWT auth module in Revel Framework. Module supports following JWT signing method `HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512`. Default is `RS512`.

Include [example application](example/jwtauth-example) and it demostrates above mentioned JWT signing method(s).

Planning to bring following enhancement to this moudle:
* Module error messages via Revel messages `/messages/<filename>.en, etc`

### Module Configuration
```ini
# default is REVEL-JWT-AUTH
auth.jwt.realm.name = "JWT-AUTH"

# use appropriate values (string, URL), default is REVEL-JWT-AUTH
auth.jwt.issuer = "JWT AUTH"

# In minutes, default is 60 minutes
auth.jwt.expiration = 30

# Signing Method
# options are - HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512
auth.jwt.sign.method = "RS512"

# RSA key files
# applicable to RS256, RS384, RS512 signing method and comment out others
auth.jwt.key.private = "/Users/jeeva/rsa_private.pem"
auth.jwt.key.public = "/Users/jeeva/rsa_public.pem"

# ECDSA key files
# Uncomment below two lines for ES256, ES384, ES512 signing method and comment out others
#auth.jwt.key.private = "/Users/jeeva/ec_private.pem"
#auth.jwt.key.public = "/Users/jeeva/ec_public.pem"

# HMAC signing Secret value
# Uncomment below line for HS256, HS384, HS512 signing method and comment out others
#auth.jwt.key.hmac = "1A39B778C0CE40B1B32585CF846F61B1"

# Valid regexp allowed for path
# Internally it will end up like this "^(/$|/token|/register|/(forgot|validate-reset|reset)-password)"
auth.jwt.anonymous = "/, /token, /register, /(forgot|validate-reset|reset)-password, /freepass/.*"
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

### Registering Auth Handler

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
		authenticated := true  			// Auth success

		return fmt.Sprintf("%d", userId), authenticated
	}))
})
```
