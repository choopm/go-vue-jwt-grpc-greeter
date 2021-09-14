package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"

	"github.com/choopm/go-vue-jwt-grpc-greeter/api/services/greeterservice"

	"github.com/choopm/go-vue-jwt-grpc-greeter/pkg/common"
	"github.com/choopm/go-vue-jwt-grpc-greeter/pkg/database"
	"github.com/choopm/go-vue-jwt-grpc-greeter/pkg/greeter"
	"github.com/choopm/go-vue-jwt-grpc-greeter/pkg/jwthelper"

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

	// create inital admin user
	if !db.HasUsers() {
		log.Println("Creating initial admin user")
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Unable to generate hash", err)
		}
		db.CreateUser("admin", string(hash), "admin,user")
	}

	jwtSecret := db.GetSetting("jwtsecret")
	if jwtSecret == "" {
		// create inital jwtSecret
		jwtSecret = db.SaveSetting("jwtsecret", common.RandStringBytesMaskImprSrc(32))
		if jwtSecret == "" {
			log.Fatalln("Unable to get/create jwtsecret")
		}
	}
	greeterBind := db.GetSetting("greeterbind")
	if greeterBind == "" {
		// create setting greeterBind
		greeterBind = db.SaveSetting("greeterbind", "127.0.0.1:50051")
		if greeterBind == "" {
			log.Fatalln("Unable to get/create greeterBind")
		}
	}

	// Start grpc greeter
	greeterServer := greeter.New(jwtSecret)
	go greeterServer.Start(greeterBind, jwtSecret, certFile, keyFile)

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

	// Setup echo
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

		token, err := jwthelper.CreateJWT(jwtSecret, req.Username, user.Roles, time.Now().Add(time.Hour*72).Unix())
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
		// Redirect any http requests to https using a second echo server
		e2 := echo.New()
		e2.HideBanner = true
		e2.Use(middleware.Logger())
		e2.Use(middleware.Recover())
		e2.Pre(middleware.HTTPSRedirect())
		go e2.Start(address + ":80")

		// Start TLS
		e.Logger.Fatal(e.StartTLS(address+":"+port, certFile, keyFile))
	} else {
		// Start without TLS
		e.Logger.Fatal(e.Start(address + ":" + port))
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
