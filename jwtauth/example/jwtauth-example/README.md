## JWT Auth Module Demo Revel Application

Ready to see JWT Auth module in action. It demonstrates the JWT Auth Module usage. 

### How to run?
```sh
$ go get github.com/jeevatkm/jwtauth

$ mv src/github.com/jeevatkm/jwtauth/example/jwtauth-example src/github.com/jeevatkm/

$ revel run github.com/jeevatkm/jwtauth-example

# now application will be running http://localhost:9000
```

### How to create Key files?
```sh
# generating private key file
$ openssl genrsa -out private.rsa 2048

# creating a public from private key we generated above
$ openssl rsa -in private.rsa -pubout > public.rsa.pub
```