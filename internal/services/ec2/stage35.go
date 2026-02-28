package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type TransitGatewayPropagation struct {
	ResourceID                             string
	ResourceType                           string
	State                                  string
	TransitGatewayAttachmentID             string
	TransitGatewayRouteTableAnnouncementID string
	TransitGatewayRouteTableID             string
}

type TransitGatewayAttachmentPropagation struct {
	State                      string
	TransitGatewayRouteTableID string
}

type TransitGatewayRouteTablePropagation struct {
	ResourceID                             string
	ResourceType                           string
	State                                  string
	TransitGatewayAttachmentID             string
	TransitGatewayRouteTableAnnouncementID string
}

func (s *Service) EnableTransitGatewayRouteTablePropagation(
	transitGatewayRouteTableID,
	transitGatewayAttachmentID,
	transitGatewayRouteTableAnnouncementID string,
) (TransitGatewayPropagation, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	transitGatewayRouteTableAnnouncementID = strings.TrimSpace(transitGatewayRouteTableAnnouncementID)
	if transitGatewayRouteTableID == "" {
		return TransitGatewayPropagation{}, ErrInvalidParameter
	}
	if transitGatewayAttachmentID == "" && transitGatewayRouteTableAnnouncementID == "" {
		return TransitGatewayPropagation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayRouteTables[transitGatewayRouteTableID] == nil {
		return TransitGatewayPropagation{}, ErrNotFound
	}

	key := transitGatewayPropagationKey(transitGatewayRouteTableID, transitGatewayAttachmentID, transitGatewayRouteTableAnnouncementID)
	propagation := s.transitGatewayPropagations[key]
	if propagation == nil {
		propagation = &TransitGatewayPropagation{
			ResourceID: firstNonEmptyString(transitGatewayAttachmentID, transitGatewayRouteTableAnnouncementID),
			ResourceType: firstNonEmptyString(func() string {
				if transitGatewayAttachmentID != "" {
					return "vpc"
				}
				return ""
			}(), ""),
			State:                                  "enabled",
			TransitGatewayAttachmentID:             transitGatewayAttachmentID,
			TransitGatewayRouteTableAnnouncementID: transitGatewayRouteTableAnnouncementID,
			TransitGatewayRouteTableID:             transitGatewayRouteTableID,
		}
		s.transitGatewayPropagations[key] = propagation
	} else {
		propagation.State = "enabled"
	}

	return *propagation, nil
}

func (s *Service) DisableTransitGatewayRouteTablePropagation(
	transitGatewayRouteTableID,
	transitGatewayAttachmentID,
	transitGatewayRouteTableAnnouncementID string,
) (TransitGatewayPropagation, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	transitGatewayRouteTableAnnouncementID = strings.TrimSpace(transitGatewayRouteTableAnnouncementID)
	if transitGatewayRouteTableID == "" {
		return TransitGatewayPropagation{}, ErrInvalidParameter
	}
	if transitGatewayAttachmentID == "" && transitGatewayRouteTableAnnouncementID == "" {
		return TransitGatewayPropagation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := transitGatewayPropagationKey(transitGatewayRouteTableID, transitGatewayAttachmentID, transitGatewayRouteTableAnnouncementID)
	propagation := s.transitGatewayPropagations[key]
	if propagation == nil {
		return TransitGatewayPropagation{}, ErrNotFound
	}
	propagation.State = "disabled"
	return *propagation, nil
}

func (s *Service) GetTransitGatewayAttachmentPropagations(
	transitGatewayAttachmentID string,
	transitGatewayRouteTableIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayAttachmentPropagation, *string, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}
	routeTableIDSet := toStringSet(transitGatewayRouteTableIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayAttachmentPropagation, 0)
	for _, propagation := range s.transitGatewayPropagations {
		if propagation.TransitGatewayAttachmentID != transitGatewayAttachmentID {
			continue
		}
		if len(routeTableIDSet) > 0 {
			if _, ok := routeTableIDSet[propagation.TransitGatewayRouteTableID]; !ok {
				continue
			}
		}
		out = append(out, TransitGatewayAttachmentPropagation{
			State:                      propagation.State,
			TransitGatewayRouteTableID: propagation.TransitGatewayRouteTableID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayRouteTableID < out[j].TransitGatewayRouteTableID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	page := append([]TransitGatewayAttachmentPropagation(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) GetTransitGatewayRouteTablePropagations(
	transitGatewayRouteTableID string,
	resourceIDs, resourceTypes, transitGatewayAttachmentIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayRouteTablePropagation, *string, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	if transitGatewayRouteTableID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	resourceIDSet := toStringSet(resourceIDs)
	resourceTypeSet := toLowerStringSet(resourceTypes)
	transitGatewayAttachmentIDSet := toStringSet(transitGatewayAttachmentIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayRouteTablePropagation, 0)
	for _, propagation := range s.transitGatewayPropagations {
		if propagation.TransitGatewayRouteTableID != transitGatewayRouteTableID {
			continue
		}
		if len(resourceIDSet) > 0 {
			if _, ok := resourceIDSet[propagation.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceTypeSet) > 0 {
			if _, ok := resourceTypeSet[strings.ToLower(propagation.ResourceType)]; !ok {
				continue
			}
		}
		if len(transitGatewayAttachmentIDSet) > 0 {
			if _, ok := transitGatewayAttachmentIDSet[propagation.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		out = append(out, TransitGatewayRouteTablePropagation{
			ResourceID:                             propagation.ResourceID,
			ResourceType:                           propagation.ResourceType,
			State:                                  propagation.State,
			TransitGatewayAttachmentID:             propagation.TransitGatewayAttachmentID,
			TransitGatewayRouteTableAnnouncementID: propagation.TransitGatewayRouteTableAnnouncementID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ResourceID < out[j].ResourceID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	page := append([]TransitGatewayRouteTablePropagation(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func transitGatewayPropagationKey(transitGatewayRouteTableID, transitGatewayAttachmentID, transitGatewayRouteTableAnnouncementID string) string {
	return strings.TrimSpace(transitGatewayRouteTableID) + "|" + strings.TrimSpace(transitGatewayAttachmentID) + "|" + strings.TrimSpace(transitGatewayRouteTableAnnouncementID)
}
