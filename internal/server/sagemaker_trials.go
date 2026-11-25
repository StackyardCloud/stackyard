package server

import (
	"math"
	"strings"
	"time"
)

// SageMaker trial components and the metrics SageMaker Metrics
// (sagemakermetrics.go) writes to them. A training job's trial component,
// "<job>-aws-training-job", comes into being the first time it is used, as
// SageMaker creates one for every training job.

const (
	smmMaxComponents         = 1000
	smmMaxMetricsPerResource = 100
	smmMaxPointsPerMetric    = 20000
	smmMaxPoints             = 500000
	smmTrainingJobSuffix     = "-aws-training-job"
)

type smmParam struct {
	StringValue *string  `json:"StringValue,omitempty"`
	NumberValue *float64 `json:"NumberValue,omitempty"`
}

type smmArtifact struct {
	MediaType string `json:"MediaType,omitempty"`
	Value     string `json:"Value"`
}

type smmMetadata struct {
	CommitId    string `json:"CommitId,omitempty"`
	Repository  string `json:"Repository,omitempty"`
	GeneratedBy string `json:"GeneratedBy,omitempty"`
	ProjectId   string `json:"ProjectId,omitempty"`
}

type smmStatus struct {
	PrimaryStatus string `json:"PrimaryStatus,omitempty"`
	Message       string `json:"Message,omitempty"`
}

// smmPoint is one metric value. Stored points are never modified: series
// only append, so a reader may keep points[:n] after the lock is released.
type smmPoint struct {
	at      float64 // epoch seconds
	step    int64
	hasStep bool
	value   float64
}

type smmSeries struct {
	points               []smmPoint
	count                int
	sum, sumSq, min, max float64
	last, lastAt         float64
}

func (s *smmSeries) add(p smmPoint) {
	if s.count == 0 || p.value < s.min {
		s.min = p.value
	}
	if s.count == 0 || p.value > s.max {
		s.max = p.value
	}
	if s.count == 0 || p.at >= s.lastAt {
		s.last, s.lastAt = p.value, p.at
	}
	s.count++
	s.sum += p.value
	s.sumSq += p.value * p.value
	s.points = append(s.points, p)
}

type smmComponent struct {
	Name, ARN, DisplayName string
	Source                 string // ARN of the job that made the component, if any
	Status                 *smmStatus
	Start, End             *float64
	Created, Modified      time.Time
	Parameters             map[string]smmParam
	Inputs, Outputs        map[string]smmArtifact
	Metadata               *smmMetadata
	metrics                map[string]*smmSeries
	metricOrder            []string
}

type smmCreateReq struct {
	TrialComponentName string
	DisplayName        string
	Status             *smmStatus
	StartTime, EndTime *sagemakerTimestamp
	Parameters         map[string]smmParam
	InputArtifacts     map[string]smmArtifact
	OutputArtifacts    map[string]smmArtifact
	MetadataProperties *smmMetadata
	Tags               []sagemakerTag
}

func smmEpochPtr(t *sagemakerTimestamp) *float64 {
	if t == nil {
		return nil
	}
	v := sagemakerEpoch(t.Time)
	return &v
}

func smmCreateTrialComponent(st *sagemakerStore, req smmCreateReq, now time.Time) (any, *sagemakerResourceError) {
	if st.trialComponents[req.TrialComponentName] != nil {
		return nil, sagemakerResErr(sagemakerErrInUse, "Trial component %s already exists.", req.TrialComponentName)
	}
	if len(st.trialComponents) >= smmMaxComponents {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d trial components.", smmMaxComponents)
	}
	for _, p := range req.Parameters {
		if (p.StringValue == nil) == (p.NumberValue == nil) {
			return nil, sagemakerResErr(sagemakerErrValidation, "Each parameter needs exactly one of StringValue and NumberValue.")
		}
	}
	c := &smmComponent{Name: req.TrialComponentName, ARN: sagemakerResourceARN("experiment-trial-component", req.TrialComponentName),
		DisplayName: req.DisplayName, Status: req.Status, Start: smmEpochPtr(req.StartTime), End: smmEpochPtr(req.EndTime),
		Created: now, Modified: now, Parameters: req.Parameters, Inputs: req.InputArtifacts, Outputs: req.OutputArtifacts,
		Metadata: req.MetadataProperties, metrics: map[string]*smmSeries{}}
	st.trialComponents[c.Name] = c
	st.setTagsLocked(c.ARN, req.Tags)
	return map[string]any{"TrialComponentArn": c.ARN}, nil
}

// smmComponentLocked resolves a trial component name or ARN, or a training
// job ARN, creating a training job's trial component on first use.
func (s *sagemakerStore) smmComponentLocked(ref string, now time.Time) (*smmComponent, *sagemakerResourceError) {
	name := ref
	if strings.Contains(ref, ":training-job/") {
		name = sagemakerNameFromARN(ref, "training-job") + smmTrainingJobSuffix
	} else {
		name = sagemakerNameFromARN(ref, "experiment-trial-component")
	}
	if c := s.trialComponents[name]; c != nil {
		return c, nil
	}
	job := s.trainingJobs[strings.TrimSuffix(name, smmTrainingJobSuffix)]
	if !strings.HasSuffix(name, smmTrainingJobSuffix) || job == nil {
		return nil, sagemakerResErr(sagemakerErrNotFound, "Trial component %s does not exist.", name)
	}
	if len(s.trialComponents) >= smmMaxComponents {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d trial components.", smmMaxComponents)
	}
	status := "InProgress"
	if job.Status == "Completed" || job.Status == "Failed" || job.Status == "Stopped" {
		status = job.Status
	}
	c := &smmComponent{Name: name, ARN: sagemakerResourceARN("experiment-trial-component", name), DisplayName: name,
		Source: job.ARN, Status: &smmStatus{PrimaryStatus: status}, Created: now, Modified: now, metrics: map[string]*smmSeries{}}
	s.trialComponents[name] = c
	return c, nil
}

