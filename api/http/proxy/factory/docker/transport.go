package docker

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/portainer/portainer/api/docker"

	"github.com/docker/docker/client"
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/http/proxy/factory/responseutils"
	"github.com/portainer/portainer/api/http/security"
)

var apiVersionRe = regexp.MustCompile(`(/v[0-9]\.[0-9]*)?`)

type (
	// Transport is a custom transport for Docker API reverse proxy. It allows
	// interception of requests and rewriting of responses.
	Transport struct {
		HTTPTransport          *http.Transport
		endpoint               *portainer.Endpoint
		resourceControlService portainer.ResourceControlService
		userService            portainer.UserService
		teamService            portainer.TeamService
		teamMembershipService  portainer.TeamMembershipService
		registryService        portainer.RegistryService
		dockerHubService       portainer.DockerHubService
		settingsService        portainer.SettingsService
		signatureService       portainer.DigitalSignatureService
		reverseTunnelService   portainer.ReverseTunnelService
		extensionService       portainer.ExtensionService
		dockerClient           *client.Client
		dockerClientFactory    *docker.ClientFactory
	}

	// TransportParameters is used to create a new Transport
	TransportParameters struct {
		Endpoint               *portainer.Endpoint
		ResourceControlService portainer.ResourceControlService
		UserService            portainer.UserService
		TeamService            portainer.TeamService
		TeamMembershipService  portainer.TeamMembershipService
		RegistryService        portainer.RegistryService
		DockerHubService       portainer.DockerHubService
		SettingsService        portainer.SettingsService
		SignatureService       portainer.DigitalSignatureService
		ReverseTunnelService   portainer.ReverseTunnelService
		ExtensionService       portainer.ExtensionService
		DockerClientFactory    *docker.ClientFactory
	}

	restrictedDockerOperationContext struct {
		isAdmin                bool
		endpointResourceAccess bool
		userID                 portainer.UserID
		userTeamIDs            []portainer.TeamID
		resourceControls       []portainer.ResourceControl
	}

	operationExecutor struct {
		operationContext *restrictedDockerOperationContext
		labelBlackList   []portainer.Pair
	}
	restrictedOperationRequest func(*http.Response, *operationExecutor) error
	operationRequest           func(*http.Request) error
)

// NewTransport returns a pointer to a new Transport instance.
func NewTransport(parameters *TransportParameters, httpTransport *http.Transport) (*Transport, error) {
	dockerClient, err := parameters.DockerClientFactory.CreateClient(parameters.Endpoint, "")
	if err != nil {
		return nil, err
	}

	transport := &Transport{
		endpoint:               parameters.Endpoint,
		resourceControlService: parameters.ResourceControlService,
		userService:            parameters.UserService,
		teamService:            parameters.TeamService,
		teamMembershipService:  parameters.TeamMembershipService,
		registryService:        parameters.RegistryService,
		dockerHubService:       parameters.DockerHubService,
		settingsService:        parameters.SettingsService,
		signatureService:       parameters.SignatureService,
		reverseTunnelService:   parameters.ReverseTunnelService,
		extensionService:       parameters.ExtensionService,
		dockerClientFactory:    parameters.DockerClientFactory,
		HTTPTransport:          httpTransport,
		dockerClient:           dockerClient,
	}

	return transport, nil
}

// RoundTrip is the implementation of the the http.RoundTripper interface
func (transport *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport.ProxyDockerRequest(request)
}

