package jwt

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/revel/revel"
	"github.com/revel/revel/cache"
	"gopkg.in/dgrijalva/jwt-go.v2"
)

const (
	Algorithm      = "alg"
	IssueKey       = "iss"
	IssuedAtKey    = "iat"
	ExpirationKey  = "exp"
	SubjectKey     = "sub"
	ExpireOffset   = 3600
	TokenClaimsKey = "jwt.auth.claims"
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
	expiration     int // in minutues
	isIssuerExists bool
	handler        AuthHandler
	anonymousPaths *regexp.Regexp
	signMethodName string

	// RS* signing elements
	jwtAuthSignMethodRSA *jwt.SigningMethodRSA
	rsaPrivateKey        *rsa.PrivateKey
	rsaPublicKey         *rsa.PublicKey

	// ES* signing elements
	jwtAuthSignMethodECDSA *jwt.SigningMethodECDSA
	ecPrivatekey           *ecdsa.PrivateKey
	ecPublickey            *ecdsa.PublicKey

	// HS* signing elements
	jwtAuthSignMethodHMAC *jwt.SigningMethodHMAC
	hmacSecretBytes       []byte
)

/*
Method Init initializes JWT auth provider based on given config values from app.conf
	//
	// JWT Auth Module configuration
	//
	// default is REVEL-JWT-AUTH
	auth.jwt.realm.name = "JWT-AUTH"

	// use appropriate values (string, URL), default is REVEL-JWT-AUTH
	auth.jwt.issuer = "JWT AUTH"

	// In minutes, default is 60 minutes
	auth.jwt.expiration = 30

	// Signing Method
	// options are - HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512
	auth.jwt.sign.method = "RS512"

	// RSA key files
	// applicable to RS256, RS384, RS512 signing method and comment out others
	auth.jwt.key.private = "/Users/jeeva/rsa_private.pem"
	auth.jwt.key.public = "/Users/jeeva/rsa_public.pem"

	// Uncomment below two lines for ES256, ES384, ES512 signing method and comment out others
	// since we  will be use ECDSA
	#auth.jwt.key.private = "/Users/jeeva/ec_private.pem"
	#auth.jwt.key.public = "/Users/jeeva/ec_public.pem"

	// HMAC signing Secret value
	// Uncomment below line for HS256, HS384, HS512 signing method and comment out others
	#auth.jwt.key.hmac = "1A39B778C0CE40B1B32585CF846F61B1"

	// Valid regexp allowed for path
	// Internally it will end up like this "^(/$|/token|/register|/(forgot|validate-reset|reset)-password)"
	auth.jwt.anonymous = "/, /token, /register, /(forgot|validate-reset|reset)-password"
*/
func Init(authHandler interface{}) {
	var (
		found bool
		err   error
	)
	Realm = revel.Config.StringDefault("auth.jwt.realm.name", "REVEL-JWT-AUTH")
	issuer = revel.Config.StringDefault("auth.jwt.issuer", "REVEL-JWT-AUTH")
	expiration = revel.Config.IntDefault("auth.jwt.expiration", 60)              // Default 60 minutes
	signMethodName = revel.Config.StringDefault("auth.jwt.sign.method", "RS512") //Default is RS512

	if strings.HasPrefix(signMethodName, "RS") || strings.HasPrefix(signMethodName, "ES") {
		var (
			privateKeyPath string
			publicKeyPath  string
		)
		privateKeyPath, found = revel.Config.String("auth.jwt.key.private")
		if !found {
			revel.ERROR.Fatal("No auth.jwt.key.private found, it's required for RS*/ES* signing method.")
		}

		publicKeyPath, found = revel.Config.String("auth.jwt.key.public")
		if !found {
			revel.ERROR.Fatal("No auth.jwt.key.public found, it's required for RS*/ES* signing method.")
		}

		if strings.HasPrefix(signMethodName, "RS") {
			// loading RSA key files
			rsaPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(readFile(privateKeyPath))
			if err != nil {
				revel.ERROR.Fatalf("Private key file read error [%v].", privateKeyPath)
			}

			rsaPublicKey, err = jwt.ParseRSAPublicKeyFromPEM(readFile(publicKeyPath))
			if err != nil {
				revel.ERROR.Fatalf("Public key file read error [%v].", publicKeyPath)
			}

			// Signing Method
			jwtAuthSignMethodRSA = identifySigningMethod().(*jwt.SigningMethodRSA)
		} else if strings.HasPrefix(signMethodName, "ES") {
			// loading ECDSA key files
			ecPrivatekey, err = jwt.ParseECPrivateKeyFromPEM(readFile(privateKeyPath))
			if err != nil {
				revel.ERROR.Fatalf("Private key file read error [%v].", privateKeyPath)
			}

			ecPublickey, err = jwt.ParseECPublicKeyFromPEM(readFile(publicKeyPath))
			if err != nil {
				revel.ERROR.Fatalf("Public key file read error [%v].", publicKeyPath)
			}

			// Signing Method
			jwtAuthSignMethodECDSA = identifySigningMethod().(*jwt.SigningMethodECDSA)
		}
	}

	if strings.HasPrefix(signMethodName, "HS") {
		var hmacSecertValue string
		hmacSecertValue, found = revel.Config.String("auth.jwt.key.hmac")
		if !found || len(strings.TrimSpace(hmacSecertValue)) == 0 {
			revel.ERROR.Fatal("No auth.jwt.key.hmac found, it's required for HS* signing method.")
		}

		// Signing Method
		jwtAuthSignMethodHMAC = identifySigningMethod().(*jwt.SigningMethodHMAC)
		hmacSecretBytes = []byte(hmacSecertValue)
	}

	if _, ok := authHandler.(AuthHandler); !ok {
		revel.ERROR.Fatal("Auth Handler doesn't implement interface jwt.AuthHandler")
	}

	Realm = fmt.Sprintf(`Bearer realm="%s"`, Realm)

	// preparing anonymous path regex
	anonymous := revel.Config.StringDefault("auth.jwt.anonymous", "/token")
	paths := strings.Split(anonymous, ",")
	regexString := ""
	for _, p := range paths {
		if strings.TrimSpace(p) == "/" { // TODO Something not right, Might need a revist here
			regexString = fmt.Sprintf("%s%s$|", regexString, strings.TrimSpace(p))
		} else {
			regexString = fmt.Sprintf("%s%s|", regexString, strings.TrimSpace(p))
		}
	}
	regexString = fmt.Sprintf("^(%s)", regexString[:len(regexString)-1])
	anonymousPaths = regexp.MustCompile(regexString)

	isIssuerExists = len(issuer) > 0
	handler = authHandler.(AuthHandler)

	revel.INFO.Printf("JWT Auth Module - Signing Method: %v", signMethodName)
}

