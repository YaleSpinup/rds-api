package actions

import (
	"errors"
	"log"

	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/YaleSpinup/rds-api/pkg/rds"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/middleware"
	"github.com/gobuffalo/envy"

	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

// AppConfig holds the configuration information for the app
var AppConfig common.Config

// RDS is a global map of RDS clients
var RDS = make(map[string]rds.Client)

var app *buffalo.App

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
			app.Use(middleware.ParameterLogger)
		}

		// load json config
		AppConfig, _ := common.LoadConfig("config/config.json")

		// create a shared RDS session for each account
		for account, c := range AppConfig.Accounts {
			RDS[account] = rds.NewSession(c) //.Akid, c.Secret, c.Region)
		}

		app.GET("/v1/rds/ping", PingPong)

		rdsV1API := app.Group("/v1/rds/{account}")
		rdsV1API.Use(sharedTokenAuth(AppConfig.Token), validateAccount)
		rdsV1API.POST("/", DatabasesPost)
		rdsV1API.GET("/", DatabasesGet)
		rdsV1API.GET("/{db}", DatabasesGet)
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

// validateAccount middleware validates the account
func validateAccount(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		if _, ok := RDS[c.Param("account")]; !ok {
			log.Printf("Account not found: %s", c.Param("account"))
			return c.Error(400, errors.New("Bad request"))
		}
		return next(c)
	}
}
