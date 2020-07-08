package edgegroups

import (
	"net/http"

	"github.com/gorilla/mux"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
)

// Handler is the HTTP handler used to handle endpoint group operations.
type Handler struct {
	*mux.Router
	EdgeGroupService        portainer.EdgeGroupService
	EdgeStackService        portainer.EdgeStackService
	EndpointService         portainer.EndpointService
	EndpointGroupService    portainer.EndpointGroupService
	EndpointRelationService portainer.EndpointRelationService
	TagService              portainer.TagService
}

// NewHandler creates a handler to manage endpoint group operations.
func NewHandler(bouncer *security.RequestBouncer) *Handler {
	h := &Handler{
		Router: mux.NewRouter(),
	}
	h.Handle("/edge_groups",
		bouncer.AdminAccess(httperror.LoggerHandler(h.edgeGroupCreate))).Methods(http.MethodPost)
	h.Handle("/edge_groups",
		bouncer.AdminAccess(httperror.LoggerHandler(h.edgeGroupList))).Methods(http.MethodGet)
	h.Handle("/edge_groups/{id}",
		bouncer.AdminAccess(httperror.LoggerHandler(h.edgeGroupInspect))).Methods(http.MethodGet)
	h.Handle("/edge_groups/{id}",
		bouncer.AdminAccess(httperror.LoggerHandler(h.edgeGroupUpdate))).Methods(http.MethodPut)
	h.Handle("/edge_groups/{id}",
		bouncer.AdminAccess(httperror.LoggerHandler(h.edgeGroupDelete))).Methods(http.MethodDelete)
	return h
}
