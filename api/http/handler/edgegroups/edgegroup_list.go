package edgegroups

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
)

type decoratedEdgeGroup struct {
	portainer.EdgeGroup
	HasEdgeStack bool `json:"HasEdgeStack"`
}

func (handler *Handler) edgeGroupList(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	edgeGroups, err := handler.EdgeGroupService.EdgeGroups()
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve Edge groups from the database", err}
	}

	edgeStacks, err := handler.EdgeStackService.EdgeStacks()
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve Edge stacks from the database", err}
	}

	usedEdgeGroups := make(map[portainer.EdgeGroupID]bool)

	for _, stack := range edgeStacks {
		for _, groupID := range stack.EdgeGroups {
			usedEdgeGroups[groupID] = true
		}
	}

	decoratedEdgeGroups := []decoratedEdgeGroup{}
	for _, orgEdgeGroup := range edgeGroups {
		edgeGroup := decoratedEdgeGroup{
			EdgeGroup: orgEdgeGroup,
		}
		if edgeGroup.Dynamic {
			endpoints, err := handler.getEndpointsByTags(edgeGroup.TagIDs, edgeGroup.PartialMatch)
			if err != nil {
				return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve endpoints and endpoint groups for Edge group", err}
			}

			edgeGroup.Endpoints = endpoints
		}

		edgeGroup.HasEdgeStack = usedEdgeGroups[edgeGroup.ID]

		decoratedEdgeGroups = append(decoratedEdgeGroups, edgeGroup)
	}

	return response.JSON(w, decoratedEdgeGroups)
}
