package main

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/xds/controller"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
)

const AppKey = "App"

type App interface {
	Start() error
}

type app struct {
	Controllers []axon.Instance `inject:"Controllers"`
}

func (a *app) Start() error {
	e := echo.New()

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	for _, v := range a.Controllers {
		c := v.GetStructPtr().(controller.Controller)

		for _, r := range c.Routes() {
			api := e.Group("/api")
			api = api.Group(c.Group())
			api.Add(r.Method, r.Path, r.Handler)
		}
	}

	return e.Start(":" + port)
}
