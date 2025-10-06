package main

import (
	"github.com/aos-cc/provisioning-service/internal/app"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		app.Module,
	).Run()
}
