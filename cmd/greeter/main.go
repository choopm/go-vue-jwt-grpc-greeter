package main

import (
	"bufio"
	"context"
	"html/template"
	"io"
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
	address         = os.Getenv("ADDRESS")
	port            = os.Getenv("PORT")
	bearerTokenFile = os.Getenv("BEARER_TOKEN")
	passwdFile      = os.Getenv("PASSWD")
	certFile        = os.Getenv("TLS_CRT")
	keyFile         = os.Getenv("TLS_KEY")
	httpsRedirect   = os.Getenv("HTTPS_REDIRECT")
	jwtSecret       = grpchelpers.GetBearerTokenFromFile(bearerTokenFile)
)

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Println(err)
	}
	return err
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"world": func() string {
			return "World"
		},
	}
}

func auth(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

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
		if user != username {
			continue
		}

		hash := strings.Join(strings.Split(line, ":")[1:], ":")
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err == nil {
			matched = true
			break
		}
	}

	// Throws unauthorized error
	if !matched {
		return echo.ErrUnauthorized
	}

	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = username
	claims["expires"] = time.Now().Add(time.Hour * 72).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return err
	}

	cookie := new(http.Cookie)
	cookie.Name = "bearer-token"
	cookie.Value = t
	cookie.Expires = time.Now().Add(time.Hour * 72)
	cookie.Secure = true
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]string{
		"token": t,
	})
}

func testjwt(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	name := claims["name"].(string)
	return c.String(http.StatusOK, "Hello "+name)
}

func main() {
	// Generate default login if passwd is missing
	_, err := os.Stat(passwdFile)
	if os.IsNotExist(err) {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Println(err)
		}
		err = os.WriteFile(passwdFile, []byte("admin"+string(hash)), 0644)
		if err != nil {
			log.Println(err)
		}
	} else {
		log.Println(err)
	}

	// Start grpc greeter
	go greeter.Start(bearerTokenFile, certFile, keyFile)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts, err := grpchelpers.GetDialOptions(bearerTokenFile, certFile, "app")
	check(err)
	err = greeterservice.RegisterGreeterServiceHandlerFromEndpoint(ctx, mux, "127.0.0.1:50051", opts)
	check(err)

	e := echo.New()
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.New("").Funcs(templateFuncs()).ParseGlob("web/template/*.html")),
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// root redirect
	e.GET("/", func(c echo.Context) (err error) {
		return c.Redirect(302, "/app/")
	})

	e.Static("/static/", "web/static")

	// Extract a bearer-token cookie and set authorization header
	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearerToken, err := r.Cookie("bearer-token")
			if err == nil {
				r.Header.Set("authorization", "Bearer "+bearerToken.Value)
			}
			next.ServeHTTP(w, r)
		})
	}))

	e.GET("/app/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "app.html", nil)
	})

	// Auth route
	// Required to test authorisation when jwt does't exist
	e.POST("/auth", auth)
	e.GET("/auth", func(c echo.Context) error {
		return c.Render(http.StatusOK, "auth.html", nil)
	})

	// JWT test
	jwt := e.Group("/testjwt")
	jwt.Use(middleware.JWT([]byte(jwtSecret))) // JWT
	jwt.GET("", testjwt)

	// grpc-gateway
	api := e.Group("/api")
	api.Use(middleware.JWT([]byte(jwtSecret))) // JWT
	api.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del("authorization") // don't pass the auth header to upstreams
			mux.ServeHTTP(w, r)
		})
	}))

	if httpsRedirect == "true" {
		e2 := echo.New()
		e2.Use(middleware.Logger())
		e2.Use(middleware.Recover())
		e2.Pre(middleware.HTTPSRedirect())
		go e2.Start(address + ":80")
		e.Logger.Fatal(e.StartTLS(address+":"+port, certFile, keyFile))
	} else {
		e.Start(address + ":80")
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