// ProxyDockerRequest intercepts a Docker API request and apply logic based
// on the requested operation.
func (transport *Transport) ProxyDockerRequest(request *http.Request) (*http.Response, error) {
	requestPath := apiVersionRe.ReplaceAllString(request.URL.Path, "")
	request.URL.Path = requestPath

	if transport.endpoint.Type == portainer.AgentOnDockerEnvironment {
		signature, err := transport.signatureService.CreateSignature(portainer.PortainerAgentSignatureMessage)
		if err != nil {
			return nil, err
		}

		request.Header.Set(portainer.PortainerAgentPublicKeyHeader, transport.signatureService.EncodedPublicKey())
		request.Header.Set(portainer.PortainerAgentSignatureHeader, signature)
	}

	switch {
	case strings.HasPrefix(requestPath, "/configs"):
		return transport.proxyConfigRequest(request)
	case strings.HasPrefix(requestPath, "/containers"):
		return transport.proxyContainerRequest(request)
	case strings.HasPrefix(requestPath, "/services"):
		return transport.proxyServiceRequest(request)
	case strings.HasPrefix(requestPath, "/volumes"):
		return transport.proxyVolumeRequest(request)
	case strings.HasPrefix(requestPath, "/networks"):
		return transport.proxyNetworkRequest(request)
	case strings.HasPrefix(requestPath, "/secrets"):
		return transport.proxySecretRequest(request)
	case strings.HasPrefix(requestPath, "/swarm"):
		return transport.proxySwarmRequest(request)
	case strings.HasPrefix(requestPath, "/nodes"):
		return transport.proxyNodeRequest(request)
	case strings.HasPrefix(requestPath, "/tasks"):
		return transport.proxyTaskRequest(request)
	case strings.HasPrefix(requestPath, "/build"):
		return transport.proxyBuildRequest(request)
	case strings.HasPrefix(requestPath, "/images"):
		return transport.proxyImageRequest(request)
	case strings.HasPrefix(requestPath, "/v2"):
		return transport.proxyAgentRequest(request)
	default:
		return transport.executeDockerRequest(request)
	}
}

func (transport *Transport) executeDockerRequest(request *http.Request) (*http.Response, error) {
	response, err := transport.HTTPTransport.RoundTrip(request)

	if transport.endpoint.Type != portainer.EdgeAgentEnvironment {
		return response, err
	}

	if err == nil {
		transport.reverseTunnelService.SetTunnelStatusToActive(transport.endpoint.ID)
	} else {
		transport.reverseTunnelService.SetTunnelStatusToIdle(transport.endpoint.ID)
	}

	return response, err
}

func (transport *Transport) proxyAgentRequest(r *http.Request) (*http.Response, error) {
	requestPath := strings.TrimPrefix(r.URL.Path, "/v2")

	switch {
	case strings.HasPrefix(requestPath, "/browse"):
		// host file browser request
		volumeIDParameter, found := r.URL.Query()["volumeID"]
		if !found || len(volumeIDParameter) < 1 {
			return transport.administratorOperation(r)
		}

		// volume browser request
		return transport.restrictedResourceOperation(r, volumeIDParameter[0], portainer.VolumeResourceControl, true)
	}

	return transport.executeDockerRequest(r)
}

func (transport *Transport) proxyConfigRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/configs/create":
		return transport.decorateGenericResourceCreationOperation(request, configObjectIdentifier, portainer.ConfigResourceControl)

	case "/configs":
		return transport.rewriteOperation(request, transport.configListOperation)

	default:
		// assume /configs/{id}
		configID := path.Base(requestPath)

		if request.Method == http.MethodGet {
			return transport.rewriteOperation(request, transport.configInspectOperation)
		} else if request.Method == http.MethodDelete {
			return transport.executeGenericResourceDeletionOperation(request, configID, portainer.ConfigResourceControl)
		}

		return transport.restrictedResourceOperation(request, configID, portainer.ConfigResourceControl, false)
	}
}

func (transport *Transport) proxyContainerRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/containers/create":
		return transport.decorateGenericResourceCreationOperation(request, containerObjectIdentifier, portainer.ContainerResourceControl)

	case "/containers/prune":
		return transport.administratorOperation(request)

	case "/containers/json":
		return transport.rewriteOperationWithLabelFiltering(request, transport.containerListOperation)

	default:
		// This section assumes /containers/**
		if match, _ := path.Match("/containers/*/*", requestPath); match {
			// Handle /containers/{id}/{action} requests
			containerID := path.Base(path.Dir(requestPath))
			action := path.Base(requestPath)

			if action == "json" {
				return transport.rewriteOperation(request, transport.containerInspectOperation)
			}
			return transport.restrictedResourceOperation(request, containerID, portainer.ContainerResourceControl, false)
		} else if match, _ := path.Match("/containers/*", requestPath); match {
			// Handle /containers/{id} requests
			containerID := path.Base(requestPath)

			if request.Method == http.MethodDelete {
				return transport.executeGenericResourceDeletionOperation(request, containerID, portainer.ContainerResourceControl)
			}

			return transport.restrictedResourceOperation(request, containerID, portainer.ContainerResourceControl, false)
		}
		return transport.executeDockerRequest(request)
	}
}

