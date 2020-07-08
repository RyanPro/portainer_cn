package dockerhub

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/response"
)

// GET request on /api/dockerhub
func (handler *Handler) dockerhubInspect(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	dockerhub, err := handler.DockerHubService.DockerHub()
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve DockerHub details from the database", err}
	}

	hideFields(dockerhub)
	return response.JSON(w, dockerhub)
}
