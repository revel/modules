package jwt

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/revel/revel"
	"github.com/revel/revel/cache"
	"gopkg.in/dgrijalva/jwt-go.v2"
)

const (
	ISSUER_KEY       = "iss"
	ISSUED_AT_KEY    = "iat"
	EXPIRATION_KEY   = "exp"
	SUBJECT_KEY      = "sub"
	EXPIRE_OFFSET    = 3600
	TOKEN_CLAIMS_KEY = "jwt.auth.claims"
)

// Objects implementing the AuthHandler interface can be
// registered to Authenticate User for application
type AuthHandler interface {
	Authenticate(username, password string) (string, bool)
}

// The AuthHandlerFunc type is an adapter to allow the use of
// ordinary functions as Auth handlers.
type AuthHandlerFunc func(string, string) (string, bool)

// Authenticate calls f(u, p).
func (f AuthHandlerFunc) Authenticate(u, p string) (string, bool) {
	return f(u, p)
}

var (
	Realm          string
	issuer         string
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	expiration     int // in minutues
	isIssuerExists bool
	handler        AuthHandler
	anonymousPaths *regexp.Regexp
)

/*
Method Init initializes JWT auth provider based on given config values from app.conf
	auth.jwt.realm.name = "REVEL-JWT-AUTH"
	auth.jwt.issuer = "REVEL-JWT-AUTH" 						// use appropriate values
	auth.jwt.expiration = 30								// In minutes
	auth.jwt.key.private = "/Users/jeeva/private.rsa"
	auth.jwt.key.public = "/Users/jeeva/public.rsa.pub"
	auth.jwt.anonymous = "/token, /free/.*"  				// Valid regexp allowed for path
*/
func Init(authHandler interface{}) {
	Realm = revel.Config.StringDefault("auth.jwt.realm.name", "REVEL-JWT-AUTH")
	issuer = revel.Config.StringDefault("auth.jwt.issuer", "")
	expiration = revel.Config.IntDefault("auth.jwt.expiration", 60) // Default 60 minutes
	anonymous := revel.Config.StringDefault("auth.jwt.anonymous", "/token")

	privateKeyPath, found := revel.Config.String("auth.jwt.key.private")
	if !found {
		revel.ERROR.Fatal("No auth.jwt.key.private found.")
	}

	publicKeyPath, found := revel.Config.String("auth.jwt.key.public")
	if !found {
		revel.ERROR.Fatal("No auth.jwt.key.public found.")
	}

	if _, ok := authHandler.(AuthHandler); !ok {
		revel.ERROR.Fatal("Auth Handler doesn't implement interface jwt.AuthenticationHandler")
	}

	Realm = fmt.Sprintf(`Bearer realm="%s"`, Realm)

	// preparing anonymous path regex
	paths := strings.Split(anonymous, ",")
	regexString := ""
	for _, p := range paths {
		regexString = fmt.Sprintf("%s^%s$|", regexString, strings.TrimSpace(p))
	}
	anonymousPaths = regexp.MustCompile(regexString[:len(regexString)-1])

	isIssuerExists = len(issuer) > 0
	handler = authHandler.(AuthHandler)
	privateKey = loadPrivateKey(privateKeyPath)
	publicKey = loadPublicKey(publicKeyPath)
}

// Method GenerateToken creates JWT signed string with given subject value
func GenerateToken(subject string) (string, error) {
	token := jwt.New(jwt.SigningMethodRS512)

	if isIssuerExists {
		token.Claims[ISSUER_KEY] = issuer
	}

	token.Claims[ISSUED_AT_KEY] = time.Now().Unix()
	token.Claims[EXPIRATION_KEY] = time.Now().Add(time.Minute * time.Duration(expiration)).Unix()
	token.Claims[SUBJECT_KEY] = subject

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		revel.ERROR.Printf("Generate token error [%v]", err)
		return "", err
	}

	return tokenString, nil
}

// Method ParseFromRequest retrives JWT token, validates against SigningMethod & Issuer
// then returns *jwt.Token object
func ParseFromRequest(req *http.Request) (*jwt.Token, error) {
	return jwt.ParseFromRequest(req, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		if token.Claims[ISSUER_KEY] != issuer {
			return nil, fmt.Errorf("Unexpected token Issuer: %v", token.Claims[ISSUER_KEY])
		}

		return publicKey, nil
	})
}

