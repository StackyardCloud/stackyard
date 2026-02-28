package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TransitGatewayOptionsInput struct {
	AmazonSideASN                   *int64
	AutoAcceptSharedAttachments     *string
	DefaultRouteTableAssociation    *string
	DefaultRouteTablePropagation    *string
	DnsSupport                      *string
	MulticastSupport                *string
	SecurityGroupReferencingSupport *string
	TransitGatewayCidrBlocks        []string
	VpnEcmpSupport                  *string
}

type TransitGatewayOptions struct {
	AmazonSideASN                   int64
	AssociationDefaultRouteTableID  string
	AutoAcceptSharedAttachments     string
	DefaultRouteTableAssociation    string
	DefaultRouteTablePropagation    string
	DnsSupport                      string
	MulticastSupport                string
	PropagationDefaultRouteTableID  string
	SecurityGroupReferencingSupport string
	TransitGatewayCidrBlocks        []string
	VpnEcmpSupport                  string
}

type TransitGateway struct {
	CreationTime time.Time
	Description  string
	ID           string
	ARN          string
	OwnerID      string
	State        string
	Options      TransitGatewayOptions
	Tags         map[string]string
}

type TransitGatewayVpcAttachmentOptionsInput struct {
	ApplianceModeSupport            *string
	DnsSupport                      *string
	Ipv6Support                     *string
	SecurityGroupReferencingSupport *string
}

type TransitGatewayVpcAttachmentOptions struct {
	ApplianceModeSupport            string
	DnsSupport                      string
	Ipv6Support                     string
	SecurityGroupReferencingSupport string
}

type TransitGatewayVpcAttachment struct {
	CreationTime     time.Time
	ID               string
	Options          TransitGatewayVpcAttachmentOptions
	State            string
	SubnetIDs        []string
	Tags             map[string]string
	TransitGatewayID string
	VpcID            string
	VpcOwnerID       string
}

type TransitGatewayPeeringAttachmentOptionsInput struct {
	DynamicRouting *string
}

type TransitGatewayPeeringAttachmentOptions struct {
	DynamicRouting string
}

type TransitGatewayPeeringInfo struct {
	OwnerID          string
	Region           string
	TransitGatewayID string
}

type TransitGatewayPeeringAttachment struct {
	AccepterTgwInfo                    TransitGatewayPeeringInfo
	AccepterTransitGatewayAttachmentID string
	CreationTime                       time.Time
	ID                                 string
	Options                            TransitGatewayPeeringAttachmentOptions
	RequesterTgwInfo                   TransitGatewayPeeringInfo
	State                              string
	Tags                               map[string]string
}

type TransitGatewayConnectOptionsInput struct {
	Protocol *string
}

type TransitGatewayConnectOptions struct {
	Protocol string
}

type TransitGatewayConnect struct {
	CreationTime                        time.Time
	ID                                  string
	Options                             TransitGatewayConnectOptions
	State                               string
	Tags                                map[string]string
	TransitGatewayID                    string
	TransportTransitGatewayAttachmentID string
}

type TransitGatewayConnectPeerInput struct {
	InsideCidrBlocks      []string
	PeerAddress           string
	PeerASN               *int64
	TransitGatewayAddress *string
}

type TransitGatewayAttachmentBgpConfiguration struct {
	BgpStatus             string
	PeerAddress           string
	PeerASN               int64
	TransitGatewayAddress string
	TransitGatewayASN     int64
}

type TransitGatewayConnectPeerConfiguration struct {
	BgpConfigurations     []TransitGatewayAttachmentBgpConfiguration
	InsideCidrBlocks      []string
	PeerAddress           string
	Protocol              string
	TransitGatewayAddress string
}

type TransitGatewayConnectPeer struct {
	Configuration               TransitGatewayConnectPeerConfiguration
	CreationTime                time.Time
	State                       string
	Tags                        map[string]string
	TransitGatewayAttachmentID  string
	TransitGatewayConnectPeerID string
}