type smmNameReq struct {
	TrialComponentName string
}

func (c *smmComponent) source() map[string]any {
	if c.Source == "" {
		return nil
	}
	return map[string]any{"SourceArn": c.Source, "SourceType": "SageMakerTrainingJob"}
}

func (c *smmComponent) summary() map[string]any {
	out := map[string]any{"TrialComponentName": c.Name, "TrialComponentArn": c.ARN,
		"CreationTime": sagemakerEpoch(c.Created), "LastModifiedTime": sagemakerEpoch(c.Modified)}
	if c.DisplayName != "" {
		out["DisplayName"] = c.DisplayName
	}
	if c.Status != nil {
		out["Status"] = c.Status
	}
	if c.Start != nil {
		out["StartTime"] = *c.Start
	}
	if c.End != nil {
		out["EndTime"] = *c.End
	}
	return out
}

func smmDescribeTrialComponent(st *sagemakerStore, req smmNameReq, now time.Time) (any, *sagemakerResourceError) {
	c, err := st.smmComponentLocked(req.TrialComponentName, now)
	if err != nil {
		return nil, err
	}
	out := c.summary()
	if src := c.source(); src != nil {
		out["Source"] = src
		out["Sources"] = []any{src}
	}
	if len(c.Parameters) > 0 {
		out["Parameters"] = c.Parameters
	}
	if len(c.Inputs) > 0 {
		out["InputArtifacts"] = c.Inputs
	}
	if len(c.Outputs) > 0 {
		out["OutputArtifacts"] = c.Outputs
	}
	if c.Metadata != nil {
		out["MetadataProperties"] = c.Metadata
	}
	if len(c.metricOrder) > 0 {
		metrics := make([]any, 0, len(c.metricOrder))
		for _, name := range c.metricOrder {
			s := c.metrics[name]
			avg := s.sum / float64(s.count)
			item := map[string]any{"MetricName": name, "TimeStamp": s.lastAt, "Max": s.max, "Min": s.min,
				"Last": s.last, "Count": s.count, "Avg": avg,
				"StdDev": math.Sqrt(math.Max(0, s.sumSq/float64(s.count)-avg*avg))}
			if c.Source != "" {
				item["SourceArn"] = c.Source
			}
			metrics = append(metrics, item)
		}
		out["Metrics"] = metrics
	}
	return out, nil
}

func smmDeleteTrialComponent(st *sagemakerStore, req smmNameReq, _ time.Time) (any, *sagemakerResourceError) {
	c := st.trialComponents[req.TrialComponentName]
	if c == nil {
		return nil, sagemakerResErr(sagemakerErrNotFound, "Trial component %s does not exist.", req.TrialComponentName)
	}
	for _, s := range c.metrics {
		st.smmPoints -= len(s.points)
	}
	delete(st.trialComponents, c.Name)
	delete(st.tags, c.ARN)
	return map[string]any{"TrialComponentArn": c.ARN}, nil
}

type smmListReq struct {
	ExperimentName string
	TrialName      string
	SourceArn      string
	CreatedAfter   *sagemakerTimestamp
	CreatedBefore  *sagemakerTimestamp
	SortBy         string
	SortOrder      string
	MaxResults     int
	NextToken      string
}

func smmListTrialComponents(st *sagemakerStore, req smmListReq, _ time.Time) (any, *sagemakerResourceError) {
	// ponytail: experiments and trials are not stored, so no component is in one.
	if req.ExperimentName != "" {
		return nil, sagemakerResErr(sagemakerErrNotFound, "Experiment %s does not exist.", req.ExperimentName)
	}
	if req.TrialName != "" {
		return nil, sagemakerResErr(sagemakerErrNotFound, "Trial %s does not exist.", req.TrialName)
	}
	names := sagemakerSortNames(st.trialComponents, req.SortBy != "Name", req.SortOrder != "Ascending",
		func(c *smmComponent) time.Time { return c.Created })
	var keep []*smmComponent
	for _, name := range names {
		c := st.trialComponents[name]
		if (req.SourceArn != "" && c.Source != req.SourceArn) ||
			(req.CreatedAfter != nil && !c.Created.After(req.CreatedAfter.Time)) ||
			(req.CreatedBefore != nil && !c.Created.Before(req.CreatedBefore.Time)) {
			continue
		}
		keep = append(keep, c)
	}
	start, end, next, err := sagemakerPage(len(keep), req.NextToken, req.MaxResults)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, c := range keep[start:end] {
		item := c.summary()
		if src := c.source(); src != nil {
			item["TrialComponentSource"] = src
		}
		items = append(items, item)
	}
	out := map[string]any{"TrialComponentSummaries": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}
