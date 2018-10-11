package grifts

import (
	"github.com/YaleSpinup/rds_api/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