func (transport *Transport) proxyServiceRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/services/create":
		return transport.replaceRegistryAuthenticationHeader(request)

	case "/services":
		return transport.rewriteOperation(request, transport.serviceListOperation)

	default:
		// This section assumes /services/**
		if match, _ := path.Match("/services/*/*", requestPath); match {
			// Handle /services/{id}/{action} requests
			serviceID := path.Base(path.Dir(requestPath))
			return transport.restrictedResourceOperation(request, serviceID, portainer.ServiceResourceControl, false)
		} else if match, _ := path.Match("/services/*", requestPath); match {
			// Handle /services/{id} requests
			serviceID := path.Base(requestPath)

			switch request.Method {
			case http.MethodGet:
				return transport.rewriteOperation(request, transport.serviceInspectOperation)
			case http.MethodDelete:
				return transport.executeGenericResourceDeletionOperation(request, serviceID, portainer.ServiceResourceControl)
			}
			return transport.restrictedResourceOperation(request, serviceID, portainer.ServiceResourceControl, false)
		}
		return transport.executeDockerRequest(request)
	}
}

func (transport *Transport) proxyVolumeRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/volumes/create":
		return transport.decorateVolumeResourceCreationOperation(request, volumeObjectIdentifier, portainer.VolumeResourceControl)

	case "/volumes/prune":
		return transport.administratorOperation(request)

	case "/volumes":
		return transport.rewriteOperation(request, transport.volumeListOperation)

	default:
		// assume /volumes/{name}
		volumeID := path.Base(requestPath)

		if request.Method == http.MethodGet {
			return transport.rewriteOperation(request, transport.volumeInspectOperation)
		} else if request.Method == http.MethodDelete {
			return transport.executeGenericResourceDeletionOperation(request, volumeID, portainer.VolumeResourceControl)
		}
		return transport.restrictedResourceOperation(request, volumeID, portainer.VolumeResourceControl, false)
	}
}

func (transport *Transport) proxyNetworkRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/networks/create":
		return transport.decorateGenericResourceCreationOperation(request, networkObjectIdentifier, portainer.NetworkResourceControl)

	case "/networks":
		return transport.rewriteOperation(request, transport.networkListOperation)

	default:
		// assume /networks/{id}
		networkID := path.Base(requestPath)

		if request.Method == http.MethodGet {
			return transport.rewriteOperation(request, transport.networkInspectOperation)
		} else if request.Method == http.MethodDelete {
			return transport.executeGenericResourceDeletionOperation(request, networkID, portainer.NetworkResourceControl)
		}
		return transport.restrictedResourceOperation(request, networkID, portainer.NetworkResourceControl, false)
	}
}

func (transport *Transport) proxySecretRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/secrets/create":
		return transport.decorateGenericResourceCreationOperation(request, secretObjectIdentifier, portainer.SecretResourceControl)

	case "/secrets":
		return transport.rewriteOperation(request, transport.secretListOperation)

	default:
		// assume /secrets/{id}
		secretID := path.Base(requestPath)

		if request.Method == http.MethodGet {
			return transport.rewriteOperation(request, transport.secretInspectOperation)
		} else if request.Method == http.MethodDelete {
			return transport.executeGenericResourceDeletionOperation(request, secretID, portainer.SecretResourceControl)
		}
		return transport.restrictedResourceOperation(request, secretID, portainer.SecretResourceControl, false)
	}
}

func (transport *Transport) proxyNodeRequest(request *http.Request) (*http.Response, error) {
	requestPath := request.URL.Path

	// assume /nodes/{id}
	if path.Base(requestPath) != "nodes" {
		return transport.administratorOperation(request)
	}

	return transport.executeDockerRequest(request)
}

