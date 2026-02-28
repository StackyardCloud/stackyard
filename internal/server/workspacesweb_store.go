package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	workspacesWebDefaultRegion    = "us-east-1"
	workspacesWebDefaultAccountID = "123456789012"
)

type workspacesWebStore struct {
	mu sync.Mutex

	nextID int64

	browserSettings           map[string]map[string]any
	dataProtectionSettings    map[string]map[string]any
	identityProviders         map[string]map[string]any
	ipAccessSettings          map[string]map[string]any
	networkSettings           map[string]map[string]any
	portals                   map[string]map[string]any
	sessionLoggers            map[string]map[string]any
	trustStores               map[string]map[string]any
	userAccessLoggingSettings map[string]map[string]any
	userSettings              map[string]map[string]any

	portalIdentityProviders map[string]map[string]struct{}
	sessions                map[string]map[string]map[string]any // portalId -> sessionId -> session
	trustStoreCertificates  map[string]map[string]map[string]any // trustStoreArn -> thumbprint -> certificate
	tags                    map[string]map[string]string
}

func newWorkSpacesWebStore() *workspacesWebStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &workspacesWebStore{
		nextID:                    2,
		browserSettings:           map[string]map[string]any{},
		dataProtectionSettings:    map[string]map[string]any{},
		identityProviders:         map[string]map[string]any{},
		ipAccessSettings:          map[string]map[string]any{},
		networkSettings:           map[string]map[string]any{},
		portals:                   map[string]map[string]any{},
		sessionLoggers:            map[string]map[string]any{},
		trustStores:               map[string]map[string]any{},
		userAccessLoggingSettings: map[string]map[string]any{},
		userSettings:              map[string]map[string]any{},
		portalIdentityProviders:   map[string]map[string]struct{}{},
		sessions:                  map[string]map[string]map[string]any{},
		trustStoreCertificates:    map[string]map[string]map[string]any{},
		tags:                      map[string]map[string]string{},
	}

	browserSettingsArn := workspacesWebResourceARN("browserSettings", "bs-000001")
	dataProtectionSettingsArn := workspacesWebResourceARN("dataProtectionSettings", "dps-000001")
	identityProviderArn := workspacesWebResourceARN("identityProvider", "idp-000001")
	ipAccessSettingsArn := workspacesWebResourceARN("ipAccessSettings", "ipa-000001")
	networkSettingsArn := workspacesWebResourceARN("networkSettings", "ns-000001")
	portalArn := workspacesWebResourceARN("portal", "p-000001")
	sessionLoggerArn := workspacesWebResourceARN("sessionLogger", "sl-000001")
	trustStoreArn := workspacesWebResourceARN("trustStore", "ts-000001")
	userAccessLoggingSettingsArn := workspacesWebResourceARN("userAccessLoggingSettings", "ual-000001")
	userSettingsArn := workspacesWebResourceARN("userSettings", "us-000001")

	s.ensureBrowserSettingsLocked(browserSettingsArn, now)
	s.ensureDataProtectionSettingsLocked(dataProtectionSettingsArn, now)
	s.ensureIdentityProviderLocked(identityProviderArn, portalArn, now)
	s.ensureIPAccessSettingsLocked(ipAccessSettingsArn, now)
	s.ensureNetworkSettingsLocked(networkSettingsArn, now)
	s.ensureSessionLoggerLocked(sessionLoggerArn, now)
	s.ensureTrustStoreLocked(trustStoreArn, now)
	s.ensureUserAccessLoggingSettingsLocked(userAccessLoggingSettingsArn, now)
	s.ensureUserSettingsLocked(userSettingsArn, now)

	portal := s.ensurePortalLocked(portalArn, now)
	portal["browserSettingsArn"] = browserSettingsArn
	portal["dataProtectionSettingsArn"] = dataProtectionSettingsArn
	portal["ipAccessSettingsArn"] = ipAccessSettingsArn
	portal["networkSettingsArn"] = networkSettingsArn
	portal["sessionLoggerArn"] = sessionLoggerArn
	portal["trustStoreArn"] = trustStoreArn
	portal["userAccessLoggingSettingsArn"] = userAccessLoggingSettingsArn
	portal["userSettingsArn"] = userSettingsArn

	s.ensurePortalIdentityProviderSetLocked(portalArn)[identityProviderArn] = struct{}{}
	s.ensureSessionLocked("p-000001", portalArn, "s-000001", now)
	s.ensureTrustStoreCertificateLocked(trustStoreArn, "aa", now)
	s.ensureTagsLocked(portalArn)["stackyard"] = "true"

	return s
}

