package ec2

import (
	"sort"
	"strings"
)

type DHCPConfiguration struct {
	Key    string
	Values []string
}

type DHCPOptions struct {
	ID             string
	Configurations []DHCPConfiguration
	OwnerID        string
	Tags           map[string]string
}

type EgressOnlyInternetGateway struct {
	ID          string
	Attachments []InternetGatewayAttachment
	Tags        map[string]string
}

type AddressAttribute struct {
	AllocationID string
	PublicIP     string
	PtrRecord    string
}

func (s *Service) CreateDhcpOptions(configurations []DHCPConfiguration, tags []Tag) (DHCPOptions, error) {
	if len(configurations) == 0 {
		return DHCPOptions{}, ErrInvalidParameter
	}
	normalized := normalizeDHCPConfigurations(configurations)
	if len(normalized) == 0 {
		return DHCPOptions{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	options := &DHCPOptions{
		ID:             s.nextIDLocked("dopt"),
		Configurations: normalized,
		OwnerID:        DefaultAccountID,
		Tags:           tagsToMap(tags),
	}
	s.dhcpOptions[options.ID] = options
	return cloneDHCPOptions(options), nil
}

func (s *Service) DescribeDhcpOptions(optionIDs, keys []string) []DHCPOptions {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(optionIDs)
	keySet := toStringSet(keys)
	out := make([]DHCPOptions, 0, len(s.dhcpOptions))
	for _, options := range s.dhcpOptions {
		if len(idSet) > 0 {
			if _, ok := idSet[options.ID]; !ok {
				continue
			}
		}
		if len(keySet) > 0 {
			matched := false
			for _, cfg := range options.Configurations {
				if _, ok := keySet[cfg.Key]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cloneDHCPOptions(options))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) AssociateDhcpOptions(optionID, vpcID string) error {
	optionID = strings.TrimSpace(optionID)
	vpcID = strings.TrimSpace(vpcID)
	if optionID == "" || vpcID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return ErrNotFound
	}
	if strings.EqualFold(optionID, "default") {
		optionID = defaultDHCPOptionsID
	}
	if s.dhcpOptions[optionID] == nil {
		return ErrNotFound
	}
	vpc.DhcpOptionsID = optionID
	return nil
}

func (s *Service) DeleteDhcpOptions(optionID string) error {
	optionID = strings.TrimSpace(optionID)
	if optionID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if optionID == defaultDHCPOptionsID {
		return ErrConflict
	}
	if s.dhcpOptions[optionID] == nil {
		return ErrNotFound
	}
	for _, vpc := range s.vpcs {
		if vpc.DhcpOptionsID == optionID {
			return ErrConflict
		}
	}
	delete(s.dhcpOptions, optionID)
	return nil
}

func (s *Service) CreateEgressOnlyInternetGateway(vpcID string, tags []Tag) (EgressOnlyInternetGateway, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return EgressOnlyInternetGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return EgressOnlyInternetGateway{}, ErrNotFound
	}
	gateway := &EgressOnlyInternetGateway{
		ID: s.nextIDLocked("eigw"),
		Attachments: []InternetGatewayAttachment{
			{VpcID: vpcID, State: "attached"},
		},
		Tags: tagsToMap(tags),
	}
	s.egressOnlyGateways[gateway.ID] = gateway
	return cloneEgressOnlyInternetGateway(gateway), nil
}

func (s *Service) DescribeEgressOnlyInternetGateways(gatewayIDs, vpcIDs []string) []EgressOnlyInternetGateway {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(gatewayIDs)
	vpcSet := toStringSet(vpcIDs)
	out := make([]EgressOnlyInternetGateway, 0, len(s.egressOnlyGateways))
	for _, gateway := range s.egressOnlyGateways {
		if len(idSet) > 0 {
			if _, ok := idSet[gateway.ID]; !ok {
				continue
			}
		}
		if len(vpcSet) > 0 {
			matched := false
			for _, attachment := range gateway.Attachments {
				if _, ok := vpcSet[attachment.VpcID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, cloneEgressOnlyInternetGateway(gateway))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) DeleteEgressOnlyInternetGateway(gatewayID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.egressOnlyGateways[gatewayID] == nil {
		return ErrNotFound
	}
	delete(s.egressOnlyGateways, gatewayID)
	return nil
}

func (s *Service) DescribeAddressesAttribute(allocationIDs []string, attribute string) ([]AddressAttribute, error) {
	attribute = strings.ToLower(strings.TrimSpace(attribute))
	if attribute != "domain-name" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(allocationIDs)
	out := make([]AddressAttribute, 0, len(s.addresses))
	for _, address := range s.addresses {
		if len(idSet) > 0 {
			if _, ok := idSet[address.AllocationID]; !ok {
				continue
			}
		}
		out = append(out, cloneAddressAttributeFromElasticAddress(address))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AllocationID < out[j].AllocationID })
	return out, nil
}

func normalizeDHCPConfigurations(configurations []DHCPConfiguration) []DHCPConfiguration {
	out := make([]DHCPConfiguration, 0, len(configurations))
	for _, configuration := range configurations {
		key := strings.TrimSpace(configuration.Key)
		if key == "" {
			continue
		}
		values := make([]string, 0, len(configuration.Values))
		for _, value := range configuration.Values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			values = append(values, value)
		}
		if len(values) == 0 {
			continue
		}
		out = append(out, DHCPConfiguration{Key: key, Values: values})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func cloneDHCPOptions(in *DHCPOptions) DHCPOptions {
	out := *in
	out.Configurations = make([]DHCPConfiguration, 0, len(in.Configurations))
	for _, cfg := range in.Configurations {
		out.Configurations = append(out.Configurations, DHCPConfiguration{
			Key:    cfg.Key,
			Values: append([]string(nil), cfg.Values...),
		})
	}
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneEgressOnlyInternetGateway(in *EgressOnlyInternetGateway) EgressOnlyInternetGateway {
	out := *in
	out.Attachments = append([]InternetGatewayAttachment(nil), in.Attachments...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}
