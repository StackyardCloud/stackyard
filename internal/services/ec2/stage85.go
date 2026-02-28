package ec2

import "strings"

func (s *Service) AssociateIpamByoasn(asn, cidr string) (ByoipAsnAssociation, error) {
	asn = strings.TrimSpace(asn)
	cidr = strings.TrimSpace(cidr)
	if asn == "" || cidr == "" {
		return ByoipAsnAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		record = &ByoipCidr{Cidr: cidr}
		s.byoipCidrs[cidr] = record
	}

	for i := range record.AsnAssociations {
		if strings.TrimSpace(record.AsnAssociations[i].Asn) != asn {
			continue
		}
		record.AsnAssociations[i].State = "associated"
		record.AsnAssociations[i].StatusMessage = "associated"
		return record.AsnAssociations[i], nil
	}

	association := ByoipAsnAssociation{
		Asn:           asn,
		Cidr:          cidr,
		State:         "associated",
		StatusMessage: "associated",
	}
	record.AsnAssociations = append(record.AsnAssociations, association)
	return association, nil
}
