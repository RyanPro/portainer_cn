package endpoints

// TODO: legacy extension management

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
)

// DELETE request on /api/endpoints/:id/extensions/:extensionType
func (handler *Handler) endpointExtensionRemove(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	endpointID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid endpoint identifier route variable", err}
	}

	endpoint, err := handler.EndpointService.Endpoint(portainer.EndpointID(endpointID))
	if err == portainer.ErrObjectNotFound {
		return &httperror.HandlerError{http.StatusNotFound, "Unable to find an endpoint with the specified identifier inside the database", err}
	} else if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find an endpoint with the specified identifier inside the database", err}
	}

	extensionType, err := request.RetrieveNumericRouteVariableValue(r, "extensionType")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid extension type route variable", err}
	}

	for idx, ext := range endpoint.Extensions {
		if ext.Type == portainer.EndpointExtensionType(extensionType) {
			endpoint.Extensions = append(endpoint.Extensions[:idx], endpoint.Extensions[idx+1:]...)
		}
	}

	err = handler.EndpointService.UpdateEndpoint(endpoint.ID, endpoint)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist endpoint changes inside the database", err}
	}

	return response.Empty(w)
}