func (transport *Transport) proxySwarmRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/swarm":
		return transport.rewriteOperation(request, swarmInspectOperation)
	default:
		// assume /swarm/{action}
		return transport.administratorOperation(request)
	}
}

func (transport *Transport) proxyTaskRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/tasks":
		return transport.rewriteOperation(request, transport.taskListOperation)
	default:
		// assume /tasks/{id}
		return transport.executeDockerRequest(request)
	}
}

func (transport *Transport) proxyBuildRequest(request *http.Request) (*http.Response, error) {
	return transport.interceptAndRewriteRequest(request, buildOperation)
}

func (transport *Transport) proxyImageRequest(request *http.Request) (*http.Response, error) {
	switch requestPath := request.URL.Path; requestPath {
	case "/images/create":
		return transport.replaceRegistryAuthenticationHeader(request)
	default:
		if path.Base(requestPath) == "push" && request.Method == http.MethodPost {
			return transport.replaceRegistryAuthenticationHeader(request)
		}
		return transport.executeDockerRequest(request)
	}
}

func (transport *Transport) replaceRegistryAuthenticationHeader(request *http.Request) (*http.Response, error) {
	accessContext, err := transport.createRegistryAccessContext(request)
	if err != nil {
		return nil, err
	}

	originalHeader := request.Header.Get("X-Registry-Auth")

	if originalHeader != "" {

		decodedHeaderData, err := base64.StdEncoding.DecodeString(originalHeader)
		if err != nil {
			return nil, err
		}

		var originalHeaderData registryAuthenticationHeader
		err = json.Unmarshal(decodedHeaderData, &originalHeaderData)
		if err != nil {
			return nil, err
		}

		authenticationHeader := createRegistryAuthenticationHeader(originalHeaderData.Serveraddress, accessContext)

		headerData, err := json.Marshal(authenticationHeader)
		if err != nil {
			return nil, err
		}

		header := base64.StdEncoding.EncodeToString(headerData)

		request.Header.Set("X-Registry-Auth", header)
	}

	return transport.decorateGenericResourceCreationOperation(request, serviceObjectIdentifier, portainer.ServiceResourceControl)
}

func (transport *Transport) restrictedResourceOperation(request *http.Request, resourceID string, resourceType portainer.ResourceControlType, volumeBrowseRestrictionCheck bool) (*http.Response, error) {
	var err error
	tokenData, err := security.RetrieveTokenData(request)
	if err != nil {
		return nil, err
	}

	if tokenData.Role != portainer.AdministratorRole {
		rbacExtension, err := transport.extensionService.Extension(portainer.RBACExtension)
		if err != nil && err != portainer.ErrObjectNotFound {
			return nil, err
		}

		user, err := transport.userService.User(tokenData.ID)
		if err != nil {
			return nil, err
		}

		if volumeBrowseRestrictionCheck {
			settings, err := transport.settingsService.Settings()
			if err != nil {
				return nil, err
			}

			if !settings.AllowVolumeBrowserForRegularUsers {
				if rbacExtension == nil {
					return responseutils.WriteAccessDeniedResponse()
				}

				// Return access denied for all roles except endpoint-administrator
				_, userCanBrowse := user.EndpointAuthorizations[transport.endpoint.ID][portainer.OperationDockerAgentBrowseList]
				if !userCanBrowse {
					return responseutils.WriteAccessDeniedResponse()
				}
			}
		}

		endpointResourceAccess := false
		_, ok := user.EndpointAuthorizations[transport.endpoint.ID][portainer.EndpointResourcesAccess]
		if ok {
			endpointResourceAccess = true
		}

		if rbacExtension != nil && endpointResourceAccess {
			return transport.executeDockerRequest(request)
		}

		teamMemberships, err := transport.teamMembershipService.TeamMembershipsByUserID(tokenData.ID)
		if err != nil {
			return nil, err
		}

		userTeamIDs := make([]portainer.TeamID, 0)
		for _, membership := range teamMemberships {
			userTeamIDs = append(userTeamIDs, membership.TeamID)
		}

		resourceControls, err := transport.resourceControlService.ResourceControls()
		if err != nil {
			return nil, err
		}

		resourceControl := portainer.GetResourceControlByResourceIDAndType(resourceID, resourceType, resourceControls)
		if resourceControl == nil {
			agentTargetHeader := request.Header.Get(portainer.PortainerAgentTargetHeader)

			// This resource was created outside of portainer,
			// is part of a Docker service or part of a Docker Swarm/Compose stack.
			inheritedResourceControl, err := transport.getInheritedResourceControlFromServiceOrStack(resourceID, agentTargetHeader, resourceType, resourceControls)
			if err != nil {
				return nil, err
			}

			if inheritedResourceControl == nil || !portainer.UserCanAccessResource(tokenData.ID, userTeamIDs, inheritedResourceControl) {
				return responseutils.WriteAccessDeniedResponse()
			}
		}

		if resourceControl != nil && !portainer.UserCanAccessResource(tokenData.ID, userTeamIDs, resourceControl) {
			return responseutils.WriteAccessDeniedResponse()
		}
	}

	return transport.executeDockerRequest(request)
}

