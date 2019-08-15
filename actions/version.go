package actions

import (
	"fmt"

	"github.com/gobuffalo/buffalo"
)

// VersionHandler returns the app version.
func VersionHandler(c buffalo.Context) error {
	return c.Render(200, r.JSON(struct {
		Version    string `json:"version"`
		GitHash    string `json:"githash"`
		BuildStamp string `json:"buildstamp"`
	}{
		fmt.Sprintf("%s%s", Version, VersionPrerelease),
		githash,
		buildstamp,
	}))
}
