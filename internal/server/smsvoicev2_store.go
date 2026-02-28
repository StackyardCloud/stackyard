package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type smsVoiceV2Store struct {
	mu sync.Mutex

	nextID int64

	configurationSets          map[string]map[string]any
	pools                      map[string]map[string]any
	protectConfigurations      map[string]map[string]any
	registrations              map[string]map[string]any
	registrationAttachments    map[string]map[string]any
	verifiedDestinationNumbers map[string]map[string]any
	phoneNumbers               map[string]map[string]any
	senderIDs                  map[string]map[string]any
	optOutLists                map[string]map[string]any
	optedOutNumbers            map[string]map[string]bool
	keywords                   map[string]string
	tags                       map[string]map[string]string
	resourcePolicies           map[string]string
	defaults                   map[string]any
	spendLimits                map[string]string
}

func newSMSVoiceV2Store() *smsVoiceV2Store {
	s := &smsVoiceV2Store{
		nextID:                     2,
		configurationSets:          map[string]map[string]any{},
		pools:                      map[string]map[string]any{},
		protectConfigurations:      map[string]map[string]any{},
		registrations:              map[string]map[string]any{},
		registrationAttachments:    map[string]map[string]any{},
		verifiedDestinationNumbers: map[string]map[string]any{},
		phoneNumbers:               map[string]map[string]any{},
		senderIDs:                  map[string]map[string]any{},
		optOutLists:                map[string]map[string]any{},
		optedOutNumbers:            map[string]map[string]bool{},
		keywords:                   map[string]string{},
		tags:                       map[string]map[string]string{},
		resourcePolicies:           map[string]string{},
		defaults: map[string]any{
			"DefaultMessageType":            "TRANSACTIONAL",
			"DefaultSenderId":               "STACKYARD",
			"DefaultMessageFeedbackEnabled": false,
		},
		spendLimits: map[string]string{
			"TextMessage":  "1000",
			"VoiceMessage": "1000",
			"MediaMessage": "1000",
		},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *smsVoiceV2Store) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)

	configSetName := smsVoiceV2PayloadString(payload, "ConfigurationSetName", "stackyard-sms-config-set")
	poolID := smsVoiceV2PayloadString(payload, "PoolId", "pool-000001")
	protectConfigurationID := smsVoiceV2PayloadString(payload, "ProtectConfigurationId", "pc-000001")
	registrationID := smsVoiceV2PayloadString(payload, "RegistrationId", "reg-000001")
	registrationAttachmentID := smsVoiceV2PayloadString(payload, "RegistrationAttachmentId", "ra-000001")
	verifiedDestinationNumberID := smsVoiceV2PayloadString(payload, "VerifiedDestinationNumberId", "vdn-000001")
	phoneNumberID := smsVoiceV2PayloadString(payload, "PhoneNumberId", "pn-000001")
	senderID := smsVoiceV2PayloadString(payload, "SenderId", "STACKYARD")
	optOutListName := smsVoiceV2PayloadString(payload, "OptOutListName", "stackyard-optout-list")
	destinationPhoneNumber := smsVoiceV2PayloadString(payload, "DestinationPhoneNumber", "+12065550101")
	originationIdentity := smsVoiceV2PayloadString(payload, "OriginationIdentity", "arn:aws:sms-voice:us-east-1:123456789012:phone-number/+12065550100")
	keyword := smsVoiceV2PayloadString(payload, "Keyword", "HELP")
	resourceArn := smsVoiceV2PayloadString(payload, "ResourceArn", smsVoiceV2ConfigurationSetARN(configSetName))

	s.ensureConfigurationSetLocked(configSetName, now)
	s.ensurePoolLocked(poolID, now)
	s.ensureProtectConfigurationLocked(protectConfigurationID, now)
	s.ensureRegistrationLocked(registrationID, now)
	s.ensureRegistrationAttachmentLocked(registrationAttachmentID, registrationID, now)
	s.ensureVerifiedDestinationNumberLocked(verifiedDestinationNumberID, destinationPhoneNumber, now)
	s.ensurePhoneNumberLocked(phoneNumberID, now)
	s.ensureSenderIDLocked(senderID, now)
	s.ensureOptOutListLocked(optOutListName, now)
	s.ensureTagMapLocked(resourceArn)

	s.applyTagMutationsLocked(action, payload, resourceArn)

	switch action {
	case "CreateConfigurationSet":
		name := smsVoiceV2PayloadString(payload, "ConfigurationSetName", fmt.Sprintf("stackyard-sms-config-set-%06d", s.nextIDLocked()))
		set := s.ensureConfigurationSetLocked(name, now)
		set["CreatedTimestamp"] = now.Format(time.RFC3339)
		set["DefaultMessageType"] = smsVoiceV2PayloadString(payload, "MessageType", smsVoiceV2StringValue(set, "DefaultMessageType", "TRANSACTIONAL"))
		set["DefaultSenderId"] = smsVoiceV2PayloadString(payload, "SenderId", smsVoiceV2StringValue(set, "DefaultSenderId", "STACKYARD"))
		return map[string]any{"ConfigurationSetName": name, "ConfigurationSetArn": smsVoiceV2StringValue(set, "ConfigurationSetArn", smsVoiceV2ConfigurationSetARN(name))}
	case "DeleteConfigurationSet", "CreateEventDestination", "DeleteEventDestination", "UpdateEventDestination":
		return map[string]any{}
	case "DescribeConfigurationSets":
		return map[string]any{"ConfigurationSets": smsVoiceV2ListMaps(s.configurationSets), "NextToken": ""}

	case "CreateOptOutList":
		list := s.ensureOptOutListLocked(optOutListName, now)
		list["CreatedTimestamp"] = now.Format(time.RFC3339)
		return map[string]any{"OptOutListArn": smsVoiceV2OptOutListARN(optOutListName), "OptOutListName": optOutListName}
	case "DeleteOptOutList":
		delete(s.optOutLists, optOutListName)
		delete(s.optedOutNumbers, optOutListName)
		return map[string]any{}
	case "DescribeOptOutLists":
		return map[string]any{"OptOutLists": smsVoiceV2ListMaps(s.optOutLists), "NextToken": ""}
	case "PutOptedOutNumber":
		numbers := s.ensureOptedOutNumbersLocked(optOutListName)
		numbers[destinationPhoneNumber] = true
		return map[string]any{}
	case "DeleteOptedOutNumber":
		numbers := s.ensureOptedOutNumbersLocked(optOutListName)
		delete(numbers, destinationPhoneNumber)
		return map[string]any{}
	case "DescribeOptedOutNumbers":
		numbers := s.ensureOptedOutNumbersLocked(optOutListName)
		out := make([]any, 0, len(numbers))
		for number := range numbers {
			out = append(out, map[string]any{"OptedOutNumber": number, "OptedOutTimestamp": now.Format(time.RFC3339)})
		}
		sort.Slice(out, func(i, j int) bool {
			li := smsVoiceV2StringValue(out[i].(map[string]any), "OptedOutNumber", "")
			lj := smsVoiceV2StringValue(out[j].(map[string]any), "OptedOutNumber", "")
			return li < lj
		})
		return map[string]any{"OptedOutNumbers": out, "NextToken": ""}

	case "CreatePool":
		id := smsVoiceV2PayloadString(payload, "PoolId", fmt.Sprintf("pool-%06d", s.nextIDLocked()))
		pool := s.ensurePoolLocked(id, now)
		pool["MessageType"] = smsVoiceV2PayloadString(payload, "MessageType", smsVoiceV2StringValue(pool, "MessageType", "TRANSACTIONAL"))
		return map[string]any{"PoolArn": smsVoiceV2PoolARN(id), "PoolId": id}
	case "UpdatePool", "AssociateOriginationIdentity", "DisassociateOriginationIdentity":
		return map[string]any{}
	case "DeletePool":
		delete(s.pools, poolID)
		return map[string]any{}
	case "DescribePools":
		return map[string]any{"Pools": smsVoiceV2ListMaps(s.pools), "NextToken": ""}
	case "ListPoolOriginationIdentities":
		return map[string]any{"OriginationIdentities": []any{originationIdentity}, "NextToken": ""}

	case "CreateProtectConfiguration":
		id := smsVoiceV2PayloadString(payload, "ProtectConfigurationId", fmt.Sprintf("pc-%06d", s.nextIDLocked()))
		s.ensureProtectConfigurationLocked(id, now)
		return map[string]any{"ProtectConfigurationArn": smsVoiceV2ProtectConfigurationARN(id), "ProtectConfigurationId": id}
	case "UpdateProtectConfiguration", "AssociateProtectConfiguration", "DisassociateProtectConfiguration", "UpdateProtectConfigurationCountryRuleSet", "PutProtectConfigurationRuleSetNumberOverride", "DeleteProtectConfigurationRuleSetNumberOverride", "SetAccountDefaultProtectConfiguration", "DeleteAccountDefaultProtectConfiguration":
		return map[string]any{}
	case "DeleteProtectConfiguration":
		delete(s.protectConfigurations, protectConfigurationID)
		return map[string]any{}
	case "DescribeProtectConfigurations":
		return map[string]any{"ProtectConfigurations": smsVoiceV2ListMaps(s.protectConfigurations), "NextToken": ""}
	case "GetProtectConfigurationCountryRuleSet":
		return map[string]any{"ProtectConfigurationId": protectConfigurationID, "IsoCountryCode": smsVoiceV2PayloadString(payload, "IsoCountryCode", "US"), "NumberCapabilities": []any{"SMS"}}
	case "ListProtectConfigurationRuleSetNumberOverrides":
		return map[string]any{"ProtectConfigurationRuleSetNumberOverrides": []any{}, "NextToken": ""}

	case "CreateRegistration":
		id := smsVoiceV2PayloadString(payload, "RegistrationId", fmt.Sprintf("reg-%06d", s.nextIDLocked()))
		reg := s.ensureRegistrationLocked(id, now)
		reg["RegistrationStatus"] = "CREATED"
		return map[string]any{"RegistrationArn": smsVoiceV2RegistrationARN(id), "RegistrationId": id}
	case "DeleteRegistration":
		delete(s.registrations, registrationID)
		return map[string]any{}
	case "DescribeRegistrations":
		return map[string]any{"Registrations": smsVoiceV2ListMaps(s.registrations), "NextToken": ""}
	case "CreateRegistrationVersion":
		return map[string]any{"RegistrationArn": smsVoiceV2RegistrationARN(registrationID), "RegistrationId": registrationID, "VersionNumber": 1}
	case "DescribeRegistrationVersions":
		return map[string]any{"RegistrationVersions": []any{map[string]any{"RegistrationId": registrationID, "VersionNumber": 1, "RegistrationVersionStatus": "DRAFT"}}, "NextToken": ""}
	case "SubmitRegistrationVersion", "DiscardRegistrationVersion":
		return map[string]any{}
	case "CreateRegistrationAssociation", "DeleteRegistrationFieldValue", "PutRegistrationFieldValue":
		return map[string]any{}
	case "ListRegistrationAssociations":
		return map[string]any{"RegistrationAssociations": []any{map[string]any{"RegistrationId": registrationID, "ResourceArn": resourceArn}}, "NextToken": ""}
	case "CreateRegistrationAttachment":
		id := smsVoiceV2PayloadString(payload, "RegistrationAttachmentId", fmt.Sprintf("ra-%06d", s.nextIDLocked()))
		s.ensureRegistrationAttachmentLocked(id, registrationID, now)
		return map[string]any{"RegistrationAttachmentId": id, "RegistrationId": registrationID}
	case "DeleteRegistrationAttachment":
		delete(s.registrationAttachments, registrationAttachmentID)
		return map[string]any{}
	case "DescribeRegistrationAttachments":
		return map[string]any{"RegistrationAttachments": smsVoiceV2ListMaps(s.registrationAttachments), "NextToken": ""}
	case "DescribeRegistrationFieldDefinitions":
		return map[string]any{"RegistrationFieldDefinitions": []any{map[string]any{"FieldPath": "company.name", "FieldRequirement": "OPTIONAL", "FieldType": "TEXT"}}, "NextToken": ""}
	case "DescribeRegistrationSectionDefinitions":
		return map[string]any{"RegistrationSectionDefinitions": []any{map[string]any{"SectionPath": "company", "DisplayHints": map[string]any{"Title": "Company"}}}, "NextToken": ""}
	case "DescribeRegistrationTypeDefinitions":
		return map[string]any{"RegistrationTypeDefinitions": []any{map[string]any{"RegistrationType": "US_A2P_10DLC", "IsoCountryCode": "US"}}, "NextToken": ""}
	case "DescribeRegistrationFieldValues":
		return map[string]any{"RegistrationFieldValues": []any{map[string]any{"FieldPath": "company.name", "FieldValue": "Stackyard"}}, "NextToken": ""}

	case "CreateVerifiedDestinationNumber":
		id := smsVoiceV2PayloadString(payload, "VerifiedDestinationNumberId", fmt.Sprintf("vdn-%06d", s.nextIDLocked()))
		number := smsVoiceV2PayloadString(payload, "DestinationPhoneNumber", destinationPhoneNumber)
		s.ensureVerifiedDestinationNumberLocked(id, number, now)
		return map[string]any{"VerifiedDestinationNumberId": id}
	case "DeleteVerifiedDestinationNumber":
		delete(s.verifiedDestinationNumbers, verifiedDestinationNumberID)
		return map[string]any{}
	case "DescribeVerifiedDestinationNumbers":
		return map[string]any{"VerifiedDestinationNumbers": smsVoiceV2ListMaps(s.verifiedDestinationNumbers), "NextToken": ""}
	case "SendDestinationNumberVerificationCode", "VerifyDestinationNumber":
		return map[string]any{}

	case "RequestPhoneNumber":
		id := smsVoiceV2PayloadString(payload, "PhoneNumberId", fmt.Sprintf("pn-%06d", s.nextIDLocked()))
		phone := s.ensurePhoneNumberLocked(id, now)
		phone["Status"] = "PENDING"
		return map[string]any{"PhoneNumberArn": smsVoiceV2PhoneNumberARN(id), "PhoneNumberId": id, "PhoneNumber": smsVoiceV2StringValue(phone, "PhoneNumber", "+12065550100")}
	case "UpdatePhoneNumber":
		return map[string]any{}
	case "ReleasePhoneNumber":
		delete(s.phoneNumbers, phoneNumberID)
		return map[string]any{}
	case "DescribePhoneNumbers":
		return map[string]any{"PhoneNumbers": smsVoiceV2ListMaps(s.phoneNumbers), "NextToken": ""}

	case "RequestSenderId":
		id := smsVoiceV2PayloadString(payload, "SenderId", senderID)
		s.ensureSenderIDLocked(id, now)
		return map[string]any{"SenderId": id}
	case "UpdateSenderId":
		return map[string]any{}
	case "ReleaseSenderId":
		delete(s.senderIDs, senderID)
		return map[string]any{}
	case "DescribeSenderIds":
		return map[string]any{"SenderIds": smsVoiceV2ListMaps(s.senderIDs), "NextToken": ""}

	case "PutKeyword":
		s.keywords[keyword] = originationIdentity
		return map[string]any{}
	case "DeleteKeyword":
		delete(s.keywords, keyword)
		return map[string]any{}
	case "DescribeKeywords":
		items := make([]any, 0, len(s.keywords))
		for key, identity := range s.keywords {
			items = append(items, map[string]any{"Keyword": key, "OriginationIdentity": identity})
		}
		sort.Slice(items, func(i, j int) bool {
			li := smsVoiceV2StringValue(items[i].(map[string]any), "Keyword", "")
			lj := smsVoiceV2StringValue(items[j].(map[string]any), "Keyword", "")
			return li < lj
		})
		return map[string]any{"Keywords": items, "NextToken": ""}

	case "PutResourcePolicy":
		s.resourcePolicies[resourceArn] = smsVoiceV2PayloadString(payload, "Policy", "{}")
		return map[string]any{}
	case "GetResourcePolicy":
		policy := s.resourcePolicies[resourceArn]
		if strings.TrimSpace(policy) == "" {
			policy = "{}"
		}
		return map[string]any{"Policy": policy}
	case "DeleteResourcePolicy":
		delete(s.resourcePolicies, resourceArn)
		return map[string]any{}

	case "SendTextMessage", "SendMediaMessage", "SendVoiceMessage":
		return map[string]any{"MessageId": fmt.Sprintf("msg-%06d", s.nextIDLocked())}
	case "PutMessageFeedback":
		return map[string]any{}

	case "SetDefaultMessageFeedbackEnabled":
		s.defaults["DefaultMessageFeedbackEnabled"] = smsVoiceV2PayloadBool(payload, "MessageFeedbackEnabled", true)
		return map[string]any{}
	case "SetDefaultMessageType":
		s.defaults["DefaultMessageType"] = smsVoiceV2PayloadString(payload, "MessageType", "TRANSACTIONAL")
		return map[string]any{}
	case "SetDefaultSenderId":
		s.defaults["DefaultSenderId"] = smsVoiceV2PayloadString(payload, "SenderId", "STACKYARD")
		return map[string]any{}
	case "DeleteDefaultMessageType":
		s.defaults["DefaultMessageType"] = "TRANSACTIONAL"
		return map[string]any{}
	case "DeleteDefaultSenderId":
		s.defaults["DefaultSenderId"] = "STACKYARD"
		return map[string]any{}

	case "SetTextMessageSpendLimitOverride":
		s.spendLimits["TextMessage"] = smsVoiceV2PayloadString(payload, "MonthlyLimit", "1000")
		return map[string]any{}
	case "SetVoiceMessageSpendLimitOverride":
		s.spendLimits["VoiceMessage"] = smsVoiceV2PayloadString(payload, "MonthlyLimit", "1000")
		return map[string]any{}
	case "SetMediaMessageSpendLimitOverride":
		s.spendLimits["MediaMessage"] = smsVoiceV2PayloadString(payload, "MonthlyLimit", "1000")
		return map[string]any{}
	case "DeleteTextMessageSpendLimitOverride":
		s.spendLimits["TextMessage"] = "1000"
		return map[string]any{}
	case "DeleteVoiceMessageSpendLimitOverride":
		s.spendLimits["VoiceMessage"] = "1000"
		return map[string]any{}
	case "DeleteMediaMessageSpendLimitOverride":
		s.spendLimits["MediaMessage"] = "1000"
		return map[string]any{}

	case "DescribeAccountAttributes":
		return map[string]any{"AccountAttributes": []any{map[string]any{"Name": "DefaultMessageType", "Value": s.defaults["DefaultMessageType"]}, map[string]any{"Name": "DefaultSenderId", "Value": s.defaults["DefaultSenderId"]}}}
	case "DescribeAccountLimits":
		return map[string]any{"AccountLimits": []any{map[string]any{"Name": "ConfigurationSets", "Used": 1, "Max": 1000}, map[string]any{"Name": "Pools", "Used": 1, "Max": 1000}}}
	case "DescribeSpendLimits":
		return map[string]any{
			"SpendLimits": []any{
				map[string]any{"Name": "TextMessageMonthlySpendLimit", "EnforcedLimit": s.spendLimits["TextMessage"]},
				map[string]any{"Name": "VoiceMessageMonthlySpendLimit", "EnforcedLimit": s.spendLimits["VoiceMessage"]},
				map[string]any{"Name": "MediaMessageMonthlySpendLimit", "EnforcedLimit": s.spendLimits["MediaMessage"]},
			},
		}

	case "TagResource", "UntagResource":
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"Tags": smsVoiceV2TagsToList(s.ensureTagMapLocked(resourceArn))}
	}

	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}
	return map[string]any{}
}