// Method GenerateToken creates JWT signed string with given subject value
func GenerateToken(subject string) (string, error) {
	token := createToken()

	token.Claims[IssueKey] = issuer
	token.Claims[IssuedAtKey] = time.Now().Unix()
	token.Claims[ExpirationKey] = time.Now().Add(time.Minute * time.Duration(expiration)).Unix()
	token.Claims[SubjectKey] = subject

	return signString(token)
}

// Method ParseFromRequest retrives JWT token, validates against SigningMethod & Issuer
// then returns *jwt.Token object
func ParseFromRequest(req *http.Request) (*jwt.Token, error) {
	return jwt.ParseFromRequest(req, func(token *jwt.Token) (interface{}, error) {
		// Signing Method verification
		// https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
		var signMethodOk bool
		var key interface{}
		if strings.HasPrefix(signMethodName, "RS") {
			_, signMethodOk = token.Method.(*jwt.SigningMethodRSA)
			key = rsaPublicKey
		} else if strings.HasPrefix(signMethodName, "HS") {
			_, signMethodOk = token.Method.(*jwt.SigningMethodHMAC)
			key = hmacSecretBytes
		} else if strings.HasPrefix(signMethodName, "ES") {
			_, signMethodOk = token.Method.(*jwt.SigningMethodECDSA)
			key = ecPublickey
		}
		if !signMethodOk {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header[Algorithm])
		}

		// Issuer verification
		if token.Claims[IssueKey] != issuer {
			return nil, fmt.Errorf("Unexpected token issuer: %v", token.Claims[IssueKey])
		}

		return key, nil
	})
}

// Method TokenRemainingValidity calculates the remaining time left out in auth token
func TokenRemainingValidity(timestamp interface{}) int {
	if validity, ok := timestamp.(float64); ok {
		tm := time.Unix(int64(validity), 0)
		remainer := tm.Sub(time.Now())
		if remainer > 0 {
			return int(remainer.Seconds() + ExpireOffset)
		}
	}

	return ExpireOffset
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
and can be accessed using c.Args[jwt.TokenClaimsKey]
*/
func AuthFilter(c *revel.Controller, fc []revel.Filter) {
	if anonymousPaths.MatchString(c.Request.URL.Path) {
		fc[0](c, fc[1:]) //not applying JWT auth filter due to anonymous path
	} else {
		token, err := ParseFromRequest(c.Request.Request)
		if err == nil && token.Valid && !IsInBlocklist(GetAuthToken(c.Request)) {
			c.Args[TokenClaimsKey] = token.Claims

			fc[0](c, fc[1:]) // everything looks good, move on
		} else {
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					revel.ERROR.Println("That's not even a token")
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					revel.ERROR.Println("Timing is everything, Token is either expired or not active yet")
				} else {
					revel.ERROR.Printf("Couldn't handle this request: %v", err)
				}
			} else {
				revel.ERROR.Printf("Couldn't handle this request: %v", err)
			}

			c.Response.Status = http.StatusUnauthorized
			c.Response.Out.Header().Add("Www-Authenticate", Realm)
			c.Result = c.RenderJson(map[string]string{
				"id":      "unauthorized",
				"message": "Invalid or token is not provided",
			})

			return
		}
	}
}

//
// Private Methods
//
func createToken() *jwt.Token {
	if strings.HasPrefix(signMethodName, "HS") {
		return jwt.New(jwtAuthSignMethodHMAC)
	} else if strings.HasPrefix(signMethodName, "ES") {
		return jwt.New(jwtAuthSignMethodECDSA)
	}

	return jwt.New(jwtAuthSignMethodRSA)
}

func signString(token *jwt.Token) (signedString string, err error) {
	switch signMethodName {
	case "RS256", "RS384", "RS512":
		signedString, err = token.SignedString(rsaPrivateKey)
	case "HS256", "HS384", "HS512":
		signedString, err = token.SignedString(hmacSecretBytes)
	case "ES256", "ES384", "ES512":
		signedString, err = token.SignedString(ecPrivatekey)
	}

	if err != nil {
		revel.ERROR.Printf("Generate token error [%v]", err)
	}

	return
}

func readFile(file string) []byte {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		revel.ERROR.Fatalf("Key file read error [%v]", file)
	}
	return bytes
}

func identifySigningMethod() (signingMethod interface{}) {
	switch signMethodName {
	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	case "RS384":
		signingMethod = jwt.SigningMethodRS384
	case "RS512":
		signingMethod = jwt.SigningMethodRS512
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS384":
		signingMethod = jwt.SigningMethodHS384
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "ES384":
		signingMethod = jwt.SigningMethodES384
	case "ES512":
		signingMethod = jwt.SigningMethodES512
	}
	return
}
