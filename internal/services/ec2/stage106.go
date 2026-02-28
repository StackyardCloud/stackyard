package ec2

import "strings"

type CoipCidr struct {
	Cidr                     string
	CoipPoolID               string
	LocalGatewayRouteTableID string
}

func (s *Service) CreateCoipCidr(cidr, coipPoolID string) (CoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	coipPoolID = strings.TrimSpace(coipPoolID)
	if cidr == "" || coipPoolID == "" {
		return CoipCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := coipPoolID + "|" + cidr
	if _, exists := s.coipCidrs[key]; exists {
		return CoipCidr{}, ErrAlreadyExists
	}

	coipCidr := &CoipCidr{
		Cidr:       cidr,
		CoipPoolID: coipPoolID,
	}
	s.coipCidrs[key] = coipCidr
	return cloneCoipCidr(coipCidr), nil
}

func cloneCoipCidr(in *CoipCidr) CoipCidr {
	if in == nil {
		return CoipCidr{}
	}
	return CoipCidr{
		Cidr:                     in.Cidr,
		CoipPoolID:               in.CoipPoolID,
		LocalGatewayRouteTableID: in.LocalGatewayRouteTableID,
	}
}
