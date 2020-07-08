package endpoints

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
)

type stackStatusResponse struct {
	ID      portainer.EdgeStackID
	Version int
}

type endpointStatusInspectResponse struct {
	Status          string                   `json:"status"`
	Port            int                      `json:"port"`
	Schedules       []portainer.EdgeSchedule `json:"schedules"`
	CheckinInterval int                      `json:"checkin"`
	Credentials     string                   `json:"credentials"`
	Stacks          []stackStatusResponse    `json:"stacks"`
}

// GET request on /api/endpoints/:id/status
func (handler *Handler) endpointStatusInspect(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
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

	err = handler.requestBouncer.AuthorizedEdgeEndpointOperation(r, endpoint)
	if err != nil {
		return &httperror.HandlerError{http.StatusForbidden, "Permission denied to access endpoint", err}
	}

	if endpoint.EdgeID == "" {
		edgeIdentifier := r.Header.Get(portainer.PortainerAgentEdgeIDHeader)

		endpoint.EdgeID = edgeIdentifier

		err := handler.EndpointService.UpdateEndpoint(endpoint.ID, endpoint)
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to Unable to persist endpoint changes inside the database", err}
		}
	}

	settings, err := handler.SettingsService.Settings()
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve settings from the database", err}
	}

	tunnel := handler.ReverseTunnelService.GetTunnelDetails(endpoint.ID)

	statusResponse := endpointStatusInspectResponse{
		Status:          tunnel.Status,
		Port:            tunnel.Port,
		Schedules:       tunnel.Schedules,
		CheckinInterval: settings.EdgeAgentCheckinInterval,
		Credentials:     tunnel.Credentials,
	}

	if tunnel.Status == portainer.EdgeAgentManagementRequired {
		handler.ReverseTunnelService.SetTunnelStatusToActive(endpoint.ID)
	}

	relation, err := handler.EndpointRelationService.EndpointRelation(endpoint.ID)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve relation object from the database", err}
	}

	edgeStacksStatus := []stackStatusResponse{}
	for stackID := range relation.EdgeStacks {
		stack, err := handler.EdgeStackService.EdgeStack(stackID)
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve edge stack from the database", err}
		}

		stackStatus := stackStatusResponse{
			ID:      stack.ID,
			Version: stack.Version,
		}

		edgeStacksStatus = append(edgeStacksStatus, stackStatus)
	}

	statusResponse.Stacks = edgeStacksStatus

	return response.JSON(w, statusResponse)
}
