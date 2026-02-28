package ec2

import (
	"sort"
	"strings"
	"time"
)

type AllowedImageCriterion struct {
	ImageProviders []string
}

type AllowedImagesSettings struct {
	ImageCriteria []AllowedImageCriterion
	ManagedBy     string
	State         string
}

type DeclarativePoliciesRegionalSummary struct {
	NumberOfMatchedAccounts   int32
	NumberOfUnmatchedAccounts int32
	RegionName                string
}

type DeclarativePoliciesAttributeSummary struct {
	AttributeName             string
	MostFrequentValue         string
	NumberOfMatchedAccounts   int32
	NumberOfUnmatchedAccounts int32
	RegionalSummaries         []DeclarativePoliciesRegionalSummary
}

type DeclarativePoliciesReport struct {
	AttributeSummaries     []DeclarativePoliciesAttributeSummary
	EndTime                *time.Time
	NumberOfAccounts       int32
	NumberOfFailedAccounts int32
	ReportID               string
	S3Bucket               string
	S3Prefix               string
	StartTime              time.Time
	Status                 string
	Tags                   map[string]string
	TargetID               string
}

type DeclarativePoliciesReportSummary struct {
	AttributeSummaries     []DeclarativePoliciesAttributeSummary
	EndTime                *time.Time
	NumberOfAccounts       int32
	NumberOfFailedAccounts int32
	ReportID               string
	S3Bucket               string
	S3Prefix               string
	StartTime              time.Time
	TargetID               string
}

func (s *Service) EnableAllowedImagesSettings(allowedImagesSettingsState string) (string, error) {
	allowedImagesSettingsState = strings.ToLower(strings.TrimSpace(allowedImagesSettingsState))
	switch allowedImagesSettingsState {
	case "enabled", "audit-mode":
	default:
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.allowedImagesSettings.ManagedBy = "account"
	s.allowedImagesSettings.State = allowedImagesSettingsState
	return s.allowedImagesSettings.State, nil
}

func (s *Service) DisableAllowedImagesSettings() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.allowedImagesSettings.ManagedBy = "account"
	s.allowedImagesSettings.State = "disabled"
	return s.allowedImagesSettings.State
}

func (s *Service) GetAllowedImagesSettings() AllowedImagesSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneAllowedImagesSettings(s.allowedImagesSettings)
}

func (s *Service) ReplaceImageCriteriaInAllowedImagesSettings(imageCriteria []AllowedImageCriterion) (bool, error) {
	if len(imageCriteria) > 10 {
		return false, ErrInvalidParameter
	}

	normalized := make([]AllowedImageCriterion, 0, len(imageCriteria))
	providerCount := 0
	for _, criterion := range imageCriteria {
		providers := dedupeTrimmedStrings(criterion.ImageProviders)
		if len(providers) == 0 {
			continue
		}
		providerCount += len(providers)
		normalized = append(normalized, AllowedImageCriterion{ImageProviders: append([]string(nil), providers...)})
	}
	if providerCount > 200 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.allowedImagesSettings.ImageCriteria = normalized
	return true, nil
}

func (s *Service) StartDeclarativePoliciesReport(targetID, s3Bucket string, s3Prefix *string, tags []Tag) (string, error) {
	targetID = strings.TrimSpace(targetID)
	s3Bucket = strings.TrimSpace(s3Bucket)
	if targetID == "" || s3Bucket == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, report := range s.declarativePoliciesReports {
		if report == nil {
			continue
		}
		if strings.EqualFold(report.Status, "running") {
			return "", ErrConflict
		}
	}

	prefix := ""
	if s3Prefix != nil {
		prefix = strings.TrimSpace(*s3Prefix)
	}

	reportID := s.nextIDLocked("dpr")
	startTime := time.Now().UTC()
	report := &DeclarativePoliciesReport{
		AttributeSummaries:     s.declarativePoliciesAttributeSummariesLocked(),
		EndTime:                nil,
		NumberOfAccounts:       1,
		NumberOfFailedAccounts: 0,
		ReportID:               reportID,
		S3Bucket:               s3Bucket,
		S3Prefix:               prefix,
		StartTime:              startTime,
		Status:                 "running",
		Tags:                   tagsToMap(tags),
		TargetID:               targetID,
	}
	s.declarativePoliciesReports[reportID] = report
	return reportID, nil
}

func (s *Service) CancelDeclarativePoliciesReport(reportID string) (bool, error) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	report := s.declarativePoliciesReports[reportID]
	if report == nil {
		return false, ErrNotFound
	}
	if !strings.EqualFold(report.Status, "running") {
		return false, ErrConflict
	}
	endTime := time.Now().UTC()
	report.Status = "cancelled"
	report.EndTime = &endTime
	return true, nil
}