// rewriteOperationWithLabelFiltering will create a new operation context with data that will be used
// to decorate the original request's response as well as retrieve all the black listed labels
// to filter the resources.
func (transport *Transport) rewriteOperationWithLabelFiltering(request *http.Request, operation restrictedOperationRequest) (*http.Response, error) {
	operationContext, err := transport.createOperationContext(request)
	if err != nil {
		return nil, err
	}

	settings, err := transport.settingsService.Settings()
	if err != nil {
		return nil, err
	}

	executor := &operationExecutor{
		operationContext: operationContext,
		labelBlackList:   settings.BlackListedLabels,
	}

	return transport.executeRequestAndRewriteResponse(request, operation, executor)
}

// rewriteOperation will create a new operation context with data that will be used
// to decorate the original request's response.
func (transport *Transport) rewriteOperation(request *http.Request, operation restrictedOperationRequest) (*http.Response, error) {
	operationContext, err := transport.createOperationContext(request)
	if err != nil {
		return nil, err
	}

	executor := &operationExecutor{
		operationContext: operationContext,
	}

	return transport.executeRequestAndRewriteResponse(request, operation, executor)
}

func (transport *Transport) interceptAndRewriteRequest(request *http.Request, operation operationRequest) (*http.Response, error) {
	err := operation(request)
	if err != nil {
		return nil, err
	}

	return transport.executeDockerRequest(request)
}

// decorateGenericResourceCreationResponse extracts the response as a JSON object, extracts the resource identifier from that object based
// on the resourceIdentifierAttribute parameter then generate a new resource control associated to that resource
// with a random token and rewrites the response by decorating the original response with a ResourceControl object.
// The generic Docker API response format is JSON object:
// https://docs.docker.com/engine/api/v1.37/#operation/ContainerCreate
// https://docs.docker.com/engine/api/v1.37/#operation/NetworkCreate
// https://docs.docker.com/engine/api/v1.37/#operation/VolumeCreate
// https://docs.docker.com/engine/api/v1.37/#operation/ServiceCreate
// https://docs.docker.com/engine/api/v1.37/#operation/SecretCreate
// https://docs.docker.com/engine/api/v1.37/#operation/ConfigCreate
func (transport *Transport) decorateGenericResourceCreationResponse(response *http.Response, resourceIdentifierAttribute string, resourceType portainer.ResourceControlType, userID portainer.UserID) error {
	responseObject, err := responseutils.GetResponseAsJSONOBject(response)
	if err != nil {
		return err
	}

	if responseObject[resourceIdentifierAttribute] == nil {
		log.Printf("[ERROR] [proxy,docker]")
		return errors.New("missing identifier in Docker resource creation response")
	}

	resourceID := responseObject[resourceIdentifierAttribute].(string)

	resourceControl, err := transport.createPrivateResourceControl(resourceID, resourceType, userID)
	if err != nil {
		return err
	}

	responseObject = decorateObject(responseObject, resourceControl)

	return responseutils.RewriteResponse(response, responseObject, http.StatusOK)
}