func (s *Service) CreateTransitGateway(
	description string,
	options TransitGatewayOptionsInput,
	tags []Tag,
) (TransitGateway, error) {
	description = strings.TrimSpace(description)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	gatewayID := s.nextIDLocked("tgw")
	defaultRouteTableID := s.nextIDLocked("tgw-rtb")

	gateway := &TransitGateway{
		CreationTime: now,
		Description:  description,
		ID:           gatewayID,
		ARN:          fmt.Sprintf("arn:aws:ec2:%s:%s:transit-gateway/%s", DefaultRegion, DefaultAccountID, gatewayID),
		OwnerID:      DefaultAccountID,
		State:        "available",
		Options: TransitGatewayOptions{
			AmazonSideASN:                   firstNonNilInt64(options.AmazonSideASN, 64512),
			AssociationDefaultRouteTableID:  defaultRouteTableID,
			AutoAcceptSharedAttachments:     normalizeOptionValue(options.AutoAcceptSharedAttachments, "disable"),
			DefaultRouteTableAssociation:    normalizeOptionValue(options.DefaultRouteTableAssociation, "enable"),
			DefaultRouteTablePropagation:    normalizeOptionValue(options.DefaultRouteTablePropagation, "enable"),
			DnsSupport:                      normalizeOptionValue(options.DnsSupport, "enable"),
			MulticastSupport:                normalizeOptionValue(options.MulticastSupport, "disable"),
			PropagationDefaultRouteTableID:  defaultRouteTableID,
			SecurityGroupReferencingSupport: normalizeOptionValue(options.SecurityGroupReferencingSupport, "disable"),
			TransitGatewayCidrBlocks:        dedupeTrimmedStrings(options.TransitGatewayCidrBlocks),
			VpnEcmpSupport:                  normalizeOptionValue(options.VpnEcmpSupport, "enable"),
		},
		Tags: tagsToMap(tags),
	}
	s.transitGateways[gateway.ID] = gateway

	s.transitGatewayRouteTables[defaultRouteTableID] = &TransitGatewayRouteTable{
		CreationTime:                 now,
		DefaultAssociationRouteTable: true,
		DefaultPropagationRouteTable: true,
		ID:                           defaultRouteTableID,
		State:                        "available",
		Tags:                         map[string]string{},
		TransitID:                    gateway.ID,
	}

	return cloneTransitGateway(gateway), nil
}

func (s *Service) DeleteTransitGateway(transitGatewayID string) (TransitGateway, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if transitGatewayID == "" {
		return TransitGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.transitGateways[transitGatewayID]
	if gateway == nil {
		return TransitGateway{}, ErrNotFound
	}

	out := cloneTransitGateway(gateway)
	out.State = "deleted"
	delete(s.transitGateways, transitGatewayID)

	for routeTableID, routeTable := range s.transitGatewayRouteTables {
		if routeTable.TransitID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayRouteTables, routeTableID)
		s.deleteTransitGatewayRouteTableAnnouncementsForRouteTableLocked(routeTableID)
		for key, association := range s.transitGatewayRouteTableAssocs {
			if association.TransitGatewayRouteTableID == routeTableID {
				delete(s.transitGatewayRouteTableAssocs, key)
			}
		}
		for key, propagation := range s.transitGatewayPropagations {
			if propagation.TransitGatewayRouteTableID == routeTableID {
				delete(s.transitGatewayPropagations, key)
			}
		}
		for key, reference := range s.transitGatewayPrefixListReferences {
			if reference.TransitGatewayRouteTableID == routeTableID {
				delete(s.transitGatewayPrefixListReferences, key)
			}
		}
	}

	for policyTableID, policyTable := range s.transitGatewayPolicyTables {
		if policyTable.TransitID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayPolicyTables, policyTableID)
		for key, association := range s.transitGatewayPolicyTableAssocs {
			if association.TransitGatewayPolicyTableID == policyTableID {
				delete(s.transitGatewayPolicyTableAssocs, key)
			}
		}
	}

	for multicastDomainID, domain := range s.transitGatewayMulticastDomains {
		if domain.TransitID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayMulticastDomains, multicastDomainID)
		s.deleteTransitGatewayMulticastGroupsForDomainLocked(multicastDomainID)
		for key, association := range s.transitGatewayMulticastDomainAssocs {
			if association.TransitGatewayMulticastDomainID == multicastDomainID {
				delete(s.transitGatewayMulticastDomainAssocs, key)
			}
		}
	}

	for attachmentID, attachment := range s.transitGatewayVpcAttachments {
		if attachment.TransitGatewayID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayVpcAttachments, attachmentID)
		s.deleteTransitGatewayAttachmentReferencesLocked(attachmentID)
	}

	for attachmentID, peering := range s.transitGatewayPeeringAttachments {
		if peering.RequesterTgwInfo.TransitGatewayID != transitGatewayID && peering.AccepterTgwInfo.TransitGatewayID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayPeeringAttachments, attachmentID)
		s.deleteTransitGatewayAttachmentReferencesLocked(attachmentID)
	}

	for connectID, connect := range s.transitGatewayConnects {
		if connect.TransitGatewayID != transitGatewayID {
			continue
		}
		delete(s.transitGatewayConnects, connectID)
		s.deleteTransitGatewayAttachmentReferencesLocked(connectID)
	}

	return out, nil
}