func (s *Service) DescribeDeclarativePoliciesReports(
	reportIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]DeclarativePoliciesReport, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(dedupeTrimmedStrings(reportIDs))

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]DeclarativePoliciesReport, 0, len(s.declarativePoliciesReports))
	for _, report := range s.declarativePoliciesReports {
		if report == nil {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[report.ReportID]; !ok {
				continue
			}
		}
		items = append(items, cloneDeclarativePoliciesReport(report))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ReportID < items[j].ReportID })
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]DeclarativePoliciesReport(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetDeclarativePoliciesReportSummary(reportID string) (DeclarativePoliciesReportSummary, error) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return DeclarativePoliciesReportSummary{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	report := s.declarativePoliciesReports[reportID]
	if report == nil {
		return DeclarativePoliciesReportSummary{}, ErrNotFound
	}

	return DeclarativePoliciesReportSummary{
		AttributeSummaries:     cloneDeclarativePoliciesAttributeSummaries(report.AttributeSummaries),
		EndTime:                cloneTimePointer(report.EndTime),
		NumberOfAccounts:       report.NumberOfAccounts,
		NumberOfFailedAccounts: report.NumberOfFailedAccounts,
		ReportID:               report.ReportID,
		S3Bucket:               report.S3Bucket,
		S3Prefix:               report.S3Prefix,
		StartTime:              report.StartTime,
		TargetID:               report.TargetID,
	}, nil
}

func (s *Service) declarativePoliciesAttributeSummariesLocked() []DeclarativePoliciesAttributeSummary {
	state := strings.TrimSpace(s.allowedImagesSettings.State)
	if state == "" {
		state = "disabled"
	}

	return []DeclarativePoliciesAttributeSummary{
		{
			AttributeName:             "allowedImagesSettingsState",
			MostFrequentValue:         state,
			NumberOfMatchedAccounts:   1,
			NumberOfUnmatchedAccounts: 0,
			RegionalSummaries: []DeclarativePoliciesRegionalSummary{
				{
					NumberOfMatchedAccounts:   1,
					NumberOfUnmatchedAccounts: 0,
					RegionName:                DefaultRegion,
				},
			},
		},
	}
}

func cloneAllowedImagesSettings(in AllowedImagesSettings) AllowedImagesSettings {
	return AllowedImagesSettings{
		ImageCriteria: cloneAllowedImageCriteria(in.ImageCriteria),
		ManagedBy:     in.ManagedBy,
		State:         in.State,
	}
}

func cloneAllowedImageCriteria(in []AllowedImageCriterion) []AllowedImageCriterion {
	out := make([]AllowedImageCriterion, 0, len(in))
	for _, criterion := range in {
		out = append(out, AllowedImageCriterion{
			ImageProviders: append([]string(nil), criterion.ImageProviders...),
		})
	}
	return out
}

func cloneDeclarativePoliciesReport(in *DeclarativePoliciesReport) DeclarativePoliciesReport {
	if in == nil {
		return DeclarativePoliciesReport{}
	}
	return DeclarativePoliciesReport{
		AttributeSummaries:     cloneDeclarativePoliciesAttributeSummaries(in.AttributeSummaries),
		EndTime:                cloneTimePointer(in.EndTime),
		NumberOfAccounts:       in.NumberOfAccounts,
		NumberOfFailedAccounts: in.NumberOfFailedAccounts,
		ReportID:               in.ReportID,
		S3Bucket:               in.S3Bucket,
		S3Prefix:               in.S3Prefix,
		StartTime:              in.StartTime,
		Status:                 in.Status,
		Tags:                   cloneStringMap(in.Tags),
		TargetID:               in.TargetID,
	}
}

func cloneDeclarativePoliciesAttributeSummaries(in []DeclarativePoliciesAttributeSummary) []DeclarativePoliciesAttributeSummary {
	out := make([]DeclarativePoliciesAttributeSummary, 0, len(in))
	for _, summary := range in {
		out = append(out, DeclarativePoliciesAttributeSummary{
			AttributeName:             summary.AttributeName,
			MostFrequentValue:         summary.MostFrequentValue,
			NumberOfMatchedAccounts:   summary.NumberOfMatchedAccounts,
			NumberOfUnmatchedAccounts: summary.NumberOfUnmatchedAccounts,
			RegionalSummaries:         cloneDeclarativePoliciesRegionalSummaries(summary.RegionalSummaries),
		})
	}
	return out
}

func cloneDeclarativePoliciesRegionalSummaries(in []DeclarativePoliciesRegionalSummary) []DeclarativePoliciesRegionalSummary {
	out := make([]DeclarativePoliciesRegionalSummary, 0, len(in))
	for _, summary := range in {
		out = append(out, DeclarativePoliciesRegionalSummary{
			NumberOfMatchedAccounts:   summary.NumberOfMatchedAccounts,
			NumberOfUnmatchedAccounts: summary.NumberOfUnmatchedAccounts,
			RegionName:                summary.RegionName,
		})
	}
	return out
}

func cloneTimePointer(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}
