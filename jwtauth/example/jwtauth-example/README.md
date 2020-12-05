## JWT Auth Module Demo Revel Application

Ready to see JWT Auth module in action. It demonstrates the JWT Auth Module usage. 

### How to run?
```sh
$ cd $GOPATH

$ go get github.com/revel/modules

$ revel run github.com/revel/modules/jwtauth/example/jwtauth-example

# now application will be running http://localhost:9000
```

### Sample Application in action
JWT Auth Module provides following routes for Revel application. It is perfectly ready to use for REST API, Web application and Single Page web application.
```sh
# JWT Auth Routes
POST	/token									JwtAuth.Token
GET		/refresh-token							JwtAuth.RefreshToken
GET		/logout									JwtAuth.Logout
```
Sample application has three user information, such as-
```sh
Username: jeeva@myjeeva.com
Password: sample1

Username: user1@myjeeva.com
Password: user1

Username: user2@myjeeva.com
Password: user2
```

#### /token
Let's get the auth token, pick any rest client by your choice and user credentials from above.

|  Field  | Value |
| ------------- | ------------- |
| URL  | http://localhost:9000/token |
| Method  | POST |
| Body    | ` { "username":"jeeva@myjeeva.com", "password":"sample1" } ` |
| Response  | ` {  "token": "<auth-token-value-come-here>" } ` |

#### /refresh-oken
Let's get refereshed auth token. **Once you execute this request, your previous auth token is no longer valid.**

|  Field  | Value |
| ------------- | ------------- |
| URL  |http://localhost:9000/refresh-token |
| Method  | GET |
| Header    | ` Authorization: Bearer <place-auth-token-you-got-from-above> ` |
| Response  | ` {  "token": "<new-refreshed-auth-token-value-come-here>" } ` |

#### /logout
I'm done, let's logout from application.

|  Field  | Value |
| ------------- | ------------- |
| URL  |http://localhost:9000/logout |
| Method  | GET |
| Header    | ` Authorization: Bearer <place-your-auth-toke> ` |
| Response  | ` { "id": "success", "message": "Successfully logged out" } ` |


### How to create RSA Key files?
```sh
# generating private key file
$ openssl genrsa -out rsa_private.pem 2048

# creating a public from private key we generated above
$ openssl rsa -in rsa_private.pem -pubout > rsa_public.pem
```

### How to create ECDSA Key files?
```sh
# generating private key file
$ openssl ecparam -name secp384r1 -genkey -noout -out ec_private.pem

# creating a public from private key we generated above
$ openssl ec -in ec_private.pem -pubout -out ec_public.pem
```
