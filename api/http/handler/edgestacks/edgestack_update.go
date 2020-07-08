package edgestacks

import (
	"net/http"
	"strconv"

	"github.com/asaskevich/govalidator"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
)

type updateEdgeStackPayload struct {
	StackFileContent string
	Version          *int
	Prune            *bool
	EdgeGroups       []portainer.EdgeGroupID
}

func (payload *updateEdgeStackPayload) Validate(r *http.Request) error {
	if govalidator.IsNull(payload.StackFileContent) {
		return portainer.Error("Invalid stack file content")
	}
	if payload.EdgeGroups != nil && len(payload.EdgeGroups) == 0 {
		return portainer.Error("Edge Groups are mandatory for an Edge stack")
	}
	return nil
}

func (handler *Handler) edgeStackUpdate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	stackID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid stack identifier route variable", err}
	}

	stack, err := handler.EdgeStackService.EdgeStack(portainer.EdgeStackID(stackID))
	if err == portainer.ErrObjectNotFound {
		return &httperror.HandlerError{http.StatusNotFound, "Unable to find a stack with the specified identifier inside the database", err}
	} else if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find a stack with the specified identifier inside the database", err}
	}

	var payload updateEdgeStackPayload
	err = request.DecodeAndValidateJSONPayload(r, &payload)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", err}
	}

	if payload.EdgeGroups != nil {
		endpoints, err := handler.EndpointService.Endpoints()
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve endpoints from database", err}
		}

		endpointGroups, err := handler.EndpointGroupService.EndpointGroups()
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve endpoint groups from database", err}
		}

		edgeGroups, err := handler.EdgeGroupService.EdgeGroups()
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve edge groups from database", err}
		}

		oldRelated, err := portainer.EdgeStackRelatedEndpoints(stack.EdgeGroups, endpoints, endpointGroups, edgeGroups)
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve edge stack related endpoints from database", err}
		}

		newRelated, err := portainer.EdgeStackRelatedEndpoints(payload.EdgeGroups, endpoints, endpointGroups, edgeGroups)
		if err != nil {
			return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve edge stack related endpoints from database", err}
		}

		oldRelatedSet := EndpointSet(oldRelated)
		newRelatedSet := EndpointSet(newRelated)

		endpointsToRemove := map[portainer.EndpointID]bool{}
		for endpointID := range oldRelatedSet {
			if !newRelatedSet[endpointID] {
				endpointsToRemove[endpointID] = true
			}
		}

		for endpointID := range endpointsToRemove {
			relation, err := handler.EndpointRelationService.EndpointRelation(endpointID)
			if err != nil {
				return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find endpoint relation in database", err}
			}

			delete(relation.EdgeStacks, stack.ID)

			err = handler.EndpointRelationService.UpdateEndpointRelation(endpointID, relation)
			if err != nil {
				return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist endpoint relation in database", err}
			}
		}

		endpointsToAdd := map[portainer.EndpointID]bool{}
		for endpointID := range newRelatedSet {
			if !oldRelatedSet[endpointID] {
				endpointsToAdd[endpointID] = true
			}
		}

		for endpointID := range endpointsToAdd {
			relation, err := handler.EndpointRelationService.EndpointRelation(endpointID)
			if err != nil {
				return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find endpoint relation in database", err}
			}

			relation.EdgeStacks[stack.ID] = true

			err = handler.EndpointRelationService.UpdateEndpointRelation(endpointID, relation)
			if err != nil {
				return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist endpoint relation in database", err}
			}
		}

		stack.EdgeGroups = payload.EdgeGroups

	}

	if payload.Prune != nil {
		stack.Prune = *payload.Prune
	}

	stackFolder := strconv.Itoa(int(stack.ID))
	_, err = handler.FileService.StoreEdgeStackFileFromBytes(stackFolder, stack.EntryPoint, []byte(payload.StackFileContent))
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist updated Compose file on disk", err}
	}

	if payload.Version != nil && *payload.Version != stack.Version {
		stack.Version = *payload.Version
		stack.Status = map[portainer.EndpointID]portainer.EdgeStackStatus{}
	}

	err = handler.EdgeStackService.UpdateEdgeStack(stack.ID, stack)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist the stack changes inside the database", err}
	}

	return response.JSON(w, stack)
}

func EndpointSet(endpointIDs []portainer.EndpointID) map[portainer.EndpointID]bool {
	set := map[portainer.EndpointID]bool{}

	for _, endpointID := range endpointIDs {
		set[endpointID] = true
	}

	return set
}
