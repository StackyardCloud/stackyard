package ec2

import "strings"

type CarrierGateway struct {
	ID      string
	OwnerID string
	State   string
	Tags    map[string]string
	VpcID   string
}

func (s *Service) CreateCarrierGateway(vpcID string, tags []Tag) (CarrierGateway, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return CarrierGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return CarrierGateway{}, ErrNotFound
	}

	gateway := &CarrierGateway{
		ID:      s.nextIDLocked("cagw"),
		OwnerID: DefaultAccountID,
		State:   "available",
		Tags:    tagsToMap(tags),
		VpcID:   vpcID,
	}
	s.carrierGateways[gateway.ID] = gateway
	return cloneCarrierGateway(gateway), nil
}

func cloneCarrierGateway(in *CarrierGateway) CarrierGateway {
	if in == nil {
		return CarrierGateway{}
	}
	return CarrierGateway{
		ID:      in.ID,
		OwnerID: in.OwnerID,
		State:   in.State,
		Tags:    cloneStringMap(in.Tags),
		VpcID:   in.VpcID,
	}
}
