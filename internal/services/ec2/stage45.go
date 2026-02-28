package ec2

import "strings"

func (s *Service) ModifyVpcTenancy(vpcID, instanceTenancy string) error {
	vpcID = strings.TrimSpace(vpcID)
	instanceTenancy = strings.ToLower(strings.TrimSpace(instanceTenancy))
	if vpcID == "" || instanceTenancy == "" {
		return ErrInvalidParameter
	}

	switch instanceTenancy {
	case "default", "dedicated", "host":
	default:
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return ErrNotFound
	}
	vpc.InstanceTenancy = instanceTenancy
	return nil
}
