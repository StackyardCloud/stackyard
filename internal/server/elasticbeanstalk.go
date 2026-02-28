package server

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	awsebtypes "github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
	ebservice "github.com/stackyard/stackyard/internal/services/elasticbeanstalk"
)

func (s *Server) handleElasticBeanstalkQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElasticBeanstalkQueryCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticbeanstalk")
	if !ok {
		respondElasticBeanstalkErrorXML(w, status, code, msg)
		return true
	}

	if err := r.ParseForm(); err != nil {
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid form")
		return true
	}

	action := strings.TrimSpace(r.Form.Get("Action"))
	if action == "" {
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "MissingAction", "missing Action")
		return true
	}
	if _, known := elasticBeanstalkOperationByName[action]; !known {
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidAction", "invalid Action")
		return true
	}

	version := strings.TrimSpace(r.Form.Get("Version"))
	if version != "" && version != "2010-12-01" {
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "invalid Version")
		return true
	}

	switch action {
	case "CreateApplication":
		app, err := s.eb.CreateApplication(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("Description")),
			parseElasticBeanstalkTags(r.Form, "Tags.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreateApplicationResult{Application: toAWSEBApplication(app)})
		return true
	case "UpdateApplication":
		app, err := s.eb.UpdateApplication(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("Description")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateApplicationResult{Application: toAWSEBApplication(app)})
		return true
	case "DeleteApplication":
		terminateEnvByForce, _ := parseOptionalBool(r.Form.Get("TerminateEnvByForce"), false)
		if err := s.eb.DeleteApplication(strings.TrimSpace(r.Form.Get("ApplicationName")), terminateEnvByForce); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDeleteApplicationResult{})
		return true
	case "DescribeApplications":
		apps := s.eb.DescribeApplications(parseElasticBeanstalkMembers(r.Form, "ApplicationNames.member"))
		out := make([]awsebtypes.ApplicationDescription, 0, len(apps))
		for _, app := range apps {
			out = append(out, toAWSEBApplication(app))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeApplicationsResult{Applications: out})
		return true
	case "CreateApplicationVersion":
		version, err := s.eb.CreateApplicationVersion(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("VersionLabel")),
			strings.TrimSpace(r.Form.Get("Description")),
			ebservice.S3Location{
				S3Bucket: strings.TrimSpace(r.Form.Get("SourceBundle.S3Bucket")),
				S3Key:    strings.TrimSpace(r.Form.Get("SourceBundle.S3Key")),
			},
			parseElasticBeanstalkTags(r.Form, "Tags.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreateApplicationVersionResult{ApplicationVersion: toAWSEBApplicationVersion(version)})
		return true
	case "UpdateApplicationVersion":
		version, err := s.eb.UpdateApplicationVersion(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("VersionLabel")),
			strings.TrimSpace(r.Form.Get("Description")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateApplicationVersionResult{ApplicationVersion: toAWSEBApplicationVersion(version)})
		return true
	case "DeleteApplicationVersion":
		if err := s.eb.DeleteApplicationVersion(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("VersionLabel")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDeleteApplicationVersionResult{})
		return true
	case "DescribeApplicationVersions":
		versions := s.eb.DescribeApplicationVersions(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			parseElasticBeanstalkMembers(r.Form, "VersionLabels.member"),
		)
		out := make([]awsebtypes.ApplicationVersionDescription, 0, len(versions))
		for _, version := range versions {
			out = append(out, toAWSEBApplicationVersion(version))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeApplicationVersionsResult{ApplicationVersions: out})
		return true
	case "CreateConfigurationTemplate":
		tpl, err := s.eb.CreateConfigurationTemplate(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("SolutionStackName")),
			strings.TrimSpace(r.Form.Get("SourceConfiguration.TemplateName")),
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreateConfigurationTemplateResult{Template: toAWSEBConfigurationSettingsFromTemplate(tpl)})
		return true
	case "UpdateConfigurationTemplate":
		tpl, err := s.eb.UpdateConfigurationTemplate(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("SolutionStackName")),
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateConfigurationTemplateResult{Template: toAWSEBConfigurationSettingsFromTemplate(tpl)})
		return true
	case "DeleteConfigurationTemplate":
		if err := s.eb.DeleteConfigurationTemplate(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDeleteConfigurationTemplateResult{})
		return true
	case "DescribeConfigurationSettings":
		settings := s.eb.DescribeConfigurationSettings(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		out := make([]awsebtypes.ConfigurationSettingsDescription, 0, len(settings))
		for _, setting := range settings {
			out = append(out, toAWSEBConfigurationSettings(setting))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeConfigurationSettingsResult{ConfigurationSettings: out})
		return true
	case "DescribeConfigurationOptions":
		solutionStackName, options, err := s.eb.DescribeConfigurationOptions(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("SolutionStackName")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			parseElasticBeanstalkOptionSpecifications(r.Form, "Options.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.ConfigurationOptionDescription, 0, len(options))
		for _, option := range options {
			out = append(out, toAWSEBConfigurationOption(option))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeConfigurationOptionsResult{SolutionStackName: solutionStackName, Options: out})
		return true
	case "ValidateConfigurationSettings":
		messages := s.eb.ValidateConfigurationSettings(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
		)
		out := make([]awsebtypes.ValidationMessage, 0, len(messages))
		for _, msg := range messages {
			out = append(out, toAWSEBValidationMessage(msg))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkValidateConfigurationSettingsResult{Messages: out})
		return true
	case "CreateEnvironment":
		env, err := s.eb.CreateEnvironment(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("CNAMEPrefix")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("SolutionStackName")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("VersionLabel")),
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
			parseElasticBeanstalkTags(r.Form, "Tags.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreateEnvironmentResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "UpdateEnvironment":
		env, err := s.eb.UpdateEnvironment(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("VersionLabel")),
			strings.TrimSpace(r.Form.Get("TemplateName")),
			strings.TrimSpace(r.Form.Get("Description")),
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateEnvironmentResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "AbortEnvironmentUpdate":
		env, err := s.eb.AbortEnvironmentUpdate(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkAbortEnvironmentUpdateResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "RebuildEnvironment":
		env, err := s.eb.RebuildEnvironment(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkRebuildEnvironmentResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "RestartAppServer":
		env, err := s.eb.RestartAppServer(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkRestartAppServerResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "TerminateEnvironment":
		env, err := s.eb.TerminateEnvironment(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkTerminateEnvironmentResult{Environment: toAWSEBEnvironment(env)})
		return true
	case "SwapEnvironmentCNAMEs":
		err := s.eb.SwapEnvironmentCNAMEs(
			strings.TrimSpace(r.Form.Get("SourceEnvironmentId")),
			strings.TrimSpace(r.Form.Get("SourceEnvironmentName")),
			strings.TrimSpace(r.Form.Get("DestinationEnvironmentId")),
			strings.TrimSpace(r.Form.Get("DestinationEnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkSwapEnvironmentCNAMEsResult{})
		return true
	case "DescribeEnvironments":
		includeDeleted, _ := parseOptionalBool(r.Form.Get("IncludeDeleted"), false)
		envs := s.eb.DescribeEnvironments(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			parseElasticBeanstalkMembers(r.Form, "EnvironmentIds.member"),
			parseElasticBeanstalkMembers(r.Form, "EnvironmentNames.member"),
			includeDeleted,
		)
		out := make([]awsebtypes.EnvironmentDescription, 0, len(envs))
		for _, env := range envs {
			out = append(out, toAWSEBEnvironment(env))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEnvironmentsResult{Environments: out})
		return true
	case "DescribeEnvironmentResources":
		resources, err := s.eb.DescribeEnvironmentResources(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEnvironmentResourcesResult{EnvironmentResources: toAWSEBEnvironmentResources(resources)})
		return true
	case "DescribeEvents":
		maxRecords := parseOptionalInt(r.Form.Get("MaxRecords"), 50)
		events := s.eb.DescribeEvents(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			maxRecords,
		)
		out := make([]awsebtypes.EventDescription, 0, len(events))
		for _, ev := range events {
			out = append(out, toAWSEBEvent(ev))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEventsResult{Events: out})
		return true
	case "RequestEnvironmentInfo":
		if err := s.eb.RequestEnvironmentInfo(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("InfoType")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkRequestEnvironmentInfoResult{})
		return true
	case "RetrieveEnvironmentInfo":
		items, err := s.eb.RetrieveEnvironmentInfo(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("InfoType")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.EnvironmentInfoDescription, 0, len(items))
		for _, item := range items {
			out = append(out, toAWSEBEnvironmentInfo(item))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkRetrieveEnvironmentInfoResult{EnvironmentInfo: out})
		return true
	case "CheckDNSAvailability":
		available, fqdn, err := s.eb.CheckDNSAvailability(strings.TrimSpace(r.Form.Get("CNAMEPrefix")))
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCheckDNSAvailabilityResult{Available: boolPtr(available), FullyQualifiedCNAME: strPtr(fqdn)})
		return true
	case "CreateStorageLocation":
		bucket := s.eb.CreateStorageLocation()
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreateStorageLocationResult{S3Bucket: strPtr(bucket)})
		return true
	case "ListAvailableSolutionStacks":
		stacks := s.eb.ListAvailableSolutionStacks()
		respondElasticBeanstalkXML(w, action, elasticBeanstalkListAvailableSolutionStacksResult{SolutionStacks: stacks})
		return true
	case "DescribeAccountAttributes":
		quotas := s.eb.DescribeAccountAttributes()
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeAccountAttributesResult{
			ResourceQuotas: toAWSEBResourceQuotas(quotas),
		})
		return true
	case "AssociateEnvironmentOperationsRole":
		if err := s.eb.AssociateEnvironmentOperationsRole(
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("OperationsRole")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkAssociateEnvironmentOperationsRoleResult{})
		return true
	case "DisassociateEnvironmentOperationsRole":
		if err := s.eb.DisassociateEnvironmentOperationsRole(
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDisassociateEnvironmentOperationsRoleResult{})
		return true
	case "ComposeEnvironments":
		envs, err := s.eb.ComposeEnvironments(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("GroupName")),
			parseElasticBeanstalkMembers(r.Form, "VersionLabels.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.EnvironmentDescription, 0, len(envs))
		for _, env := range envs {
			out = append(out, toAWSEBEnvironment(env))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkComposeEnvironmentsResult{Environments: out})
		return true
	case "CreatePlatformVersion":
		platform, err := s.eb.CreatePlatformVersion(
			strings.TrimSpace(r.Form.Get("PlatformName")),
			strings.TrimSpace(r.Form.Get("PlatformVersion")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			ebservice.S3Location{
				S3Bucket: strings.TrimSpace(r.Form.Get("PlatformDefinitionBundle.S3Bucket")),
				S3Key:    strings.TrimSpace(r.Form.Get("PlatformDefinitionBundle.S3Key")),
			},
			parseElasticBeanstalkOptionSettings(r.Form, "OptionSettings.member"),
			parseElasticBeanstalkTags(r.Form, "Tags.member"),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkCreatePlatformVersionResult{
			Builder:         toAWSEBBuilder(platform.BuilderARN),
			PlatformSummary: toAWSEBPlatformSummary(platform),
		})
		return true
	case "DeletePlatformVersion":
		platform, err := s.eb.DeletePlatformVersion(strings.TrimSpace(r.Form.Get("PlatformArn")))
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDeletePlatformVersionResult{
			PlatformSummary: toAWSEBPlatformSummary(platform),
		})
		return true
	case "DescribePlatformVersion":
		platform, err := s.eb.DescribePlatformVersion(strings.TrimSpace(r.Form.Get("PlatformArn")))
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribePlatformVersionResult{
			PlatformDescription: toAWSEBPlatformDescription(platform),
		})
		return true
	case "ListPlatformVersions":
		maxRecords := parseOptionalInt(r.Form.Get("MaxRecords"), 100)
		platforms, nextToken, err := s.eb.ListPlatformVersions(
			parseElasticBeanstalkPlatformFilters(r.Form, "Filters.member"),
			maxRecords,
			strings.TrimSpace(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.PlatformSummary, 0, len(platforms))
		for _, platform := range platforms {
			out = append(out, toAWSEBPlatformSummary(platform))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkListPlatformVersionsResult{
			PlatformSummaryList: out,
			NextToken:           strPtr(nextToken),
		})
		return true
	case "ListPlatformBranches":
		maxRecords := parseOptionalInt(r.Form.Get("MaxRecords"), 100)
		branches, nextToken, err := s.eb.ListPlatformBranches(
			parseElasticBeanstalkSearchFilters(r.Form, "Filters.member"),
			maxRecords,
			strings.TrimSpace(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.PlatformBranchSummary, 0, len(branches))
		for _, branch := range branches {
			out = append(out, toAWSEBPlatformBranchSummary(branch))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkListPlatformBranchesResult{
			PlatformBranchSummaryList: out,
			NextToken:                 strPtr(nextToken),
		})
		return true
	case "DescribeEnvironmentHealth":
		health, err := s.eb.DescribeEnvironmentHealth(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEnvironmentHealthResult{
			EnvironmentName:    strPtr(health.EnvironmentName),
			Color:              strPtr(health.Color),
			HealthStatus:       strPtr(health.HealthStatus),
			Status:             awsebtypes.EnvironmentHealth(health.Status),
			Causes:             append([]string(nil), health.Causes...),
			ApplicationMetrics: toAWSEBApplicationMetrics(health.ApplicationMetrics),
			InstancesHealth:    toAWSEBInstanceHealthSummary(health.InstancesHealth),
			RefreshedAt:        timePtr(health.RefreshedAt),
		})
		return true
	case "DescribeInstancesHealth":
		items, nextToken, refreshedAt, err := s.eb.DescribeInstancesHealth(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.SingleInstanceHealth, 0, len(items))
		for _, item := range items {
			out = append(out, toAWSEBSingleInstanceHealth(item))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeInstancesHealthResult{
			InstanceHealthList: out,
			NextToken:          strPtr(nextToken),
			RefreshedAt:        timePtr(refreshedAt),
		})
		return true
	case "DeleteEnvironmentConfiguration":
		if err := s.eb.DeleteEnvironmentConfiguration(
			strings.TrimSpace(r.Form.Get("ApplicationName")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDeleteEnvironmentConfigurationResult{})
		return true
	case "ApplyEnvironmentManagedAction":
		out, err := s.eb.ApplyEnvironmentManagedAction(
			strings.TrimSpace(r.Form.Get("ActionId")),
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkApplyEnvironmentManagedActionResult{
			ActionDescription: strPtr(out.ActionDescription),
			ActionId:          strPtr(out.ActionID),
			ActionType:        awsebtypes.ActionType(out.ActionType),
			Status:            strPtr(out.Status),
		})
		return true
	case "DescribeEnvironmentManagedActions":
		items, err := s.eb.DescribeEnvironmentManagedActions(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			strings.TrimSpace(r.Form.Get("Status")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.ManagedAction, 0, len(items))
		for _, item := range items {
			out = append(out, toAWSEBManagedAction(item))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEnvironmentManagedActionsResult{
			ManagedActions: out,
		})
		return true
	case "DescribeEnvironmentManagedActionHistory":
		maxItems := parseOptionalInt(r.Form.Get("MaxItems"), 100)
		items, nextToken, err := s.eb.DescribeEnvironmentManagedActionHistory(
			strings.TrimSpace(r.Form.Get("EnvironmentId")),
			strings.TrimSpace(r.Form.Get("EnvironmentName")),
			maxItems,
			strings.TrimSpace(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		out := make([]awsebtypes.ManagedActionHistoryItem, 0, len(items))
		for _, item := range items {
			out = append(out, toAWSEBManagedActionHistoryItem(item))
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkDescribeEnvironmentManagedActionHistoryResult{
			ManagedActionHistoryItems: out,
			NextToken:                 strPtr(nextToken),
		})
		return true
	case "UpdateApplicationResourceLifecycle":
		cfg, ok, err := parseElasticBeanstalkApplicationResourceLifecycleConfig(r.Form)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		if !ok {
			respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", "ResourceLifecycleConfig is required")
			return true
		}
		appName := strings.TrimSpace(r.Form.Get("ApplicationName"))
		updated, err := s.eb.UpdateApplicationResourceLifecycle(appName, cfg)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateApplicationResourceLifecycleResult{
			ApplicationName:         strPtr(appName),
			ResourceLifecycleConfig: toAWSEBApplicationResourceLifecycleConfig(updated),
		})
		return true
	case "ListTagsForResource":
		resourceARN := strings.TrimSpace(r.Form.Get("ResourceArn"))
		tags, err := s.eb.ListTagsForResource(resourceARN)
		if err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkListTagsForResourceResult{ResourceArn: strPtr(resourceARN), ResourceTags: toAWSEBTags(tags)})
		return true
	case "UpdateTagsForResource":
		if err := s.eb.UpdateTagsForResource(
			strings.TrimSpace(r.Form.Get("ResourceArn")),
			parseElasticBeanstalkTags(r.Form, "TagsToAdd.member"),
			parseElasticBeanstalkMembers(r.Form, "TagsToRemove.member"),
		); err != nil {
			respondElasticBeanstalkErrorForErr(w, err)
			return true
		}
		respondElasticBeanstalkXML(w, action, elasticBeanstalkUpdateTagsForResourceResult{})
		return true
	default:
		respondElasticBeanstalkErrorXML(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func respondElasticBeanstalkErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ebservice.ErrInvalidParameter):
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ebservice.ErrAlreadyExists):
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ebservice.ErrNotFound):
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	case errors.Is(err, ebservice.ErrConflict):
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	default:
		respondElasticBeanstalkErrorXML(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
	}
}

func isElasticBeanstalkQueryCandidate(r *http.Request) bool {
	action := strings.TrimSpace(r.URL.Query().Get("Action"))
	if action != "" {
		if !isElasticBeanstalkAction(action) {
			return false
		}
		if version := strings.TrimSpace(r.URL.Query().Get("Version")); version != "" && version != "2010-12-01" {
			return false
		}
		if service := sigV4ServiceHint(r); service != "" && service != "elasticbeanstalk" {
			return false
		}
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		return false
	}
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return false
	}
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return false
	}
	action = strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return false
	}
	if !isElasticBeanstalkAction(action) {
		return false
	}
	if version := strings.TrimSpace(values.Get("Version")); version != "" && version != "2010-12-01" {
		return false
	}
	if service := sigV4ServiceHint(r); service != "" && service != "elasticbeanstalk" {
		return false
	}
	return true
}

func isElasticBeanstalkAction(action string) bool {
	_, ok := elasticBeanstalkOperationByName[strings.TrimSpace(action)]
	return ok
}

func sigV4ServiceHint(r *http.Request) string {
	if req, err := parseSigV4Authorization(r.Header.Get("Authorization")); err == nil {
		return strings.TrimSpace(req.Service)
	}
	if req, err := parseSigV4Query(r.URL.Query()); err == nil {
		return strings.TrimSpace(req.Service)
	}
	return ""
}

func respondElasticBeanstalkXML(w http.ResponseWriter, action string, result any) {
	env := elasticBeanstalkResponseEnvelope{
		XMLName: xml.Name{Local: action + "Response"},
		Xmlns:   elasticBeanstalkNamespace,
		Result:  result,
		Metadata: elasticBeanstalkResponseMetadata{
			RequestID: "stackyard-request",
		},
	}
	respondXML(w, http.StatusOK, env)
}

func respondElasticBeanstalkErrorXML(w http.ResponseWriter, status int, code, message string) {
	respondXML(w, status, elasticBeanstalkErrorResponse{
		Xmlns: elasticBeanstalkNamespace,
		Error: elasticBeanstalkErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: "stackyard-request",
	})
}

func parseElasticBeanstalkMembers(values url.Values, prefix string) []string {
	type indexed struct {
		idx int
		val string
	}
	items := make([]indexed, 0)
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) == 0 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		val := strings.TrimSpace(vals[0])
		if val == "" {
			continue
		}
		items = append(items, indexed{idx: idx, val: val})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].idx < items[j].idx })
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.val)
	}
	return out
}

