import getEndpointsTotalCount from './transform/getEndpointsTotalCount';

angular.module('portainer.app').factory('Endpoints', [
  '$resource',
  'API_ENDPOINT_ENDPOINTS',
  function EndpointsFactory($resource, API_ENDPOINT_ENDPOINTS) {
    'use strict';
    return $resource(
      API_ENDPOINT_ENDPOINTS + '/:id/:action',
      {},
      {
        query: {
          method: 'GET',
          params: { start: '@start', limit: '@limit', search: '@search', groupId: '@groupId' },
          transformResponse: getEndpointsTotalCount,
        },
        get: { method: 'GET', params: { id: '@id' } },
        update: { method: 'PUT', params: { id: '@id' } },
        updateAccess: { method: 'PUT', params: { id: '@id', action: 'access' } },
        remove: { method: 'DELETE', params: { id: '@id' } },
        snapshots: { method: 'POST', params: { action: 'snapshot' } },
        snapshot: { method: 'POST', params: { id: '@id', action: 'snapshot' } },
        executeJob: { method: 'POST', ignoreLoadingBar: true, params: { id: '@id', action: 'job' } },
        status: { method: 'GET', params: { id: '@id', action: 'status' } },
      }
    );
  },
]);
