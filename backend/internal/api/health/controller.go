package health

import (
	"context"
	"fmt"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Controller struct {
	pinger Pinger
}

func New(pinger Pinger) Controller {
	return Controller{pinger: pinger}
}

func (controller Controller) CheckHealth(
	ctx context.Context,
	_ generated.CheckHealthRequestObject,
) (generated.CheckHealthResponseObject, error) {
	err := controller.pinger.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrUnavailable, err)
	}

	return generated.CheckHealth200JSONResponse{Status: "ok"}, nil
}
