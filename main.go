package main

import (
	"log"

	"github.com/YaleSpinup/rds-api/actions"
)

func main() {
	app := actions.App()
	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}

