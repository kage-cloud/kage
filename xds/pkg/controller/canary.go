package controller

import (
	"github.com/kage-cloud/kage/xds/pkg/exchange"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

const CanaryControllerKey = "CanaryController"

type CanaryController interface {
	Controller
	Create(ctx echo.Context) error
	Delete(ctx echo.Context) error
}

type canaryController struct {
	CanaryControllerService service.CanaryControllerService `inject:"CanaryControllerService"`
}

func (c *canaryController) Create(ctx echo.Context) error {
	req := new(exchange.CreateCanaryRequest)
	if err := ctx.Bind(req); err != nil {
		return err
	}

	if err := req.Validate(); err != nil {
		return err
	}

	res, err := c.CanaryControllerService.Create(req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, res)
}

func (c *canaryController) Delete(ctx echo.Context) error {
	req := new(exchange.DeleteCanaryRequest)
	if err := ctx.Bind(req); err != nil {
		return err
	}

	err := c.CanaryControllerService.Delete(req)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (c *canaryController) Routes() []Route {
	return []Route{
		{
			Path:    "/:namespace",
			Method:  http.MethodPost,
			Handler: c.Create,
		},
		{
			Path:    "/:namespace/:name",
			Method:  http.MethodDelete,
			Handler: c.Delete,
		},
	}
}

func (c *canaryController) Group() string {
	return "canary"
}
