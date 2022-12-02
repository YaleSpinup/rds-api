package actions

import (
	"encoding/json"
	"flag"
	"fmt"
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

	// ConfigFile is the name of the json config file
	ConfigFile = "config/config.json"

	// The org for this instance of the app
	Org string

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
	client *rds.Client
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

		app.ErrorHandlers = buffalo.ErrorHandlers{
			0:                              defaultErrorHandler,
			http.StatusNotFound:            defaultErrorHandler,
			http.StatusInternalServerError: defaultErrorHandler,
		}

		if ENV == "development" {
			app.Use(paramlogger.ParameterLogger)
		}

		// override values for test runs
		if flag.Lookup("test.v") != nil {
			ConfigFile = "../config/config.example.json"
		}

		// load json config
		appCfg, err := common.LoadConfig(ConfigFile)
		if err != nil {
			log.Fatalf("Failed to load config %s: %+v", ConfigFile, err)
		}

		Org = appCfg.Org

		s := newServer(appCfg)

		app.GET("/v1/rds/ping", PingPong)
		app.GET("/v1/rds/version", VersionHandler)

		rdsV1API := app.Group("/v1/rds/{account}")
		rdsV1API.Use(s.authHandler)
		rdsV1API.POST("/", s.DatabasesPost)
		rdsV1API.GET("/", s.DatabasesList)
		rdsV1API.GET("/{db}", s.DatabasesGet)
		rdsV1API.PUT("/{db}", s.DatabasesPut)
		rdsV1API.PUT("/{db}/power", s.DatabasesPutState)
		rdsV1API.DELETE("/{db}", s.DatabasesDelete)
		rdsV1API.POST("/{db}/snapshots", s.SnapshotsPost)
		rdsV1API.GET("/{db}/snapshots", s.SnapshotsList)
		rdsV1API.GET("/{db}/snapshots", s.SnapshotsVersionList)
		rdsV1API.GET("/snapshots/{snap}", s.SnapshotsGet)
		rdsV1API.DELETE("/snapshots/{snap}", s.SnapshotsDelete)

		log.Printf("Started rds-api in org %s", Org)
	}

	return app
}

// authHandler middleware validates the auth token
func (s *server) authHandler(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		htoken := c.Request().Header.Get("X-Auth-Token")
		if htoken == "" {
			log.Println("Missing token header for request", c.Request().URL)
			return c.Error(403, errors.New("Forbidden"))
		}
		if err := bcrypt.CompareHashAndPassword([]byte(htoken), s.token); err != nil {
			log.Println("Bad token for request", c.Request().URL)
			return c.Error(403, errors.New("Forbidden"))
		}
		return next(c)
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

// defaultErrorHandler takes an error and returns a JSON representation for
// easier consumption.
func defaultErrorHandler(status int, origErr error, c buffalo.Context) error {
	c.LogField("status", status)
	c.Logger().Error(origErr)

	c.Response().WriteHeader(status)
	c.Response().Header().Set("content-type", "application/json")

	resp := struct {
		Error   string `json:"error"`
		Message string `json:"message,omitempty"`
	}{
		Error: origErr.Error(),
	}

	// if the error is an apierror (return from handleError())
	// else if it's a buffalo error (return from c.Error()) with an
	// an apierror as the cause.  this should probably be more consistent
	if aerr, ok := origErr.(apierror.Error); ok {
		c.Logger().Debugf("orig error is an apierr: %+v", origErr)
		resp.Error = aerr.Error()
		resp.Message = aerr.Message
	} else if berr, ok := origErr.(buffalo.HTTPError); ok {
		c.Logger().Debugf("error is a buffalo error: %+v", berr)

		if aerr, ok := berr.Cause.(apierror.Error); ok {
			c.Logger().Debugf("error cause is an apierr: %+v", berr)
			resp.Error = aerr.Error()
			resp.Message = aerr.Message
		}
	}

	j, err := json.Marshal(resp)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal error into JSON: %s", err.Error())
		c.Response().Write([]byte(msg))
		return nil
	}

	c.Response().Write(j)

	return nil
}
