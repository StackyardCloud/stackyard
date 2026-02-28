package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type artifactStore struct {
	mu sync.Mutex

	nextID int64

	accountSettings map[string]any
	agreements      map[string]*artifactAgreement
	customerAgmts   map[string]*artifactCustomerAgreement
	reports         map[string]*artifactReport
}

type artifactAgreement struct {
	ID          string
	Name        string
	Description string
	Status      string
	Accepted    bool
	NdaAccepted bool
	AcceptedAt  string
}

type artifactCustomerAgreement struct {
	ID           string
	AgreementID  string
	Name         string
	Status       string
	CreatedAt    string
	TerminatedAt string
}

type artifactReport struct {
	ID            string
	Name          string
	Description   string
	Category      string
	LatestVersion string
	Versions      []string
}

func newArtifactStore() *artifactStore {
	now := time.Now().UTC().Format(time.RFC3339)
	agreement := &artifactAgreement{
		ID:          "agr-000001",
		Name:        "AWS Artifact Terms and Conditions",
		Description: "Seeded agreement for local Artifact emulation",
		Status:      "ACTIVE",
	}
	customer := &artifactCustomerAgreement{
		ID:          "cagr-000001",
		AgreementID: agreement.ID,
		Name:        "Seeded customer agreement",
		Status:      "ACTIVE",
		CreatedAt:   now,
	}
	report := &artifactReport{
		ID:            "rpt-000001",
		Name:          "SOC Report",
		Description:   "Seeded compliance report",
		Category:      "compliance",
		LatestVersion: "2",
		Versions:      []string{"1", "2"},
	}

	return &artifactStore{
		nextID: 2,
		accountSettings: map[string]any{
			"notificationsEnabled": true,
			"defaultReportFormat":  "PDF",
			"lastUpdatedTime":      now,
		},
		agreements: map[string]*artifactAgreement{
			agreement.ID: agreement,
		},
		customerAgmts: map[string]*artifactCustomerAgreement{
			customer.ID: customer,
		},
		reports: map[string]*artifactReport{
			report.ID: report,
		},
	}
}

