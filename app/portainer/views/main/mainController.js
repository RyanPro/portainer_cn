angular.module('portainer.app').controller('MainController', [
  '$scope',
  '$cookieStore',
  'StateManager',
  'EndpointProvider',
  function ($scope, $cookieStore, StateManager, EndpointProvider) {
    /**
     * Sidebar Toggle & Cookie Control
     */
    var mobileView = 992;
    $scope.getWidth = function () {
      return window.innerWidth;
    };

    $scope.applicationState = StateManager.getState();
    $scope.endpointState = EndpointProvider.endpoint();

    $scope.$watch($scope.getWidth, function (newValue) {
      if (newValue >= mobileView) {
        if (angular.isDefined($cookieStore.get('toggle'))) {
          $scope.toggle = !$cookieStore.get('toggle') ? false : true;
        } else {
          $scope.toggle = true;
        }
      } else {
        $scope.toggle = false;
      }
    });

    $scope.toggleSidebar = function () {
      $scope.toggle = !$scope.toggle;
      $cookieStore.put('toggle', $scope.toggle);
    };

    window.onresize = function () {
      $scope.$apply();
    };
  },
]);
