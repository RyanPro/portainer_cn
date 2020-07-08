package resourcecontrols

import (
	"errors"
	"net/http"

	"github.com/asaskevich/govalidator"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
)

type resourceControlCreatePayload struct {
	ResourceID         string
	Type               string
	Public             bool
	AdministratorsOnly bool
	Users              []int
	Teams              []int
	SubResourceIDs     []string
}

func (payload *resourceControlCreatePayload) Validate(r *http.Request) error {
	if govalidator.IsNull(payload.ResourceID) {
		return errors.New("invalid payload: invalid resource identifier")
	}

	if govalidator.IsNull(payload.Type) {
		return errors.New("invalid payload: invalid type")
	}

	if len(payload.Users) == 0 && len(payload.Teams) == 0 && !payload.Public && !payload.AdministratorsOnly {
		return errors.New("invalid payload: must specify Users, Teams, Public or AdministratorsOnly")
	}

	if payload.Public && payload.AdministratorsOnly {
		return errors.New("invalid payload: cannot set both public and administrators only flags to true")
	}
	return nil
}

// POST request on /api/resource_controls
func (handler *Handler) resourceControlCreate(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	var payload resourceControlCreatePayload
	err := request.DecodeAndValidateJSONPayload(r, &payload)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", err}
	}

	var resourceControlType portainer.ResourceControlType
	switch payload.Type {
	case "container":
		resourceControlType = portainer.ContainerResourceControl
	case "service":
		resourceControlType = portainer.ServiceResourceControl
	case "volume":
		resourceControlType = portainer.VolumeResourceControl
	case "network":
		resourceControlType = portainer.NetworkResourceControl
	case "secret":
		resourceControlType = portainer.SecretResourceControl
	case "stack":
		resourceControlType = portainer.StackResourceControl
	case "config":
		resourceControlType = portainer.ConfigResourceControl
	default:
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid type value. Value must be one of: container, service, volume, network, secret, stack or config", portainer.ErrInvalidResourceControlType}
	}

	rc, err := handler.ResourceControlService.ResourceControlByResourceIDAndType(payload.ResourceID, resourceControlType)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to retrieve resource controls from the database", err}
	}
	if rc != nil {
		return &httperror.HandlerError{http.StatusConflict, "A resource control is already associated to this resource", portainer.ErrResourceControlAlreadyExists}
	}

	var userAccesses = make([]portainer.UserResourceAccess, 0)
	for _, v := range payload.Users {
		userAccess := portainer.UserResourceAccess{
			UserID:      portainer.UserID(v),
			AccessLevel: portainer.ReadWriteAccessLevel,
		}
		userAccesses = append(userAccesses, userAccess)
	}

	var teamAccesses = make([]portainer.TeamResourceAccess, 0)
	for _, v := range payload.Teams {
		teamAccess := portainer.TeamResourceAccess{
			TeamID:      portainer.TeamID(v),
			AccessLevel: portainer.ReadWriteAccessLevel,
		}
		teamAccesses = append(teamAccesses, teamAccess)
	}

	resourceControl := portainer.ResourceControl{
		ResourceID:         payload.ResourceID,
		SubResourceIDs:     payload.SubResourceIDs,
		Type:               resourceControlType,
		Public:             payload.Public,
		AdministratorsOnly: payload.AdministratorsOnly,
		UserAccesses:       userAccesses,
		TeamAccesses:       teamAccesses,
	}

	err = handler.ResourceControlService.CreateResourceControl(&resourceControl)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to persist the resource control inside the database", err}
	}

	return response.JSON(w, resourceControl)
}
