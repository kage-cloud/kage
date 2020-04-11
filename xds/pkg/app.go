package pkg

import (
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/controller"
	"github.com/kage-cloud/kage/xds/pkg/controlplane"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

const AppKey = "App"

type App interface {
	Start() error
}

type app struct {
	Controllers       []axon.Instance          `inject:"Controllers"`
	Config            *config.Config           `inject:"Config"`
	EnvoyControlPlane controlplane.Envoy       `inject:"EnvoyControlPlane"`
	StateSyncService  service.StateSyncService `inject:"StateSyncService"`
}

func (a *app) Start() error {
	if err := a.EnvoyControlPlane.StartAsync(); err != nil {
		return err
	}

	if err := a.StateSyncService.Start(); err != nil {
		return err
	}

	e := echo.New()
	if log.GetLevel() >= log.DebugLevel {
		e.Use(middleware.Logger(), middleware.Recover())
	}

	e.Use(middleware.CORS())
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = customHTTPErrorHandler(e.DefaultHTTPErrorHandler)

	for _, v := range a.Controllers {
		c := v.GetStructPtr().(controller.Controller)

		for _, r := range c.Routes() {
			api := e.Group("/api")
			api = api.Group(c.Group())
			api.Add(r.Method, r.Path, r.Handler)
		}
	}

	log.WithField("port", a.Config.Server.Port).Info("Starting API server")
	return e.Start(fmt.Sprintf(":%d", a.Config.Server.Port))
}

func customHTTPErrorHandler(defaultHandler echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(err error, context echo.Context) {
		defaultHandler(echo.NewHTTPError(except.ToHttpStatus(err), err.Error()), context)
	}
}
