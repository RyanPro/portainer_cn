import angular from 'angular';
import _ from 'lodash-es';

class EdgeGroupFormController {
  /* @ngInject */
  constructor(EndpointService, $async, $scope) {
    this.EndpointService = EndpointService;
    this.$async = $async;

    this.state = {
      available: {
        limit: '10',
        filter: '',
        pageNumber: 1,
        totalCount: 0,
      },
      associated: {
        limit: '10',
        filter: '',
        pageNumber: 1,
        totalCount: 0,
      },
    };

    this.endpoints = {
      associated: [],
      available: null,
    };

    this.associateEndpoint = this.associateEndpoint.bind(this);
    this.dissociateEndpoint = this.dissociateEndpoint.bind(this);
    this.getPaginatedEndpointsAsync = this.getPaginatedEndpointsAsync.bind(this);
    this.getPaginatedEndpoints = this.getPaginatedEndpoints.bind(this);

    $scope.$watch(
      () => this.model,
      () => {
        this.getPaginatedEndpoints(this.pageType, 'associated');
      },
      true
    );
  }

  associateEndpoint(endpoint) {
    if (!_.includes(this.model.Endpoints, endpoint.Id)) {
      this.endpoints.associated.push(endpoint);
      this.model.Endpoints.push(endpoint.Id);
      _.remove(this.endpoints.available, { Id: endpoint.Id });
    }
  }

  dissociateEndpoint(endpoint) {
    _.remove(this.endpoints.associated, { Id: endpoint.Id });
    _.remove(this.model.Endpoints, (id) => id === endpoint.Id);
    this.endpoints.available.push(endpoint);
  }

  getPaginatedEndpoints(pageType, tableType) {
    return this.$async(this.getPaginatedEndpointsAsync, pageType, tableType);
  }

  async getPaginatedEndpointsAsync(pageType, tableType) {
    const { pageNumber, limit, search } = this.state[tableType];
    const start = (pageNumber - 1) * limit + 1;
    const query = { search, type: 4 };
    if (tableType === 'associated') {
      if (this.model.Dynamic) {
        query.tagIds = this.model.TagIds;
        query.tagsPartialMatch = this.model.PartialMatch;
      } else {
        query.endpointIds = this.model.Endpoints;
      }
    }
    const response = await this.fetchEndpoints(start, limit, query);
    const totalCount = parseInt(response.totalCount, 10);
    this.endpoints[tableType] = response.value;
    this.state[tableType].totalCount = totalCount;

    if (tableType === 'available') {
      this.noEndpoints = totalCount === 0;
      this.endpoints[tableType] = _.filter(response.value, (endpoint) => !_.includes(this.model.Endpoints, endpoint.Id));
    }
  }

  fetchEndpoints(start, limit, query) {
    if (query.tagIds && !query.tagIds.length) {
      return { value: [], totalCount: 0 };
    }
    return this.EndpointService.endpoints(start, limit, query);
  }
}

angular.module('portainer.edge').controller('EdgeGroupFormController', EdgeGroupFormController);
export default EdgeGroupFormController;
