package controller

import (
	"github.com/gogo/protobuf/jsonpb"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/exchange"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/labstack/echo/v4"
	"net/http"
)

const AdminControllerKey = "AdminController"

type AdminController interface {
	Controller
	Get(ctx echo.Context) error
}

type adminController struct {
	KageMeshService service.KageMeshService `inject:"KageMeshService"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
}

func (a *adminController) Routes() []Route {
	return []Route{
		{
			Handler: a.Get,
			Method:  http.MethodGet,
			Path:    "/:namespace/:canary_name",
		},
	}
}

func (a *adminController) Group() string {
	return "admin"
}

func (a *adminController) Get(ctx echo.Context) error {
	req := new(exchange.GetAdminRequest)
	if err := ctx.Bind(req); err != nil {
		return err
	}

	v, err := a.KageMeshService.Fetch(req.CanaryName, kconfig.Opt{Namespace: req.Namespace})
	if err != nil {
		return err
	}

	state, err := a.StoreClient.Get(v.MeshConfig.NodeId)
	if err != nil {
		return err
	}

	str, err := new(jsonpb.Marshaler).MarshalToString(state)
	if err != nil {
		return err
	}

	return ctx.JSONBlob(http.StatusOK, []byte(str))
}