func (s *smsVoiceV2Store) seedLocked(now time.Time) {
	set := s.ensureConfigurationSetLocked("stackyard-sms-config-set", now)
	pool := s.ensurePoolLocked("pool-000001", now)
	protect := s.ensureProtectConfigurationLocked("pc-000001", now)
	registration := s.ensureRegistrationLocked("reg-000001", now)
	attachment := s.ensureRegistrationAttachmentLocked("ra-000001", "reg-000001", now)
	verified := s.ensureVerifiedDestinationNumberLocked("vdn-000001", "+12065550101", now)
	phone := s.ensurePhoneNumberLocked("pn-000001", now)
	sender := s.ensureSenderIDLocked("STACKYARD", now)
	optOut := s.ensureOptOutListLocked("stackyard-optout-list", now)

	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(set, "ConfigurationSetArn", smsVoiceV2ConfigurationSetARN("stackyard-sms-config-set")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(pool, "PoolArn", smsVoiceV2PoolARN("pool-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(protect, "ProtectConfigurationArn", smsVoiceV2ProtectConfigurationARN("pc-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(registration, "RegistrationArn", smsVoiceV2RegistrationARN("reg-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(attachment, "RegistrationAttachmentArn", smsVoiceV2RegistrationAttachmentARN("ra-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(verified, "VerifiedDestinationNumberArn", smsVoiceV2VerifiedDestinationNumberARN("vdn-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(phone, "PhoneNumberArn", smsVoiceV2PhoneNumberARN("pn-000001")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(sender, "SenderIdArn", smsVoiceV2SenderIDARN("STACKYARD")))
	_ = s.ensureTagMapLocked(smsVoiceV2StringValue(optOut, "OptOutListArn", smsVoiceV2OptOutListARN("stackyard-optout-list")))

	if _, ok := s.resourcePolicies[smsVoiceV2ConfigurationSetARN("stackyard-sms-config-set")]; !ok {
		s.resourcePolicies[smsVoiceV2ConfigurationSetARN("stackyard-sms-config-set")] = "{}"
	}
	if _, ok := s.keywords["HELP"]; !ok {
		s.keywords["HELP"] = "arn:aws:sms-voice:us-east-1:123456789012:phone-number/+12065550100"
	}
}

func (s *smsVoiceV2Store) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *smsVoiceV2Store) ensureConfigurationSetLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-sms-config-set"
	}
	if existing := s.configurationSets[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ConfigurationSetName": name,
		"ConfigurationSetArn":  smsVoiceV2ConfigurationSetARN(name),
		"CreatedTimestamp":     now.Format(time.RFC3339),
		"DefaultMessageType":   "TRANSACTIONAL",
		"DefaultSenderId":      "STACKYARD",
	}
	s.configurationSets[name] = item
	return item
}

func (s *smsVoiceV2Store) ensurePoolLocked(poolID string, now time.Time) map[string]any {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
		poolID = "pool-000001"
	}
	if existing := s.pools[poolID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"PoolId":           poolID,
		"PoolArn":          smsVoiceV2PoolARN(poolID),
		"Status":           "ACTIVE",
		"MessageType":      "TRANSACTIONAL",
		"CreatedTimestamp": now.Format(time.RFC3339),
	}
	s.pools[poolID] = item
	return item
}

func (s *smsVoiceV2Store) ensureProtectConfigurationLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "pc-000001"
	}
	if existing := s.protectConfigurations[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ProtectConfigurationId":    id,
		"ProtectConfigurationArn":   smsVoiceV2ProtectConfigurationARN(id),
		"AccountDefault":            false,
		"CreatedTimestamp":          now.Format(time.RFC3339),
		"ProtectStatus":             "ACTIVE",
		"DeletionProtectionEnabled": false,
	}
	s.protectConfigurations[id] = item
	return item
}

func (s *smsVoiceV2Store) ensureRegistrationLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "reg-000001"
	}
	if existing := s.registrations[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"RegistrationId":            id,
		"RegistrationArn":           smsVoiceV2RegistrationARN(id),
		"RegistrationStatus":        "ACTIVE",
		"RegistrationType":          "US_A2P_10DLC",
		"IsoCountryCode":            "US",
		"CurrentVersionNumber":      1,
		"LatestDeniedVersionNumber": 0,
		"CreatedTimestamp":          now.Format(time.RFC3339),
	}
	s.registrations[id] = item
	return item
}

