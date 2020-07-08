angular.module('portainer.app').directive('rdHeaderContent', [
  'Authentication',
  function rdHeaderContent(Authentication) {
    var directive = {
      requires: '^rdHeader',
      transclude: true,
      link: function (scope) {
        scope.username = Authentication.getUserDetails().username;
      },
      template:
        '<div class="breadcrumb-links"><div class="pull-left" ng-transclude></div><div class="pull-right" ng-if="username"><a ui-sref="portainer.account" style="margin-right: 5px;"><u><i class="fa fa-wrench" aria-hidden="true"></i> 我的账户 </u></a><a ui-sref="portainer.auth({logout: true})" class="text-danger" style="margin-right: 25px;"><u><i class="fa fa-sign-out-alt" aria-hidden="true"></i> 退出登录</u></a></div></div>',
      restrict: 'E',
    };
    return directive;
  },
]);
