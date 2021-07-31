package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"

	"gitlab.0pointer.org/choopm/greeter/api/services/greeterservice"

	"gitlab.0pointer.org/choopm/greeter/internal/common"
	"gitlab.0pointer.org/choopm/greeter/internal/database"
	"gitlab.0pointer.org/choopm/greeter/internal/greeter"

	"gitlab.0pointer.org/choopm/grpchelpers"
)

var (
	address      = os.Getenv("ADDRESS")
	port         = os.Getenv("PORT")
	databaseFile = os.Getenv("DATABASE")
	certFile     = os.Getenv("TLS_CRT")
	keyFile      = os.Getenv("TLS_KEY")
	httpRedirect = os.Getenv("HTTP_REDIRECT")
	cookieAuth   = os.Getenv("COOKIE_AUTH")
)

func main() {
	// Open DB
	db, err := database.New(databaseFile)
	if err != nil {
		log.Fatalln("Unable to open database", err)
	}

	if !db.HasUsers() {
		log.Println("Creating initial admin user")
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Unable to generate hash", err)
		}
		db.CreateUser("admin", string(hash))
	}

	jwtSecret := db.GetSetting("jwtsecret")
	if jwtSecret == "" {
		jwtSecret = db.SaveSetting("jwtsecret", common.RandStringBytesMaskImprSrc(32))
		if jwtSecret == "" {
			log.Fatalln("Unable to get/create jwtsecret")
		}
	}

	// Start grpc greeter
	go greeter.StartServer(jwtSecret, certFile, keyFile)

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
	e.HideBanner = true
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

	if cookieAuth == "true" {
		// Extract a present bearer-token cookie and inject it as authorization header
		e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bearerToken, err := r.Cookie("bearer-token")
				if err == nil {
					r.Header.Set("authorization", "Bearer "+bearerToken.Value)
				}
				next.ServeHTTP(w, r)
			})
		}))
	}

	// Serve vue
	e.Static("/", "/static")

	// Auth route
	e.POST("/auth", func(c echo.Context) error {
		type Req struct {
			Username string `json:"username" form:"username" query:"username"`
			Password string `json:"password" form:"password" query:"password"`
		}
		req := new(Req)
		if err := c.Bind(req); err != nil {
			return err
		}

		user := db.GetUserByName(req.Username)
		if user == nil {
			return echo.ErrUnauthorized
		}

		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
		if err != nil {
			return echo.ErrUnauthorized
		}
		// success

		token, err := createJWT(jwtSecret, req.Username, time.Now().Add(time.Hour*72).Unix())
		if err != nil {
			return err
		}

		if cookieAuth == "true" {
			cookie := new(http.Cookie)
			cookie.Name = "bearer-token"
			cookie.Value = token
			cookie.Expires = time.Now().Add(time.Hour * 72)
			cookie.Secure = true
			cookie.HttpOnly = true
			c.SetCookie(cookie)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"token":    token,
			"username": req.Username,
		})
	})

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
		e2.HideBanner = true
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

func createJWT(jwtSecret, username string, expires int64) (string, error) {
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
