package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"

	"gitlab.0pointer.org/choopm/greeter"
	"gitlab.0pointer.org/choopm/greeter/api/services/greeterservice"

	"gitlab.0pointer.org/choopm/grpchelpers"
)

var (
	address      = os.Getenv("ADDRESS")
	port         = os.Getenv("PORT")
	passwdFile   = os.Getenv("PASSWD")
	certFile     = os.Getenv("TLS_CRT")
	keyFile      = os.Getenv("TLS_KEY")
	httpRedirect = os.Getenv("HTTP_REDIRECT")
	jwtSecret    = grpchelpers.GetTrimmedStringFromFile(os.Getenv("JWT_SECRET"))
)

func auth(c echo.Context) error {
	type Req struct {
		Username string `json:"username" form:"username" query:"username"`
		Password string `json:"password" form:"password" query:"password"`
	}
	req := new(Req)
	if err := c.Bind(req); err != nil {
		return err
	}

	file, err := os.Open(passwdFile)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	matched := false
	for scanner.Scan() {
		line := scanner.Text()
		user := strings.Split(line, ":")[0]
		if user != req.Username {
			continue
		}

		hash := strings.Join(strings.Split(line, ":")[1:], ":")
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
		if err == nil {
			matched = true
			break
		}
	}

	// Throws unauthorized error
	if !matched {
		return echo.ErrUnauthorized
	}

	token, err := createJWT(req.Username, time.Now().Add(time.Hour*72).Unix())
	if err != nil {
		return err
	}

	cookie := new(http.Cookie)
	cookie.Name = "bearer-token"
	cookie.Value = token
	cookie.Expires = time.Now().Add(time.Hour * 72)
	cookie.Secure = true
	//c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{
		"token":    token,
		"username": req.Username,
	})
}

func createJWT(username string, expires int64) (string, error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Hour * 72).Unix(),
		IssuedAt:  time.Now().Unix(),
		Issuer:    "greeter",
	}
	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}
	return string(t), nil
}
func main() {
	// Generate default login if passwd is missing
	_, err := os.Stat(passwdFile)
	if os.IsNotExist(err) {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Unable to generate hash", err)
		}
		err = os.WriteFile(passwdFile, []byte("admin:"+string(hash)), 0644)
		if err != nil {
			log.Println("Unable to write "+passwdFile, err)
		}
	} else if err != nil {
		log.Fatalln("Can't stat "+passwdFile, err)
	}

	// Start grpc greeter
	go greeter.Start(jwtSecret, certFile, keyFile)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts, err := grpchelpers.GetDialOptions("/dev/null", certFile, "app")
	check(err)
	err = greeterservice.RegisterGreeterServiceHandlerFromEndpoint(ctx, mux, "127.0.0.1:50051", opts)
	check(err)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Redirect any 404 to /
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		if code == 404 {
			c.Redirect(302, "/")
		} else {
			e.DefaultHTTPErrorHandler(err, c)
		}
	}

	// Serve vue
	e.Static("/", "/static")

	// Auth route
	e.POST("/auth", auth)

	// grpc-gateway
	api := e.Group("/api")
	api.Use(middleware.JWT([]byte(jwtSecret))) // JWT
	api.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// r.Header.Del("authorization") // don't pass the auth header to upstreams
			mux.ServeHTTP(w, r)
		})
	}))

	if httpRedirect == "true" {
		e2 := echo.New()
		e2.Use(middleware.Logger())
		e2.Use(middleware.Recover())
		e2.Pre(middleware.HTTPSRedirect())
		go e2.Start(address + ":80")
		e.Logger.Fatal(e.StartTLS(address+":"+port, certFile, keyFile))
	} else {
		e.Logger.Fatal(e.Start(address + ":" + port))
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
