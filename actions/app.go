package actions

import (
	"log"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/YaleSpinup/rds-api/pkg/rds"
	"github.com/YaleSpinup/rds-api/rdsapi"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
)

var (
	app *buffalo.App

	// ENV is used to help switch settings based on where the
	// application is being run. Default is "development".
	ENV = envy.Get("GO_ENV", "development")

	// AppConfig holds the configuration information for the app
	AppConfig common.Config

	// RDS is a global map of RDS clients
	RDS = make(map[string]rds.Client)

	// Version is the main version number
	Version = rdsapi.Version

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = rdsapi.VersionPrerelease

	// BuildStamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	BuildStamp = rdsapi.BuildStamp

	// GitHash is the git sha of the built binary, it should be set at buildtime with ldflags
	GitHash = rdsapi.GitHash
)

type rdsOrchestrator struct {
	client rds.Client
}

// App is where all routes and middleware for buffalo should be defined
func App() *buffalo.App {
	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			PreWares: []buffalo.PreWare{
				cors.Default().Handler,
			},
			SessionName: "_rdsapi_session",
		})

		if ENV == "development" {
			app.Use(paramlogger.ParameterLogger)
		}

		// load json config
		AppConfig, _ := common.LoadConfig("config/config.json")

		// create a shared RDS session for each account
		for account, c := range AppConfig.Accounts {
			RDS[account] = rds.NewSession(c)
		}

		app.GET("/v1/rds/ping", PingPong)
		app.GET("/v1/rds/version", VersionHandler)

		rdsV1API := app.Group("/v1/rds/{account}")
		rdsV1API.Use(sharedTokenAuth([]byte(AppConfig.Token)))
		rdsV1API.POST("/", DatabasesPost)
		rdsV1API.GET("/", DatabasesList)
		rdsV1API.GET("/{db}", DatabasesGet)
		rdsV1API.PUT("/{db}", DatabasesPut)
		rdsV1API.PUT("/{db}/power", DatabasesPutState)
		rdsV1API.DELETE("/{db}", DatabasesDelete)
	}

	return app
}

// sharedTokenAuth middleware validates the auth token
func sharedTokenAuth(token []byte) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			htoken, ok := c.Request().Header["X-Auth-Token"]

			if !ok || len(htoken) == 0 {
				log.Println("Missing token header for request", c.Request().URL)
				return c.Error(403, errors.New("Forbidden"))
			}
			if err := bcrypt.CompareHashAndPassword([]byte(htoken[0]), token); err != nil {
				log.Println("Bad token for request", c.Request().URL)
				return c.Error(403, errors.New("Forbidden"))
			}

			return next(c)
		}
	}
}

// handleError handles standard apierror return codes
func handleError(c buffalo.Context, err error) error {
	log.Println(err.Error())
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		switch aerr.Code {
		case apierror.ErrForbidden:
			return c.Error(http.StatusForbidden, aerr)
		case apierror.ErrNotFound:
			return c.Error(http.StatusNotFound, aerr)
		case apierror.ErrConflict:
			return c.Error(http.StatusConflict, aerr)
		case apierror.ErrBadRequest:
			return c.Error(http.StatusBadRequest, aerr)
		case apierror.ErrLimitExceeded:
			return c.Error(http.StatusTooManyRequests, aerr)
		default:
			return c.Error(http.StatusInternalServerError, aerr)
		}
	} else {
		return c.Error(http.StatusInternalServerError, err)
	}
}