func parseElasticBeanstalkTags(values url.Values, prefix string) []ebservice.Tag {
	type tagItem struct {
		idx   int
		key   string
		value string
	}
	items := map[int]*tagItem{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item := items[idx]
		if item == nil {
			item = &tagItem{idx: idx}
			items[idx] = item
		}
		switch parts[1] {
		case "Key":
			item.key = strings.TrimSpace(vals[0])
		case "Value":
			item.value = vals[0]
		}
	}
	ordered := make([]*tagItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].idx < ordered[j].idx })
	out := make([]ebservice.Tag, 0, len(ordered))
	for _, item := range ordered {
		if item.key == "" {
			continue
		}
		out = append(out, ebservice.Tag{Key: item.key, Value: item.value})
	}
	return out
}

func parseElasticBeanstalkOptionSettings(values url.Values, prefix string) []ebservice.OptionSetting {
	type optionItem struct {
		idx          int
		namespace    string
		optionName   string
		resourceName string
		value        string
	}
	items := map[int]*optionItem{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item := items[idx]
		if item == nil {
			item = &optionItem{idx: idx}
			items[idx] = item
		}
		switch parts[1] {
		case "Namespace":
			item.namespace = strings.TrimSpace(vals[0])
		case "OptionName":
			item.optionName = strings.TrimSpace(vals[0])
		case "ResourceName":
			item.resourceName = strings.TrimSpace(vals[0])
		case "Value":
			item.value = vals[0]
		}
	}
	ordered := make([]*optionItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].idx < ordered[j].idx })
	out := make([]ebservice.OptionSetting, 0, len(ordered))
	for _, item := range ordered {
		if item.namespace == "" || item.optionName == "" {
			continue
		}
		out = append(out, ebservice.OptionSetting{
			Namespace:    item.namespace,
			OptionName:   item.optionName,
			ResourceName: item.resourceName,
			Value:        item.value,
		})
	}
	return out
}