func (s *smsVoiceV2Store) ensureRegistrationAttachmentLocked(id, registrationID string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ra-000001"
	}
	if existing := s.registrationAttachments[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"RegistrationAttachmentId":  id,
		"RegistrationAttachmentArn": smsVoiceV2RegistrationAttachmentARN(id),
		"RegistrationId":            registrationID,
		"AttachmentStatus":          "APPROVED",
		"CreatedTimestamp":          now.Format(time.RFC3339),
	}
	s.registrationAttachments[id] = item
	return item
}

func (s *smsVoiceV2Store) ensureVerifiedDestinationNumberLocked(id, number string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "vdn-000001"
	}
	if existing := s.verifiedDestinationNumbers[id]; existing != nil {
		return existing
	}
	if strings.TrimSpace(number) == "" {
		number = "+12065550101"
	}
	item := map[string]any{
		"VerifiedDestinationNumberId":  id,
		"VerifiedDestinationNumberArn": smsVoiceV2VerifiedDestinationNumberARN(id),
		"DestinationPhoneNumber":       number,
		"VerificationStatus":           "VERIFIED",
		"CreatedTimestamp":             now.Format(time.RFC3339),
	}
	s.verifiedDestinationNumbers[id] = item
	return item
}

func (s *smsVoiceV2Store) ensurePhoneNumberLocked(id string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "pn-000001"
	}
	if existing := s.phoneNumbers[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"PhoneNumberId":      id,
		"PhoneNumberArn":     smsVoiceV2PhoneNumberARN(id),
		"PhoneNumber":        "+12065550100",
		"NumberCapabilities": []any{"SMS", "VOICE"},
		"Status":             "ACTIVE",
		"IsoCountryCode":     "US",
		"CreatedTimestamp":   now.Format(time.RFC3339),
	}
	s.phoneNumbers[id] = item
	return item
}

