package actions

import (
	"errors"
	"log"

	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/YaleSpinup/rds-api/pkg/rds"
	"github.com/YaleSpinup/rds-api/rdsapi"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	paramlogger "github.com/gobuffalo/mw-paramlogger"

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
		rdsV1API.Use(sharedTokenAuth(AppConfig.Token))
		rdsV1API.POST("/", DatabasesPost)
		rdsV1API.GET("/", DatabasesList)
		rdsV1API.GET("/{db}", DatabasesGet)
		rdsV1API.PUT("/{db}", DatabasesPut)
		rdsV1API.DELETE("/{db}", DatabasesDelete)
	}

	return app
}

// sharedTokenAuth middleware validates the auth token
func sharedTokenAuth(token string) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			headers, ok := c.Request().Header["X-Auth-Token"]
			if !ok || len(headers) == 0 || headers[0] != token {
				log.Println("Missing or bad token header for request", c.Request().URL)
				return c.Error(403, errors.New("Forbidden"))
			}
			return next(c)
		}
	}
}
