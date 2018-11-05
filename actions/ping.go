package actions

import "github.com/gobuffalo/buffalo"

// PingPong responds to a ping
func PingPong(c buffalo.Context) error {
	return c.Render(200, r.String("pong"))
}