func (s *smsVoiceV2Store) ensureSenderIDLocked(senderID string, now time.Time) map[string]any {
	senderID = strings.TrimSpace(senderID)
	if senderID == "" {
		senderID = "STACKYARD"
	}
	if existing := s.senderIDs[senderID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"SenderId":         senderID,
		"SenderIdArn":      smsVoiceV2SenderIDARN(senderID),
		"IsoCountryCode":   "US",
		"Status":           "ACTIVE",
		"CreatedTimestamp": now.Format(time.RFC3339),
	}
	s.senderIDs[senderID] = item
	return item
}

func (s *smsVoiceV2Store) ensureOptOutListLocked(name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-optout-list"
	}
	if existing := s.optOutLists[name]; existing != nil {
		return existing
	}
	item := map[string]any{
		"OptOutListName":   name,
		"OptOutListArn":    smsVoiceV2OptOutListARN(name),
		"CreatedTimestamp": now.Format(time.RFC3339),
	}
	s.optOutLists[name] = item
	return item
}

func (s *smsVoiceV2Store) ensureOptedOutNumbersLocked(optOutListName string) map[string]bool {
	optOutListName = strings.TrimSpace(optOutListName)
	if optOutListName == "" {
		optOutListName = "stackyard-optout-list"
	}
	if existing := s.optedOutNumbers[optOutListName]; existing != nil {
		return existing
	}
	out := map[string]bool{}
	s.optedOutNumbers[optOutListName] = out
	return out
}

