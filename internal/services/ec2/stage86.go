package ec2

import (
	"fmt"
	"strings"
)

func (s *Service) AssociateIpamResourceDiscovery(
	ipamID, ipamResourceDiscoveryID string,
	tags []Tag,
) (IpamResourceDiscoveryAssociation, error) {
	ipamID = strings.TrimSpace(ipamID)
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	if ipamID == "" || ipamResourceDiscoveryID == "" {
		return IpamResourceDiscoveryAssociation{}, ErrInvalidParameter
	}
	tags = normalizeEC2Tags(tags)

	s.mu.Lock()
	defer s.mu.Unlock()

	pairKey := ipamID + "|" + ipamResourceDiscoveryID
	if associationID, ok := s.ipamResourceDiscoveryAssociationByPair[pairKey]; ok {
		existing := s.ipamResourceDiscoveryAssociations[associationID]
		if existing == nil {
			return IpamResourceDiscoveryAssociation{}, ErrNotFound
		}
		if len(tags) > 0 {
			existing.Tags = cloneEC2Tags(tags)
		}
		return cloneIpamResourceDiscoveryAssociation(existing), nil
	}

	associationID := s.nextIDLocked("ipam-rd-assoc")
	association := &IpamResourceDiscoveryAssociation{
		IpamARN:                             fmt.Sprintf("arn:aws:ec2:%s:%s:ipam/%s", DefaultRegion, DefaultAccountID, ipamID),
		IpamID:                              ipamID,
		IpamRegion:                          DefaultRegion,
		IpamResourceDiscoveryAssociationARN: fmt.Sprintf("arn:aws:ec2:%s:%s:ipam-resource-discovery-association/%s", DefaultRegion, DefaultAccountID, associationID),
		IpamResourceDiscoveryAssociationID:  associationID,
		IpamResourceDiscoveryID:             ipamResourceDiscoveryID,
		IsDefault:                           false,
		OwnerID:                             DefaultAccountID,
		ResourceDiscoveryStatus:             "active",
		State:                               "associate-complete",
		Tags:                                cloneEC2Tags(tags),
	}
	s.ipamResourceDiscoveryAssociations[associationID] = association
	s.ipamResourceDiscoveryAssociationByPair[pairKey] = associationID
	return cloneIpamResourceDiscoveryAssociation(association), nil
}

func cloneIpamResourceDiscoveryAssociation(in *IpamResourceDiscoveryAssociation) IpamResourceDiscoveryAssociation {
	if in == nil {
		return IpamResourceDiscoveryAssociation{}
	}
	out := *in
	out.Tags = cloneEC2Tags(in.Tags)
	return out
}
