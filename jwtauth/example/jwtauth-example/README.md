## JWT Auth Module Demo Revel Application

Ready to see JWT Auth module in action. It demonstrates the JWT Auth Module usage. 

### How to run?
```sh
$ cd $GOPATH

$ go get github.com/jeevatkm/jwtauth

$ ln -s $GOPATH/src/github.com/jeevatkm/jwtauth/example/jwtauth-example $GOPATH/src/github.com/jeevatkm/jwtauth-example

$ revel run github.com/jeevatkm/jwtauth-example

# now application will be running http://localhost:9000
```

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
