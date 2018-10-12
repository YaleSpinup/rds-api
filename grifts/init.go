package grifts

import (
	"github.com/YaleSpinup/rds-api/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
