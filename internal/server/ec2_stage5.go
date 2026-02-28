package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage5Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateDhcpOptions":
		options, err := s.ec2.CreateDhcpOptions(
			parseEC2DHCPConfigurations(r.Form),
			parseEC2TagSpecificationsForResource(r.Form, "dhcp-options"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateDhcpOptionsResponse{
			XMLName:     xml.Name{Local: "CreateDhcpOptionsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			DhcpOptions: ec2DHCPOptionsItemFrom(options),
		})
		return true
	case "DescribeDhcpOptions":
		optionIDs := parseEC2Members(r.Form, "DhcpOptionsId.")
		if len(optionIDs) == 0 {
			optionIDs = parseEC2FilterValues(r.Form, "dhcp-options-id")
		}
		keys := parseEC2FilterValues(r.Form, "key")
		options := s.ec2.DescribeDhcpOptions(optionIDs, keys)
		respondEC2XML(w, ec2DescribeDhcpOptionsResponse{
			XMLName:        xml.Name{Local: "DescribeDhcpOptionsResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			DhcpOptionsSet: ec2DHCPOptionsSet{Items: ec2DHCPOptionsItems(options)},
		})
		return true
	case "AssociateDhcpOptions":
		if err := s.ec2.AssociateDhcpOptions(
			strings.TrimSpace(r.Form.Get("DhcpOptionsId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "AssociateDhcpOptionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteDhcpOptions":
		if err := s.ec2.DeleteDhcpOptions(strings.TrimSpace(r.Form.Get("DhcpOptionsId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteDhcpOptionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateEgressOnlyInternetGateway":
		gateway, err := s.ec2.CreateEgressOnlyInternetGateway(
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2TagSpecificationsForResource(r.Form, "egress-only-internet-gateway"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateEgressOnlyInternetGatewayResponse{
			XMLName:                   xml.Name{Local: "CreateEgressOnlyInternetGatewayResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			EgressOnlyInternetGateway: ec2EgressOnlyInternetGatewayItems([]ec2svc.EgressOnlyInternetGateway{gateway})[0],
		})
		return true
	case "DescribeEgressOnlyInternetGateways":
		gatewayIDs := parseEC2Members(r.Form, "EgressOnlyInternetGatewayId.")
		vpcIDs := parseEC2FilterValues(r.Form, "attachment.vpc-id")
		gateways := s.ec2.DescribeEgressOnlyInternetGateways(gatewayIDs, vpcIDs)
		respondEC2XML(w, ec2DescribeEgressOnlyInternetGatewaysResponse{
			XMLName:   xml.Name{Local: "DescribeEgressOnlyInternetGatewaysResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			EgressOnlyInternetGatewaySet: ec2EgressOnlyInternetGatewaySet{
				Items: ec2EgressOnlyInternetGatewayItems(gateways),
			},
		})
		return true
	case "DeleteEgressOnlyInternetGateway":
		if err := s.ec2.DeleteEgressOnlyInternetGateway(strings.TrimSpace(r.Form.Get("EgressOnlyInternetGatewayId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteEgressOnlyInternetGatewayResponse{
			XMLName:    xml.Name{Local: "DeleteEgressOnlyInternetGatewayResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			ReturnCode: true,
		})
		return true
	case "DescribeAddressesAttribute":
		attributes, err := s.ec2.DescribeAddressesAttribute(
			parseEC2Members(r.Form, "AllocationId."),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeAddressesAttributeResponse{
			XMLName:   xml.Name{Local: "DescribeAddressesAttributeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Addresses: ec2AddressAttributeSet{Items: ec2AddressAttributeItems(attributes)},
		})
		return true
	default:
		return false
	}
}

func parseEC2DHCPConfigurations(values url.Values) []ec2svc.DHCPConfiguration {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "DhcpConfiguration.") || !strings.HasSuffix(key, ".Key") {
			continue
		}
		rest := strings.TrimPrefix(key, "DhcpConfiguration.")
		rest = strings.TrimSuffix(rest, ".Key")
		index, err := strconv.Atoi(rest)
		if err != nil || index <= 0 {
			continue
		}
		indices[index] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)

	out := make([]ec2svc.DHCPConfiguration, 0, len(ordered))
	for _, idx := range ordered {
		key := strings.TrimSpace(values.Get("DhcpConfiguration." + strconv.Itoa(idx) + ".Key"))
		vals := parseEC2Members(values, "DhcpConfiguration."+strconv.Itoa(idx)+".Value.")
		if key == "" || len(vals) == 0 {
			continue
		}
		out = append(out, ec2svc.DHCPConfiguration{Key: key, Values: vals})
	}
	return out
}

func ec2DHCPOptionsItems(in []ec2svc.DHCPOptions) []ec2DHCPOptionsItem {
	out := make([]ec2DHCPOptionsItem, 0, len(in))
	for _, options := range in {
		out = append(out, ec2DHCPOptionsItemFrom(options))
	}
	return out
}

func ec2DHCPOptionsItemFrom(options ec2svc.DHCPOptions) ec2DHCPOptionsItem {
	tags := make([]ec2TagItem, 0, len(options.Tags))
	for key, value := range options.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	cfgs := make([]ec2DHCPConfigurationItem, 0, len(options.Configurations))
	for _, cfg := range options.Configurations {
		values := make([]ec2DHCPValueItem, 0, len(cfg.Values))
		for _, value := range cfg.Values {
			values = append(values, ec2DHCPValueItem{Value: value})
		}
		cfgs = append(cfgs, ec2DHCPConfigurationItem{
			Key:      cfg.Key,
			ValueSet: ec2DHCPValueSet{Items: values},
		})
	}
	return ec2DHCPOptionsItem{
		DhcpOptionsID:        options.ID,
		OwnerID:              options.OwnerID,
		DhcpConfigurationSet: ec2DHCPConfigurationSet{Items: cfgs},
		TagSet:               ec2TagSet{Items: tags},
	}
}

func ec2EgressOnlyInternetGatewayItems(in []ec2svc.EgressOnlyInternetGateway) []ec2EgressOnlyInternetGatewayItem {
	out := make([]ec2EgressOnlyInternetGatewayItem, 0, len(in))
	for _, gateway := range in {
		attachments := make([]ec2InternetGatewayAttachmentItem, 0, len(gateway.Attachments))
		for _, attachment := range gateway.Attachments {
			attachments = append(attachments, ec2InternetGatewayAttachmentItem{
				VpcID: attachment.VpcID,
				State: attachment.State,
			})
		}
		tags := make([]ec2TagItem, 0, len(gateway.Tags))
		for key, value := range gateway.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		out = append(out, ec2EgressOnlyInternetGatewayItem{
			EgressOnlyInternetGatewayID: gateway.ID,
			AttachmentSet:               ec2InternetGatewayAttachmentSet{Items: attachments},
			TagSet:                      ec2TagSet{Items: tags},
		})
	}
	return out
}

func ec2AddressAttributeItems(in []ec2svc.AddressAttribute) []ec2AddressAttributeItem {
	out := make([]ec2AddressAttributeItem, 0, len(in))
	for _, attr := range in {
		out = append(out, ec2AddressAttributeItem{
			AllocationID: attr.AllocationID,
			PublicIP:     attr.PublicIP,
			PtrRecord:    attr.PtrRecord,
		})
	}
	return out
}

type ec2CreateDhcpOptionsResponse struct {
	XMLName     xml.Name
	Xmlns       string             `xml:"xmlns,attr"`
	RequestID   string             `xml:"requestId"`
	DhcpOptions ec2DHCPOptionsItem `xml:"dhcpOptions"`
}

type ec2DescribeDhcpOptionsResponse struct {
	XMLName        xml.Name          `xml:"DescribeDhcpOptionsResponse"`
	Xmlns          string            `xml:"xmlns,attr"`
	RequestID      string            `xml:"requestId"`
	DhcpOptionsSet ec2DHCPOptionsSet `xml:"dhcpOptionsSet"`
}

type ec2DHCPOptionsSet struct {
	Items []ec2DHCPOptionsItem `xml:"item"`
}

type ec2DHCPOptionsItem struct {
	DhcpOptionsID        string                  `xml:"dhcpOptionsId"`
	OwnerID              string                  `xml:"ownerId,omitempty"`
	DhcpConfigurationSet ec2DHCPConfigurationSet `xml:"dhcpConfigurationSet"`
	TagSet               ec2TagSet               `xml:"tagSet"`
}

type ec2DHCPConfigurationSet struct {
	Items []ec2DHCPConfigurationItem `xml:"item"`
}

type ec2DHCPConfigurationItem struct {
	Key      string          `xml:"key"`
	ValueSet ec2DHCPValueSet `xml:"valueSet"`
}

type ec2DHCPValueSet struct {
	Items []ec2DHCPValueItem `xml:"item"`
}

type ec2DHCPValueItem struct {
	Value string `xml:"value"`
}

type ec2CreateEgressOnlyInternetGatewayResponse struct {
	XMLName                   xml.Name
	Xmlns                     string                           `xml:"xmlns,attr"`
	RequestID                 string                           `xml:"requestId"`
	EgressOnlyInternetGateway ec2EgressOnlyInternetGatewayItem `xml:"egressOnlyInternetGateway"`
}

type ec2DeleteEgressOnlyInternetGatewayResponse struct {
	XMLName    xml.Name
	Xmlns      string `xml:"xmlns,attr"`
	RequestID  string `xml:"requestId"`
	ReturnCode bool   `xml:"returnCode"`
}

type ec2DescribeEgressOnlyInternetGatewaysResponse struct {
	XMLName                      xml.Name                        `xml:"DescribeEgressOnlyInternetGatewaysResponse"`
	Xmlns                        string                          `xml:"xmlns,attr"`
	RequestID                    string                          `xml:"requestId"`
	EgressOnlyInternetGatewaySet ec2EgressOnlyInternetGatewaySet `xml:"egressOnlyInternetGatewaySet"`
}

type ec2EgressOnlyInternetGatewaySet struct {
	Items []ec2EgressOnlyInternetGatewayItem `xml:"item"`
}

type ec2EgressOnlyInternetGatewayItem struct {
	EgressOnlyInternetGatewayID string                          `xml:"egressOnlyInternetGatewayId"`
	AttachmentSet               ec2InternetGatewayAttachmentSet `xml:"attachmentSet"`
	TagSet                      ec2TagSet                       `xml:"tagSet"`
}

type ec2DescribeAddressesAttributeResponse struct {
	XMLName   xml.Name               `xml:"DescribeAddressesAttributeResponse"`
	Xmlns     string                 `xml:"xmlns,attr"`
	RequestID string                 `xml:"requestId"`
	Addresses ec2AddressAttributeSet `xml:"addressSet"`
}

type ec2AddressAttributeSet struct {
	Items []ec2AddressAttributeItem `xml:"item"`
}

type ec2AddressAttributeItem struct {
	AllocationID string `xml:"allocationId,omitempty"`
	PublicIP     string `xml:"publicIp,omitempty"`
	PtrRecord    string `xml:"ptrRecord,omitempty"`
}
