package tags

import (
	"net/http"

	"github.com/gorilla/mux"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
)

// Handler is the HTTP handler used to handle tag operations.
type Handler struct {
	*mux.Router
	TagService              portainer.TagService
	EdgeGroupService        portainer.EdgeGroupService
	EdgeStackService        portainer.EdgeStackService
	EndpointService         portainer.EndpointService
	EndpointGroupService    portainer.EndpointGroupService
	EndpointRelationService portainer.EndpointRelationService
}

// NewHandler creates a handler to manage tag operations.
func NewHandler(bouncer *security.RequestBouncer) *Handler {
	h := &Handler{
		Router: mux.NewRouter(),
	}
	h.Handle("/tags",
		bouncer.AdminAccess(httperror.LoggerHandler(h.tagCreate))).Methods(http.MethodPost)
	h.Handle("/tags",
		bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.tagList))).Methods(http.MethodGet)
	h.Handle("/tags/{id}",
		bouncer.AdminAccess(httperror.LoggerHandler(h.tagDelete))).Methods(http.MethodDelete)

	return h
}
