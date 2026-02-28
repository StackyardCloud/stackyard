package ec2

import (
	"sort"
	"strings"
	"time"
)

type TransitGatewayRouteTableAnnouncement struct {
	AnnouncementDirection                  string
	CoreNetworkID                          string
	CreationTime                           time.Time
	PeerCoreNetworkID                      string
	PeerTransitGatewayID                   string
	PeeringAttachmentID                    string
	State                                  string
	Tags                                   map[string]string
	TransitGatewayID                       string
	TransitGatewayRouteTableAnnouncementID string
	TransitGatewayRouteTableID             string
}

func (s *Service) CreateTransitGatewayRouteTableAnnouncement(
	transitGatewayRouteTableID,
	peeringAttachmentID string,
	tags []Tag,
) (TransitGatewayRouteTableAnnouncement, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	peeringAttachmentID = strings.TrimSpace(peeringAttachmentID)
	if transitGatewayRouteTableID == "" || peeringAttachmentID == "" {
		return TransitGatewayRouteTableAnnouncement{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	routeTable := s.transitGatewayRouteTables[transitGatewayRouteTableID]
	if routeTable == nil {
		return TransitGatewayRouteTableAnnouncement{}, ErrNotFound
	}
	peeringAttachment := s.transitGatewayPeeringAttachments[peeringAttachmentID]
	if peeringAttachment == nil {
		return TransitGatewayRouteTableAnnouncement{}, ErrNotFound
	}
	if routeTable.TransitID != "" && peeringAttachment.RequesterTgwInfo.TransitGatewayID != "" && routeTable.TransitID != peeringAttachment.RequesterTgwInfo.TransitGatewayID {
		return TransitGatewayRouteTableAnnouncement{}, ErrInvalidParameter
	}

	for _, existing := range s.transitGatewayRouteTableAnnouncements {
		if existing.TransitGatewayRouteTableID == transitGatewayRouteTableID && existing.PeeringAttachmentID == peeringAttachmentID {
			return TransitGatewayRouteTableAnnouncement{}, ErrAlreadyExists
		}
	}

	announcement := &TransitGatewayRouteTableAnnouncement{
		AnnouncementDirection:                  "outgoing",
		CreationTime:                           time.Now().UTC(),
		PeerTransitGatewayID:                   peeringAttachment.AccepterTgwInfo.TransitGatewayID,
		PeeringAttachmentID:                    peeringAttachmentID,
		State:                                  "available",
		Tags:                                   tagsToMap(tags),
		TransitGatewayID:                       routeTable.TransitID,
		TransitGatewayRouteTableAnnouncementID: s.nextIDLocked("tgw-rtb-announcement"),
		TransitGatewayRouteTableID:             transitGatewayRouteTableID,
	}
	s.transitGatewayRouteTableAnnouncements[announcement.TransitGatewayRouteTableAnnouncementID] = announcement
	return cloneTransitGatewayRouteTableAnnouncement(announcement), nil
}

func (s *Service) DeleteTransitGatewayRouteTableAnnouncement(
	transitGatewayRouteTableAnnouncementID string,
) (TransitGatewayRouteTableAnnouncement, error) {
	transitGatewayRouteTableAnnouncementID = strings.TrimSpace(transitGatewayRouteTableAnnouncementID)
	if transitGatewayRouteTableAnnouncementID == "" {
		return TransitGatewayRouteTableAnnouncement{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	announcement := s.transitGatewayRouteTableAnnouncements[transitGatewayRouteTableAnnouncementID]
	if announcement == nil {
		return TransitGatewayRouteTableAnnouncement{}, ErrNotFound
	}

	out := cloneTransitGatewayRouteTableAnnouncement(announcement)
	out.State = "deleted"
	delete(s.transitGatewayRouteTableAnnouncements, transitGatewayRouteTableAnnouncementID)
	s.deleteTransitGatewayPropagationsByAnnouncementIDLocked(transitGatewayRouteTableAnnouncementID)
	return out, nil
}

func (s *Service) DescribeTransitGatewayRouteTableAnnouncements(
	transitGatewayRouteTableAnnouncementIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayRouteTableAnnouncement, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	announcementIDSet := toStringSet(append(
		dedupeTrimmedStrings(transitGatewayRouteTableAnnouncementIDs),
		standard["transit-gateway-route-table-announcement-id"]...,
	))
	announcementDirectionSet := toLowerStringSet(standard["announcement-direction"])
	coreNetworkIDSet := toStringSet(standard["core-network-id"])
	peerCoreNetworkIDSet := toStringSet(standard["peer-core-network-id"])
	peerTransitGatewayIDSet := toStringSet(standard["peer-transit-gateway-id"])
	peeringAttachmentIDSet := toStringSet(standard["peering-attachment-id"])
	stateSet := toLowerStringSet(standard["state"])
	transitGatewayIDSet := toStringSet(standard["transit-gateway-id"])
	transitGatewayRouteTableIDSet := toStringSet(standard["transit-gateway-route-table-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayRouteTableAnnouncement, 0, len(s.transitGatewayRouteTableAnnouncements))
	for _, announcement := range s.transitGatewayRouteTableAnnouncements {
		if len(announcementIDSet) > 0 {
			if _, ok := announcementIDSet[announcement.TransitGatewayRouteTableAnnouncementID]; !ok {
				continue
			}
		}
		if len(announcementDirectionSet) > 0 {
			if _, ok := announcementDirectionSet[strings.ToLower(announcement.AnnouncementDirection)]; !ok {
				continue
			}
		}
		if len(coreNetworkIDSet) > 0 {
			if _, ok := coreNetworkIDSet[announcement.CoreNetworkID]; !ok {
				continue
			}
		}
		if len(peerCoreNetworkIDSet) > 0 {
			if _, ok := peerCoreNetworkIDSet[announcement.PeerCoreNetworkID]; !ok {
				continue
			}
		}
		if len(peerTransitGatewayIDSet) > 0 {
			if _, ok := peerTransitGatewayIDSet[announcement.PeerTransitGatewayID]; !ok {
				continue
			}
		}
		if len(peeringAttachmentIDSet) > 0 {
			if _, ok := peeringAttachmentIDSet[announcement.PeeringAttachmentID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(announcement.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[announcement.TransitGatewayID]; !ok {
				continue
			}
		}
		if len(transitGatewayRouteTableIDSet) > 0 {
			if _, ok := transitGatewayRouteTableIDSet[announcement.TransitGatewayRouteTableID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(announcement.Tags, tagKeys, tagFilters) {
			continue
		}

		out = append(out, cloneTransitGatewayRouteTableAnnouncement(announcement))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].TransitGatewayRouteTableAnnouncementID < out[j].TransitGatewayRouteTableAnnouncementID
	})

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayRouteTableAnnouncement(nil), out[start:end]...), outputToken, nil
}

func (s *Service) deleteTransitGatewayPropagationsByAnnouncementIDLocked(transitGatewayRouteTableAnnouncementID string) {
	for key, propagation := range s.transitGatewayPropagations {
		if propagation.TransitGatewayRouteTableAnnouncementID == transitGatewayRouteTableAnnouncementID {
			delete(s.transitGatewayPropagations, key)
		}
	}
}

func (s *Service) deleteTransitGatewayRouteTableAnnouncementsForRouteTableLocked(transitGatewayRouteTableID string) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	if transitGatewayRouteTableID == "" {
		return
	}
	for announcementID, announcement := range s.transitGatewayRouteTableAnnouncements {
		if announcement.TransitGatewayRouteTableID == transitGatewayRouteTableID {
			delete(s.transitGatewayRouteTableAnnouncements, announcementID)
			s.deleteTransitGatewayPropagationsByAnnouncementIDLocked(announcementID)
		}
	}
}

func (s *Service) deleteTransitGatewayRouteTableAnnouncementsForPeeringAttachmentLocked(peeringAttachmentID string) {
	peeringAttachmentID = strings.TrimSpace(peeringAttachmentID)
	if peeringAttachmentID == "" {
		return
	}
	for announcementID, announcement := range s.transitGatewayRouteTableAnnouncements {
		if announcement.PeeringAttachmentID == peeringAttachmentID {
			delete(s.transitGatewayRouteTableAnnouncements, announcementID)
			s.deleteTransitGatewayPropagationsByAnnouncementIDLocked(announcementID)
		}
	}
}

func cloneTransitGatewayRouteTableAnnouncement(in *TransitGatewayRouteTableAnnouncement) TransitGatewayRouteTableAnnouncement {
	if in == nil {
		return TransitGatewayRouteTableAnnouncement{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
