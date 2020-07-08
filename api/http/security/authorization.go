package security

import (
	"github.com/portainer/portainer/api"
)

// AuthorizedResourceControlAccess checks whether the user can alter an existing resource control.
func AuthorizedResourceControlAccess(resourceControl *portainer.ResourceControl, context *RestrictedRequestContext) bool {
	if context.IsAdmin || resourceControl.Public {
		return true
	}

	for _, access := range resourceControl.TeamAccesses {
		for _, membership := range context.UserMemberships {
			if membership.TeamID == access.TeamID {
				return true
			}
		}
	}

	for _, access := range resourceControl.UserAccesses {
		if context.UserID == access.UserID {
			return true
		}
	}

	return false
}

// AuthorizedResourceControlUpdate ensure that the user can update a resource control object.
// A non-administrator user cannot create a resource control where:
// * the Public flag is set false
// * the AdministatorsOnly flag is set to true
// * he wants to create a resource control without any user/team accesses
// * he wants to add more than one user in the user accesses
// * he wants to add a user in the user accesses that is not corresponding to its id
// * he wants to add a team he is not a member of
func AuthorizedResourceControlUpdate(resourceControl *portainer.ResourceControl, context *RestrictedRequestContext) bool {
	if context.IsAdmin || resourceControl.Public {
		return true
	}

	if resourceControl.AdministratorsOnly {
		return false
	}

	userAccessesCount := len(resourceControl.UserAccesses)
	teamAccessesCount := len(resourceControl.TeamAccesses)

	if userAccessesCount == 0 && teamAccessesCount == 0 {
		return false
	}

	if userAccessesCount > 1 || (userAccessesCount == 1 && teamAccessesCount == 1) {
		return false
	}

	if userAccessesCount == 1 {
		access := resourceControl.UserAccesses[0]
		if access.UserID == context.UserID {
			return true
		}
	}

	if teamAccessesCount > 0 {
		for _, access := range resourceControl.TeamAccesses {
			for _, membership := range context.UserMemberships {
				if membership.TeamID == access.TeamID {
					return true
				}
			}
		}
	}

	return false
}

// AuthorizedTeamManagement ensure that access to the management of the specified team is granted.
// It will check if the user is either administrator or leader of that team.
func AuthorizedTeamManagement(teamID portainer.TeamID, context *RestrictedRequestContext) bool {
	if context.IsAdmin {
		return true
	}

	for _, membership := range context.UserMemberships {
		if membership.TeamID == teamID && membership.Role == portainer.TeamLeader {
			return true
		}
	}

	return false
}

// authorizedEndpointAccess ensure that the user can access the specified endpoint.
// It will check if the user is part of the authorized users or part of a team that is
// listed in the authorized teams of the endpoint and the associated group.
func authorizedEndpointAccess(endpoint *portainer.Endpoint, endpointGroup *portainer.EndpointGroup, userID portainer.UserID, memberships []portainer.TeamMembership) bool {
	groupAccess := authorizedAccess(userID, memberships, endpointGroup.UserAccessPolicies, endpointGroup.TeamAccessPolicies)
	if !groupAccess {
		return authorizedAccess(userID, memberships, endpoint.UserAccessPolicies, endpoint.TeamAccessPolicies)
	}
	return true
}

// authorizedEndpointGroupAccess ensure that the user can access the specified endpoint group.
// It will check if the user is part of the authorized users or part of a team that is
// listed in the authorized teams.
func authorizedEndpointGroupAccess(endpointGroup *portainer.EndpointGroup, userID portainer.UserID, memberships []portainer.TeamMembership) bool {
	return authorizedAccess(userID, memberships, endpointGroup.UserAccessPolicies, endpointGroup.TeamAccessPolicies)
}

// AuthorizedRegistryAccess ensure that the user can access the specified registry.
// It will check if the user is part of the authorized users or part of a team that is
// listed in the authorized teams.
func AuthorizedRegistryAccess(registry *portainer.Registry, userID portainer.UserID, memberships []portainer.TeamMembership) bool {
	return authorizedAccess(userID, memberships, registry.UserAccessPolicies, registry.TeamAccessPolicies)
}

func authorizedAccess(userID portainer.UserID, memberships []portainer.TeamMembership, userAccessPolicies portainer.UserAccessPolicies, teamAccessPolicies portainer.TeamAccessPolicies) bool {
	_, userAccess := userAccessPolicies[userID]
	if userAccess {
		return true
	}

	for _, membership := range memberships {
		_, teamAccess := teamAccessPolicies[membership.TeamID]
		if teamAccess {
			return true
		}
	}

	return false
}