func parseElasticBeanstalkOptionSpecifications(values url.Values, prefix string) []ebservice.OptionSpecification {
	type optionItem struct {
		idx          int
		namespace    string
		optionName   string
		resourceName string
	}
	items := map[int]*optionItem{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item := items[idx]
		if item == nil {
			item = &optionItem{idx: idx}
			items[idx] = item
		}
		switch parts[1] {
		case "Namespace":
			item.namespace = strings.TrimSpace(vals[0])
		case "OptionName":
			item.optionName = strings.TrimSpace(vals[0])
		case "ResourceName":
			item.resourceName = strings.TrimSpace(vals[0])
		}
	}
	ordered := make([]*optionItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].idx < ordered[j].idx })
	out := make([]ebservice.OptionSpecification, 0, len(ordered))
	for _, item := range ordered {
		if item.namespace == "" || item.optionName == "" {
			continue
		}
		out = append(out, ebservice.OptionSpecification{
			Namespace:    item.namespace,
			OptionName:   item.optionName,
			ResourceName: item.resourceName,
		})
	}
	return out
}

func parseElasticBeanstalkPlatformFilters(values url.Values, prefix string) []ebservice.PlatformFilter {
	type filterItem struct {
		idx      int
		typ      string
		operator string
		values   []string
	}
	items := map[int]*filterItem{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item := items[idx]
		if item == nil {
			item = &filterItem{idx: idx}
			items[idx] = item
		}
		switch parts[1] {
		case "Type":
			item.typ = strings.TrimSpace(vals[0])
		case "Operator":
			item.operator = strings.TrimSpace(vals[0])
		case "Values":
			if len(parts) == 4 && parts[2] == "member" {
				v := strings.TrimSpace(vals[0])
				if v != "" {
					item.values = append(item.values, v)
				}
			}
		}
	}
	ordered := make([]*filterItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].idx < ordered[j].idx })
	out := make([]ebservice.PlatformFilter, 0, len(ordered))
	for _, item := range ordered {
		if item.typ == "" {
			continue
		}
		out = append(out, ebservice.PlatformFilter{
			Type:     item.typ,
			Operator: item.operator,
			Values:   append([]string(nil), item.values...),
		})
	}
	return out
}