func (transport *Transport) decorateGenericResourceCreationOperation(request *http.Request, resourceIdentifierAttribute string, resourceType portainer.ResourceControlType) (*http.Response, error) {
	tokenData, err := security.RetrieveTokenData(request)
	if err != nil {
		return nil, err
	}

	response, err := transport.executeDockerRequest(request)
	if err != nil {
		return response, err
	}

	if response.StatusCode == http.StatusCreated {
		err = transport.decorateGenericResourceCreationResponse(response, resourceIdentifierAttribute, resourceType, tokenData.ID)
	}

	return response, err
}

func (transport *Transport) executeGenericResourceDeletionOperation(request *http.Request, resourceIdentifierAttribute string, resourceType portainer.ResourceControlType) (*http.Response, error) {
	response, err := transport.restrictedResourceOperation(request, resourceIdentifierAttribute, resourceType, false)
	if err != nil {
		return response, err
	}

	resourceControl, err := transport.resourceControlService.ResourceControlByResourceIDAndType(resourceIdentifierAttribute, resourceType)
	if err != nil {
		return response, err
	}

	if resourceControl != nil {
		err = transport.resourceControlService.DeleteResourceControl(resourceControl.ID)
		if err != nil {
			return response, err
		}
	}

	return response, err
}

func (transport *Transport) executeRequestAndRewriteResponse(request *http.Request, operation restrictedOperationRequest, executor *operationExecutor) (*http.Response, error) {
	response, err := transport.executeDockerRequest(request)
	if err != nil {
		return response, err
	}

	err = operation(response, executor)
	return response, err
}

// administratorOperation ensures that the user has administrator privileges
// before executing the original request.
func (transport *Transport) administratorOperation(request *http.Request) (*http.Response, error) {
	tokenData, err := security.RetrieveTokenData(request)
	if err != nil {
		return nil, err
	}

	if tokenData.Role != portainer.AdministratorRole {
		return responseutils.WriteAccessDeniedResponse()
	}

	return transport.executeDockerRequest(request)
}

func (transport *Transport) createRegistryAccessContext(request *http.Request) (*registryAccessContext, error) {
	tokenData, err := security.RetrieveTokenData(request)
	if err != nil {
		return nil, err
	}

	accessContext := &registryAccessContext{
		isAdmin: true,
		userID:  tokenData.ID,
	}

	hub, err := transport.dockerHubService.DockerHub()
	if err != nil {
		return nil, err
	}
	accessContext.dockerHub = hub

	registries, err := transport.registryService.Registries()
	if err != nil {
		return nil, err
	}
	accessContext.registries = registries

	if tokenData.Role != portainer.AdministratorRole {
		accessContext.isAdmin = false

		teamMemberships, err := transport.teamMembershipService.TeamMembershipsByUserID(tokenData.ID)
		if err != nil {
			return nil, err
		}

		accessContext.teamMemberships = teamMemberships
	}

	return accessContext, nil
}

func (transport *Transport) createOperationContext(request *http.Request) (*restrictedDockerOperationContext, error) {
	var err error
	tokenData, err := security.RetrieveTokenData(request)
	if err != nil {
		return nil, err
	}

	resourceControls, err := transport.resourceControlService.ResourceControls()
	if err != nil {
		return nil, err
	}

	operationContext := &restrictedDockerOperationContext{
		isAdmin:                true,
		userID:                 tokenData.ID,
		resourceControls:       resourceControls,
		endpointResourceAccess: false,
	}

	if tokenData.Role != portainer.AdministratorRole {
		operationContext.isAdmin = false

		user, err := transport.userService.User(operationContext.userID)
		if err != nil {
			return nil, err
		}

		_, ok := user.EndpointAuthorizations[transport.endpoint.ID][portainer.EndpointResourcesAccess]
		if ok {
			operationContext.endpointResourceAccess = true
		}

		teamMemberships, err := transport.teamMembershipService.TeamMembershipsByUserID(tokenData.ID)
		if err != nil {
			return nil, err
		}

		userTeamIDs := make([]portainer.TeamID, 0)
		for _, membership := range teamMemberships {
			userTeamIDs = append(userTeamIDs, membership.TeamID)
		}
		operationContext.userTeamIDs = userTeamIDs
	}

	return operationContext, nil
}