func (s *artifactStore) Handle(
	action string,
	payload map[string]any,
	_ map[string]string,
	query map[string][]string,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "ListAgreements":
		items := make([]any, 0, len(s.agreements))
		for _, agr := range s.sortedAgreementsLocked() {
			items = append(items, map[string]any{
				"agreementId":    agr.ID,
				"name":           agr.Name,
				"description":    agr.Description,
				"status":         agr.Status,
				"accepted":       agr.Accepted,
				"acceptanceTime": agr.AcceptedAt,
			})
		}
		return map[string]any{"agreements": items, "nextToken": ""}

	case "GetAgreement":
		agreementID := artifactResolveString(payload, query, "agreementId", s.firstAgreementIDLocked())
		agr := s.ensureAgreementLocked(agreementID)
		return map[string]any{
			"agreementId":    agr.ID,
			"name":           agr.Name,
			"description":    agr.Description,
			"status":         agr.Status,
			"accepted":       agr.Accepted,
			"ndaAccepted":    agr.NdaAccepted,
			"acceptanceTime": agr.AcceptedAt,
		}

	case "AcceptAgreement":
		agreementID := artifactResolveString(payload, query, "agreementId", s.firstAgreementIDLocked())
		agr := s.ensureAgreementLocked(agreementID)
		agr.Accepted = true
		agr.AcceptedAt = now
		return map[string]any{"agreementId": agr.ID, "status": "ACCEPTED"}

	case "AcceptNdaForAgreement":
		agreementID := artifactResolveString(payload, query, "agreementId", s.firstAgreementIDLocked())
		agr := s.ensureAgreementLocked(agreementID)
		agr.NdaAccepted = true
		if agr.AcceptedAt == "" {
			agr.AcceptedAt = now
		}
		return map[string]any{"agreementId": agr.ID, "ndaAccepted": true}

	case "GetNdaForAgreement":
		agreementID := artifactResolveString(payload, query, "agreementId", s.firstAgreementIDLocked())
		agr := s.ensureAgreementLocked(agreementID)
		return map[string]any{
			"agreementId": agreementID,
			"version":     "2024-01-01",
			"content":     fmt.Sprintf("NDA terms for %s", agr.Name),
		}

	case "ListCustomerAgreements":
		items := make([]any, 0, len(s.customerAgmts))
		for _, ca := range s.sortedCustomerAgreementsLocked() {
			items = append(items, map[string]any{
				"customerAgreementId": ca.ID,
				"agreementId":         ca.AgreementID,
				"name":                ca.Name,
				"status":              ca.Status,
				"createdAt":           ca.CreatedAt,
				"terminatedAt":        ca.TerminatedAt,
			})
		}
		return map[string]any{"customerAgreements": items, "nextToken": ""}

	case "GetCustomerAgreement":
		id := artifactResolveString(payload, query, "customerAgreementId", s.firstCustomerAgreementIDLocked())
		ca := s.ensureCustomerAgreementLocked(id)
		return map[string]any{
			"customerAgreementId": ca.ID,
			"agreementId":         ca.AgreementID,
			"name":                ca.Name,
			"status":              ca.Status,
			"createdAt":           ca.CreatedAt,
			"terminatedAt":        ca.TerminatedAt,
		}

	case "TerminateAgreement":
		id := artifactResolveString(payload, query, "customerAgreementId", s.firstCustomerAgreementIDLocked())
		ca := s.ensureCustomerAgreementLocked(id)
		ca.Status = "TERMINATED"
		ca.TerminatedAt = now
		return map[string]any{
			"terminatedCustomerAgreement": map[string]any{
				"customerAgreementId": ca.ID,
				"agreementId":         ca.AgreementID,
				"status":              ca.Status,
				"terminatedAt":        ca.TerminatedAt,
			},
		}

	case "ListReports":
		items := make([]any, 0, len(s.reports))
		for _, r := range s.sortedReportsLocked() {
			items = append(items, map[string]any{
				"reportId":          r.ID,
				"name":              r.Name,
				"description":       r.Description,
				"category":          r.Category,
				"latestVersion":     r.LatestVersion,
				"availableVersions": r.Versions,
			})
		}
		return map[string]any{"reports": items, "nextToken": ""}

	case "GetReport":
		reportID := artifactResolveString(payload, query, "reportId", s.firstReportIDLocked())
		report := s.ensureReportLocked(reportID)
		return map[string]any{
			"reportId":    report.ID,
			"name":        report.Name,
			"version":     report.LatestVersion,
			"downloadUrl": fmt.Sprintf("https://artifact.stackyard.local/report/%s/%s", report.ID, report.LatestVersion),
			"expiresAt":   time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		}

	case "GetReportMetadata":
		reportID := artifactResolveString(payload, query, "reportId", s.firstReportIDLocked())
		report := s.ensureReportLocked(reportID)
		return map[string]any{
			"reportId":      report.ID,
			"name":          report.Name,
			"description":   report.Description,
			"category":      report.Category,
			"latestVersion": report.LatestVersion,
		}

	case "GetTermForReport":
		reportID := artifactResolveString(payload, query, "reportId", s.firstReportIDLocked())
		report := s.ensureReportLocked(reportID)
		return map[string]any{
			"reportId": report.ID,
			"version":  report.LatestVersion,
			"termText": fmt.Sprintf("Terms for report %s version %s", report.Name, report.LatestVersion),
		}

	case "ListReportVersions":
		reportID := artifactResolveString(payload, query, "reportId", s.firstReportIDLocked())
		report := s.ensureReportLocked(reportID)
		versions := make([]any, 0, len(report.Versions))
		for _, version := range report.Versions {
			versions = append(versions, map[string]any{
				"reportId": report.ID,
				"version":  version,
				"status":   "AVAILABLE",
			})
		}
		return map[string]any{
			"reportId":       report.ID,
			"reportVersions": versions,
			"nextToken":      "",
		}

	case "GetAccountSettings":
		return map[string]any{"accountSettings": copyMapAny(s.accountSettings)}

	case "PutAccountSettings":
		for key, value := range payload {
			s.accountSettings[key] = value
		}
		s.accountSettings["lastUpdatedTime"] = now
		return map[string]any{"accountSettings": copyMapAny(s.accountSettings)}
	}

	return map[string]any{}
}

