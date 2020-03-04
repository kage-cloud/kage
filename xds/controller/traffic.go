package controller

import (
	"github.com/eddieowens/kage/xds/exchange"
	"github.com/eddieowens/kage/xds/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

const TrafficControllerKey = "TrafficController"

type TrafficController interface {
	Controller
	Direct(ctx echo.Context) error
}

type trafficController struct {
	TrafficControllerService service.TrafficControllerService `inject:"TrafficControllerService"`
}

func (t *trafficController) Direct(ctx echo.Context) error {
	req := new(exchange.DirectTrafficRequest)
	if err := ctx.Bind(req); err != nil {
		return err
	}

	resp, err := t.TrafficControllerService.Direct(req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (t *trafficController) Routes() []Route {
	return []Route{
		{
			Path:    "/:endpoint_name/direct",
			Method:  http.MethodPost,
			Handler: t.Direct,
		},
	}
}

func (t *trafficController) Group() string {
	return "traffic"
}
