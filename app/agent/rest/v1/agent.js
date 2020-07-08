angular.module('portainer.agent').factory('AgentVersion1', [
  '$resource',
  'API_ENDPOINT_ENDPOINTS',
  'EndpointProvider',
  function AgentFactory($resource, API_ENDPOINT_ENDPOINTS, EndpointProvider) {
    'use strict';
    return $resource(
      API_ENDPOINT_ENDPOINTS + '/:endpointId/docker/agents',
      {
        endpointId: EndpointProvider.endpointID,
      },
      {
        query: { method: 'GET', isArray: true },
      }
    );
  },
]);
