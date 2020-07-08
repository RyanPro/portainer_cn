package endpointedge

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"

	"github.com/gorilla/mux"
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
)

// Handler is the HTTP handler used to handle edge endpoint operations.
type Handler struct {
	*mux.Router
	requestBouncer   *security.RequestBouncer
	EndpointService  portainer.EndpointService
	EdgeStackService portainer.EdgeStackService
	FileService      portainer.FileService
}

// NewHandler creates a handler to manage endpoint operations.
func NewHandler(bouncer *security.RequestBouncer) *Handler {
	h := &Handler{
		Router:         mux.NewRouter(),
		requestBouncer: bouncer,
	}

	h.Handle("/{id}/edge/stacks/{stackId}",
		bouncer.PublicAccess(httperror.LoggerHandler(h.endpointEdgeStackInspect))).Methods(http.MethodGet)

	return h
}