func parseElasticBeanstalkSearchFilters(values url.Values, prefix string) []ebservice.SearchFilter {
	type filterItem struct {
		idx       int
		attribute string
		operator  string
		values    []string
	}
	items := map[int]*filterItem{}
	for key, vals := range values {
		if !strings.HasPrefix(key, prefix+".") || len(vals) == 0 {
			continue
		}
		rest := strings.TrimPrefix(key, prefix+".")
		parts := strings.Split(rest, ".")
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		item := items[idx]
		if item == nil {
			item = &filterItem{idx: idx}
			items[idx] = item
		}
		switch parts[1] {
		case "Attribute":
			item.attribute = strings.TrimSpace(vals[0])
		case "Operator":
			item.operator = strings.TrimSpace(vals[0])
		case "Values":
			if len(parts) == 4 && parts[2] == "member" {
				v := strings.TrimSpace(vals[0])
				if v != "" {
					item.values = append(item.values, v)
				}
			}
		}
	}
	ordered := make([]*filterItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].idx < ordered[j].idx })
	out := make([]ebservice.SearchFilter, 0, len(ordered))
	for _, item := range ordered {
		if item.attribute == "" {
			continue
		}
		out = append(out, ebservice.SearchFilter{
			Attribute: item.attribute,
			Operator:  item.operator,
			Values:    append([]string(nil), item.values...),
		})
	}
	return out
}

func parseElasticBeanstalkApplicationResourceLifecycleConfig(values url.Values) (ebservice.ApplicationResourceLifecycleConfig, bool, error) {
	const root = "ResourceLifecycleConfig."
	if !hasElasticBeanstalkFormPrefix(values, root) {
		return ebservice.ApplicationResourceLifecycleConfig{}, false, nil
	}

	out := ebservice.ApplicationResourceLifecycleConfig{
		ServiceRole: strings.TrimSpace(values.Get("ResourceLifecycleConfig.ServiceRole")),
	}
	versionPrefix := "ResourceLifecycleConfig.VersionLifecycleConfig."
	if hasElasticBeanstalkFormPrefix(values, versionPrefix) {
		versionCfg := &ebservice.ApplicationVersionLifecycleConfig{}
		maxAgeRule, err := parseElasticBeanstalkMaxAgeRule(values, versionPrefix+"MaxAgeRule.")
		if err != nil {
			return ebservice.ApplicationResourceLifecycleConfig{}, true, err
		}
		versionCfg.MaxAgeRule = maxAgeRule
		maxCountRule, err := parseElasticBeanstalkMaxCountRule(values, versionPrefix+"MaxCountRule.")
		if err != nil {
			return ebservice.ApplicationResourceLifecycleConfig{}, true, err
		}
		versionCfg.MaxCountRule = maxCountRule
		out.VersionLifecycleConfig = versionCfg
	}
	return out, true, nil
}

func parseElasticBeanstalkMaxAgeRule(values url.Values, prefix string) (*ebservice.MaxAgeRule, error) {
	if !hasElasticBeanstalkFormPrefix(values, prefix) {
		return nil, nil
	}
	enabled, enabledSet, err := parseElasticBeanstalkBoolValue(values, prefix+"Enabled")
	if err != nil {
		return nil, err
	}
	if !enabledSet {
		return nil, ebservice.ErrInvalidParameter
	}
	rule := &ebservice.MaxAgeRule{Enabled: enabled}
	if v, set, err := parseElasticBeanstalkBoolValue(values, prefix+"DeleteSourceFromS3"); err != nil {
		return nil, err
	} else if set {
		rule.DeleteSourceFromS3 = v
	}
	if v, set, err := parseElasticBeanstalkInt32Value(values, prefix+"MaxAgeInDays"); err != nil {
		return nil, err
	} else if set {
		rule.MaxAgeInDays = v
	}
	return rule, nil
}

func parseElasticBeanstalkMaxCountRule(values url.Values, prefix string) (*ebservice.MaxCountRule, error) {
	if !hasElasticBeanstalkFormPrefix(values, prefix) {
		return nil, nil
	}
	enabled, enabledSet, err := parseElasticBeanstalkBoolValue(values, prefix+"Enabled")
	if err != nil {
		return nil, err
	}
	if !enabledSet {
		return nil, ebservice.ErrInvalidParameter
	}
	rule := &ebservice.MaxCountRule{Enabled: enabled}
	if v, set, err := parseElasticBeanstalkBoolValue(values, prefix+"DeleteSourceFromS3"); err != nil {
		return nil, err
	} else if set {
		rule.DeleteSourceFromS3 = v
	}
	if v, set, err := parseElasticBeanstalkInt32Value(values, prefix+"MaxCount"); err != nil {
		return nil, err
	} else if set {
		rule.MaxCount = v
	}
	return rule, nil
}

func parseElasticBeanstalkBoolValue(values url.Values, key string) (bool, bool, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		if _, ok := values[key]; ok {
			return false, true, ebservice.ErrInvalidParameter
		}
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, true, ebservice.ErrInvalidParameter
	}
	return parsed, true, nil
}

func parseElasticBeanstalkInt32Value(values url.Values, key string) (int32, bool, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		if _, ok := values[key]; ok {
			return 0, true, ebservice.ErrInvalidParameter
		}
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0, true, ebservice.ErrInvalidParameter
	}
	return int32(parsed), true, nil
}

func hasElasticBeanstalkFormPrefix(values url.Values, prefix string) bool {
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func parseOptionalInt(raw string, defaultValue int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}

func parseOptionalBool(raw string, defaultValue bool) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(raw)
}

func toAWSEBApplication(in ebservice.Application) awsebtypes.ApplicationDescription {
	return awsebtypes.ApplicationDescription{
		ApplicationArn:         strPtr(in.ARN),
		ApplicationName:        strPtr(in.Name),
		Description:            strPtr(in.Description),
		DateCreated:            timePtr(in.DateCreated),
		DateUpdated:            timePtr(in.DateUpdated),
		Versions:               append([]string(nil), in.Versions...),
		ConfigurationTemplates: append([]string(nil), in.ConfigurationTemplates...),
	}
}