func (s *workspacesWebStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	portalArn := workspacesWebLookupString(payload, pathParams, query, "portalArn")
	if portalArn == "" {
		portalArn = workspacesWebResourceARN("portal", "p-000001")
	}
	portal := s.ensurePortalLocked(portalArn, now)
	portalID := workspacesWebLookupString(payload, pathParams, query, "portalId")
	if portalID == "" {
		portalID = workspacesWebString(portal, "portalId", "p-000001")
	}
	resourceArn := workspacesWebLookupString(payload, pathParams, query, "resourceArn")
	if resourceArn == "" {
		resourceArn = portalArn
	}

	s.ensureTagsLocked(resourceArn)
	s.ensureSessionsMapLocked(portalID)

	s.applyGenericPayloadUpdatesLocked(action, payload, now)

	switch action {
	case "CreateBrowserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "browserSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("browserSettings", s.nextTokenLocked("bs"))
		}
		item := s.ensureBrowserSettingsLocked(arn, now)
		return map[string]any{"browserSettingsArn": item["browserSettingsArn"]}
	case "CreateDataProtectionSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "dataProtectionSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("dataProtectionSettings", s.nextTokenLocked("dps"))
		}
		item := s.ensureDataProtectionSettingsLocked(arn, now)
		return map[string]any{"dataProtectionSettingsArn": item["dataProtectionSettingsArn"]}
	case "CreateIdentityProvider":
		arn := workspacesWebLookupString(payload, pathParams, query, "identityProviderArn")
		if arn == "" {
			arn = workspacesWebResourceARN("identityProvider", s.nextTokenLocked("idp"))
		}
		item := s.ensureIdentityProviderLocked(arn, portalArn, now)
		s.ensurePortalIdentityProviderSetLocked(portalArn)[arn] = struct{}{}
		return map[string]any{"identityProviderArn": item["identityProviderArn"]}
	case "CreateIpAccessSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "ipAccessSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("ipAccessSettings", s.nextTokenLocked("ipa"))
		}
		item := s.ensureIPAccessSettingsLocked(arn, now)
		return map[string]any{"ipAccessSettingsArn": item["ipAccessSettingsArn"]}
	case "CreateNetworkSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "networkSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("networkSettings", s.nextTokenLocked("ns"))
		}
		item := s.ensureNetworkSettingsLocked(arn, now)
		return map[string]any{"networkSettingsArn": item["networkSettingsArn"]}
	case "CreatePortal":
		arn := workspacesWebLookupString(payload, pathParams, query, "portalArn")
		if arn == "" {
			arn = workspacesWebResourceARN("portal", s.nextTokenLocked("p"))
		}
		item := s.ensurePortalLocked(arn, now)
		s.ensureSessionsMapLocked(workspacesWebString(item, "portalId", "p-000001"))
		return map[string]any{"portalArn": item["portalArn"], "portalId": item["portalId"]}
	case "CreateSessionLogger":
		arn := workspacesWebLookupString(payload, pathParams, query, "sessionLoggerArn")
		if arn == "" {
			arn = workspacesWebResourceARN("sessionLogger", s.nextTokenLocked("sl"))
		}
		item := s.ensureSessionLoggerLocked(arn, now)
		return map[string]any{"sessionLoggerArn": item["sessionLoggerArn"]}
	case "CreateTrustStore":
		arn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		if arn == "" {
			arn = workspacesWebResourceARN("trustStore", s.nextTokenLocked("ts"))
		}
		item := s.ensureTrustStoreLocked(arn, now)
		s.ensureTrustStoreCertificateLocked(arn, "aa", now)
		return map[string]any{"trustStoreArn": item["trustStoreArn"]}
	case "CreateUserAccessLoggingSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userAccessLoggingSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("userAccessLoggingSettings", s.nextTokenLocked("ual"))
		}
		item := s.ensureUserAccessLoggingSettingsLocked(arn, now)
		return map[string]any{"userAccessLoggingSettingsArn": item["userAccessLoggingSettingsArn"]}
	case "CreateUserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userSettingsArn")
		if arn == "" {
			arn = workspacesWebResourceARN("userSettings", s.nextTokenLocked("us"))
		}
		item := s.ensureUserSettingsLocked(arn, now)
		return map[string]any{"userSettingsArn": item["userSettingsArn"]}

	case "UpdateBrowserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "browserSettingsArn")
		item := s.ensureBrowserSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"browserSettingsArn": item["browserSettingsArn"]}
	case "UpdateDataProtectionSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "dataProtectionSettingsArn")
		item := s.ensureDataProtectionSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"dataProtectionSettingsArn": item["dataProtectionSettingsArn"]}
	case "UpdateIdentityProvider":
		arn := workspacesWebLookupString(payload, pathParams, query, "identityProviderArn")
		item := s.ensureIdentityProviderLocked(arn, portalArn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"identityProviderArn": item["identityProviderArn"]}
	case "UpdateIpAccessSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "ipAccessSettingsArn")
		item := s.ensureIPAccessSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"ipAccessSettingsArn": item["ipAccessSettingsArn"]}
	case "UpdateNetworkSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "networkSettingsArn")
		item := s.ensureNetworkSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"networkSettingsArn": item["networkSettingsArn"]}
	case "UpdatePortal":
		item := s.ensurePortalLocked(portalArn, now)
		item["lastUpdatedTime"] = now
		if v := workspacesWebLookupString(payload, pathParams, query, "displayName", "portalDisplayName"); v != "" {
			item["displayName"] = v
		}
		return map[string]any{"portalArn": item["portalArn"], "portalId": item["portalId"]}
	case "UpdateSessionLogger":
		arn := workspacesWebLookupString(payload, pathParams, query, "sessionLoggerArn")
		item := s.ensureSessionLoggerLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"sessionLoggerArn": item["sessionLoggerArn"]}
	case "UpdateTrustStore":
		arn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		item := s.ensureTrustStoreLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"trustStoreArn": item["trustStoreArn"]}
	case "UpdateUserAccessLoggingSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userAccessLoggingSettingsArn")
		item := s.ensureUserAccessLoggingSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"userAccessLoggingSettingsArn": item["userAccessLoggingSettingsArn"]}
	case "UpdateUserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userSettingsArn")
		item := s.ensureUserSettingsLocked(arn, now)
		item["lastUpdatedTime"] = now
		return map[string]any{"userSettingsArn": item["userSettingsArn"]}

	case "DeleteBrowserSettings":
		delete(s.browserSettings, workspacesWebLookupString(payload, pathParams, query, "browserSettingsArn"))
		return map[string]any{}
	case "DeleteDataProtectionSettings":
		delete(s.dataProtectionSettings, workspacesWebLookupString(payload, pathParams, query, "dataProtectionSettingsArn"))
		return map[string]any{}
	case "DeleteIdentityProvider":
		arn := workspacesWebLookupString(payload, pathParams, query, "identityProviderArn")
		delete(s.identityProviders, arn)
		for portalKey := range s.portalIdentityProviders {
			delete(s.portalIdentityProviders[portalKey], arn)
		}
		return map[string]any{}
	case "DeleteIpAccessSettings":
		delete(s.ipAccessSettings, workspacesWebLookupString(payload, pathParams, query, "ipAccessSettingsArn"))
		return map[string]any{}
	case "DeleteNetworkSettings":
		delete(s.networkSettings, workspacesWebLookupString(payload, pathParams, query, "networkSettingsArn"))
		return map[string]any{}
	case "DeletePortal":
		id := workspacesWebString(portal, "portalId", "p-000001")
		delete(s.portals, portalArn)
		delete(s.portalIdentityProviders, portalArn)
		delete(s.sessions, id)
		return map[string]any{}
	case "DeleteSessionLogger":
		delete(s.sessionLoggers, workspacesWebLookupString(payload, pathParams, query, "sessionLoggerArn"))
		return map[string]any{}
	case "DeleteTrustStore":
		arn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		delete(s.trustStores, arn)
		delete(s.trustStoreCertificates, arn)
		return map[string]any{}
	case "DeleteUserAccessLoggingSettings":
		delete(s.userAccessLoggingSettings, workspacesWebLookupString(payload, pathParams, query, "userAccessLoggingSettingsArn"))
		return map[string]any{}
	case "DeleteUserSettings":
		delete(s.userSettings, workspacesWebLookupString(payload, pathParams, query, "userSettingsArn"))
		return map[string]any{}

	case "AssociateBrowserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "browserSettingsArn")
		portal["browserSettingsArn"] = s.ensureBrowserSettingsLocked(arn, now)["browserSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateDataProtectionSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "dataProtectionSettingsArn")
		portal["dataProtectionSettingsArn"] = s.ensureDataProtectionSettingsLocked(arn, now)["dataProtectionSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateIpAccessSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "ipAccessSettingsArn")
		portal["ipAccessSettingsArn"] = s.ensureIPAccessSettingsLocked(arn, now)["ipAccessSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateNetworkSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "networkSettingsArn")
		portal["networkSettingsArn"] = s.ensureNetworkSettingsLocked(arn, now)["networkSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateSessionLogger":
		arn := workspacesWebLookupString(payload, pathParams, query, "sessionLoggerArn")
		portal["sessionLoggerArn"] = s.ensureSessionLoggerLocked(arn, now)["sessionLoggerArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateTrustStore":
		arn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		portal["trustStoreArn"] = s.ensureTrustStoreLocked(arn, now)["trustStoreArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateUserAccessLoggingSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userAccessLoggingSettingsArn")
		portal["userAccessLoggingSettingsArn"] = s.ensureUserAccessLoggingSettingsLocked(arn, now)["userAccessLoggingSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "AssociateUserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userSettingsArn")
		portal["userSettingsArn"] = s.ensureUserSettingsLocked(arn, now)["userSettingsArn"]
		portal["lastUpdatedTime"] = now
		return map[string]any{}

	case "DisassociateBrowserSettings":
		delete(portal, "browserSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateDataProtectionSettings":
		delete(portal, "dataProtectionSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateIpAccessSettings":
		delete(portal, "ipAccessSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateNetworkSettings":
		delete(portal, "networkSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateSessionLogger":
		delete(portal, "sessionLoggerArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateTrustStore":
		delete(portal, "trustStoreArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateUserAccessLoggingSettings":
		delete(portal, "userAccessLoggingSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}
	case "DisassociateUserSettings":
		delete(portal, "userSettingsArn")
		portal["lastUpdatedTime"] = now
		return map[string]any{}

	case "GetBrowserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "browserSettingsArn")
		return workspacesWebCloneMap(s.ensureBrowserSettingsLocked(arn, now))
	case "GetDataProtectionSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "dataProtectionSettingsArn")
		return workspacesWebCloneMap(s.ensureDataProtectionSettingsLocked(arn, now))
	case "GetIdentityProvider":
		arn := workspacesWebLookupString(payload, pathParams, query, "identityProviderArn")
		return workspacesWebCloneMap(s.ensureIdentityProviderLocked(arn, portalArn, now))
	case "GetIpAccessSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "ipAccessSettingsArn")
		return workspacesWebCloneMap(s.ensureIPAccessSettingsLocked(arn, now))
	case "GetNetworkSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "networkSettingsArn")
		return workspacesWebCloneMap(s.ensureNetworkSettingsLocked(arn, now))
	case "GetPortal":
		return workspacesWebCloneMap(portal)
	case "GetPortalServiceProviderMetadata":
		return map[string]any{
			"portalArn":                   portalArn,
			"serviceProviderSamlMetadata": fmt.Sprintf("<EntityDescriptor entityID=\"%s\"></EntityDescriptor>", portalArn),
		}
	case "GetSession":
		sessionID := workspacesWebLookupString(payload, pathParams, query, "sessionId")
		if sessionID == "" {
			sessionID = "s-000001"
		}
		return workspacesWebCloneMap(s.ensureSessionLocked(portalID, portalArn, sessionID, now))
	case "GetSessionLogger":
		arn := workspacesWebLookupString(payload, pathParams, query, "sessionLoggerArn")
		return workspacesWebCloneMap(s.ensureSessionLoggerLocked(arn, now))
	case "GetTrustStore":
		arn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		return workspacesWebCloneMap(s.ensureTrustStoreLocked(arn, now))
	case "GetTrustStoreCertificate":
		trustStoreArn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		thumbprint := workspacesWebLookupString(payload, pathParams, query, "thumbprint")
		if thumbprint == "" {
			thumbprint = "aa"
		}
		return workspacesWebCloneMap(s.ensureTrustStoreCertificateLocked(trustStoreArn, thumbprint, now))
	case "GetUserAccessLoggingSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userAccessLoggingSettingsArn")
		return workspacesWebCloneMap(s.ensureUserAccessLoggingSettingsLocked(arn, now))
	case "GetUserSettings":
		arn := workspacesWebLookupString(payload, pathParams, query, "userSettingsArn")
		return workspacesWebCloneMap(s.ensureUserSettingsLocked(arn, now))

	case "ListBrowserSettings":
		return map[string]any{"browserSettings": s.listResourcesLocked(s.browserSettings), "nextToken": ""}
	case "ListDataProtectionSettings":
		return map[string]any{"dataProtectionSettings": s.listResourcesLocked(s.dataProtectionSettings), "nextToken": ""}
	case "ListIdentityProviders":
		identityProviders := []any{}
		set := s.ensurePortalIdentityProviderSetLocked(portalArn)
		arns := make([]string, 0, len(set))
		for arn := range set {
			arns = append(arns, arn)
		}
		sort.Strings(arns)
		for _, arn := range arns {
			identityProviders = append(identityProviders, workspacesWebCloneMap(s.ensureIdentityProviderLocked(arn, portalArn, now)))
		}
		return map[string]any{"identityProviders": identityProviders, "nextToken": ""}
	case "ListIpAccessSettings":
		return map[string]any{"ipAccessSettings": s.listResourcesLocked(s.ipAccessSettings), "nextToken": ""}
	case "ListNetworkSettings":
		return map[string]any{"networkSettings": s.listResourcesLocked(s.networkSettings), "nextToken": ""}
	case "ListPortals":
		return map[string]any{"portals": s.listResourcesLocked(s.portals), "nextToken": ""}
	case "ListSessionLoggers":
		return map[string]any{"sessionLoggers": s.listResourcesLocked(s.sessionLoggers), "nextToken": ""}
	case "ListSessions":
		sessions := []any{}
		for _, sessionID := range s.sortedSessionIDsLocked(portalID) {
			session := s.sessions[portalID][sessionID]
			if session == nil {
				continue
			}
			if requested := workspacesWebLookupString(payload, pathParams, query, "status"); requested != "" && !strings.EqualFold(workspacesWebString(session, "status", ""), requested) {
				continue
			}
			if requested := workspacesWebLookupString(payload, pathParams, query, "sessionId"); requested != "" && !strings.EqualFold(workspacesWebString(session, "sessionId", ""), requested) {
				continue
			}
			if requested := workspacesWebLookupString(payload, pathParams, query, "username"); requested != "" && !strings.EqualFold(workspacesWebString(session, "username", ""), requested) {
				continue
			}
			sessions = append(sessions, workspacesWebCloneMap(session))
		}
		if len(sessions) == 0 {
			sessions = append(sessions, workspacesWebCloneMap(s.ensureSessionLocked(portalID, portalArn, "s-000001", now)))
		}
		return map[string]any{"sessions": sessions, "nextToken": ""}
	case "ListTagsForResource":
		return map[string]any{"tags": workspacesWebCloneStringMap(s.ensureTagsLocked(resourceArn))}
	case "ListTrustStoreCertificates":
		trustStoreArn := workspacesWebLookupString(payload, pathParams, query, "trustStoreArn")
		certsByThumbprint := s.ensureTrustStoreCertificatesLocked(trustStoreArn)
		thumbprints := make([]string, 0, len(certsByThumbprint))
		for thumbprint := range certsByThumbprint {
			thumbprints = append(thumbprints, thumbprint)
		}
		sort.Strings(thumbprints)
		certificates := make([]any, 0, len(thumbprints))
		for _, thumbprint := range thumbprints {
			certificates = append(certificates, workspacesWebCloneMap(certsByThumbprint[thumbprint]))
		}
		return map[string]any{"certificates": certificates, "nextToken": ""}
	case "ListTrustStores":
		return map[string]any{"trustStores": s.listResourcesLocked(s.trustStores), "nextToken": ""}
	case "ListUserAccessLoggingSettings":
		return map[string]any{"userAccessLoggingSettings": s.listResourcesLocked(s.userAccessLoggingSettings), "nextToken": ""}
	case "ListUserSettings":
		return map[string]any{"userSettings": s.listResourcesLocked(s.userSettings), "nextToken": ""}

	case "ExpireSession":
		sessionID := workspacesWebLookupString(payload, pathParams, query, "sessionId")
		if sessionID == "" {
			sessionID = "s-000001"
		}
		session := s.ensureSessionLocked(portalID, portalArn, sessionID, now)
		session["status"] = "EXPIRED"
		session["endTime"] = now
		session["lastUpdatedTime"] = now
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for key, value := range workspacesWebExtractTags(payload) {
			tags[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range workspacesWebExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *workspacesWebStore) applyGenericPayloadUpdatesLocked(action string, payload map[string]any, now string) {
	_ = action
	description := workspacesWebLookupPayloadString(payload, "description", "Description")
	if description == "" {
		return
	}
	for _, resources := range []map[string]map[string]any{
		s.browserSettings,
		s.dataProtectionSettings,
		s.identityProviders,
		s.ipAccessSettings,
		s.networkSettings,
		s.portals,
		s.sessionLoggers,
		s.trustStores,
		s.userAccessLoggingSettings,
		s.userSettings,
	} {
		for _, resource := range resources {
			resource["description"] = description
			resource["lastUpdatedTime"] = now
		}
	}
}

func (s *workspacesWebStore) ensureBrowserSettingsLocked(arn, now string) map[string]any {
	return s.ensureResourceLocked(s.browserSettings, "browserSettings", "bs", "browserSettingsArn", arn, now)
}

func (s *workspacesWebStore) ensureDataProtectionSettingsLocked(arn, now string) map[string]any {
	return s.ensureResourceLocked(s.dataProtectionSettings, "dataProtectionSettings", "dps", "dataProtectionSettingsArn", arn, now)
}

func (s *workspacesWebStore) ensureIdentityProviderLocked(arn, portalArn, now string) map[string]any {
	item := s.ensureResourceLocked(s.identityProviders, "identityProvider", "idp", "identityProviderArn", arn, now)
	if portalArn != "" {
		item["portalArn"] = portalArn
	}
	item["identityProviderType"] = "SAML"
	return item
}

func (s *workspacesWebStore) ensureIPAccessSettingsLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.ipAccessSettings, "ipAccessSettings", "ipa", "ipAccessSettingsArn", arn, now)
	item["ipRules"] = []any{map[string]any{"ipRange": "10.0.0.0/24", "description": "stackyard"}}
	return item
}

func (s *workspacesWebStore) ensureNetworkSettingsLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.networkSettings, "networkSettings", "ns", "networkSettingsArn", arn, now)
	item["vpcId"] = "vpc-000001"
	item["subnetIds"] = []any{"subnet-000001"}
	return item
}

func (s *workspacesWebStore) ensurePortalLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.portals, "portal", "p", "portalArn", arn, now)
	item["portalId"] = workspacesWebIDFromARN(workspacesWebString(item, "portalArn", arn))
	if workspacesWebString(item, "status", "") == "" {
		item["status"] = "ACTIVE"
	}
	item["portalEndpoint"] = fmt.Sprintf("https://%s.workspaces-web.%s.amazonaws.com", item["portalId"], workspacesWebDefaultRegion)
	return item
}

func (s *workspacesWebStore) ensureSessionLoggerLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.sessionLoggers, "sessionLogger", "sl", "sessionLoggerArn", arn, now)
	item["logConfiguration"] = map[string]any{"logType": "WebAccess", "enabled": true}
	return item
}

func (s *workspacesWebStore) ensureTrustStoreLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.trustStores, "trustStore", "ts", "trustStoreArn", arn, now)
	item["status"] = "ACTIVE"
	return item
}

func (s *workspacesWebStore) ensureUserAccessLoggingSettingsLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.userAccessLoggingSettings, "userAccessLoggingSettings", "ual", "userAccessLoggingSettingsArn", arn, now)
	item["kinesisStreamArn"] = "arn:aws:kinesis:us-east-1:123456789012:stream/workspaces-web"
	return item
}

func (s *workspacesWebStore) ensureUserSettingsLocked(arn, now string) map[string]any {
	item := s.ensureResourceLocked(s.userSettings, "userSettings", "us", "userSettingsArn", arn, now)
	item["copyAllowed"] = "Enabled"
	item["pasteAllowed"] = "Enabled"
	item["downloadAllowed"] = "Enabled"
	return item
}

func (s *workspacesWebStore) ensureSessionLocked(portalID, portalArn, sessionID, now string) map[string]any {
	portalID = strings.TrimSpace(portalID)
	if portalID == "" {
		portalID = "p-000001"
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "s-000001"
	}
	s.ensureSessionsMapLocked(portalID)
	if existing := s.sessions[portalID][sessionID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"sessionId":       sessionID,
		"portalId":        portalID,
		"portalArn":       portalArn,
		"username":        "stackyard-user",
		"status":          "ACTIVE",
		"startTime":       now,
		"lastUpdatedTime": now,
	}
	s.sessions[portalID][sessionID] = item
	return item
}

func (s *workspacesWebStore) ensureTrustStoreCertificateLocked(trustStoreArn, thumbprint, now string) map[string]any {
	trustStoreArn = strings.TrimSpace(trustStoreArn)
	if trustStoreArn == "" {
		trustStoreArn = workspacesWebResourceARN("trustStore", "ts-000001")
	}
	thumbprint = strings.TrimSpace(thumbprint)
	if thumbprint == "" {
		thumbprint = "aa"
	}
	certs := s.ensureTrustStoreCertificatesLocked(trustStoreArn)
	if existing := certs[thumbprint]; existing != nil {
		return existing
	}
	item := map[string]any{
		"trustStoreArn":  trustStoreArn,
		"thumbprint":     thumbprint,
		"subject":        "CN=stackyard-workspaces-web",
		"issuer":         "CN=stackyard-ca",
		"validNotBefore": now,
		"validNotAfter":  now,
	}
	certs[thumbprint] = item
	return item
}

func (s *workspacesWebStore) ensureTrustStoreCertificatesLocked(trustStoreArn string) map[string]map[string]any {
	trustStoreArn = strings.TrimSpace(trustStoreArn)
	if trustStoreArn == "" {
		trustStoreArn = workspacesWebResourceARN("trustStore", "ts-000001")
	}
	if existing := s.trustStoreCertificates[trustStoreArn]; existing != nil {
		return existing
	}
	created := map[string]map[string]any{}
	s.trustStoreCertificates[trustStoreArn] = created
	return created
}

func (s *workspacesWebStore) ensurePortalIdentityProviderSetLocked(portalArn string) map[string]struct{} {
	portalArn = strings.TrimSpace(portalArn)
	if portalArn == "" {
		portalArn = workspacesWebResourceARN("portal", "p-000001")
	}
	if existing := s.portalIdentityProviders[portalArn]; existing != nil {
		return existing
	}
	created := map[string]struct{}{}
	s.portalIdentityProviders[portalArn] = created
	return created
}

func (s *workspacesWebStore) ensureSessionsMapLocked(portalID string) map[string]map[string]any {
	portalID = strings.TrimSpace(portalID)
	if portalID == "" {
		portalID = "p-000001"
	}
	if existing := s.sessions[portalID]; existing != nil {
		return existing
	}
	created := map[string]map[string]any{}
	s.sessions[portalID] = created
	return created
}

func (s *workspacesWebStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = workspacesWebResourceARN("portal", "p-000001")
	}
	if existing := s.tags[resourceArn]; existing != nil {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceArn] = created
	return created
}

