function ChannelListCtrl($scope, $http) {
	$http.get('/=/channels').success(function(data) {
		$scope.channels = data;
	});
}
// ChannelListCtrl.$inject = ['$scope', '$http'];


function ChannelDetailCtrl($scope, $routeParams, $http, $location) {
	$scope.id = $routeParams.channelId;

	$http.get('/=/channel/' + $routeParams.channelId).success(function(data) {
		$scope.title = data.title;
		$scope.token = data.token;
		$scope.description = data.description;
		$scope.identity = data.identity;
	});


	$scope.updateChannel = function() {
		$http.post('/=/channel/' + $scope.id, {
			title: $scope.title,
			description: $scope.description,
			identity: $scope.identity,
			token: $scope.token
		}).success(function(data) {
			$location.path('/channels')
		});
		// $scope..push({text:$scope.todoText, done:false});
	};
}
// ChannelDetailCtrl.$inject = ['$scope','$routeParams','$http', '$location'];

angular.module('channelcat', []).
config(['$routeProvider', function($routeProvider) {
	$routeProvider.
		when('/channels', {templateUrl: '/partials/channel-list.html',   controller: ChannelListCtrl}).
		when('/channels/:channelId', {templateUrl: '/partials/channel-detail.html', controller: ChannelDetailCtrl}).
		otherwise({redirectTo: '/channels'});
}]);