// Method TokenRemainingValidity calculates the remaining time left out in auth token
func TokenRemainingValidity(timestamp interface{}) int {
	if validity, ok := timestamp.(float64); ok {
		tm := time.Unix(int64(validity), 0)
		remainer := tm.Sub(time.Now())
		if remainer > 0 {
			return int(remainer.Seconds() + EXPIRE_OFFSET)
		}
	}

	return EXPIRE_OFFSET
}

func Authenticate(username, password string) (string, bool) {
	return handler.Authenticate(username, password)
}

// Method GetAuthToken retrives Auth Token from revel.Request
// 		Authorization: Bearer <auth-token>
func GetAuthToken(req *revel.Request) string {
	authToken := req.Header.Get("Authorization")

	if len(authToken) > 7 { // char count "Bearer " ==> 7
		return authToken[7:]
	}

	return ""
}

// Method IsInBlocklist is checks against logged out tokens
func IsInBlocklist(token string) bool {
	var existingToken string
	cache.Get(token, &existingToken)

	if len(existingToken) > 0 {
		revel.WARN.Printf("Yes, blocklisted token [%v]", existingToken)
		return true
	}

	return false
}

/*
Filter AuthFilter is Revel Filter for JWT Auth Token verification
Register it in the revel.Filters in <APP_PATH>/app/init.go

Add jwt.AuthFilter anywhere deemed appropriate, it must be register after revel.PanicFilter

	revel.Filters = []revel.Filter{
		revel.PanicFilter,
		...
		jwt.AuthFilter,		// JWT Auth Token verification for Request Paths
		...
	}

Note: If everything looks good then Claims map made available via c.Args
and can be accessed using c.Args[jwt.TOKEN_CLAIMS_KEY]
*/
func AuthFilter(c *revel.Controller, fc []revel.Filter) {
	if !anonymousPaths.MatchString(c.Request.URL.Path) {
		token, err := ParseFromRequest(c.Request.Request)
		if err == nil && token.Valid && !IsInBlocklist(GetAuthToken(c.Request)) {
			c.Args[TOKEN_CLAIMS_KEY] = token.Claims

			fc[0](c, fc[1:]) // everything looks good, move on
		} else {
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					revel.ERROR.Println("That's not even a token")
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					revel.ERROR.Println("Timing is everything, Token is either expired or not active yet")
				} else {
					revel.ERROR.Printf("Couldn't handle this token: %v", err)
				}
			} else {
				revel.ERROR.Printf("Couldn't handle this token: %v", err)
			}

			c.Response.Status = http.StatusUnauthorized
			c.Response.Out.Header().Add("WWW-Authenticate", Realm)
			c.Result = c.RenderJson(map[string]string{
				"id":      "unauthorized",
				"message": "Invalid or token is not provided",
			})

			return
		}
	}

	fc[0](c, fc[1:]) //not applying JWT auth filter due to anonymous path
}

// Private Methods
func loadPrivateKey(keyPath string) *rsa.PrivateKey {
	keyData := readKeyFile(keyPath)

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(keyData.Bytes)
	if err != nil {
		revel.ERROR.Fatalf("Private key import error [%v]", keyPath)
	}

	return privateKeyImported
}

func loadPublicKey(keyPath string) *rsa.PublicKey {
	keyData := readKeyFile(keyPath)

	publicKeyImported, err := x509.ParsePKIXPublicKey(keyData.Bytes)
	if err != nil {
		revel.ERROR.Fatalf("Public key import error [%v]", keyPath)
	}

	rsaPublicKey, ok := publicKeyImported.(*rsa.PublicKey)
	if !ok {
		revel.ERROR.Fatalf("Public key assert error [%v]", keyPath)
	}

	return rsaPublicKey
}

func readKeyFile(keyPath string) *pem.Block {
	keyFile, err := os.Open(keyPath)
	defer keyFile.Close()
	if err != nil {
		revel.ERROR.Fatalf("Key file open error [%v]", keyPath)
	}

	pemFileInfo, _ := keyFile.Stat()
	var size int64 = pemFileInfo.Size()
	pemBytes := make([]byte, size)

	buffer := bufio.NewReader(keyFile)
	_, err = buffer.Read(pemBytes)
	if err != nil {
		revel.ERROR.Fatalf("Key file read error [%v]", keyPath)
	}

	keyData, _ := pem.Decode([]byte(pemBytes))

	return keyData
}