func (s *workspacesWebStore) ensureResourceLocked(resources map[string]map[string]any, resourceType, idPrefix, arnKey, arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = workspacesWebResourceARN(resourceType, idPrefix+"-000001")
	}
	if existing := resources[arn]; existing != nil {
		return existing
	}
	id := workspacesWebIDFromARN(arn)
	if id == "" {
		id = idPrefix + "-000001"
		arn = workspacesWebResourceARN(resourceType, id)
	}
	item := map[string]any{
		arnKey:            arn,
		"displayName":     "stackyard-" + resourceType + "-" + id,
		"description":     "stackyard " + resourceType,
		"createdTime":     now,
		"lastUpdatedTime": now,
	}
	resources[arn] = item
	return item
}

func (s *workspacesWebStore) listResourcesLocked(resources map[string]map[string]any) []any {
	arns := make([]string, 0, len(resources))
	for arn := range resources {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, workspacesWebCloneMap(resources[arn]))
	}
	return out
}

func (s *workspacesWebStore) sortedSessionIDsLocked(portalID string) []string {
	sessionMap := s.ensureSessionsMapLocked(portalID)
	ids := make([]string, 0, len(sessionMap))
	for id := range sessionMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *workspacesWebStore) nextTokenLocked(prefix string) string {
	id := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return id
}

