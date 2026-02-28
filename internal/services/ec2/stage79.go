package ec2

import "strings"

type ByoipAsnAssociation struct {
	Asn           string
	Cidr          string
	State         string
	StatusMessage string
}

type ByoipCidr struct {
	AsnAssociations    []ByoipAsnAssociation
	Cidr               string
	Description        string
	NetworkBorderGroup string
	State              string
	StatusMessage      string
}

func (s *Service) AdvertiseByoipCidr(cidr string, asn *string, networkBorderGroup *string) (ByoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}

	var normalizedASN string
	if asn != nil {
		normalizedASN = strings.TrimSpace(*asn)
		if normalizedASN == "" {
			return ByoipCidr{}, ErrInvalidParameter
		}
	}

	var normalizedBorderGroup string
	if networkBorderGroup != nil {
		normalizedBorderGroup = strings.TrimSpace(*networkBorderGroup)
		if normalizedBorderGroup == "" {
			return ByoipCidr{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		record = &ByoipCidr{
			Cidr: cidr,
		}
		s.byoipCidrs[cidr] = record
	}

	record.State = "advertised"
	record.StatusMessage = "advertised"
	if normalizedBorderGroup != "" {
		record.NetworkBorderGroup = normalizedBorderGroup
	}

	if normalizedASN != "" {
		record.AsnAssociations = []ByoipAsnAssociation{
			{
				Asn:           normalizedASN,
				Cidr:          cidr,
				State:         "advertised",
				StatusMessage: "advertised",
			},
		}
	}

	return cloneByoipCidr(record), nil
}

func cloneByoipCidr(in *ByoipCidr) ByoipCidr {
	if in == nil {
		return ByoipCidr{}
	}
	out := *in
	out.AsnAssociations = append([]ByoipAsnAssociation(nil), in.AsnAssociations...)
	return out
}