func toAWSEBApplicationVersion(in ebservice.ApplicationVersion) awsebtypes.ApplicationVersionDescription {
	status := awsebtypes.ApplicationVersionStatus(in.Status)
	return awsebtypes.ApplicationVersionDescription{
		ApplicationVersionArn: strPtr(in.ARN),
		ApplicationName:       strPtr(in.ApplicationName),
		VersionLabel:          strPtr(in.VersionLabel),
		Description:           strPtr(in.Description),
		DateCreated:           timePtr(in.DateCreated),
		DateUpdated:           timePtr(in.DateUpdated),
		Status:                status,
		SourceBundle: &awsebtypes.S3Location{
			S3Bucket: strPtr(in.SourceBundle.S3Bucket),
			S3Key:    strPtr(in.SourceBundle.S3Key),
		},
	}
}

func toAWSEBEnvironment(in ebservice.Environment) awsebtypes.EnvironmentDescription {
	status := awsebtypes.EnvironmentStatus(in.Status)
	health := awsebtypes.EnvironmentHealth(in.Health)
	healthStatus := awsebtypes.EnvironmentHealthStatus(in.HealthStatus)
	return awsebtypes.EnvironmentDescription{
		EnvironmentArn:               strPtr(in.ARN),
		EnvironmentId:                strPtr(in.ID),
		EnvironmentName:              strPtr(in.Name),
		ApplicationName:              strPtr(in.ApplicationName),
		VersionLabel:                 strPtr(in.VersionLabel),
		Description:                  strPtr(in.Description),
		CNAME:                        strPtr(in.CNAME),
		EndpointURL:                  strPtr(in.EndpointURL),
		TemplateName:                 strPtr(in.TemplateName),
		SolutionStackName:            strPtr(in.SolutionStackName),
		DateCreated:                  timePtr(in.DateCreated),
		DateUpdated:                  timePtr(in.DateUpdated),
		Status:                       status,
		Health:                       health,
		HealthStatus:                 healthStatus,
		AbortableOperationInProgress: boolPtr(in.AbortableOperationInProgress),
		OperationsRole:               strPtr(in.OperationsRole),
		Tier: &awsebtypes.EnvironmentTier{
			Name: strPtr(in.TierName),
			Type: strPtr(in.TierType),
		},
	}
}

func toAWSEBEvent(in ebservice.Event) awsebtypes.EventDescription {
	severity := awsebtypes.EventSeverity(in.Severity)
	return awsebtypes.EventDescription{
		ApplicationName: strPtr(in.ApplicationName),
		EnvironmentName: strPtr(in.EnvironmentName),
		EventDate:       timePtr(in.EventDate),
		Message:         strPtr(in.Message),
		Severity:        severity,
		RequestId:       strPtr(in.RequestID),
		VersionLabel:    strPtr(in.VersionLabel),
	}
}

func toAWSEBEnvironmentResources(in ebservice.EnvironmentResourceDescription) awsebtypes.EnvironmentResourceDescription {
	asg := make([]awsebtypes.AutoScalingGroup, 0, len(in.AutoScalingGroups))
	for _, name := range in.AutoScalingGroups {
		asg = append(asg, awsebtypes.AutoScalingGroup{Name: strPtr(name)})
	}
	instances := make([]awsebtypes.Instance, 0, len(in.Instances))
	for _, id := range in.Instances {
		instances = append(instances, awsebtypes.Instance{Id: strPtr(id)})
	}
	lcfg := make([]awsebtypes.LaunchConfiguration, 0, len(in.LaunchConfigurations))
	for _, name := range in.LaunchConfigurations {
		lcfg = append(lcfg, awsebtypes.LaunchConfiguration{Name: strPtr(name)})
	}
	lbs := make([]awsebtypes.LoadBalancer, 0, len(in.LoadBalancers))
	for _, name := range in.LoadBalancers {
		lbs = append(lbs, awsebtypes.LoadBalancer{Name: strPtr(name)})
	}
	queues := make([]awsebtypes.Queue, 0, len(in.Queues))
	for _, name := range in.Queues {
		queues = append(queues, awsebtypes.Queue{Name: strPtr(name)})
	}
	triggers := make([]awsebtypes.Trigger, 0, len(in.Triggers))
	for _, name := range in.Triggers {
		triggers = append(triggers, awsebtypes.Trigger{Name: strPtr(name)})
	}
	return awsebtypes.EnvironmentResourceDescription{
		EnvironmentName:      strPtr(in.EnvironmentName),
		AutoScalingGroups:    asg,
		Instances:            instances,
		LaunchConfigurations: lcfg,
		LoadBalancers:        lbs,
		Queues:               queues,
		Triggers:             triggers,
	}
}

func toAWSEBConfigurationSettings(in ebservice.ConfigurationSettingsDescription) awsebtypes.ConfigurationSettingsDescription {
	out := awsebtypes.ConfigurationSettingsDescription{
		ApplicationName:   strPtr(in.ApplicationName),
		TemplateName:      strPtr(in.TemplateName),
		EnvironmentName:   strPtr(in.EnvironmentName),
		Description:       strPtr(in.Description),
		SolutionStackName: strPtr(in.SolutionStackName),
		DateCreated:       timePtr(in.DateCreated),
		DateUpdated:       timePtr(in.DateUpdated),
	}
	for _, setting := range in.OptionSettings {
		out.OptionSettings = append(out.OptionSettings, awsebtypes.ConfigurationOptionSetting{
			Namespace:    strPtr(setting.Namespace),
			OptionName:   strPtr(setting.OptionName),
			ResourceName: strPtr(setting.ResourceName),
			Value:        strPtr(setting.Value),
		})
	}
	return out
}

func toAWSEBConfigurationSettingsFromTemplate(in ebservice.ConfigurationTemplate) awsebtypes.ConfigurationSettingsDescription {
	return toAWSEBConfigurationSettings(ebservice.ConfigurationSettingsDescription{
		ApplicationName:   in.ApplicationName,
		TemplateName:      in.TemplateName,
		Description:       in.Description,
		SolutionStackName: in.SolutionStackName,
		DateCreated:       in.DateCreated,
		DateUpdated:       in.DateUpdated,
		OptionSettings:    in.OptionSettings,
	})
}

func toAWSEBConfigurationOption(in ebservice.ConfigurationOptionDescription) awsebtypes.ConfigurationOptionDescription {
	valueType := awsebtypes.ConfigurationOptionValueType(in.ValueType)
	return awsebtypes.ConfigurationOptionDescription{
		Namespace:      strPtr(in.Namespace),
		Name:           strPtr(in.Name),
		DefaultValue:   strPtr(in.DefaultValue),
		ValueType:      valueType,
		ChangeSeverity: strPtr(in.ChangeSeverity),
		UserDefined:    boolPtr(in.UserDefined),
	}
}

func toAWSEBValidationMessage(in ebservice.ValidationMessage) awsebtypes.ValidationMessage {
	severity := awsebtypes.ValidationSeverity(in.Severity)
	return awsebtypes.ValidationMessage{
		Message:    strPtr(in.Message),
		Namespace:  strPtr(in.Namespace),
		OptionName: strPtr(in.OptionName),
		Severity:   severity,
	}
}

