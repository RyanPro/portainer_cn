package settings

import (
	"net/http"

	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/libhttp/response"
	"github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/filesystem"
)

type settingsLDAPCheckPayload struct {
	LDAPSettings portainer.LDAPSettings
}

func (payload *settingsLDAPCheckPayload) Validate(r *http.Request) error {
	return nil
}

// PUT request on /settings/ldap/check
func (handler *Handler) settingsLDAPCheck(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	var payload settingsLDAPCheckPayload
	err := request.DecodeAndValidateJSONPayload(r, &payload)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid request payload", err}
	}

	if (payload.LDAPSettings.TLSConfig.TLS || payload.LDAPSettings.StartTLS) && !payload.LDAPSettings.TLSConfig.TLSSkipVerify {
		caCertPath, _ := handler.FileService.GetPathForTLSFile(filesystem.LDAPStorePath, portainer.TLSFileCA)
		payload.LDAPSettings.TLSConfig.TLSCACertPath = caCertPath
	}

	err = handler.LDAPService.TestConnectivity(&payload.LDAPSettings)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to connect to LDAP server", err}
	}

	return response.Empty(w)
}
