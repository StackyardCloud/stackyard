package ec2

import (
	"fmt"
	"strings"
	"time"
)

type TransitGatewayMulticastDomain struct {
	CreationTime time.Time
	ID           string
	ARN          string
	OwnerID      string
	State        string
	Tags         map[string]string
	TransitID    string
}

type TransitGatewayPolicyTable struct {
	CreationTime time.Time
	ID           string
	State        string
	Tags         map[string]string
	TransitID    string
}

type TransitGatewayRouteTable struct {
	CreationTime                 time.Time
	DefaultAssociationRouteTable bool
	DefaultPropagationRouteTable bool
	ID                           string
	State                        string
	Tags                         map[string]string
	TransitID                    string
}

func (s *Service) CreateTransitGatewayMulticastDomain(transitGatewayID string, tags []Tag) (TransitGatewayMulticastDomain, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if transitGatewayID == "" {
		return TransitGatewayMulticastDomain{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextIDLocked("tgw-mcast-domain")
	domain := &TransitGatewayMulticastDomain{
		CreationTime: time.Now().UTC(),
		ID:           id,
		ARN:          fmt.Sprintf("arn:aws:ec2:%s:%s:transit-gateway-multicast-domain/%s", DefaultRegion, DefaultAccountID, id),
		OwnerID:      DefaultAccountID,
		State:        "available",
		Tags:         tagsToMap(tags),
		TransitID:    transitGatewayID,
	}
	s.transitGatewayMulticastDomains[domain.ID] = domain

	return cloneTransitGatewayMulticastDomain(domain), nil
}

func (s *Service) DeleteTransitGatewayMulticastDomain(transitGatewayMulticastDomainID string) (TransitGatewayMulticastDomain, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	if transitGatewayMulticastDomainID == "" {
		return TransitGatewayMulticastDomain{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	domain := s.transitGatewayMulticastDomains[transitGatewayMulticastDomainID]
	if domain == nil {
		return TransitGatewayMulticastDomain{}, ErrNotFound
	}

	out := cloneTransitGatewayMulticastDomain(domain)
	out.State = "deleted"
	delete(s.transitGatewayMulticastDomains, transitGatewayMulticastDomainID)
	s.deleteTransitGatewayMulticastGroupsForDomainLocked(transitGatewayMulticastDomainID)
	for key, association := range s.transitGatewayMulticastDomainAssocs {
		if association.TransitGatewayMulticastDomainID == transitGatewayMulticastDomainID {
			delete(s.transitGatewayMulticastDomainAssocs, key)
		}
	}

	return out, nil
}

func (s *Service) CreateTransitGatewayPolicyTable(transitGatewayID string, tags []Tag) (TransitGatewayPolicyTable, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if transitGatewayID == "" {
		return TransitGatewayPolicyTable{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := &TransitGatewayPolicyTable{
		CreationTime: time.Now().UTC(),
		ID:           s.nextIDLocked("tgw-ptb"),
		State:        "available",
		Tags:         tagsToMap(tags),
		TransitID:    transitGatewayID,
	}
	s.transitGatewayPolicyTables[table.ID] = table

	return cloneTransitGatewayPolicyTable(table), nil
}

func (s *Service) DeleteTransitGatewayPolicyTable(transitGatewayPolicyTableID string) (TransitGatewayPolicyTable, error) {
	transitGatewayPolicyTableID = strings.TrimSpace(transitGatewayPolicyTableID)
	if transitGatewayPolicyTableID == "" {
		return TransitGatewayPolicyTable{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.transitGatewayPolicyTables[transitGatewayPolicyTableID]
	if table == nil {
		return TransitGatewayPolicyTable{}, ErrNotFound
	}

	out := cloneTransitGatewayPolicyTable(table)
	out.State = "deleted"
	delete(s.transitGatewayPolicyTables, transitGatewayPolicyTableID)
	for key, association := range s.transitGatewayPolicyTableAssocs {
		if association.TransitGatewayPolicyTableID == transitGatewayPolicyTableID {
			delete(s.transitGatewayPolicyTableAssocs, key)
		}
	}

	return out, nil
}

func (s *Service) CreateTransitGatewayRouteTable(transitGatewayID string, tags []Tag) (TransitGatewayRouteTable, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if transitGatewayID == "" {
		return TransitGatewayRouteTable{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := &TransitGatewayRouteTable{
		CreationTime:                 time.Now().UTC(),
		DefaultAssociationRouteTable: false,
		DefaultPropagationRouteTable: false,
		ID:                           s.nextIDLocked("tgw-rtb"),
		State:                        "available",
		Tags:                         tagsToMap(tags),
		TransitID:                    transitGatewayID,
	}
	s.transitGatewayRouteTables[table.ID] = table

	return cloneTransitGatewayRouteTable(table), nil
}

func (s *Service) DeleteTransitGatewayRouteTable(transitGatewayRouteTableID string) (TransitGatewayRouteTable, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	if transitGatewayRouteTableID == "" {
		return TransitGatewayRouteTable{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.transitGatewayRouteTables[transitGatewayRouteTableID]
	if table == nil {
		return TransitGatewayRouteTable{}, ErrNotFound
	}

	out := cloneTransitGatewayRouteTable(table)
	out.State = "deleted"
	delete(s.transitGatewayRouteTables, transitGatewayRouteTableID)
	s.deleteTransitGatewayRouteTableAnnouncementsForRouteTableLocked(transitGatewayRouteTableID)
	routeKeyPrefix := transitGatewayRouteTableID + "|"
	for key := range s.transitGatewayRoutes {
		if strings.HasPrefix(key, routeKeyPrefix) {
			delete(s.transitGatewayRoutes, key)
		}
	}
	for key, association := range s.transitGatewayRouteTableAssocs {
		if association.TransitGatewayRouteTableID == transitGatewayRouteTableID {
			delete(s.transitGatewayRouteTableAssocs, key)
		}
	}

	return out, nil
}

func cloneTransitGatewayMulticastDomain(in *TransitGatewayMulticastDomain) TransitGatewayMulticastDomain {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayPolicyTable(in *TransitGatewayPolicyTable) TransitGatewayPolicyTable {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayRouteTable(in *TransitGatewayRouteTable) TransitGatewayRouteTable {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