func toAWSEBEnvironmentInfo(in ebservice.EnvironmentInfo) awsebtypes.EnvironmentInfoDescription {
	infoType := awsebtypes.EnvironmentInfoType(in.InfoType)
	return awsebtypes.EnvironmentInfoDescription{
		Ec2InstanceId:   strPtr(in.Ec2InstanceID),
		InfoType:        infoType,
		Message:         strPtr(in.Message),
		SampleTimestamp: timePtr(in.SampleTimestamp),
	}
}

func toAWSEBTags(in []ebservice.Tag) []awsebtypes.Tag {
	out := make([]awsebtypes.Tag, 0, len(in))
	for _, tag := range in {
		out = append(out, awsebtypes.Tag{Key: strPtr(tag.Key), Value: strPtr(tag.Value)})
	}
	return out
}

func toAWSEBResourceQuotas(in ebservice.ResourceQuotas) *awsebtypes.ResourceQuotas {
	return &awsebtypes.ResourceQuotas{
		ApplicationQuota: &awsebtypes.ResourceQuota{
			Maximum: int32Ptr(in.ApplicationQuota.Maximum),
		},
		ApplicationVersionQuota: &awsebtypes.ResourceQuota{
			Maximum: int32Ptr(in.ApplicationVersionQuota.Maximum),
		},
		ConfigurationTemplateQuota: &awsebtypes.ResourceQuota{
			Maximum: int32Ptr(in.ConfigurationTemplateQuota.Maximum),
		},
		CustomPlatformQuota: &awsebtypes.ResourceQuota{
			Maximum: int32Ptr(in.CustomPlatformQuota.Maximum),
		},
		EnvironmentQuota: &awsebtypes.ResourceQuota{
			Maximum: int32Ptr(in.EnvironmentQuota.Maximum),
		},
	}
}

func toAWSEBBuilder(arn string) *awsebtypes.Builder {
	if strings.TrimSpace(arn) == "" {
		return nil
	}
	return &awsebtypes.Builder{ARN: strPtr(arn)}
}

func toAWSEBPlatformSummary(in ebservice.PlatformVersion) awsebtypes.PlatformSummary {
	return awsebtypes.PlatformSummary{
		OperatingSystemName:          strPtr(in.OperatingSystemName),
		OperatingSystemVersion:       strPtr(in.OperatingSystemVersion),
		PlatformArn:                  strPtr(in.PlatformARN),
		PlatformBranchLifecycleState: strPtr(in.PlatformBranchLifecycleState),
		PlatformBranchName:           strPtr(in.PlatformBranchName),
		PlatformCategory:             strPtr(in.PlatformCategory),
		PlatformLifecycleState:       strPtr(in.PlatformLifecycleState),
		PlatformOwner:                strPtr(in.PlatformOwner),
		PlatformStatus:               awsebtypes.PlatformStatus(in.PlatformStatus),
		PlatformVersion:              strPtr(in.PlatformVersion),
		SupportedAddonList:           append([]string(nil), in.SupportedAddonList...),
		SupportedTierList:            append([]string(nil), in.SupportedTierList...),
	}
}

func toAWSEBPlatformDescription(in ebservice.PlatformVersion) *awsebtypes.PlatformDescription {
	return &awsebtypes.PlatformDescription{
		DateCreated:                  timePtr(in.DateCreated),
		DateUpdated:                  timePtr(in.DateUpdated),
		Description:                  strPtr(in.Description),
		Maintainer:                   strPtr(in.Maintainer),
		OperatingSystemName:          strPtr(in.OperatingSystemName),
		OperatingSystemVersion:       strPtr(in.OperatingSystemVersion),
		PlatformArn:                  strPtr(in.PlatformARN),
		PlatformBranchLifecycleState: strPtr(in.PlatformBranchLifecycleState),
		PlatformBranchName:           strPtr(in.PlatformBranchName),
		PlatformCategory:             strPtr(in.PlatformCategory),
		PlatformLifecycleState:       strPtr(in.PlatformLifecycleState),
		PlatformName:                 strPtr(in.PlatformName),
		PlatformOwner:                strPtr(in.PlatformOwner),
		PlatformStatus:               awsebtypes.PlatformStatus(in.PlatformStatus),
		PlatformVersion:              strPtr(in.PlatformVersion),
		SolutionStackName:            strPtr(in.SolutionStackName),
		SupportedAddonList:           append([]string(nil), in.SupportedAddonList...),
		SupportedTierList:            append([]string(nil), in.SupportedTierList...),
	}
}

func toAWSEBPlatformBranchSummary(in ebservice.PlatformBranch) awsebtypes.PlatformBranchSummary {
	return awsebtypes.PlatformBranchSummary{
		BranchName:        strPtr(in.BranchName),
		BranchOrder:       in.BranchOrder,
		LifecycleState:    strPtr(in.LifecycleState),
		PlatformName:      strPtr(in.PlatformName),
		SupportedTierList: append([]string(nil), in.SupportedTierList...),
	}
}

func toAWSEBApplicationMetrics(in ebservice.HealthApplicationMetrics) *awsebtypes.ApplicationMetrics {
	return &awsebtypes.ApplicationMetrics{
		Duration:     int32Ptr(in.Duration),
		RequestCount: in.RequestCount,
		StatusCodes: &awsebtypes.StatusCodes{
			Status2xx: int32Ptr(in.StatusCodes.Status2xx),
			Status3xx: int32Ptr(in.StatusCodes.Status3xx),
			Status4xx: int32Ptr(in.StatusCodes.Status4xx),
			Status5xx: int32Ptr(in.StatusCodes.Status5xx),
		},
	}
}

func toAWSEBInstanceHealthSummary(in ebservice.HealthInstanceSummary) *awsebtypes.InstanceHealthSummary {
	return &awsebtypes.InstanceHealthSummary{
		Degraded: int32Ptr(in.Degraded),
		Info:     int32Ptr(in.Info),
		NoData:   int32Ptr(in.NoData),
		Ok:       int32Ptr(in.Ok),
		Pending:  int32Ptr(in.Pending),
		Severe:   int32Ptr(in.Severe),
		Unknown:  int32Ptr(in.Unknown),
		Warning:  int32Ptr(in.Warning),
	}
}

func toAWSEBSingleInstanceHealth(in ebservice.InstanceHealth) awsebtypes.SingleInstanceHealth {
	return awsebtypes.SingleInstanceHealth{
		ApplicationMetrics: toAWSEBApplicationMetrics(in.ApplicationMetrics),
		AvailabilityZone:   strPtr(in.AvailabilityZone),
		Causes:             append([]string(nil), in.Causes...),
		Color:              strPtr(in.Color),
		HealthStatus:       strPtr(in.HealthStatus),
		InstanceId:         strPtr(in.InstanceID),
		InstanceType:       strPtr(in.InstanceType),
		LaunchedAt:         timePtr(in.LaunchedAt),
	}
}

func toAWSEBManagedAction(in ebservice.ManagedAction) awsebtypes.ManagedAction {
	return awsebtypes.ManagedAction{
		ActionDescription: strPtr(in.ActionDescription),
		ActionId:          strPtr(in.ActionID),
		ActionType:        awsebtypes.ActionType(in.ActionType),
		Status:            awsebtypes.ActionStatus(in.Status),
		WindowStartTime:   timePtr(in.WindowStartTime),
	}
}

func toAWSEBManagedActionHistoryItem(in ebservice.ManagedActionHistoryItem) awsebtypes.ManagedActionHistoryItem {
	return awsebtypes.ManagedActionHistoryItem{
		ActionDescription:  strPtr(in.ActionDescription),
		ActionId:           strPtr(in.ActionID),
		ActionType:         awsebtypes.ActionType(in.ActionType),
		ExecutedTime:       timePtr(in.ExecutedTime),
		FailureDescription: strPtr(in.FailureDescription),
		FailureType:        awsebtypes.FailureType(in.FailureType),
		FinishedTime:       timePtr(in.FinishedTime),
		Status:             awsebtypes.ActionHistoryStatus(in.Status),
	}
}

