package ec2

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
)

type TransitGatewayRouteAttachment struct {
	ResourceID                 string
	ResourceType               string
	TransitGatewayAttachmentID string
}

type TransitGatewayRoute struct {
	DestinationCidrBlock                   string
	PrefixListID                           string
	State                                  string
	TransitGatewayAttachments              []TransitGatewayRouteAttachment
	TransitGatewayRouteTableAnnouncementID string
	Type                                   string
}

func (s *Service) CreateTransitGatewayRoute(
	transitGatewayRouteTableID,
	destinationCidrBlock string,
	blackhole *bool,
	transitGatewayAttachmentID *string,
) (TransitGatewayRoute, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	destinationCidrBlock = strings.TrimSpace(destinationCidrBlock)
	if transitGatewayRouteTableID == "" || destinationCidrBlock == "" {
		return TransitGatewayRoute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.transitGatewayRouteTables[transitGatewayRouteTableID]
	if table == nil {
		return TransitGatewayRoute{}, ErrNotFound
	}

	key := transitGatewayRouteKey(transitGatewayRouteTableID, destinationCidrBlock)
	if s.transitGatewayRoutes[key] != nil {
		return TransitGatewayRoute{}, ErrAlreadyExists
	}

	route, err := s.newTransitGatewayRouteLocked(table.TransitID, destinationCidrBlock, blackhole, transitGatewayAttachmentID)
	if err != nil {
		return TransitGatewayRoute{}, err
	}
	s.transitGatewayRoutes[key] = route
	return cloneTransitGatewayRoute(route), nil
}

func (s *Service) DeleteTransitGatewayRoute(
	transitGatewayRouteTableID,
	destinationCidrBlock string,
) (TransitGatewayRoute, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	destinationCidrBlock = strings.TrimSpace(destinationCidrBlock)
	if transitGatewayRouteTableID == "" || destinationCidrBlock == "" {
		return TransitGatewayRoute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayRouteTables[transitGatewayRouteTableID] == nil {
		return TransitGatewayRoute{}, ErrNotFound
	}

	key := transitGatewayRouteKey(transitGatewayRouteTableID, destinationCidrBlock)
	existing := s.transitGatewayRoutes[key]
	if existing == nil {
		return TransitGatewayRoute{}, ErrNotFound
	}
	out := cloneTransitGatewayRoute(existing)
	out.State = "deleted"
	delete(s.transitGatewayRoutes, key)
	return out, nil
}

func (s *Service) ReplaceTransitGatewayRoute(
	transitGatewayRouteTableID,
	destinationCidrBlock string,
	blackhole *bool,
	transitGatewayAttachmentID *string,
) (TransitGatewayRoute, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	destinationCidrBlock = strings.TrimSpace(destinationCidrBlock)
	if transitGatewayRouteTableID == "" || destinationCidrBlock == "" {
		return TransitGatewayRoute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.transitGatewayRouteTables[transitGatewayRouteTableID]
	if table == nil {
		return TransitGatewayRoute{}, ErrNotFound
	}

	key := transitGatewayRouteKey(transitGatewayRouteTableID, destinationCidrBlock)
	if s.transitGatewayRoutes[key] == nil {
		return TransitGatewayRoute{}, ErrNotFound
	}

	route, err := s.newTransitGatewayRouteLocked(table.TransitID, destinationCidrBlock, blackhole, transitGatewayAttachmentID)
	if err != nil {
		return TransitGatewayRoute{}, err
	}
	s.transitGatewayRoutes[key] = route
	return cloneTransitGatewayRoute(route), nil
}

func (s *Service) SearchTransitGatewayRoutes(
	transitGatewayRouteTableID string,
	filters map[string][]string,
	maxResults *int32,
) ([]TransitGatewayRoute, *bool, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	if transitGatewayRouteTableID == "" || len(filters) == 0 {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayRouteTables[transitGatewayRouteTableID] == nil {
		return nil, nil, ErrNotFound
	}

	routes := s.collectTransitGatewayRoutesForTableLocked(transitGatewayRouteTableID)
	filtered := filterTransitGatewayRoutes(routes, filters)

	limit := 1000
	if maxResults != nil {
		limit = int(*maxResults)
	}
	if limit < 0 {
		return nil, nil, ErrInvalidParameter
	}
	additionalRoutesAvailable := false
	if len(filtered) > limit {
		additionalRoutesAvailable = true
		filtered = filtered[:limit]
	}

	out := make([]TransitGatewayRoute, 0, len(filtered))
	for _, route := range filtered {
		out = append(out, cloneTransitGatewayRoute(&route))
	}
	return out, &additionalRoutesAvailable, nil
}

func (s *Service) ExportTransitGatewayRoutes(
	transitGatewayRouteTableID,
	s3Bucket string,
	filters map[string][]string,
) (string, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	s3Bucket = strings.TrimSpace(s3Bucket)
	if transitGatewayRouteTableID == "" || s3Bucket == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayRouteTables[transitGatewayRouteTableID] == nil {
		return "", ErrNotFound
	}

	_ = filterTransitGatewayRoutes(s.collectTransitGatewayRoutesForTableLocked(transitGatewayRouteTableID), filters)

	return fmt.Sprintf("s3://%s/VPCTransitGateway/TransitGatewayRouteTables/%s.json", s3Bucket, transitGatewayRouteTableID), nil
}

func (s *Service) newTransitGatewayRouteLocked(
	transitGatewayID,
	destinationCidrBlock string,
	blackhole *bool,
	transitGatewayAttachmentID *string,
) (*TransitGatewayRoute, error) {
	useBlackhole := blackhole != nil && *blackhole
	attachmentID := ""
	if transitGatewayAttachmentID != nil {
		attachmentID = strings.TrimSpace(*transitGatewayAttachmentID)
	}
	if !useBlackhole && attachmentID == "" {
		return nil, ErrInvalidParameter
	}

	route := &TransitGatewayRoute{
		DestinationCidrBlock: destinationCidrBlock,
		State:                "active",
		Type:                 "static",
	}
	if useBlackhole {
		route.State = "blackhole"
		return route, nil
	}

	attachment, attachmentTransitGatewayID, ok := s.resolveTransitGatewayRouteAttachmentLocked(attachmentID)
	if !ok {
		return nil, ErrNotFound
	}
	if transitGatewayID != "" && attachmentTransitGatewayID != "" && transitGatewayID != attachmentTransitGatewayID {
		return nil, ErrInvalidParameter
	}
	route.TransitGatewayAttachments = []TransitGatewayRouteAttachment{attachment}
	return route, nil
}

func (s *Service) resolveTransitGatewayRouteAttachmentLocked(transitGatewayAttachmentID string) (TransitGatewayRouteAttachment, string, bool) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayRouteAttachment{}, "", false
	}
	if attachment := s.transitGatewayVpcAttachments[transitGatewayAttachmentID]; attachment != nil {
		return TransitGatewayRouteAttachment{
			ResourceID:                 attachment.VpcID,
			ResourceType:               "vpc",
			TransitGatewayAttachmentID: attachment.ID,
		}, attachment.TransitGatewayID, true
	}
	if attachment := s.transitGatewayPeeringAttachments[transitGatewayAttachmentID]; attachment != nil {
		return TransitGatewayRouteAttachment{
			ResourceID:                 attachment.AccepterTgwInfo.TransitGatewayID,
			ResourceType:               "peering",
			TransitGatewayAttachmentID: attachment.ID,
		}, attachment.RequesterTgwInfo.TransitGatewayID, true
	}
	if attachment := s.transitGatewayConnects[transitGatewayAttachmentID]; attachment != nil {
		return TransitGatewayRouteAttachment{
			ResourceID:                 attachment.TransportTransitGatewayAttachmentID,
			ResourceType:               "connect",
			TransitGatewayAttachmentID: attachment.ID,
		}, attachment.TransitGatewayID, true
	}
	return TransitGatewayRouteAttachment{}, "", false
}

func (s *Service) collectTransitGatewayRoutesForTableLocked(transitGatewayRouteTableID string) []TransitGatewayRoute {
	prefix := strings.TrimSpace(transitGatewayRouteTableID) + "|"
	out := make([]TransitGatewayRoute, 0)
	for key, route := range s.transitGatewayRoutes {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		out = append(out, cloneTransitGatewayRoute(route))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DestinationCidrBlock < out[j].DestinationCidrBlock })
	return out
}

func filterTransitGatewayRoutes(routes []TransitGatewayRoute, filters map[string][]string) []TransitGatewayRoute {
	standard, _, _ := splitEC2Filters(filters)

	attachmentIDSet := toStringSet(standard["attachment.transit-gateway-attachment-id"])
	attachmentResourceIDSet := toStringSet(standard["attachment.resource-id"])
	attachmentResourceTypeSet := toLowerStringSet(standard["attachment.resource-type"])
	destinationCIDRSet := toStringSet(standard["transit-gateway-route-destination-cidr-block"])
	stateSet := toLowerStringSet(standard["state"])
	typeSet := toLowerStringSet(standard["type"])
	exactMatchSet := toStringSet(standard["route-search.exact-match"])
	longestPrefixMatches := dedupeTrimmedStrings(standard["route-search.longest-prefix-match"])
	subnetOfMatches := dedupeTrimmedStrings(standard["route-search.subnet-of-match"])
	supernetOfMatches := dedupeTrimmedStrings(standard["route-search.supernet-of-match"])

	filtered := make([]TransitGatewayRoute, 0, len(routes))
	for _, route := range routes {
		if len(destinationCIDRSet) > 0 {
			if _, ok := destinationCIDRSet[route.DestinationCidrBlock]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(route.State)]; !ok {
				continue
			}
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[strings.ToLower(route.Type)]; !ok {
				continue
			}
		}
		if len(exactMatchSet) > 0 {
			if _, ok := exactMatchSet[route.DestinationCidrBlock]; !ok {
				continue
			}
		}
		if len(longestPrefixMatches) > 0 && !matchesRoutePrefixSearch(route.DestinationCidrBlock, longestPrefixMatches, "longest-prefix-match") {
			continue
		}
		if len(subnetOfMatches) > 0 && !matchesRoutePrefixSearch(route.DestinationCidrBlock, subnetOfMatches, "subnet-of-match") {
			continue
		}
		if len(supernetOfMatches) > 0 && !matchesRoutePrefixSearch(route.DestinationCidrBlock, supernetOfMatches, "supernet-of-match") {
			continue
		}
		if len(attachmentIDSet) > 0 || len(attachmentResourceIDSet) > 0 || len(attachmentResourceTypeSet) > 0 {
			if !matchesRouteAttachments(route.TransitGatewayAttachments, attachmentIDSet, attachmentResourceIDSet, attachmentResourceTypeSet) {
				continue
			}
		}
		filtered = append(filtered, route)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].DestinationCidrBlock < filtered[j].DestinationCidrBlock })
	return filtered
}

