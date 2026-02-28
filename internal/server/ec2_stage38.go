package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage38Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateTransitGateway":
		amazonSideASN, ok := parseEC2OptionalInt64(r.Form.Get("Options.AmazonSideAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGateway, err := s.ec2.CreateTransitGateway(
			strings.TrimSpace(r.Form.Get("Description")),
			ec2svc.TransitGatewayOptionsInput{
				AmazonSideASN:                   amazonSideASN,
				AutoAcceptSharedAttachments:     parseEC2OptionalString(r.Form.Get("Options.AutoAcceptSharedAttachments")),
				DefaultRouteTableAssociation:    parseEC2OptionalString(r.Form.Get("Options.DefaultRouteTableAssociation")),
				DefaultRouteTablePropagation:    parseEC2OptionalString(r.Form.Get("Options.DefaultRouteTablePropagation")),
				DnsSupport:                      parseEC2OptionalString(r.Form.Get("Options.DnsSupport")),
				MulticastSupport:                parseEC2OptionalString(r.Form.Get("Options.MulticastSupport")),
				SecurityGroupReferencingSupport: parseEC2OptionalString(r.Form.Get("Options.SecurityGroupReferencingSupport")),
				TransitGatewayCidrBlocks:        parseEC2MembersOrItemList(r.Form, "Options.TransitGatewayCidrBlocks"),
				VpnEcmpSupport:                  parseEC2OptionalString(r.Form.Get("Options.VpnEcmpSupport")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayResponse{
			XMLName:        xml.Name{Local: "CreateTransitGatewayResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			TransitGateway: ec2TransitGatewayItemFrom(transitGateway),
		})
		return true
	case "DeleteTransitGateway":
		transitGateway, err := s.ec2.DeleteTransitGateway(strings.TrimSpace(r.Form.Get("TransitGatewayId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayResponse{
			XMLName:        xml.Name{Local: "DeleteTransitGatewayResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			TransitGateway: ec2TransitGatewayItemFrom(transitGateway),
		})
		return true
	case "DescribeTransitGateways":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGateways, nextToken, err := s.ec2.DescribeTransitGateways(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewaysResponse{
			XMLName:           xml.Name{Local: "DescribeTransitGatewaysResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			TransitGatewaySet: ec2TransitGatewaySet{Items: ec2TransitGatewayItems(transitGateways)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateTransitGatewayVpcAttachment":
		transitGatewayVpcAttachment, err := s.ec2.CreateTransitGatewayVpcAttachment(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2MembersOrItemList(r.Form, "SubnetIds"),
			ec2svc.TransitGatewayVpcAttachmentOptionsInput{
				ApplianceModeSupport:            parseEC2OptionalString(r.Form.Get("Options.ApplianceModeSupport")),
				DnsSupport:                      parseEC2OptionalString(r.Form.Get("Options.DnsSupport")),
				Ipv6Support:                     parseEC2OptionalString(r.Form.Get("Options.Ipv6Support")),
				SecurityGroupReferencingSupport: parseEC2OptionalString(r.Form.Get("Options.SecurityGroupReferencingSupport")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-attachment"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayVpcAttachmentResponse{
			XMLName:                     xml.Name{Local: "CreateTransitGatewayVpcAttachmentResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			TransitGatewayVpcAttachment: ec2TransitGatewayVpcAttachmentItemFrom(transitGatewayVpcAttachment),
		})
		return true
	case "DeleteTransitGatewayVpcAttachment":
		transitGatewayVpcAttachment, err := s.ec2.DeleteTransitGatewayVpcAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayVpcAttachmentResponse{
			XMLName:                     xml.Name{Local: "DeleteTransitGatewayVpcAttachmentResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			TransitGatewayVpcAttachment: ec2TransitGatewayVpcAttachmentItemFrom(transitGatewayVpcAttachment),
		})
		return true
	case "DescribeTransitGatewayVpcAttachments":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGatewayVpcAttachments, nextToken, err := s.ec2.DescribeTransitGatewayVpcAttachments(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayAttachmentIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayVpcAttachmentsResponse{
			XMLName:                      xml.Name{Local: "DescribeTransitGatewayVpcAttachmentsResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			TransitGatewayVpcAttachments: ec2TransitGatewayVpcAttachmentSet{Items: ec2TransitGatewayVpcAttachmentItems(transitGatewayVpcAttachments)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateTransitGatewayPeeringAttachment":
		transitGatewayPeeringAttachment, err := s.ec2.CreateTransitGatewayPeeringAttachment(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			strings.TrimSpace(r.Form.Get("PeerTransitGatewayId")),
			strings.TrimSpace(r.Form.Get("PeerAccountId")),
			strings.TrimSpace(r.Form.Get("PeerRegion")),
			ec2svc.TransitGatewayPeeringAttachmentOptionsInput{
				DynamicRouting: parseEC2OptionalString(r.Form.Get("Options.DynamicRouting")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-attachment"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayPeeringAttachmentResponse{
			XMLName:                         xml.Name{Local: "CreateTransitGatewayPeeringAttachmentResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			TransitGatewayPeeringAttachment: ec2TransitGatewayPeeringAttachmentItemFrom(transitGatewayPeeringAttachment),
		})
		return true
	case "DeleteTransitGatewayPeeringAttachment":
		transitGatewayPeeringAttachment, err := s.ec2.DeleteTransitGatewayPeeringAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayPeeringAttachmentResponse{
			XMLName:                         xml.Name{Local: "DeleteTransitGatewayPeeringAttachmentResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			TransitGatewayPeeringAttachment: ec2TransitGatewayPeeringAttachmentItemFrom(transitGatewayPeeringAttachment),
		})
		return true
	case "DescribeTransitGatewayPeeringAttachments":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGatewayPeeringAttachments, nextToken, err := s.ec2.DescribeTransitGatewayPeeringAttachments(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayAttachmentIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayPeeringAttachmentsResponse{
			XMLName:                          xml.Name{Local: "DescribeTransitGatewayPeeringAttachmentsResponse"},
			Xmlns:                            ec2Namespace,
			RequestID:                        "stackyard-request",
			TransitGatewayPeeringAttachments: ec2TransitGatewayPeeringAttachmentSet{Items: ec2TransitGatewayPeeringAttachmentItems(transitGatewayPeeringAttachments)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateTransitGatewayConnect":
		transitGatewayConnect, err := s.ec2.CreateTransitGatewayConnect(
			strings.TrimSpace(r.Form.Get("TransportTransitGatewayAttachmentId")),
			ec2svc.TransitGatewayConnectOptionsInput{
				Protocol: parseEC2OptionalString(r.Form.Get("Options.Protocol")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-attachment"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayConnectResponse{
			XMLName:               xml.Name{Local: "CreateTransitGatewayConnectResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			TransitGatewayConnect: ec2TransitGatewayConnectItemFrom(transitGatewayConnect),
		})
		return true
	case "DeleteTransitGatewayConnect":
		transitGatewayConnect, err := s.ec2.DeleteTransitGatewayConnect(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayConnectResponse{
			XMLName:               xml.Name{Local: "DeleteTransitGatewayConnectResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			TransitGatewayConnect: ec2TransitGatewayConnectItemFrom(transitGatewayConnect),
		})
		return true
	case "DescribeTransitGatewayConnects":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGatewayConnects, nextToken, err := s.ec2.DescribeTransitGatewayConnects(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayAttachmentIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayConnectsResponse{
			XMLName:                  xml.Name{Local: "DescribeTransitGatewayConnectsResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			TransitGatewayConnectSet: ec2TransitGatewayConnectSet{Items: ec2TransitGatewayConnectItems(transitGatewayConnects)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateTransitGatewayConnectPeer":
		peerASN, ok := parseEC2OptionalInt64(r.Form.Get("BgpOptions.PeerAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGatewayConnectPeer, err := s.ec2.CreateTransitGatewayConnectPeer(
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			ec2svc.TransitGatewayConnectPeerInput{
				InsideCidrBlocks:      parseEC2MembersOrItemList(r.Form, "InsideCidrBlocks"),
				PeerAddress:           strings.TrimSpace(r.Form.Get("PeerAddress")),
				PeerASN:               peerASN,
				TransitGatewayAddress: parseEC2OptionalString(r.Form.Get("TransitGatewayAddress")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "transit-gateway-connect-peer"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayConnectPeerResponse{
			XMLName:                   xml.Name{Local: "CreateTransitGatewayConnectPeerResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayConnectPeer: ec2TransitGatewayConnectPeerItemFrom(transitGatewayConnectPeer),
		})
		return true
	case "DeleteTransitGatewayConnectPeer":
		transitGatewayConnectPeer, err := s.ec2.DeleteTransitGatewayConnectPeer(strings.TrimSpace(r.Form.Get("TransitGatewayConnectPeerId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayConnectPeerResponse{
			XMLName:                   xml.Name{Local: "DeleteTransitGatewayConnectPeerResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayConnectPeer: ec2TransitGatewayConnectPeerItemFrom(transitGatewayConnectPeer),
		})
		return true
	case "DescribeTransitGatewayConnectPeers":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGatewayConnectPeers, nextToken, err := s.ec2.DescribeTransitGatewayConnectPeers(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayConnectPeerIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayConnectPeersResponse{
			XMLName:                      xml.Name{Local: "DescribeTransitGatewayConnectPeersResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			TransitGatewayConnectPeerSet: ec2TransitGatewayConnectPeerSet{Items: ec2TransitGatewayConnectPeerItems(transitGatewayConnectPeers)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2TransitGatewayItemFrom(in ec2svc.TransitGateway) ec2TransitGatewayItem {
	return ec2TransitGatewayItem{
		CreationTime: in.CreationTime,
		Description:  in.Description,
		Options: ec2TransitGatewayOptionsItem{
			AmazonSideAsn:                   in.Options.AmazonSideASN,
			AssociationDefaultRouteTableID:  in.Options.AssociationDefaultRouteTableID,
			AutoAcceptSharedAttachments:     in.Options.AutoAcceptSharedAttachments,
			DefaultRouteTableAssociation:    in.Options.DefaultRouteTableAssociation,
			DefaultRouteTablePropagation:    in.Options.DefaultRouteTablePropagation,
			DnsSupport:                      in.Options.DnsSupport,
			MulticastSupport:                in.Options.MulticastSupport,
			PropagationDefaultRouteTableID:  in.Options.PropagationDefaultRouteTableID,
			SecurityGroupReferencingSupport: in.Options.SecurityGroupReferencingSupport,
			TransitGatewayCidrBlocks:        ec2StringSet{Items: append([]string(nil), in.Options.TransitGatewayCidrBlocks...)},
			VpnEcmpSupport:                  in.Options.VpnEcmpSupport,
		},
		OwnerID:           in.OwnerID,
		State:             in.State,
		TagSet:            ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayArn: in.ARN,
		TransitGatewayID:  in.ID,
	}
}

func ec2TransitGatewayItems(in []ec2svc.TransitGateway) []ec2TransitGatewayItem {
	out := make([]ec2TransitGatewayItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TransitGatewayItemFrom(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayID < out[j].TransitGatewayID })
	return out
}

func ec2TransitGatewayVpcAttachmentItemFrom(in ec2svc.TransitGatewayVpcAttachment) ec2TransitGatewayVpcAttachmentItem {
	return ec2TransitGatewayVpcAttachmentItem{
		CreationTime: in.CreationTime,
		Options: ec2TransitGatewayVpcAttachmentOptionsItem{
			ApplianceModeSupport:            in.Options.ApplianceModeSupport,
			DnsSupport:                      in.Options.DnsSupport,
			Ipv6Support:                     in.Options.Ipv6Support,
			SecurityGroupReferencingSupport: in.Options.SecurityGroupReferencingSupport,
		},
		State:                      in.State,
		SubnetIDs:                  ec2StringSet{Items: append([]string(nil), in.SubnetIDs...)},
		TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayAttachmentID: in.ID,
		TransitGatewayID:           in.TransitGatewayID,
		VpcID:                      in.VpcID,
		VpcOwnerID:                 in.VpcOwnerID,
	}
}

func ec2TransitGatewayVpcAttachmentItems(in []ec2svc.TransitGatewayVpcAttachment) []ec2TransitGatewayVpcAttachmentItem {
	out := make([]ec2TransitGatewayVpcAttachmentItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TransitGatewayVpcAttachmentItemFrom(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })
	return out
}

func ec2TransitGatewayPeeringAttachmentItemFrom(in ec2svc.TransitGatewayPeeringAttachment) ec2TransitGatewayPeeringAttachmentItem {
	return ec2TransitGatewayPeeringAttachmentItem{
		AccepterTgwInfo: ec2PeeringTgwInfoItem{
			OwnerID:          in.AccepterTgwInfo.OwnerID,
			Region:           in.AccepterTgwInfo.Region,
			TransitGatewayID: in.AccepterTgwInfo.TransitGatewayID,
		},
		AccepterTransitGatewayAttachmentID: in.AccepterTransitGatewayAttachmentID,
		CreationTime:                       in.CreationTime,
		Options: ec2TransitGatewayPeeringAttachmentOptionsItem{
			DynamicRouting: in.Options.DynamicRouting,
		},
		RequesterTgwInfo: ec2PeeringTgwInfoItem{
			OwnerID:          in.RequesterTgwInfo.OwnerID,
			Region:           in.RequesterTgwInfo.Region,
			TransitGatewayID: in.RequesterTgwInfo.TransitGatewayID,
		},
		State:                      in.State,
		TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayAttachmentID: in.ID,
	}
}

func ec2TransitGatewayPeeringAttachmentItems(in []ec2svc.TransitGatewayPeeringAttachment) []ec2TransitGatewayPeeringAttachmentItem {
	out := make([]ec2TransitGatewayPeeringAttachmentItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TransitGatewayPeeringAttachmentItemFrom(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })
	return out
}

func ec2TransitGatewayConnectItemFrom(in ec2svc.TransitGatewayConnect) ec2TransitGatewayConnectItem {
	return ec2TransitGatewayConnectItem{
		CreationTime: in.CreationTime,
		Options: ec2TransitGatewayConnectOptionsItem{
			Protocol: in.Options.Protocol,
		},
		State:                               in.State,
		TagSet:                              ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayAttachmentID:          in.ID,
		TransitGatewayID:                    in.TransitGatewayID,
		TransportTransitGatewayAttachmentID: in.TransportTransitGatewayAttachmentID,
	}
}

func ec2TransitGatewayConnectItems(in []ec2svc.TransitGatewayConnect) []ec2TransitGatewayConnectItem {
	out := make([]ec2TransitGatewayConnectItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TransitGatewayConnectItemFrom(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })
	return out
}

func ec2TransitGatewayConnectPeerItemFrom(in ec2svc.TransitGatewayConnectPeer) ec2TransitGatewayConnectPeerItem {
	bgpConfigurations := make([]ec2TransitGatewayAttachmentBgpConfigurationItem, 0, len(in.Configuration.BgpConfigurations))
	for _, bgpConfiguration := range in.Configuration.BgpConfigurations {
		bgpConfigurations = append(bgpConfigurations, ec2TransitGatewayAttachmentBgpConfigurationItem{
			BgpStatus:             bgpConfiguration.BgpStatus,
			PeerAddress:           bgpConfiguration.PeerAddress,
			PeerAsn:               bgpConfiguration.PeerASN,
			TransitGatewayAddress: bgpConfiguration.TransitGatewayAddress,
			TransitGatewayAsn:     bgpConfiguration.TransitGatewayASN,
		})
	}

	return ec2TransitGatewayConnectPeerItem{
		ConnectPeerConfiguration: ec2TransitGatewayConnectPeerConfigurationItem{
			BgpConfigurations:     ec2TransitGatewayAttachmentBgpConfigurationSet{Items: bgpConfigurations},
			InsideCidrBlocks:      ec2StringSet{Items: append([]string(nil), in.Configuration.InsideCidrBlocks...)},
			PeerAddress:           in.Configuration.PeerAddress,
			Protocol:              in.Configuration.Protocol,
			TransitGatewayAddress: in.Configuration.TransitGatewayAddress,
		},
		CreationTime:                in.CreationTime,
		State:                       in.State,
		TagSet:                      ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayAttachmentID:  in.TransitGatewayAttachmentID,
		TransitGatewayConnectPeerID: in.TransitGatewayConnectPeerID,
	}
}

func ec2TransitGatewayConnectPeerItems(in []ec2svc.TransitGatewayConnectPeer) []ec2TransitGatewayConnectPeerItem {
	out := make([]ec2TransitGatewayConnectPeerItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2TransitGatewayConnectPeerItemFrom(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayConnectPeerID < out[j].TransitGatewayConnectPeerID })
	return out
}

func ec2TagItemsFromMap(tags map[string]string) []ec2TagItem {
	out := make([]ec2TagItem, 0, len(tags))
	for key, value := range tags {
		out = append(out, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func parseEC2Filters(values url.Values) map[string][]string {
	indexByName := map[int]string{}
	for key := range values {
		if !strings.HasPrefix(key, "Filter.") || !strings.HasSuffix(key, ".Name") {
			continue
		}
		rest := strings.TrimPrefix(key, "Filter.")
		rest = strings.TrimSuffix(rest, ".Name")
		index, err := strconv.Atoi(rest)
		if err != nil || index <= 0 {
			continue
		}
		name := strings.TrimSpace(values.Get(key))
		if name == "" {
			continue
		}
		indexByName[index] = name
	}

	ordered := make([]int, 0, len(indexByName))
	for index := range indexByName {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	out := map[string][]string{}
	for _, index := range ordered {
		name := indexByName[index]
		if name == "" {
			continue
		}
		out[name] = append(out[name], parseEC2Members(values, "Filter."+strconv.Itoa(index)+".Value.")...)
	}
	return out
}

type ec2CreateTransitGatewayResponse struct {
	XMLName        xml.Name              `xml:"CreateTransitGatewayResponse"`
	Xmlns          string                `xml:"xmlns,attr"`
	RequestID      string                `xml:"requestId"`
	TransitGateway ec2TransitGatewayItem `xml:"transitGateway"`
}

type ec2DeleteTransitGatewayResponse struct {
	XMLName        xml.Name              `xml:"DeleteTransitGatewayResponse"`
	Xmlns          string                `xml:"xmlns,attr"`
	RequestID      string                `xml:"requestId"`
	TransitGateway ec2TransitGatewayItem `xml:"transitGateway"`
}

type ec2DescribeTransitGatewaysResponse struct {
	XMLName           xml.Name             `xml:"DescribeTransitGatewaysResponse"`
	Xmlns             string               `xml:"xmlns,attr"`
	RequestID         string               `xml:"requestId"`
	NextToken         string               `xml:"nextToken,omitempty"`
	TransitGatewaySet ec2TransitGatewaySet `xml:"transitGatewaySet"`
}

type ec2TransitGatewaySet struct {
	Items []ec2TransitGatewayItem `xml:"item"`
}

type ec2TransitGatewayItem struct {
	CreationTime      time.Time                    `xml:"creationTime,omitempty"`
	Description       string                       `xml:"description,omitempty"`
	Options           ec2TransitGatewayOptionsItem `xml:"options"`
	OwnerID           string                       `xml:"ownerId,omitempty"`
	State             string                       `xml:"state,omitempty"`
	TagSet            ec2TagSet                    `xml:"tagSet"`
	TransitGatewayArn string                       `xml:"transitGatewayArn,omitempty"`
	TransitGatewayID  string                       `xml:"transitGatewayId,omitempty"`
}

type ec2TransitGatewayOptionsItem struct {
	AmazonSideAsn                   int64        `xml:"amazonSideAsn,omitempty"`
	AssociationDefaultRouteTableID  string       `xml:"associationDefaultRouteTableId,omitempty"`
	AutoAcceptSharedAttachments     string       `xml:"autoAcceptSharedAttachments,omitempty"`
	DefaultRouteTableAssociation    string       `xml:"defaultRouteTableAssociation,omitempty"`
	DefaultRouteTablePropagation    string       `xml:"defaultRouteTablePropagation,omitempty"`
	DnsSupport                      string       `xml:"dnsSupport,omitempty"`
	MulticastSupport                string       `xml:"multicastSupport,omitempty"`
	PropagationDefaultRouteTableID  string       `xml:"propagationDefaultRouteTableId,omitempty"`
	SecurityGroupReferencingSupport string       `xml:"securityGroupReferencingSupport,omitempty"`
	TransitGatewayCidrBlocks        ec2StringSet `xml:"transitGatewayCidrBlocks"`
	VpnEcmpSupport                  string       `xml:"vpnEcmpSupport,omitempty"`
}

type ec2CreateTransitGatewayVpcAttachmentResponse struct {
	XMLName                     xml.Name                           `xml:"CreateTransitGatewayVpcAttachmentResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	TransitGatewayVpcAttachment ec2TransitGatewayVpcAttachmentItem `xml:"transitGatewayVpcAttachment"`
}

type ec2DeleteTransitGatewayVpcAttachmentResponse struct {
	XMLName                     xml.Name                           `xml:"DeleteTransitGatewayVpcAttachmentResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	TransitGatewayVpcAttachment ec2TransitGatewayVpcAttachmentItem `xml:"transitGatewayVpcAttachment"`
}

type ec2DescribeTransitGatewayVpcAttachmentsResponse struct {
	XMLName                      xml.Name                          `xml:"DescribeTransitGatewayVpcAttachmentsResponse"`
	Xmlns                        string                            `xml:"xmlns,attr"`
	RequestID                    string                            `xml:"requestId"`
	NextToken                    string                            `xml:"nextToken,omitempty"`
	TransitGatewayVpcAttachments ec2TransitGatewayVpcAttachmentSet `xml:"transitGatewayVpcAttachments"`
}

type ec2TransitGatewayVpcAttachmentSet struct {
	Items []ec2TransitGatewayVpcAttachmentItem `xml:"item"`
}

type ec2TransitGatewayVpcAttachmentItem struct {
	CreationTime               time.Time                                 `xml:"creationTime,omitempty"`
	Options                    ec2TransitGatewayVpcAttachmentOptionsItem `xml:"options"`
	State                      string                                    `xml:"state,omitempty"`
	SubnetIDs                  ec2StringSet                              `xml:"subnetIds"`
	TagSet                     ec2TagSet                                 `xml:"tagSet"`
	TransitGatewayAttachmentID string                                    `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayID           string                                    `xml:"transitGatewayId,omitempty"`
	VpcID                      string                                    `xml:"vpcId,omitempty"`
	VpcOwnerID                 string                                    `xml:"vpcOwnerId,omitempty"`
}

type ec2TransitGatewayVpcAttachmentOptionsItem struct {
	ApplianceModeSupport            string `xml:"applianceModeSupport,omitempty"`
	DnsSupport                      string `xml:"dnsSupport,omitempty"`
	Ipv6Support                     string `xml:"ipv6Support,omitempty"`
	SecurityGroupReferencingSupport string `xml:"securityGroupReferencingSupport,omitempty"`
}

type ec2CreateTransitGatewayPeeringAttachmentResponse struct {
	XMLName                         xml.Name                               `xml:"CreateTransitGatewayPeeringAttachmentResponse"`
	Xmlns                           string                                 `xml:"xmlns,attr"`
	RequestID                       string                                 `xml:"requestId"`
	TransitGatewayPeeringAttachment ec2TransitGatewayPeeringAttachmentItem `xml:"transitGatewayPeeringAttachment"`
}

type ec2DeleteTransitGatewayPeeringAttachmentResponse struct {
	XMLName                         xml.Name                               `xml:"DeleteTransitGatewayPeeringAttachmentResponse"`
	Xmlns                           string                                 `xml:"xmlns,attr"`
	RequestID                       string                                 `xml:"requestId"`
	TransitGatewayPeeringAttachment ec2TransitGatewayPeeringAttachmentItem `xml:"transitGatewayPeeringAttachment"`
}

type ec2DescribeTransitGatewayPeeringAttachmentsResponse struct {
	XMLName                          xml.Name                              `xml:"DescribeTransitGatewayPeeringAttachmentsResponse"`
	Xmlns                            string                                `xml:"xmlns,attr"`
	RequestID                        string                                `xml:"requestId"`
	NextToken                        string                                `xml:"nextToken,omitempty"`
	TransitGatewayPeeringAttachments ec2TransitGatewayPeeringAttachmentSet `xml:"transitGatewayPeeringAttachments"`
}

type ec2TransitGatewayPeeringAttachmentSet struct {
	Items []ec2TransitGatewayPeeringAttachmentItem `xml:"item"`
}

type ec2TransitGatewayPeeringAttachmentItem struct {
	AccepterTgwInfo                    ec2PeeringTgwInfoItem                         `xml:"accepterTgwInfo"`
	AccepterTransitGatewayAttachmentID string                                        `xml:"accepterTransitGatewayAttachmentId,omitempty"`
	CreationTime                       time.Time                                     `xml:"creationTime,omitempty"`
	Options                            ec2TransitGatewayPeeringAttachmentOptionsItem `xml:"options"`
	RequesterTgwInfo                   ec2PeeringTgwInfoItem                         `xml:"requesterTgwInfo"`
	State                              string                                        `xml:"state,omitempty"`
	TagSet                             ec2TagSet                                     `xml:"tagSet"`
	TransitGatewayAttachmentID         string                                        `xml:"transitGatewayAttachmentId,omitempty"`
}

type ec2PeeringTgwInfoItem struct {
	OwnerID          string `xml:"ownerId,omitempty"`
	Region           string `xml:"region,omitempty"`
	TransitGatewayID string `xml:"transitGatewayId,omitempty"`
}

type ec2TransitGatewayPeeringAttachmentOptionsItem struct {
	DynamicRouting string `xml:"dynamicRouting,omitempty"`
}

type ec2CreateTransitGatewayConnectResponse struct {
	XMLName               xml.Name                     `xml:"CreateTransitGatewayConnectResponse"`
	Xmlns                 string                       `xml:"xmlns,attr"`
	RequestID             string                       `xml:"requestId"`
	TransitGatewayConnect ec2TransitGatewayConnectItem `xml:"transitGatewayConnect"`
}

type ec2DeleteTransitGatewayConnectResponse struct {
	XMLName               xml.Name                     `xml:"DeleteTransitGatewayConnectResponse"`
	Xmlns                 string                       `xml:"xmlns,attr"`
	RequestID             string                       `xml:"requestId"`
	TransitGatewayConnect ec2TransitGatewayConnectItem `xml:"transitGatewayConnect"`
}

type ec2DescribeTransitGatewayConnectsResponse struct {
	XMLName                  xml.Name                    `xml:"DescribeTransitGatewayConnectsResponse"`
	Xmlns                    string                      `xml:"xmlns,attr"`
	RequestID                string                      `xml:"requestId"`
	NextToken                string                      `xml:"nextToken,omitempty"`
	TransitGatewayConnectSet ec2TransitGatewayConnectSet `xml:"transitGatewayConnectSet"`
}

type ec2TransitGatewayConnectSet struct {
	Items []ec2TransitGatewayConnectItem `xml:"item"`
}

type ec2TransitGatewayConnectItem struct {
	CreationTime                        time.Time                           `xml:"creationTime,omitempty"`
	Options                             ec2TransitGatewayConnectOptionsItem `xml:"options"`
	State                               string                              `xml:"state,omitempty"`
	TagSet                              ec2TagSet                           `xml:"tagSet"`
	TransitGatewayAttachmentID          string                              `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayID                    string                              `xml:"transitGatewayId,omitempty"`
	TransportTransitGatewayAttachmentID string                              `xml:"transportTransitGatewayAttachmentId,omitempty"`
}

type ec2TransitGatewayConnectOptionsItem struct {
	Protocol string `xml:"protocol,omitempty"`
}

type ec2CreateTransitGatewayConnectPeerResponse struct {
	XMLName                   xml.Name                         `xml:"CreateTransitGatewayConnectPeerResponse"`
	Xmlns                     string                           `xml:"xmlns,attr"`
	RequestID                 string                           `xml:"requestId"`
	TransitGatewayConnectPeer ec2TransitGatewayConnectPeerItem `xml:"transitGatewayConnectPeer"`
}

type ec2DeleteTransitGatewayConnectPeerResponse struct {
	XMLName                   xml.Name                         `xml:"DeleteTransitGatewayConnectPeerResponse"`
	Xmlns                     string                           `xml:"xmlns,attr"`
	RequestID                 string                           `xml:"requestId"`
	TransitGatewayConnectPeer ec2TransitGatewayConnectPeerItem `xml:"transitGatewayConnectPeer"`
}

type ec2DescribeTransitGatewayConnectPeersResponse struct {
	XMLName                      xml.Name                        `xml:"DescribeTransitGatewayConnectPeersResponse"`
	Xmlns                        string                          `xml:"xmlns,attr"`
	RequestID                    string                          `xml:"requestId"`
	NextToken                    string                          `xml:"nextToken,omitempty"`
	TransitGatewayConnectPeerSet ec2TransitGatewayConnectPeerSet `xml:"transitGatewayConnectPeerSet"`
}

type ec2TransitGatewayConnectPeerSet struct {
	Items []ec2TransitGatewayConnectPeerItem `xml:"item"`
}

type ec2TransitGatewayConnectPeerItem struct {
	ConnectPeerConfiguration    ec2TransitGatewayConnectPeerConfigurationItem `xml:"connectPeerConfiguration"`
	CreationTime                time.Time                                     `xml:"creationTime,omitempty"`
	State                       string                                        `xml:"state,omitempty"`
	TagSet                      ec2TagSet                                     `xml:"tagSet"`
	TransitGatewayAttachmentID  string                                        `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayConnectPeerID string                                        `xml:"transitGatewayConnectPeerId,omitempty"`
}

type ec2TransitGatewayConnectPeerConfigurationItem struct {
	BgpConfigurations     ec2TransitGatewayAttachmentBgpConfigurationSet `xml:"bgpConfigurations"`
	InsideCidrBlocks      ec2StringSet                                   `xml:"insideCidrBlocks"`
	PeerAddress           string                                         `xml:"peerAddress,omitempty"`
	Protocol              string                                         `xml:"protocol,omitempty"`
	TransitGatewayAddress string                                         `xml:"transitGatewayAddress,omitempty"`
}

type ec2TransitGatewayAttachmentBgpConfigurationSet struct {
	Items []ec2TransitGatewayAttachmentBgpConfigurationItem `xml:"item"`
}

type ec2TransitGatewayAttachmentBgpConfigurationItem struct {
	BgpStatus             string `xml:"bgpStatus,omitempty"`
	PeerAddress           string `xml:"peerAddress,omitempty"`
	PeerAsn               int64  `xml:"peerAsn,omitempty"`
	TransitGatewayAddress string `xml:"transitGatewayAddress,omitempty"`
	TransitGatewayAsn     int64  `xml:"transitGatewayAsn,omitempty"`
}

type ec2StringSet struct {
	Items []string `xml:"item"`
}