func toAWSEBApplicationResourceLifecycleConfig(in ebservice.ApplicationResourceLifecycleConfig) *awsebtypes.ApplicationResourceLifecycleConfig {
	return &awsebtypes.ApplicationResourceLifecycleConfig{
		ServiceRole:            strPtr(in.ServiceRole),
		VersionLifecycleConfig: toAWSEBApplicationVersionLifecycleConfig(in.VersionLifecycleConfig),
	}
}

func toAWSEBApplicationVersionLifecycleConfig(in *ebservice.ApplicationVersionLifecycleConfig) *awsebtypes.ApplicationVersionLifecycleConfig {
	if in == nil {
		return nil
	}
	return &awsebtypes.ApplicationVersionLifecycleConfig{
		MaxAgeRule:   toAWSEBMaxAgeRule(in.MaxAgeRule),
		MaxCountRule: toAWSEBMaxCountRule(in.MaxCountRule),
	}
}

func toAWSEBMaxAgeRule(in *ebservice.MaxAgeRule) *awsebtypes.MaxAgeRule {
	if in == nil {
		return nil
	}
	return &awsebtypes.MaxAgeRule{
		Enabled:            boolPtr(in.Enabled),
		DeleteSourceFromS3: boolPtr(in.DeleteSourceFromS3),
		MaxAgeInDays:       int32Ptr(in.MaxAgeInDays),
	}
}

func toAWSEBMaxCountRule(in *ebservice.MaxCountRule) *awsebtypes.MaxCountRule {
	if in == nil {
		return nil
	}
	return &awsebtypes.MaxCountRule{
		Enabled:            boolPtr(in.Enabled),
		DeleteSourceFromS3: boolPtr(in.DeleteSourceFromS3),
		MaxCount:           int32Ptr(in.MaxCount),
	}
}

func strPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}

func timePtr(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	vv := v.UTC()
	return &vv
}

type elasticBeanstalkResponseEnvelope struct {
	XMLName  xml.Name                         `xml:""`
	Xmlns    string                           `xml:"xmlns,attr,omitempty"`
	Result   any                              `xml:",any"`
	Metadata elasticBeanstalkResponseMetadata `xml:"ResponseMetadata"`
}

type elasticBeanstalkResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type elasticBeanstalkErrorResponse struct {
	XMLName   xml.Name                  `xml:"ErrorResponse"`
	Xmlns     string                    `xml:"xmlns,attr,omitempty"`
	Error     elasticBeanstalkErrorBody `xml:"Error"`
	RequestID string                    `xml:"RequestId"`
}

type elasticBeanstalkErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type elasticBeanstalkCreateApplicationResult struct {
	XMLName     xml.Name                          `xml:"CreateApplicationResult"`
	Application awsebtypes.ApplicationDescription `xml:"Application"`
}

type elasticBeanstalkUpdateApplicationResult struct {
	XMLName     xml.Name                          `xml:"UpdateApplicationResult"`
	Application awsebtypes.ApplicationDescription `xml:"Application"`
}

type elasticBeanstalkDeleteApplicationResult struct {
	XMLName xml.Name `xml:"DeleteApplicationResult"`
}

type elasticBeanstalkDescribeApplicationsResult struct {
	XMLName      xml.Name                            `xml:"DescribeApplicationsResult"`
	Applications []awsebtypes.ApplicationDescription `xml:"Applications>member,omitempty"`
}

type elasticBeanstalkCreateApplicationVersionResult struct {
	XMLName            xml.Name                                 `xml:"CreateApplicationVersionResult"`
	ApplicationVersion awsebtypes.ApplicationVersionDescription `xml:"ApplicationVersion"`
}

type elasticBeanstalkUpdateApplicationVersionResult struct {
	XMLName            xml.Name                                 `xml:"UpdateApplicationVersionResult"`
	ApplicationVersion awsebtypes.ApplicationVersionDescription `xml:"ApplicationVersion"`
}

type elasticBeanstalkDeleteApplicationVersionResult struct {
	XMLName xml.Name `xml:"DeleteApplicationVersionResult"`
}

type elasticBeanstalkDescribeApplicationVersionsResult struct {
	XMLName             xml.Name                                   `xml:"DescribeApplicationVersionsResult"`
	ApplicationVersions []awsebtypes.ApplicationVersionDescription `xml:"ApplicationVersions>member,omitempty"`
}

type elasticBeanstalkCreateConfigurationTemplateResult struct {
	XMLName  xml.Name                                    `xml:"CreateConfigurationTemplateResult"`
	Template awsebtypes.ConfigurationSettingsDescription `xml:"Template"`
}

type elasticBeanstalkUpdateConfigurationTemplateResult struct {
	XMLName  xml.Name                                    `xml:"UpdateConfigurationTemplateResult"`
	Template awsebtypes.ConfigurationSettingsDescription `xml:"Template"`
}

type elasticBeanstalkDeleteConfigurationTemplateResult struct {
	XMLName xml.Name `xml:"DeleteConfigurationTemplateResult"`
}

type elasticBeanstalkDescribeConfigurationSettingsResult struct {
	XMLName               xml.Name                                      `xml:"DescribeConfigurationSettingsResult"`
	ConfigurationSettings []awsebtypes.ConfigurationSettingsDescription `xml:"ConfigurationSettings>member,omitempty"`
}

type elasticBeanstalkDescribeConfigurationOptionsResult struct {
	XMLName           xml.Name                                    `xml:"DescribeConfigurationOptionsResult"`
	SolutionStackName string                                      `xml:"SolutionStackName,omitempty"`
	Options           []awsebtypes.ConfigurationOptionDescription `xml:"Options>member,omitempty"`
}

type elasticBeanstalkValidateConfigurationSettingsResult struct {
	XMLName  xml.Name                       `xml:"ValidateConfigurationSettingsResult"`
	Messages []awsebtypes.ValidationMessage `xml:"Messages>member,omitempty"`
}

