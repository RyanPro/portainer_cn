package endpointproxy

// TODO: legacy extension management

import (
	"strconv"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/portainer/api"

	"net/http"
)

func (handler *Handler) proxyRequestsToStoridgeAPI(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	err = handler.requestBouncer.AuthorizedEndpointOperation(r, endpoint, false)
	if err != nil {
		return &httperror.HandlerError{http.StatusForbidden, "Permission denied to access endpoint", err}
	}

	var storidgeExtension *portainer.EndpointExtension
	for _, extension := range endpoint.Extensions {
		if extension.Type == portainer.StoridgeEndpointExtension {
			storidgeExtension = &extension
		}
	}

	if storidgeExtension == nil {
		return &httperror.HandlerError{http.StatusServiceUnavailable, "Storidge extension not supported on this endpoint", portainer.ErrEndpointExtensionNotSupported}
	}

	proxyExtensionKey := strconv.Itoa(endpointID) + "_" + strconv.Itoa(int(portainer.StoridgeEndpointExtension)) + "_" + storidgeExtension.URL

	var proxy http.Handler
	proxy = handler.ProxyManager.GetLegacyExtensionProxy(proxyExtensionKey)
	if proxy == nil {
		proxy, err = handler.ProxyManager.CreateLegacyExtensionProxy(proxyExtensionKey, storidgeExtension.URL)
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to create extension proxy", err}
		}
	}

	id := strconv.Itoa(endpointID)
	http.StripPrefix("/"+id+"/storidge", proxy).ServeHTTP(w, r)
	return nil
}