func (s *smsVoiceV2Store) ensureTagMapLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = smsVoiceV2ConfigurationSetARN("stackyard-sms-config-set")
	}
	if existing := s.tags[resourceArn]; existing != nil {
		return existing
	}
	out := map[string]string{"stackyard": "true"}
	s.tags[resourceArn] = out
	return out
}

func (s *smsVoiceV2Store) applyTagMutationsLocked(action string, payload map[string]any, resourceArn string) {
	tags := s.ensureTagMapLocked(resourceArn)
	switch action {
	case "TagResource":
		for key, value := range smsVoiceV2PayloadTags(payload) {
			tags[key] = value
		}
	case "UntagResource":
		for _, key := range smsVoiceV2PayloadTagKeys(payload) {
			delete(tags, key)
		}
	}
}

func smsVoiceV2PayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func smsVoiceV2PayloadString(payload map[string]any, key, fallback string) string {
	value, ok := smsVoiceV2PayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	case fmt.Stringer:
		out := strings.TrimSpace(typed.String())
		if out == "" {
			return fallback
		}
		return out
	default:
		out := strings.TrimSpace(fmt.Sprint(value))
		if out == "" || out == "<nil>" {
			return fallback
		}
		return out
	}
}

func smsVoiceV2PayloadBool(payload map[string]any, key string, fallback bool) bool {
	value, ok := smsVoiceV2PayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		v := strings.TrimSpace(strings.ToLower(typed))
		if v == "true" || v == "1" || v == "yes" {
			return true
		}
		if v == "false" || v == "0" || v == "no" {
			return false
		}
	}
	return fallback
}

