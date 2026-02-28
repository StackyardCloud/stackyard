package ec2

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

type AsnAuthorizationContext struct {
	Message   string
	Signature string
}

type IpamCidrAuthorizationContext struct {
	Message   string
	Signature string
}

type PublicIpv4PoolRange struct {
	AddressCount          *int32
	AvailableAddressCount *int32
	FirstAddress          string
	LastAddress           string
}

type PurchaseScheduledInstanceRequest struct {
	InstanceCount int32
	PurchaseToken string
}

func (s *Service) ProvisionIpamByoasn(asn string, asnAuthorizationContext *AsnAuthorizationContext, ipamID string) (Byoasn, error) {
	asn = strings.TrimSpace(asn)
	ipamID = strings.TrimSpace(ipamID)
	if asn == "" || ipamID == "" || asnAuthorizationContext == nil {
		return Byoasn{}, ErrInvalidParameter
	}
	if strings.TrimSpace(asnAuthorizationContext.Message) == "" || strings.TrimSpace(asnAuthorizationContext.Signature) == "" {
		return Byoasn{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ipams[ipamID] == nil {
		return Byoasn{}, ErrNotFound
	}

	found := false
	for _, record := range s.byoipCidrs {
		if record == nil {
			continue
		}
		for i := range record.AsnAssociations {
			if !strings.EqualFold(strings.TrimSpace(record.AsnAssociations[i].Asn), asn) {
				continue
			}
			record.AsnAssociations[i].Asn = asn
			record.AsnAssociations[i].State = "provisioned"
			record.AsnAssociations[i].StatusMessage = "provisioned"
			record.State = "provisioned"
			record.StatusMessage = "provisioned"
			found = true
		}
	}

	if !found {
		cidr := stage131SyntheticByoasnCidr(asn, 0)
		for salt := 0; salt < 256; salt++ {
			candidate := stage131SyntheticByoasnCidr(asn, salt)
			if _, exists := s.byoipCidrs[candidate]; exists {
				continue
			}
			cidr = candidate
			break
		}

		record := s.byoipCidrs[cidr]
		if record == nil {
			record = &ByoipCidr{Cidr: cidr}
			s.byoipCidrs[cidr] = record
		}
		record.State = "provisioned"
		record.StatusMessage = "provisioned"
		record.AsnAssociations = append(record.AsnAssociations, ByoipAsnAssociation{
			Asn:           asn,
			Cidr:          cidr,
			State:         "provisioned",
			StatusMessage: "provisioned",
		})
	}

	return Byoasn{
		Asn:           asn,
		IpamID:        ipamID,
		State:         "provisioned",
		StatusMessage: "provisioned",
	}, nil
}

func (s *Service) ProvisionIpamPoolCidr(
	ipamPoolID string,
	cidr *string,
	netmaskLength *int32,
	cidrAuthorizationContext *IpamCidrAuthorizationContext,
	ipamExternalResourceVerificationTokenID *string,
	verificationMethod string,
	clientToken *string,
) (IpamPoolCidr, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	if ipamPoolID == "" {
		return IpamPoolCidr{}, ErrInvalidParameter
	}
	if cidrAuthorizationContext != nil {
		if strings.TrimSpace(cidrAuthorizationContext.Message) == "" || strings.TrimSpace(cidrAuthorizationContext.Signature) == "" {
			return IpamPoolCidr{}, ErrInvalidParameter
		}
	}
	_ = strings.TrimSpace(derefString(ipamExternalResourceVerificationTokenID))
	_ = strings.TrimSpace(verificationMethod)
	_ = strings.TrimSpace(derefString(clientToken))

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ipamPools[ipamPoolID]
	if pool == nil {
		return IpamPoolCidr{}, ErrNotFound
	}

	cidrValue := strings.TrimSpace(derefString(cidr))
	if cidrValue == "" && netmaskLength == nil {
		return IpamPoolCidr{}, ErrInvalidParameter
	}

	effectiveNetmask := cloneInt32Pointer(netmaskLength)
	if effectiveNetmask == nil {
		effectiveNetmask = parseCIDRNetmaskLength(cidrValue)
	}
	if effectiveNetmask != nil && !validStage108Netmask(effectiveNetmask, pool.AddressFamily) {
		return IpamPoolCidr{}, ErrInvalidParameter
	}

	if cidrValue == "" {
		if effectiveNetmask == nil {
			return IpamPoolCidr{}, ErrInvalidParameter
		}
		cidrValue = stage131GeneratedIpamPoolCidr(pool.AddressFamily, *effectiveNetmask)
	}
	if effectiveNetmask == nil {
		effectiveNetmask = parseCIDRNetmaskLength(cidrValue)
	}

	allocationID := s.nextIDLocked("ipam-pool-alloc")
	s.ipamPoolAllocations[allocationID] = &IpamPoolAllocation{
		Cidr:                 cidrValue,
		Description:          "",
		IpamPoolAllocationID: allocationID,
		ResourceID:           ipamPoolID,
		ResourceOwner:        DefaultAccountID,
		ResourceRegion:       DefaultRegion,
		ResourceType:         "ipam-pool",
	}

	return IpamPoolCidr{
		Cidr:           cidrValue,
		FailureReason:  nil,
		IpamPoolCidrID: allocationID,
		NetmaskLength:  cloneInt32Pointer(effectiveNetmask),
		State:          "provisioned",
	}, nil
}

func (s *Service) ProvisionPublicIpv4PoolCidr(
	ipamPoolID string,
	netmaskLength int32,
	poolID string,
	networkBorderGroup *string,
) (PublicIpv4PoolRange, string, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	poolID = strings.TrimSpace(poolID)
	if ipamPoolID == "" || poolID == "" {
		return PublicIpv4PoolRange{}, "", ErrInvalidParameter
	}
	if netmaskLength < 24 || netmaskLength > 32 {
		return PublicIpv4PoolRange{}, "", ErrInvalidParameter
	}
	if networkBorderGroup != nil && strings.TrimSpace(*networkBorderGroup) == "" {
		return PublicIpv4PoolRange{}, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ipamPools[ipamPoolID] == nil {
		return PublicIpv4PoolRange{}, "", ErrNotFound
	}
	pool := s.publicIpv4Pools[poolID]
	if pool == nil {
		return PublicIpv4PoolRange{}, "", ErrNotFound
	}
	if networkBorderGroup != nil {
		pool.NetworkBorderGroup = strings.TrimSpace(*networkBorderGroup)
	}

	cidr := stage131PublicPoolCidr(poolID, netmaskLength)
	firstAddress, lastAddress, addressCount := stage131Ipv4RangeFromCidr(cidr)
	availableAddressCount := addressCount

	return PublicIpv4PoolRange{
		AddressCount:          cloneInt32Pointer(&addressCount),
		AvailableAddressCount: cloneInt32Pointer(&availableAddressCount),
		FirstAddress:          firstAddress,
		LastAddress:           lastAddress,
	}, poolID, nil
}

func (s *Service) PurchaseCapacityBlock(
	capacityBlockOfferingID string,
	instancePlatform string,
	tags []Tag,
) ([]CapacityBlock, CapacityReservation, error) {
	capacityBlockOfferingID = strings.TrimSpace(capacityBlockOfferingID)
	instancePlatform = strings.TrimSpace(instancePlatform)
	if capacityBlockOfferingID == "" || instancePlatform == "" {
		return nil, CapacityReservation{}, ErrInvalidParameter
	}

	now := time.Now().UTC()
	normalizedTags := tagsToMap(normalizeEC2Tags(tags))

	s.mu.Lock()
	reservationID := s.nextIDLocked("cr")
	reservation := &CapacityReservation{
		ID:                     reservationID,
		AvailabilityZone:       "us-east-1a",
		AvailabilityZoneID:     "use1-az1",
		InstanceType:           "p5.48xlarge",
		InstancePlatform:       instancePlatform,
		InstanceMatchCriteria:  "targeted",
		Tenancy:                "default",
		State:                  "active",
		TotalInstanceCount:     1,
		AvailableInstanceCount: 1,
		EbsOptimized:           nil,
		EphemeralStorage:       nil,
		OwnerID:                DefaultAccountID,
		CreateDate:             now,
		Tags:                   cloneStringMap(normalizedTags),
	}
	s.capacityReservations[reservationID] = reservation
	s.mu.Unlock()

	capacityBlock := CapacityBlock{
		AvailabilityZone:       reservation.AvailabilityZone,
		AvailabilityZoneID:     reservation.AvailabilityZoneID,
		CapacityBlockID:        "cb-" + strings.TrimPrefix(reservationID, "cr-"),
		CapacityReservationIDs: []string{reservationID},
		CreateDate:             now,
		EndDate:                now.Add(24 * time.Hour),
		StartDate:              now,
		State:                  "active",
		Tags:                   cloneStringMap(normalizedTags),
		UltraserverType:        "instances",
	}
	return []CapacityBlock{capacityBlock}, cloneCapacityReservation(reservation), nil
}

func (s *Service) PurchaseCapacityBlockExtension(
	capacityBlockExtensionOfferingID string,
	capacityReservationID string,
) ([]CapacityBlockExtension, error) {
	capacityBlockExtensionOfferingID = strings.TrimSpace(capacityBlockExtensionOfferingID)
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityBlockExtensionOfferingID == "" || capacityReservationID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	reservation := s.capacityReservations[capacityReservationID]
	s.mu.Unlock()
	if reservation == nil {
		return nil, ErrNotFound
	}

	now := time.Now().UTC()
	start := now.Add(24 * time.Hour)
	duration := int32(24)
	end := start.Add(time.Duration(duration) * time.Hour)
	currencyCode := "USD"
	status := "payment-succeeded"
	upfrontFee := "0.00"

	item := CapacityBlockExtension{
		AvailabilityZone:                    cloneStringPointer(&reservation.AvailabilityZone),
		AvailabilityZoneID:                  cloneStringPointer(&reservation.AvailabilityZoneID),
		CapacityBlockExtensionDurationHours: cloneInt32Pointer(&duration),
		CapacityBlockExtensionEndDate:       cloneTimePointer(&end),
		CapacityBlockExtensionOfferingID:    cloneStringPointer(&capacityBlockExtensionOfferingID),
		CapacityBlockExtensionPurchaseDate:  cloneTimePointer(&now),
		CapacityBlockExtensionStartDate:     cloneTimePointer(&start),
		CapacityBlockExtensionStatus:        status,
		CapacityReservationID:               cloneStringPointer(&capacityReservationID),
		CurrencyCode:                        cloneStringPointer(&currencyCode),
		InstanceCount:                       cloneInt32Pointer(&reservation.TotalInstanceCount),
		InstanceType:                        cloneStringPointer(&reservation.InstanceType),
		UpfrontFee:                          cloneStringPointer(&upfrontFee),
	}
	return []CapacityBlockExtension{item}, nil
}

func (s *Service) PurchaseHostReservation(
	hostIDs []string,
	offeringID string,
	clientToken *string,
	currencyCode string,
	limitPrice *string,
	tags []Tag,
) (*string, string, []HostReservationPurchase, error) {
	hostIDs = dedupeTrimmedStrings(hostIDs)
	offeringID = strings.TrimSpace(offeringID)
	currencyCode = strings.ToUpper(strings.TrimSpace(currencyCode))
	if len(hostIDs) == 0 || offeringID == "" {
		return nil, "", nil, ErrInvalidParameter
	}
	if currencyCode == "" {
		currencyCode = "USD"
	}
	if currencyCode != "USD" {
		return nil, "", nil, ErrInvalidParameter
	}
	if limitPrice != nil && strings.TrimSpace(*limitPrice) == "" {
		return nil, "", nil, ErrInvalidParameter
	}
	_ = tags

	s.mu.Lock()
	tokenValue := strings.TrimSpace(derefString(clientToken))
	if tokenValue == "" {
		tokenValue = s.nextIDLocked("hres-token")
	}
	hostReservationID := s.nextIDLocked("hr")
	instanceFamily := "m5"
	if host := s.dedicatedHosts[hostIDs[0]]; host != nil {
		instanceFamily = strings.TrimSpace(host.InstanceFamily)
		if instanceFamily == "" {
			instanceFamily = stage116InstanceFamilyFromType(host.InstanceType)
		}
	}
	s.mu.Unlock()

	duration := int32(31536000)
	purchase := HostReservationPurchase{
		CurrencyCode:      currencyCode,
		Duration:          duration,
		HostIDs:           append([]string(nil), hostIDs...),
		HostReservationID: hostReservationID,
		HourlyPrice:       "0.75",
		InstanceFamily:    firstNonEmptyString(instanceFamily, "m5"),
		PaymentOption:     "NoUpfront",
		UpfrontPrice:      "0.00",
	}

	return cloneStringPointer(&tokenValue), currencyCode, []HostReservationPurchase{purchase}, nil
}

func (s *Service) PurchaseReservedInstancesOffering(
	instanceCount int32,
	reservedInstancesOfferingID string,
	purchaseTime *time.Time,
) (string, error) {
	reservedInstancesOfferingID = strings.TrimSpace(reservedInstancesOfferingID)
	if instanceCount <= 0 || reservedInstancesOfferingID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reservationID := s.nextIDLocked("ri")
	suffix := strings.TrimPrefix(reservationID, "ri-")
	listingID := "ril-" + suffix

	at := time.Now().UTC()
	if purchaseTime != nil && !purchaseTime.IsZero() {
		at = purchaseTime.UTC()
	}
	s.reservedInstancesListingCreatedAt[listingID] = at
	s.reservedInstancesListingStates[listingID] = "active"
	return reservationID, nil
}

func (s *Service) PurchaseScheduledInstances(
	purchaseRequests []PurchaseScheduledInstanceRequest,
	clientToken *string,
) ([]ScheduledInstance, error) {
	if len(purchaseRequests) == 0 {
		return nil, ErrInvalidParameter
	}
	_ = strings.TrimSpace(derefString(clientToken))

	now := time.Now().UTC().Truncate(time.Hour)
	slotDuration := int32(1)
	totalHours := int32(24)
	recurrence := &ScheduledInstanceRecurrence{
		Frequency:      "Weekly",
		OccurrenceDays: []int32{1},
		OccurrenceUnit: "day-of-week",
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ScheduledInstance, 0)
	for _, request := range purchaseRequests {
		request.PurchaseToken = strings.TrimSpace(request.PurchaseToken)
		if request.InstanceCount <= 0 || request.PurchaseToken == "" {
			return nil, ErrInvalidParameter
		}
		for i := int32(0); i < request.InstanceCount; i++ {
			scheduledID := s.nextIDLocked("sci")
			nextSlotStart := now.Add(1 * time.Hour)
			previousSlotEnd := now.Add(-1 * time.Hour)
			termStart := now
			termEnd := now.Add(365 * 24 * time.Hour)
			instanceCount := int32(1)
			out = append(out, ScheduledInstance{
				AvailabilityZone:            "us-east-1a",
				CreateDate:                  cloneTimePointer(&now),
				HourlyPrice:                 "0.050",
				InstanceCount:               cloneInt32Pointer(&instanceCount),
				InstanceType:                "c5.large",
				NetworkPlatform:             "EC2-VPC",
				NextSlotStartTime:           cloneTimePointer(&nextSlotStart),
				Platform:                    "Linux/UNIX",
				PreviousSlotEndTime:         cloneTimePointer(&previousSlotEnd),
				Recurrence:                  cloneStage121ScheduledInstanceRecurrence(recurrence),
				ScheduledInstanceID:         scheduledID,
				SlotDurationInHours:         cloneInt32Pointer(&slotDuration),
				TermEndDate:                 cloneTimePointer(&termEnd),
				TermStartDate:               cloneTimePointer(&termStart),
				TotalScheduledInstanceHours: cloneInt32Pointer(&totalHours),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ScheduledInstanceID < out[j].ScheduledInstanceID
	})
	return out, nil
}

func (s *Service) RegisterImage(
	name string,
	architecture string,
	description *string,
	imageLocation *string,
	rootDeviceName *string,
	virtualizationType *string,
	tags []Tag,
) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidParameter
	}

	architecture = strings.TrimSpace(architecture)
	if architecture == "" {
		architecture = "x86_64"
	}
	imageLocationValue := strings.TrimSpace(derefString(imageLocation))
	if imageLocationValue == "" {
		imageLocationValue = DefaultAccountID + "/" + name
	}
	rootDeviceNameValue := strings.TrimSpace(derefString(rootDeviceName))
	if rootDeviceNameValue == "" {
		rootDeviceNameValue = "/dev/sda1"
	}
	virtualizationTypeValue := strings.TrimSpace(derefString(virtualizationType))
	if virtualizationTypeValue == "" {
		virtualizationTypeValue = "hvm"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	imageID := s.nextIDLocked("ami")
	s.images[imageID] = &Image{
		ID:                       imageID,
		Name:                     name,
		Description:              strings.TrimSpace(derefString(description)),
		DeprecationTime:          nil,
		DeregistrationProtection: "disabled",
		State:                    "available",
		OwnerID:                  DefaultAccountID,
		ImageLocation:            imageLocationValue,
		Architecture:             architecture,
		ImageType:                "machine",
		RootDeviceType:           "ebs",
		RootDeviceName:           rootDeviceNameValue,
		VirtualizationType:       virtualizationTypeValue,
		CreationDate:             time.Now().UTC(),
		Tags:                     tagsToMap(normalizeEC2Tags(tags)),
		LaunchPermissions:        nil,
	}
	return imageID, nil
}

func (s *Service) RegisterInstanceEventNotificationAttributes(
	includeAllTagsOfInstance *bool,
	instanceTagKeys []string,
) (InstanceTagNotificationAttribute, error) {
	instanceTagKeys = dedupeTrimmedStrings(instanceTagKeys)
	if includeAllTagsOfInstance == nil && len(instanceTagKeys) == 0 {
		return InstanceTagNotificationAttribute{}, ErrInvalidParameter
	}

	if includeAllTagsOfInstance != nil && *includeAllTagsOfInstance {
		includeAll := true
		return InstanceTagNotificationAttribute{
			IncludeAllTagsOfInstance: &includeAll,
			InstanceTagKeys:          nil,
		}, nil
	}

	if len(instanceTagKeys) == 0 {
		return InstanceTagNotificationAttribute{}, ErrInvalidParameter
	}

	includeAll := false
	return InstanceTagNotificationAttribute{
		IncludeAllTagsOfInstance: &includeAll,
		InstanceTagKeys:          instanceTagKeys,
	}, nil
}

func stage131SyntheticByoasnCidr(asn string, salt int) string {
	h := 0
	for _, r := range asn {
		h = ((h * 33) + int(r)) % 251
	}
	octet := (h + salt) % 251
	if octet <= 0 {
		octet = 1
	}
	return fmt.Sprintf("198.19.%d.0/24", octet)
}

func stage131GeneratedIpamPoolCidr(addressFamily string, netmaskLength int32) string {
	if strings.EqualFold(strings.TrimSpace(addressFamily), "ipv6") || netmaskLength > 32 {
		segment := int(netmaskLength)%65535 + 1
		if netmaskLength <= 0 {
			netmaskLength = 64
		}
		return fmt.Sprintf("2001:db8:%x::/%d", segment, netmaskLength)
	}
	if netmaskLength <= 0 {
		netmaskLength = 24
	}
	secondOctet := int(netmaskLength)%200 + 10
	return fmt.Sprintf("10.%d.0.0/%d", secondOctet, netmaskLength)
}

func stage131PublicPoolCidr(poolID string, netmaskLength int32) string {
	h := 0
	for _, r := range poolID {
		h = ((h * 31) + int(r)) % 250
	}
	octet := h + 1
	if octet > 250 {
		octet = 250
	}
	return fmt.Sprintf("198.51.%d.0/%d", octet, netmaskLength)
}

func stage131Ipv4RangeFromCidr(cidr string) (string, string, int32) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
	if err != nil || !prefix.Addr().Is4() {
		return "198.51.100.0", "198.51.100.255", 256
	}

	base := prefix.Masked().Addr().As4()
	baseUint := uint32(base[0])<<24 | uint32(base[1])<<16 | uint32(base[2])<<8 | uint32(base[3])
	hostBits := 32 - prefix.Bits()
	if hostBits < 0 || hostBits > 8 {
		hostBits = 8
	}
	count := int32(1 << hostBits)
	lastUint := baseUint + uint32(count) - 1

	first := netip.AddrFrom4(base).String()
	lastBytes := [4]byte{
		byte(lastUint >> 24),
		byte(lastUint >> 16),
		byte(lastUint >> 8),
		byte(lastUint),
	}
	last := netip.AddrFrom4(lastBytes).String()
	return first, last, count
}