func (s *Service) DescribeTransitGateways(
	transitGatewayIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGateway, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayIDs), standard["transit-gateway-id"]...))
	ownerIDSet := toStringSet(standard["owner-id"])
	stateSet := toLowerStringSet(standard["state"])
	assocDefaultRouteTableIDSet := toStringSet(standard["options.association-default-route-table-id"])
	propDefaultRouteTableIDSet := toStringSet(standard["options.propagation-default-route-table-id"])
	amazonSideASNSet := toStringSet(standard["options.amazon-side-asn"])
	autoAcceptSet := toLowerStringSet(standard["options.auto-accept-shared-attachments"])
	defaultAssociationSet := toLowerStringSet(standard["options.default-route-table-association"])
	defaultPropagationSet := toLowerStringSet(standard["options.default-route-table-propagation"])
	dnsSupportSet := toLowerStringSet(standard["options.dns-support"])
	multicastSupportSet := toLowerStringSet(standard["options.multicast-support"])
	securityGroupRefSet := toLowerStringSet(standard["options.security-group-referencing-support"])
	transitGatewayCIDRBlockSet := toStringSet(standard["options.transit-gateway-cidr-block"])
	vpnEcmpSupportSet := toLowerStringSet(standard["options.vpn-ecmp-support"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGateway, 0, len(s.transitGateways))
	for _, gateway := range s.transitGateways {
		if len(idSet) > 0 {
			if _, ok := idSet[gateway.ID]; !ok {
				continue
			}
		}
		if len(ownerIDSet) > 0 {
			if _, ok := ownerIDSet[gateway.OwnerID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(gateway.State)]; !ok {
				continue
			}
		}
		if len(assocDefaultRouteTableIDSet) > 0 {
			if _, ok := assocDefaultRouteTableIDSet[gateway.Options.AssociationDefaultRouteTableID]; !ok {
				continue
			}
		}
		if len(propDefaultRouteTableIDSet) > 0 {
			if _, ok := propDefaultRouteTableIDSet[gateway.Options.PropagationDefaultRouteTableID]; !ok {
				continue
			}
		}
		if len(amazonSideASNSet) > 0 {
			if _, ok := amazonSideASNSet[strconv.FormatInt(gateway.Options.AmazonSideASN, 10)]; !ok {
				continue
			}
		}
		if len(autoAcceptSet) > 0 {
			if _, ok := autoAcceptSet[strings.ToLower(gateway.Options.AutoAcceptSharedAttachments)]; !ok {
				continue
			}
		}
		if len(defaultAssociationSet) > 0 {
			if _, ok := defaultAssociationSet[strings.ToLower(gateway.Options.DefaultRouteTableAssociation)]; !ok {
				continue
			}
		}
		if len(defaultPropagationSet) > 0 {
			if _, ok := defaultPropagationSet[strings.ToLower(gateway.Options.DefaultRouteTablePropagation)]; !ok {
				continue
			}
		}
		if len(dnsSupportSet) > 0 {
			if _, ok := dnsSupportSet[strings.ToLower(gateway.Options.DnsSupport)]; !ok {
				continue
			}
		}
		if len(multicastSupportSet) > 0 {
			if _, ok := multicastSupportSet[strings.ToLower(gateway.Options.MulticastSupport)]; !ok {
				continue
			}
		}
		if len(securityGroupRefSet) > 0 {
			if _, ok := securityGroupRefSet[strings.ToLower(gateway.Options.SecurityGroupReferencingSupport)]; !ok {
				continue
			}
		}
		if len(transitGatewayCIDRBlockSet) > 0 {
			if !containsAnyString(gateway.Options.TransitGatewayCidrBlocks, transitGatewayCIDRBlockSet) {
				continue
			}
		}
		if len(vpnEcmpSupportSet) > 0 {
			if _, ok := vpnEcmpSupportSet[strings.ToLower(gateway.Options.VpnEcmpSupport)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(gateway.Tags, tagKeys, tagFilters) {
			continue
		}
		out = append(out, cloneTransitGateway(gateway))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGateway(nil), out[start:end]...), outputToken, nil
}

func (s *Service) CreateTransitGatewayVpcAttachment(
	transitGatewayID,
	vpcID string,
	subnetIDs []string,
	options TransitGatewayVpcAttachmentOptionsInput,
	tags []Tag,
) (TransitGatewayVpcAttachment, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	vpcID = strings.TrimSpace(vpcID)
	if transitGatewayID == "" || vpcID == "" {
		return TransitGatewayVpcAttachment{}, ErrInvalidParameter
	}

	resolvedSubnetIDs := dedupeTrimmedStrings(subnetIDs)
	if len(resolvedSubnetIDs) == 0 {
		return TransitGatewayVpcAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGateways[transitGatewayID] == nil || s.vpcs[vpcID] == nil {
		return TransitGatewayVpcAttachment{}, ErrNotFound
	}
	for _, subnetID := range resolvedSubnetIDs {
		subnet := s.subnets[subnetID]
		if subnet == nil || subnet.VpcID != vpcID {
			return TransitGatewayVpcAttachment{}, ErrNotFound
		}
	}

	attachment := &TransitGatewayVpcAttachment{
		CreationTime: time.Now().UTC(),
		ID:           s.nextIDLocked("tgw-attach"),
		Options: TransitGatewayVpcAttachmentOptions{
			ApplianceModeSupport:            normalizeOptionValue(options.ApplianceModeSupport, "disable"),
			DnsSupport:                      normalizeOptionValue(options.DnsSupport, "enable"),
			Ipv6Support:                     normalizeOptionValue(options.Ipv6Support, "disable"),
			SecurityGroupReferencingSupport: normalizeOptionValue(options.SecurityGroupReferencingSupport, "disable"),
		},
		State:            "available",
		SubnetIDs:        append([]string(nil), resolvedSubnetIDs...),
		Tags:             tagsToMap(tags),
		TransitGatewayID: transitGatewayID,
		VpcID:            vpcID,
		VpcOwnerID:       DefaultAccountID,
	}
	s.transitGatewayVpcAttachments[attachment.ID] = attachment
	return cloneTransitGatewayVpcAttachment(attachment), nil
}

func (s *Service) DeleteTransitGatewayVpcAttachment(transitGatewayAttachmentID string) (TransitGatewayVpcAttachment, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayVpcAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	attachment := s.transitGatewayVpcAttachments[transitGatewayAttachmentID]
	if attachment == nil {
		return TransitGatewayVpcAttachment{}, ErrNotFound
	}

	out := cloneTransitGatewayVpcAttachment(attachment)
	out.State = "deleted"
	delete(s.transitGatewayVpcAttachments, transitGatewayAttachmentID)
	for connectID, connect := range s.transitGatewayConnects {
		if connect.TransportTransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayConnects, connectID)
			s.deleteTransitGatewayAttachmentReferencesLocked(connectID)
		}
	}
	s.deleteTransitGatewayAttachmentReferencesLocked(transitGatewayAttachmentID)

	return out, nil
}

func (s *Service) DescribeTransitGatewayVpcAttachments(
	transitGatewayAttachmentIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayVpcAttachment, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayAttachmentIDs), standard["transit-gateway-attachment-id"]...))
	stateSet := toLowerStringSet(standard["state"])
	transitGatewayIDSet := toStringSet(standard["transit-gateway-id"])
	vpcIDSet := toStringSet(standard["vpc-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayVpcAttachment, 0, len(s.transitGatewayVpcAttachments))
	for _, attachment := range s.transitGatewayVpcAttachments {
		if len(idSet) > 0 {
			if _, ok := idSet[attachment.ID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(attachment.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[attachment.TransitGatewayID]; !ok {
				continue
			}
		}
		if len(vpcIDSet) > 0 {
			if _, ok := vpcIDSet[attachment.VpcID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(attachment.Tags, tagKeys, tagFilters) {
			continue
		}
		out = append(out, cloneTransitGatewayVpcAttachment(attachment))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayVpcAttachment(nil), out[start:end]...), outputToken, nil
}

func (s *Service) CreateTransitGatewayPeeringAttachment(
	transitGatewayID,
	peerTransitGatewayID,
	peerAccountID,
	peerRegion string,
	options TransitGatewayPeeringAttachmentOptionsInput,
	tags []Tag,
) (TransitGatewayPeeringAttachment, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	peerTransitGatewayID = strings.TrimSpace(peerTransitGatewayID)
	peerAccountID = strings.TrimSpace(peerAccountID)
	peerRegion = strings.TrimSpace(peerRegion)
	if transitGatewayID == "" || peerTransitGatewayID == "" || peerAccountID == "" || peerRegion == "" {
		return TransitGatewayPeeringAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGateways[transitGatewayID] == nil {
		return TransitGatewayPeeringAttachment{}, ErrNotFound
	}

	attachment := &TransitGatewayPeeringAttachment{
		AccepterTgwInfo: TransitGatewayPeeringInfo{
			OwnerID:          peerAccountID,
			Region:           peerRegion,
			TransitGatewayID: peerTransitGatewayID,
		},
		AccepterTransitGatewayAttachmentID: s.nextIDLocked("tgw-attach"),
		CreationTime:                       time.Now().UTC(),
		ID:                                 s.nextIDLocked("tgw-attach"),
		Options: TransitGatewayPeeringAttachmentOptions{
			DynamicRouting: normalizeOptionValue(options.DynamicRouting, "disable"),
		},
		RequesterTgwInfo: TransitGatewayPeeringInfo{
			OwnerID:          DefaultAccountID,
			Region:           DefaultRegion,
			TransitGatewayID: transitGatewayID,
		},
		State: "available",
		Tags:  tagsToMap(tags),
	}
	s.transitGatewayPeeringAttachments[attachment.ID] = attachment
	return cloneTransitGatewayPeeringAttachment(attachment), nil
}

func (s *Service) DeleteTransitGatewayPeeringAttachment(transitGatewayAttachmentID string) (TransitGatewayPeeringAttachment, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayPeeringAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	attachment := s.transitGatewayPeeringAttachments[transitGatewayAttachmentID]
	if attachment == nil {
		return TransitGatewayPeeringAttachment{}, ErrNotFound
	}

	out := cloneTransitGatewayPeeringAttachment(attachment)
	out.State = "deleted"
	delete(s.transitGatewayPeeringAttachments, transitGatewayAttachmentID)
	s.deleteTransitGatewayAttachmentReferencesLocked(transitGatewayAttachmentID)

	return out, nil
}

func (s *Service) DescribeTransitGatewayPeeringAttachments(
	transitGatewayAttachmentIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayPeeringAttachment, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayAttachmentIDs), standard["transit-gateway-attachment-id"]...))
	localOwnerSet := toStringSet(standard["local-owner-id"])
	remoteOwnerSet := toStringSet(standard["remote-owner-id"])
	stateSet := toLowerStringSet(standard["state"])
	transitGatewayIDSet := toStringSet(standard["transit-gateway-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayPeeringAttachment, 0, len(s.transitGatewayPeeringAttachments))
	for _, attachment := range s.transitGatewayPeeringAttachments {
		if len(idSet) > 0 {
			if _, ok := idSet[attachment.ID]; !ok {
				continue
			}
		}
		if len(localOwnerSet) > 0 {
			if _, ok := localOwnerSet[attachment.RequesterTgwInfo.OwnerID]; !ok {
				continue
			}
		}
		if len(remoteOwnerSet) > 0 {
			if _, ok := remoteOwnerSet[attachment.AccepterTgwInfo.OwnerID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(attachment.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, requester := transitGatewayIDSet[attachment.RequesterTgwInfo.TransitGatewayID]; !requester {
				if _, accepter := transitGatewayIDSet[attachment.AccepterTgwInfo.TransitGatewayID]; !accepter {
					continue
				}
			}
		}
		if !matchesTagFilters(attachment.Tags, tagKeys, tagFilters) {
			continue
		}
		out = append(out, cloneTransitGatewayPeeringAttachment(attachment))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayPeeringAttachment(nil), out[start:end]...), outputToken, nil
}

func (s *Service) CreateTransitGatewayConnect(
	transportTransitGatewayAttachmentID string,
	options TransitGatewayConnectOptionsInput,
	tags []Tag,
) (TransitGatewayConnect, error) {
	transportTransitGatewayAttachmentID = strings.TrimSpace(transportTransitGatewayAttachmentID)
	if transportTransitGatewayAttachmentID == "" {
		return TransitGatewayConnect{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	transitGatewayID := s.findTransitGatewayIDForAttachmentLocked(transportTransitGatewayAttachmentID)
	if transitGatewayID == "" {
		return TransitGatewayConnect{}, ErrNotFound
	}

	connect := &TransitGatewayConnect{
		CreationTime: time.Now().UTC(),
		ID:           s.nextIDLocked("tgw-attach"),
		Options: TransitGatewayConnectOptions{
			Protocol: normalizeOptionValue(options.Protocol, "gre"),
		},
		State:                               "available",
		Tags:                                tagsToMap(tags),
		TransitGatewayID:                    transitGatewayID,
		TransportTransitGatewayAttachmentID: transportTransitGatewayAttachmentID,
	}
	s.transitGatewayConnects[connect.ID] = connect
	return cloneTransitGatewayConnect(connect), nil
}

func (s *Service) DeleteTransitGatewayConnect(transitGatewayAttachmentID string) (TransitGatewayConnect, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayConnect{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connect := s.transitGatewayConnects[transitGatewayAttachmentID]
	if connect == nil {
		return TransitGatewayConnect{}, ErrNotFound
	}
	for _, peer := range s.transitGatewayConnectPeers {
		if peer.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			return TransitGatewayConnect{}, ErrConflict
		}
	}

	out := cloneTransitGatewayConnect(connect)
	out.State = "deleted"
	delete(s.transitGatewayConnects, transitGatewayAttachmentID)
	s.deleteTransitGatewayAttachmentReferencesLocked(transitGatewayAttachmentID)

	return out, nil
}

func (s *Service) DescribeTransitGatewayConnects(
	transitGatewayAttachmentIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayConnect, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayAttachmentIDs), standard["transit-gateway-attachment-id"]...))
	protocolSet := toLowerStringSet(standard["options.protocol"])
	stateSet := toLowerStringSet(standard["state"])
	transitGatewayIDSet := toStringSet(standard["transit-gateway-id"])
	transportAttachmentIDSet := toStringSet(standard["transport-transit-gateway-attachment-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayConnect, 0, len(s.transitGatewayConnects))
	for _, connect := range s.transitGatewayConnects {
		if len(idSet) > 0 {
			if _, ok := idSet[connect.ID]; !ok {
				continue
			}
		}
		if len(protocolSet) > 0 {
			if _, ok := protocolSet[strings.ToLower(connect.Options.Protocol)]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(connect.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[connect.TransitGatewayID]; !ok {
				continue
			}
		}
		if len(transportAttachmentIDSet) > 0 {
			if _, ok := transportAttachmentIDSet[connect.TransportTransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(connect.Tags, tagKeys, tagFilters) {
			continue
		}
		out = append(out, cloneTransitGatewayConnect(connect))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayConnect(nil), out[start:end]...), outputToken, nil
}

func (s *Service) CreateTransitGatewayConnectPeer(
	transitGatewayAttachmentID string,
	input TransitGatewayConnectPeerInput,
	tags []Tag,
) (TransitGatewayConnectPeer, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	input.PeerAddress = strings.TrimSpace(input.PeerAddress)
	if transitGatewayAttachmentID == "" || input.PeerAddress == "" {
		return TransitGatewayConnectPeer{}, ErrInvalidParameter
	}

	insideCIDRBlocks := dedupeTrimmedStrings(input.InsideCidrBlocks)
	if len(insideCIDRBlocks) == 0 {
		return TransitGatewayConnectPeer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	connect := s.transitGatewayConnects[transitGatewayAttachmentID]
	if connect == nil {
		return TransitGatewayConnectPeer{}, ErrNotFound
	}

	transitGatewayAddress := "169.254.100.1"
	if input.TransitGatewayAddress != nil {
		if trimmed := strings.TrimSpace(*input.TransitGatewayAddress); trimmed != "" {
			transitGatewayAddress = trimmed
		}
	}
	peerASN := firstNonNilInt64(input.PeerASN, 65000)
	transitGatewayASN := int64(64512)
	if gateway := s.transitGateways[connect.TransitGatewayID]; gateway != nil && gateway.Options.AmazonSideASN > 0 {
		transitGatewayASN = gateway.Options.AmazonSideASN
	}

	peer := &TransitGatewayConnectPeer{
		Configuration: TransitGatewayConnectPeerConfiguration{
			BgpConfigurations: []TransitGatewayAttachmentBgpConfiguration{
				{
					BgpStatus:             "up",
					PeerAddress:           input.PeerAddress,
					PeerASN:               peerASN,
					TransitGatewayAddress: transitGatewayAddress,
					TransitGatewayASN:     transitGatewayASN,
				},
			},
			InsideCidrBlocks:      append([]string(nil), insideCIDRBlocks...),
			PeerAddress:           input.PeerAddress,
			Protocol:              connect.Options.Protocol,
			TransitGatewayAddress: transitGatewayAddress,
		},
		CreationTime:                time.Now().UTC(),
		State:                       "available",
		Tags:                        tagsToMap(tags),
		TransitGatewayAttachmentID:  transitGatewayAttachmentID,
		TransitGatewayConnectPeerID: s.nextIDLocked("tgw-connect-peer"),
	}
	s.transitGatewayConnectPeers[peer.TransitGatewayConnectPeerID] = peer
	return cloneTransitGatewayConnectPeer(peer), nil
}

func (s *Service) DeleteTransitGatewayConnectPeer(transitGatewayConnectPeerID string) (TransitGatewayConnectPeer, error) {
	transitGatewayConnectPeerID = strings.TrimSpace(transitGatewayConnectPeerID)
	if transitGatewayConnectPeerID == "" {
		return TransitGatewayConnectPeer{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	peer := s.transitGatewayConnectPeers[transitGatewayConnectPeerID]
	if peer == nil {
		return TransitGatewayConnectPeer{}, ErrNotFound
	}

	out := cloneTransitGatewayConnectPeer(peer)
	out.State = "deleted"
	delete(s.transitGatewayConnectPeers, transitGatewayConnectPeerID)
	return out, nil
}

func (s *Service) DescribeTransitGatewayConnectPeers(
	transitGatewayConnectPeerIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayConnectPeer, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayConnectPeerIDs), standard["transit-gateway-connect-peer-id"]...))
	stateSet := toLowerStringSet(standard["state"])
	attachmentIDSet := toStringSet(standard["transit-gateway-attachment-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayConnectPeer, 0, len(s.transitGatewayConnectPeers))
	for _, peer := range s.transitGatewayConnectPeers {
		if len(idSet) > 0 {
			if _, ok := idSet[peer.TransitGatewayConnectPeerID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(peer.State)]; !ok {
				continue
			}
		}
		if len(attachmentIDSet) > 0 {
			if _, ok := attachmentIDSet[peer.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(peer.Tags, tagKeys, tagFilters) {
			continue
		}
		out = append(out, cloneTransitGatewayConnectPeer(peer))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayConnectPeerID < out[j].TransitGatewayConnectPeerID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayConnectPeer(nil), out[start:end]...), outputToken, nil
}

func (s *Service) findTransitGatewayIDForAttachmentLocked(transitGatewayAttachmentID string) string {
	if attachment := s.transitGatewayVpcAttachments[transitGatewayAttachmentID]; attachment != nil {
		return attachment.TransitGatewayID
	}
	if attachment := s.transitGatewayPeeringAttachments[transitGatewayAttachmentID]; attachment != nil {
		return attachment.RequesterTgwInfo.TransitGatewayID
	}
	return ""
}

func (s *Service) deleteTransitGatewayAttachmentReferencesLocked(transitGatewayAttachmentID string) {
	s.deleteTransitGatewayRouteTableAnnouncementsForPeeringAttachmentLocked(transitGatewayAttachmentID)

	for key, association := range s.transitGatewayPolicyTableAssocs {
		if association.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayPolicyTableAssocs, key)
		}
	}
	for key, association := range s.transitGatewayRouteTableAssocs {
		if association.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayRouteTableAssocs, key)
		}
	}
	for key, association := range s.transitGatewayMulticastDomainAssocs {
		if association.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayMulticastDomainAssocs, key)
		}
	}
	for key, propagation := range s.transitGatewayPropagations {
		if propagation.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayPropagations, key)
		}
	}
	for key, reference := range s.transitGatewayPrefixListReferences {
		if reference.TransitGatewayAttachment != nil && reference.TransitGatewayAttachment.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayPrefixListReferences, key)
		}
	}
	for peerID, peer := range s.transitGatewayConnectPeers {
		if peer.TransitGatewayAttachmentID == transitGatewayAttachmentID {
			delete(s.transitGatewayConnectPeers, peerID)
		}
	}
}

func cloneTransitGateway(in *TransitGateway) TransitGateway {
	if in == nil {
		return TransitGateway{}
	}
	out := *in
	out.Options = cloneTransitGatewayOptions(in.Options)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayOptions(in TransitGatewayOptions) TransitGatewayOptions {
	out := in
	out.TransitGatewayCidrBlocks = append([]string(nil), in.TransitGatewayCidrBlocks...)
	return out
}

func cloneTransitGatewayVpcAttachment(in *TransitGatewayVpcAttachment) TransitGatewayVpcAttachment {
	if in == nil {
		return TransitGatewayVpcAttachment{}
	}
	out := *in
	out.SubnetIDs = append([]string(nil), in.SubnetIDs...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayPeeringAttachment(in *TransitGatewayPeeringAttachment) TransitGatewayPeeringAttachment {
	if in == nil {
		return TransitGatewayPeeringAttachment{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayConnect(in *TransitGatewayConnect) TransitGatewayConnect {
	if in == nil {
		return TransitGatewayConnect{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayConnectPeer(in *TransitGatewayConnectPeer) TransitGatewayConnectPeer {
	if in == nil {
		return TransitGatewayConnectPeer{}
	}
	out := *in
	out.Configuration = cloneTransitGatewayConnectPeerConfiguration(in.Configuration)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneTransitGatewayConnectPeerConfiguration(in TransitGatewayConnectPeerConfiguration) TransitGatewayConnectPeerConfiguration {
	out := in
	out.BgpConfigurations = append([]TransitGatewayAttachmentBgpConfiguration(nil), in.BgpConfigurations...)
	out.InsideCidrBlocks = append([]string(nil), in.InsideCidrBlocks...)
	return out
}

func normalizeOptionValue(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if trimmed == "" {
		return defaultValue
	}
	return trimmed
}

func firstNonNilInt64(value *int64, defaultValue int64) int64 {
	if value == nil {
		return defaultValue
	}
	return *value
}

func containsAnyString(values []string, filterSet map[string]struct{}) bool {
	if len(filterSet) == 0 {
		return true
	}
	for _, value := range values {
		if _, ok := filterSet[strings.TrimSpace(value)]; ok {
			return true
		}
	}
	return false
}

func splitEC2Filters(filters map[string][]string) (map[string][]string, []string, map[string][]string) {
	standard := map[string][]string{}
	tagKeyValues := []string{}
	tagFilters := map[string][]string{}

	for name, values := range filters {
		cleanedValues := dedupeTrimmedStrings(values)
		if len(cleanedValues) == 0 {
			continue
		}
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}
		lowerName := strings.ToLower(trimmedName)
		if lowerName == "tag-key" {
			tagKeyValues = append(tagKeyValues, cleanedValues...)
			continue
		}
		if strings.HasPrefix(lowerName, "tag:") {
			colonIndex := strings.Index(trimmedName, ":")
			if colonIndex < 0 || colonIndex == len(trimmedName)-1 {
				continue
			}
			tagKey := strings.TrimSpace(trimmedName[colonIndex+1:])
			if tagKey == "" {
				continue
			}
			tagFilters[tagKey] = append(tagFilters[tagKey], cleanedValues...)
			continue
		}
		standard[lowerName] = append(standard[lowerName], cleanedValues...)
	}

	for key, values := range standard {
		standard[key] = dedupeTrimmedStrings(values)
	}
	for key, values := range tagFilters {
		tagFilters[key] = dedupeTrimmedStrings(values)
	}

	return standard, dedupeTrimmedStrings(tagKeyValues), tagFilters
}

func matchesTagFilters(tags map[string]string, tagKeys []string, tagFilters map[string][]string) bool {
	if len(tagKeys) > 0 {
		if len(tags) == 0 {
			return false
		}
		matchedKey := false
		for _, key := range tagKeys {
			if _, ok := tags[key]; ok {
				matchedKey = true
				break
			}
		}
		if !matchedKey {
			return false
		}
	}

	for key, values := range tagFilters {
		tagValue, ok := tags[key]
		if !ok {
			return false
		}
		if len(values) == 0 {
			continue
		}
		if _, ok := toStringSet(values)[tagValue]; !ok {
			return false
		}
	}

	return true
}

func ec2PageStart(nextToken *string) (int, error) {
	start := 0
	if nextToken == nil {
		return start, nil
	}
	token := strings.TrimSpace(*nextToken)
	if token == "" {
		return start, nil
	}
	parsed, err := strconv.Atoi(token)
	if err != nil || parsed < 0 {
		return 0, ErrInvalidParameter
	}
	return parsed, nil
}

func ec2PageWindow(total, start int, maxResults *int32) (int, int, *string, error) {
	if start > total {
		return 0, 0, nil, ErrInvalidParameter
	}
	end := total
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > total {
			end = total
		}
	}
	var outputToken *string
	if end < total {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return start, end, outputToken, nil
}