func smsVoiceV2PayloadTags(payload map[string]any) map[string]string {
	value, ok := smsVoiceV2PayloadValue(payload, "Tags")
	if !ok {
		value, ok = smsVoiceV2PayloadValue(payload, "tags")
		if !ok {
			return map[string]string{}
		}
	}

	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, val := range typed {
			out[key] = val
		}
	case map[string]any:
		for key, val := range typed {
			out[key] = strings.TrimSpace(fmt.Sprint(val))
		}
	case []any:
		for _, entry := range typed {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			key := smsVoiceV2StringValue(item, "Key", smsVoiceV2StringValue(item, "key", ""))
			if key == "" {
				continue
			}
			val := smsVoiceV2StringValue(item, "Value", smsVoiceV2StringValue(item, "value", ""))
			out[key] = val
		}
	}
	return out
}

func smsVoiceV2PayloadTagKeys(payload map[string]any) []string {
	value, ok := smsVoiceV2PayloadValue(payload, "TagKeys")
	if !ok {
		value, ok = smsVoiceV2PayloadValue(payload, "tagKeys")
		if !ok {
			return []string{}
		}
	}
	out := []string{}
	switch typed := value.(type) {
	case []string:
		for _, key := range typed {
			key = strings.TrimSpace(key)
			if key != "" {
				out = append(out, key)
			}
		}
	case []any:
		for _, item := range typed {
			key := strings.TrimSpace(fmt.Sprint(item))
			if key != "" && key != "<nil>" {
				out = append(out, key)
			}
		}
	}
	return out
}

