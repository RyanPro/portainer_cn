import $ from 'jquery';
import '@babel/polyfill';

angular.module('portainer').run([
  '$rootScope',
  '$state',
  '$interval',
  'LocalStorage',
  'EndpointProvider',
  'SystemService',
  'cfpLoadingBar',
  '$transitions',
  'HttpRequestHelper',
  function ($rootScope, $state, $interval, LocalStorage, EndpointProvider, SystemService, cfpLoadingBar, $transitions, HttpRequestHelper) {
    'use strict';

    EndpointProvider.initialize();

    $rootScope.$state = $state;

    // Workaround to prevent the loading bar from going backward
    // https://github.com/chieffancypants/angular-loading-bar/issues/273
    var originalSet = cfpLoadingBar.set;
    cfpLoadingBar.set = function overrideSet(n) {
      if (n > cfpLoadingBar.status()) {
        originalSet.apply(cfpLoadingBar, arguments);
      }
    };

    $transitions.onBefore({}, function () {
      HttpRequestHelper.resetAgentHeaders();
    });

    $state.defaultErrorHandler(function () {
      // Do not log transitionTo errors
    });

    // Keep-alive Edge endpoints by sending a ping request every minute
    $interval(function () {
      ping(EndpointProvider, SystemService);
    }, 60 * 1000);

    $(document).ajaxSend(function (event, jqXhr, jqOpts) {
      const type = jqOpts.type === 'POST' || jqOpts.type === 'PUT' || jqOpts.type === 'PATCH';
      const hasNoContentType = jqOpts.contentType !== 'application/json' && jqOpts.headers && !jqOpts.headers['Content-Type'];
      if (type && hasNoContentType) {
        jqXhr.setRequestHeader('Content-Type', 'application/json');
      }
      jqXhr.setRequestHeader('Authorization', 'Bearer ' + LocalStorage.getJWT());
    });
  },
]);

function ping(EndpointProvider, SystemService) {
  let endpoint = EndpointProvider.currentEndpoint();
  if (endpoint !== undefined && endpoint.Type === 4) {
    SystemService.ping(endpoint.Id);
  }
}
