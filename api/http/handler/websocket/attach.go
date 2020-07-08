package websocket

import (
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/websocket"
	httperror "github.com/portainer/libhttp/error"
	"github.com/portainer/libhttp/request"
	"github.com/portainer/portainer/api"
)

// websocketAttach handles GET requests on /websocket/attach?id=<attachID>&endpointId=<endpointID>&nodeName=<nodeName>&token=<token>
// If the nodeName query parameter is present, the request will be proxied to the underlying agent endpoint.
// If the nodeName query parameter is not specified, the request will be upgraded to the websocket protocol and
// an AttachStart operation HTTP request will be created and hijacked.
// Authentication and access is controled via the mandatory token query parameter.
func (handler *Handler) websocketAttach(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	attachID, err := request.RetrieveQueryParameter(r, "id", false)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid query parameter: id", err}
	}
	if !govalidator.IsHexadecimal(attachID) {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid query parameter: id (must be hexadecimal identifier)", err}
	}

	endpointID, err := request.RetrieveNumericQueryParameter(r, "endpointId", false)
	if err != nil {
		return &httperror.HandlerError{http.StatusBadRequest, "Invalid query parameter: endpointId", err}
	}

	endpoint, err := handler.EndpointService.Endpoint(portainer.EndpointID(endpointID))
	if err == portainer.ErrObjectNotFound {
		return &httperror.HandlerError{http.StatusNotFound, "Unable to find the endpoint associated to the stack inside the database", err}
	} else if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "Unable to find the endpoint associated to the stack inside the database", err}
	}

	err = handler.requestBouncer.AuthorizedEndpointOperation(r, endpoint, true)
	if err != nil {
		return &httperror.HandlerError{http.StatusForbidden, "Permission denied to access endpoint", err}
	}

	params := &webSocketRequestParams{
		endpoint: endpoint,
		ID:       attachID,
		nodeName: r.FormValue("nodeName"),
	}

	err = handler.handleAttachRequest(w, r, params)
	if err != nil {
		return &httperror.HandlerError{http.StatusInternalServerError, "An error occured during websocket attach operation", err}
	}

	return nil
}

func (handler *Handler) handleAttachRequest(w http.ResponseWriter, r *http.Request, params *webSocketRequestParams) error {

	r.Header.Del("Origin")

	if params.endpoint.Type == portainer.AgentOnDockerEnvironment {
		return handler.proxyAgentWebsocketRequest(w, r, params)
	} else if params.endpoint.Type == portainer.EdgeAgentEnvironment {
		return handler.proxyEdgeAgentWebsocketRequest(w, r, params)
	}

	websocketConn, err := handler.connectionUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer websocketConn.Close()

	return hijackAttachStartOperation(websocketConn, params.endpoint, params.ID)
}

func hijackAttachStartOperation(websocketConn *websocket.Conn, endpoint *portainer.Endpoint, attachID string) error {
	dial, err := initDial(endpoint)
	if err != nil {
		return err
	}

	// When we set up a TCP connection for hijack, there could be long periods
	// of inactivity (a long running command with no output) that in certain
	// network setups may cause ECONNTIMEOUT, leaving the client in an unknown
	// state. Setting TCP KeepAlive on the socket connection will prohibit
	// ECONNTIMEOUT unless the socket connection truly is broken
	if tcpConn, ok := dial.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	httpConn := httputil.NewClientConn(dial, nil)
	defer httpConn.Close()

	attachStartRequest, err := createAttachStartRequest(attachID)
	if err != nil {
		return err
	}

	err = hijackRequest(websocketConn, httpConn, attachStartRequest)
	if err != nil {
		return err
	}

	return nil
}

func createAttachStartRequest(attachID string) (*http.Request, error) {

	request, err := http.NewRequest("POST", "/containers/"+attachID+"/attach?stdin=1&stdout=1&stderr=1&stream=1", nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "tcp")

	return request, nil
}