func smsVoiceV2ListMaps(values map[string]map[string]any) []any {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, smsVoiceV2CloneMap(values[key]))
	}
	return out
}

func smsVoiceV2CloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func smsVoiceV2StringValue(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for current, value := range payload {
		if strings.EqualFold(current, key) {
			out := strings.TrimSpace(fmt.Sprint(value))
			if out != "" && out != "<nil>" {
				return out
			}
		}
	}
	return fallback
}

func smsVoiceV2TagsToList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func smsVoiceV2ConfigurationSetARN(name string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:configuration-set/" + strings.TrimSpace(name)
}

func smsVoiceV2PoolARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:pool/" + strings.TrimSpace(id)
}

func smsVoiceV2ProtectConfigurationARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:protect-configuration/" + strings.TrimSpace(id)
}

func smsVoiceV2RegistrationARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:registration/" + strings.TrimSpace(id)
}

func smsVoiceV2RegistrationAttachmentARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:registration-attachment/" + strings.TrimSpace(id)
}

func smsVoiceV2VerifiedDestinationNumberARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:verified-destination-number/" + strings.TrimSpace(id)
}

func smsVoiceV2PhoneNumberARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:phone-number/" + strings.TrimSpace(id)
}

func smsVoiceV2SenderIDARN(id string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:sender-id/" + strings.TrimSpace(id)
}

func smsVoiceV2OptOutListARN(name string) string {
	return "arn:aws:sms-voice:us-east-1:123456789012:opt-out-list/" + strings.TrimSpace(name)
}