type elasticBeanstalkCreateEnvironmentResult struct {
	XMLName     xml.Name                          `xml:"CreateEnvironmentResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkUpdateEnvironmentResult struct {
	XMLName     xml.Name                          `xml:"UpdateEnvironmentResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkAbortEnvironmentUpdateResult struct {
	XMLName     xml.Name                          `xml:"AbortEnvironmentUpdateResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkRebuildEnvironmentResult struct {
	XMLName     xml.Name                          `xml:"RebuildEnvironmentResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkRestartAppServerResult struct {
	XMLName     xml.Name                          `xml:"RestartAppServerResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkTerminateEnvironmentResult struct {
	XMLName     xml.Name                          `xml:"TerminateEnvironmentResult"`
	Environment awsebtypes.EnvironmentDescription `xml:"Environment"`
}

type elasticBeanstalkSwapEnvironmentCNAMEsResult struct {
	XMLName xml.Name `xml:"SwapEnvironmentCNAMEsResult"`
}

type elasticBeanstalkDescribeEnvironmentsResult struct {
	XMLName      xml.Name                            `xml:"DescribeEnvironmentsResult"`
	Environments []awsebtypes.EnvironmentDescription `xml:"Environments>member,omitempty"`
}

type elasticBeanstalkDescribeEnvironmentResourcesResult struct {
	XMLName              xml.Name                                  `xml:"DescribeEnvironmentResourcesResult"`
	EnvironmentResources awsebtypes.EnvironmentResourceDescription `xml:"EnvironmentResources"`
}

type elasticBeanstalkDescribeEventsResult struct {
	XMLName xml.Name                      `xml:"DescribeEventsResult"`
	Events  []awsebtypes.EventDescription `xml:"Events>member,omitempty"`
}

type elasticBeanstalkRequestEnvironmentInfoResult struct {
	XMLName xml.Name `xml:"RequestEnvironmentInfoResult"`
}

type elasticBeanstalkRetrieveEnvironmentInfoResult struct {
	XMLName         xml.Name                                `xml:"RetrieveEnvironmentInfoResult"`
	EnvironmentInfo []awsebtypes.EnvironmentInfoDescription `xml:"EnvironmentInfo>member,omitempty"`
}

type elasticBeanstalkCheckDNSAvailabilityResult struct {
	XMLName             xml.Name `xml:"CheckDNSAvailabilityResult"`
	Available           *bool    `xml:"Available,omitempty"`
	FullyQualifiedCNAME *string  `xml:"FullyQualifiedCNAME,omitempty"`
}

type elasticBeanstalkCreateStorageLocationResult struct {
	XMLName  xml.Name `xml:"CreateStorageLocationResult"`
	S3Bucket *string  `xml:"S3Bucket,omitempty"`
}

type elasticBeanstalkListAvailableSolutionStacksResult struct {
	XMLName        xml.Name `xml:"ListAvailableSolutionStacksResult"`
	SolutionStacks []string `xml:"SolutionStacks>member,omitempty"`
}

type elasticBeanstalkDescribeAccountAttributesResult struct {
	XMLName        xml.Name                   `xml:"DescribeAccountAttributesResult"`
	ResourceQuotas *awsebtypes.ResourceQuotas `xml:"ResourceQuotas,omitempty"`
}

type elasticBeanstalkAssociateEnvironmentOperationsRoleResult struct {
	XMLName xml.Name `xml:"AssociateEnvironmentOperationsRoleResult"`
}

type elasticBeanstalkDisassociateEnvironmentOperationsRoleResult struct {
	XMLName xml.Name `xml:"DisassociateEnvironmentOperationsRoleResult"`
}

type elasticBeanstalkComposeEnvironmentsResult struct {
	XMLName      xml.Name                            `xml:"ComposeEnvironmentsResult"`
	Environments []awsebtypes.EnvironmentDescription `xml:"Environments>member,omitempty"`
	NextToken    *string                             `xml:"NextToken,omitempty"`
}

type elasticBeanstalkCreatePlatformVersionResult struct {
	XMLName         xml.Name                   `xml:"CreatePlatformVersionResult"`
	Builder         *awsebtypes.Builder        `xml:"Builder,omitempty"`
	PlatformSummary awsebtypes.PlatformSummary `xml:"PlatformSummary"`
}

type elasticBeanstalkDeletePlatformVersionResult struct {
	XMLName         xml.Name                   `xml:"DeletePlatformVersionResult"`
	PlatformSummary awsebtypes.PlatformSummary `xml:"PlatformSummary"`
}

type elasticBeanstalkDescribePlatformVersionResult struct {
	XMLName             xml.Name                        `xml:"DescribePlatformVersionResult"`
	PlatformDescription *awsebtypes.PlatformDescription `xml:"PlatformDescription,omitempty"`
}

type elasticBeanstalkListPlatformVersionsResult struct {
	XMLName             xml.Name                     `xml:"ListPlatformVersionsResult"`
	PlatformSummaryList []awsebtypes.PlatformSummary `xml:"PlatformSummaryList>member,omitempty"`
	NextToken           *string                      `xml:"NextToken,omitempty"`
}

type elasticBeanstalkListPlatformBranchesResult struct {
	XMLName                   xml.Name                           `xml:"ListPlatformBranchesResult"`
	PlatformBranchSummaryList []awsebtypes.PlatformBranchSummary `xml:"PlatformBranchSummaryList>member,omitempty"`
	NextToken                 *string                            `xml:"NextToken,omitempty"`
}

type elasticBeanstalkDescribeEnvironmentHealthResult struct {
	XMLName            xml.Name                          `xml:"DescribeEnvironmentHealthResult"`
	ApplicationMetrics *awsebtypes.ApplicationMetrics    `xml:"ApplicationMetrics,omitempty"`
	Causes             []string                          `xml:"Causes>member,omitempty"`
	Color              *string                           `xml:"Color,omitempty"`
	EnvironmentName    *string                           `xml:"EnvironmentName,omitempty"`
	HealthStatus       *string                           `xml:"HealthStatus,omitempty"`
	InstancesHealth    *awsebtypes.InstanceHealthSummary `xml:"InstancesHealth,omitempty"`
	RefreshedAt        *time.Time                        `xml:"RefreshedAt,omitempty"`
	Status             awsebtypes.EnvironmentHealth      `xml:"Status,omitempty"`
}

type elasticBeanstalkDescribeInstancesHealthResult struct {
	XMLName            xml.Name                          `xml:"DescribeInstancesHealthResult"`
	InstanceHealthList []awsebtypes.SingleInstanceHealth `xml:"InstanceHealthList>member,omitempty"`
	NextToken          *string                           `xml:"NextToken,omitempty"`
	RefreshedAt        *time.Time                        `xml:"RefreshedAt,omitempty"`
}

type elasticBeanstalkDeleteEnvironmentConfigurationResult struct {
	XMLName xml.Name `xml:"DeleteEnvironmentConfigurationResult"`
}

type elasticBeanstalkApplyEnvironmentManagedActionResult struct {
	XMLName           xml.Name              `xml:"ApplyEnvironmentManagedActionResult"`
	ActionDescription *string               `xml:"ActionDescription,omitempty"`
	ActionId          *string               `xml:"ActionId,omitempty"`
	ActionType        awsebtypes.ActionType `xml:"ActionType,omitempty"`
	Status            *string               `xml:"Status,omitempty"`
}

type elasticBeanstalkDescribeEnvironmentManagedActionsResult struct {
	XMLName        xml.Name                   `xml:"DescribeEnvironmentManagedActionsResult"`
	ManagedActions []awsebtypes.ManagedAction `xml:"ManagedActions>member,omitempty"`
}

type elasticBeanstalkDescribeEnvironmentManagedActionHistoryResult struct {
	XMLName                   xml.Name                              `xml:"DescribeEnvironmentManagedActionHistoryResult"`
	ManagedActionHistoryItems []awsebtypes.ManagedActionHistoryItem `xml:"ManagedActionHistoryItems>member,omitempty"`
	NextToken                 *string                               `xml:"NextToken,omitempty"`
}

type elasticBeanstalkUpdateApplicationResourceLifecycleResult struct {
	XMLName                 xml.Name                                       `xml:"UpdateApplicationResourceLifecycleResult"`
	ApplicationName         *string                                        `xml:"ApplicationName,omitempty"`
	ResourceLifecycleConfig *awsebtypes.ApplicationResourceLifecycleConfig `xml:"ResourceLifecycleConfig,omitempty"`
}

type elasticBeanstalkListTagsForResourceResult struct {
	XMLName      xml.Name         `xml:"ListTagsForResourceResult"`
	ResourceArn  *string          `xml:"ResourceArn,omitempty"`
	ResourceTags []awsebtypes.Tag `xml:"ResourceTags>member,omitempty"`
}

type elasticBeanstalkUpdateTagsForResourceResult struct {
	XMLName xml.Name `xml:"UpdateTagsForResourceResult"`
}