func workspacesWebResourceARN(resourceType, id string) string {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		resourceType = "portal"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = "p-000001"
	}
	return fmt.Sprintf("arn:aws:workspaces-web:%s:%s:%s/%s", workspacesWebDefaultRegion, workspacesWebDefaultAccountID, resourceType, id)
}

func workspacesWebIDFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func workspacesWebLookupString(payload map[string]any, pathParams map[string]string, query url.Values, keys ...string) string {
	for _, key := range keys {
		if value := workspacesWebLookupMapString(pathParams, key); value != "" {
			return value
		}
		if value := workspacesWebLookupMapString(pathParams, strings.TrimSuffix(key, "+")); value != "" {
			return value
		}
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
		if value := strings.TrimSpace(query.Get(strings.TrimSuffix(key, "+"))); value != "" {
			return value
		}
		if value := workspacesWebLookupPayloadString(payload, key); value != "" {
			return value
		}
		if value := workspacesWebLookupPayloadString(payload, strings.TrimSuffix(key, "+")); value != "" {
			return value
		}
	}
	return ""
}

func workspacesWebLookupMapString(values map[string]string, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func workspacesWebLookupPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			if out := workspacesWebAnyToString(value); out != "" {
				return out
			}
		}
		for k, value := range payload {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				if out := workspacesWebAnyToString(value); out != "" {
					return out
				}
			}
		}
	}
	return ""
}

func workspacesWebAnyToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []string:
		if len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
	case []any:
		if len(v) > 0 {
			return workspacesWebAnyToString(v[0])
		}
	}
	return ""
}

func workspacesWebExtractTags(payload map[string]any) map[string]string {
	tags := map[string]string{}

	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case map[string]any:
			for tagKey, tagValue := range v {
				tagKey = strings.TrimSpace(tagKey)
				if tagKey == "" {
					continue
				}
				tags[tagKey] = workspacesWebAnyToString(tagValue)
			}
		case map[string]string:
			for tagKey, tagValue := range v {
				tagKey = strings.TrimSpace(tagKey)
				if tagKey == "" {
					continue
				}
				tags[tagKey] = strings.TrimSpace(tagValue)
			}
		case []any:
			for _, entry := range v {
				entryMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				tagKey := workspacesWebLookupPayloadString(entryMap, "key", "Key")
				if tagKey == "" {
					continue
				}
				tags[tagKey] = workspacesWebLookupPayloadString(entryMap, "value", "Value")
			}
		}
	}

	if len(tags) == 0 {
		tags["env"] = "coverage"
	}
	return tags
}

func workspacesWebExtractTagKeys(payload map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := []string{}
	appendKey := func(value string) {
		value = strings.TrimSpace(strings.Trim(value, "[]\"'"))
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, key := range []string{"tagKeys", "TagKeys"} {
		for _, raw := range query[key] {
			for _, part := range strings.Split(raw, ",") {
				appendKey(part)
			}
		}
	}
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if raw, ok := payload[key]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					appendKey(workspacesWebAnyToString(item))
				}
			case []string:
				for _, item := range v {
					appendKey(item)
				}
			default:
				appendKey(workspacesWebAnyToString(v))
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "env")
	}
	return out
}

func workspacesWebString(item map[string]any, key, fallback string) string {
	if item == nil {
		return fallback
	}
	if value, ok := item[key]; ok {
		if out := workspacesWebAnyToString(value); out != "" {
			return out
		}
	}
	return fallback
}

func workspacesWebCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = workspacesWebCloneMap(typed)
		case map[string]string:
			out[k] = workspacesWebCloneStringMap(typed)
		case []any:
			copied := make([]any, len(typed))
			copy(copied, typed)
			out[k] = copied
		default:
			out[k] = v
		}
	}
	return out
}

func workspacesWebCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