func copyMapAny(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *artifactStore) ensureAgreementLocked(id string) *artifactAgreement {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstAgreementIDLocked()
	}
	if agr, ok := s.agreements[id]; ok {
		return agr
	}
	agr := &artifactAgreement{
		ID:          id,
		Name:        "Generated Agreement " + id,
		Description: "Generated agreement",
		Status:      "ACTIVE",
	}
	s.agreements[id] = agr
	return agr
}

func (s *artifactStore) ensureCustomerAgreementLocked(id string) *artifactCustomerAgreement {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstCustomerAgreementIDLocked()
	}
	if ca, ok := s.customerAgmts[id]; ok {
		return ca
	}
	agreementID := s.firstAgreementIDLocked()
	ca := &artifactCustomerAgreement{
		ID:          id,
		AgreementID: agreementID,
		Name:        "Generated Customer Agreement " + id,
		Status:      "ACTIVE",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.customerAgmts[id] = ca
	return ca
}

func (s *artifactStore) ensureReportLocked(id string) *artifactReport {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstReportIDLocked()
	}
	if report, ok := s.reports[id]; ok {
		return report
	}
	report := &artifactReport{
		ID:            id,
		Name:          "Generated Report " + id,
		Description:   "Generated report",
		Category:      "compliance",
		LatestVersion: "1",
		Versions:      []string{"1"},
	}
	s.reports[id] = report
	return report
}

func (s *artifactStore) firstAgreementIDLocked() string {
	keys := make([]string, 0, len(s.agreements))
	for k := range s.agreements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		id := fmt.Sprintf("agr-%06d", s.nextID)
		s.nextID++
		return id
	}
	return keys[0]
}

func (s *artifactStore) firstCustomerAgreementIDLocked() string {
	keys := make([]string, 0, len(s.customerAgmts))
	for k := range s.customerAgmts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		id := fmt.Sprintf("cagr-%06d", s.nextID)
		s.nextID++
		return id
	}
	return keys[0]
}

func (s *artifactStore) firstReportIDLocked() string {
	keys := make([]string, 0, len(s.reports))
	for k := range s.reports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		id := fmt.Sprintf("rpt-%06d", s.nextID)
		s.nextID++
		return id
	}
	return keys[0]
}

func (s *artifactStore) sortedAgreementsLocked() []*artifactAgreement {
	keys := make([]string, 0, len(s.agreements))
	for k := range s.agreements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*artifactAgreement, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.agreements[k])
	}
	return out
}

func (s *artifactStore) sortedCustomerAgreementsLocked() []*artifactCustomerAgreement {
	keys := make([]string, 0, len(s.customerAgmts))
	for k := range s.customerAgmts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*artifactCustomerAgreement, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.customerAgmts[k])
	}
	return out
}

func (s *artifactStore) sortedReportsLocked() []*artifactReport {
	keys := make([]string, 0, len(s.reports))
	for k := range s.reports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*artifactReport, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.reports[k])
	}
	return out
}

func artifactResolveString(payload map[string]any, query map[string][]string, key, fallback string) string {
	if payload != nil {
		if raw, ok := payload[key]; ok && raw != nil {
			if value := strings.TrimSpace(fmt.Sprintf("%v", raw)); value != "" {
				return value
			}
		}
		alt := strings.ToUpper(key[:1]) + key[1:]
		if raw, ok := payload[alt]; ok && raw != nil {
			if value := strings.TrimSpace(fmt.Sprintf("%v", raw)); value != "" {
				return value
			}
		}
	}
	if query != nil {
		if values := query[key]; len(values) > 0 {
			if value := strings.TrimSpace(values[0]); value != "" {
				return value
			}
		}
		alt := strings.ToUpper(key[:1]) + key[1:]
		if values := query[alt]; len(values) > 0 {
			if value := strings.TrimSpace(values[0]); value != "" {
				return value
			}
		}
	}
	return fallback
}