func matchesRouteAttachments(
	attachments []TransitGatewayRouteAttachment,
	attachmentIDSet,
	attachmentResourceIDSet,
	attachmentResourceTypeSet map[string]struct{},
) bool {
	if len(attachments) == 0 {
		return false
	}
	for _, attachment := range attachments {
		if len(attachmentIDSet) > 0 {
			if _, ok := attachmentIDSet[attachment.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		if len(attachmentResourceIDSet) > 0 {
			if _, ok := attachmentResourceIDSet[attachment.ResourceID]; !ok {
				continue
			}
		}
		if len(attachmentResourceTypeSet) > 0 {
			if _, ok := attachmentResourceTypeSet[strings.ToLower(attachment.ResourceType)]; !ok {
				continue
			}
		}
		return true
	}
	return false
}

func matchesRoutePrefixSearch(routeCIDR string, candidateCIDRs []string, mode string) bool {
	routePrefix, ok := parseCIDRPrefix(routeCIDR)
	if !ok {
		return false
	}

	for _, candidate := range candidateCIDRs {
		candidatePrefix, ok := parseCIDRPrefix(candidate)
		if !ok {
			continue
		}
		switch mode {
		case "longest-prefix-match":
			if routePrefix.Contains(candidatePrefix.Addr()) {
				return true
			}
		case "subnet-of-match":
			if candidatePrefix.Contains(routePrefix.Addr()) && routePrefix.Bits() >= candidatePrefix.Bits() {
				return true
			}
		case "supernet-of-match":
			if routePrefix.Contains(candidatePrefix.Addr()) && routePrefix.Bits() <= candidatePrefix.Bits() {
				return true
			}
		}
	}

	return false
}

func parseCIDRPrefix(value string) (netip.Prefix, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Prefix{}, false
	}
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		return netip.Prefix{}, false
	}
	return prefix.Masked(), true
}

func transitGatewayRouteKey(transitGatewayRouteTableID, destinationCidrBlock string) string {
	return strings.TrimSpace(transitGatewayRouteTableID) + "|" + strings.TrimSpace(destinationCidrBlock)
}

func cloneTransitGatewayRoute(in *TransitGatewayRoute) TransitGatewayRoute {
	if in == nil {
		return TransitGatewayRoute{}
	}
	out := *in
	out.TransitGatewayAttachments = append([]TransitGatewayRouteAttachment(nil), in.TransitGatewayAttachments...)
	return out
}
