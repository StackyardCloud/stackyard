package ec2

import (
	"strings"
)

type VpcEndpointServiceAddedPrincipal struct {
	Principal           string
	PrincipalType       string
	ServiceID           string
	ServicePermissionID string
}

func (s *Service) ModifyVpcEndpointServicePermissions(
	serviceID string,
	addAllowedPrincipals,
	removeAllowedPrincipals []string,
) ([]VpcEndpointServiceAddedPrincipal, bool, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return nil, false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcEndpointServicePayerResponsibility[serviceID]; !ok {
		return nil, false, ErrNotFound
	}
	principalsByService := s.vpcEndpointServicePermissions[serviceID]
	if principalsByService == nil {
		principalsByService = map[string]string{}
		s.vpcEndpointServicePermissions[serviceID] = principalsByService
	}

	for _, principal := range dedupeTrimmedStrings(removeAllowedPrincipals) {
		delete(principalsByService, principal)
	}

	added := make([]VpcEndpointServiceAddedPrincipal, 0, len(addAllowedPrincipals))
	for _, principal := range dedupeTrimmedStrings(addAllowedPrincipals) {
		permissionID := principalsByService[principal]
		if permissionID == "" {
			permissionID = s.nextIDLocked("vpce-svc-perm")
			principalsByService[principal] = permissionID
		}
		added = append(added, VpcEndpointServiceAddedPrincipal{
			Principal:           principal,
			PrincipalType:       ec2PrincipalTypeFromPrincipal(principal),
			ServiceID:           serviceID,
			ServicePermissionID: permissionID,
		})
	}

	return added, true, nil
}

func ec2PrincipalTypeFromPrincipal(principal string) string {
	principal = strings.TrimSpace(principal)
	switch {
	case principal == "*":
		return "All"
	case strings.Contains(principal, ":role/"):
		return "Role"
	case strings.Contains(principal, ":user/"):
		return "User"
	case strings.Contains(principal, "ou-"):
		return "OrganizationUnit"
	case strings.Contains(principal, ".amazonaws.com"):
		return "Service"
	default:
		return "Account"
	}
}
