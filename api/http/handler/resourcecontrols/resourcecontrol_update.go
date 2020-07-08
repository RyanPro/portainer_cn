package resourcecontrols

import (
	"errors"
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/security"
)

type resourceControlUpdatePayload struct {
	Public             bool
	Users              []int
	Teams              []int
	AdministratorsOnly bool
}

func (payload *resourceControlUpdatePayload) Validate(r *http.Request) error {
	if len(payload.Users) == 0 && len(payload.Teams) == 0 && !payload.Public && !payload.AdministratorsOnly {
		return errors.New("invalid payload: must specify Users, Teams, Public or AdministratorsOnly")
	}

	if payload.Public && payload.AdministratorsOnly {
		return errors.New("invalid payload: cannot set public and administrators only")
	}
	return nil
}

// PUT request on /api/resource_controls/:id
func (handler *Handler) resourceControlUpdate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	resourceControlID, err := request.RetrieveNumericRouteVariableValue(r, "id")
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid resource control identifier route variable", err}
	}

	var payload resourceControlUpdatePayload
	err = request.DecodeAndValidateJSONPayload(r, &payload)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", err}
	}

	resourceControl, err := handler.ResourceControlService.ResourceControl(portainer.ResourceControlID(resourceControlID))
	if err == portainer.ErrObjectNotFound {
		return &httperror.HandlerError{http.StatusNotFound, "Unable to find a resource control with the specified identifier inside the database", err}
	} else if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find a resource control with with the specified identifier inside the database", err}
	}

	securityContext, err := security.RetrieveRestrictedRequestContext(r)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve info from request context", err}
	}

	if !security.AuthorizedResourceControlAccess(resourceControl, securityContext) {
		return &httperror.HandlerError{http.StatusForbidden, "Permission denied to access the resource control", portainer.ErrResourceAccessDenied}
	}

	resourceControl.Public = payload.Public
	resourceControl.AdministratorsOnly = payload.AdministratorsOnly

	var userAccesses = make([]portainer.UserResourceAccess, 0)
	for _, v := range payload.Users {
		userAccess := portainer.UserResourceAccess{
			UserID:      portainer.UserID(v),
			AccessLevel: portainer.ReadWriteAccessLevel,
		}
		userAccesses = append(userAccesses, userAccess)
	}
	resourceControl.UserAccesses = userAccesses

	var teamAccesses = make([]portainer.TeamResourceAccess, 0)
	for _, v := range payload.Teams {
		teamAccess := portainer.TeamResourceAccess{
			TeamID:      portainer.TeamID(v),
			AccessLevel: portainer.ReadWriteAccessLevel,
		}
		teamAccesses = append(teamAccesses, teamAccess)
	}
	resourceControl.TeamAccesses = teamAccesses

	if !security.AuthorizedResourceControlUpdate(resourceControl, securityContext) {
		return &httperror.HandlerError{http.StatusForbidden, "Permission denied to update the resource control", portainer.ErrResourceAccessDenied}
	}

	err = handler.ResourceControlService.UpdateResourceControl(resourceControl.ID, resourceControl)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist resource control changes inside the database", err}
	}

	return response.JSON(w, resourceControl)
}
