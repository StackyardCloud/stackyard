package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	deviceFarmDefaultRegion    = "us-east-1"
	deviceFarmDefaultAccountID = "123456789012"
)

type deviceFarmStore struct {
	mu sync.Mutex

	nextID int64

	accountSettings map[string]any

	projects             map[string]map[string]any
	devicePools          map[string]map[string]any
	uploads              map[string]map[string]any
	runs                 map[string]map[string]any
	jobs                 map[string]map[string]any
	suites               map[string]map[string]any
	tests                map[string]map[string]any
	remoteAccessSessions map[string]map[string]any
	instanceProfiles     map[string]map[string]any
	networkProfiles      map[string]map[string]any
	vpceConfigurations   map[string]map[string]any
	deviceInstances      map[string]map[string]any
	testGridProjects     map[string]map[string]any
	testGridSessions     map[string]map[string]any
	offerings            map[string]map[string]any
	offeringTransactions map[string]map[string]any
	tags                 map[string]map[string]string
}

func newDeviceFarmStore() *deviceFarmStore {
	now := time.Now().UTC().Format(time.RFC3339)

	s := &deviceFarmStore{
		nextID: 2,
		accountSettings: map[string]any{
			"awsAccountNumber":     deviceFarmDefaultAccountID,
			"unmeteredDevices":     map[string]any{},
			"maxJobTimeoutMinutes": 600,
			"trialMinutes":         map[string]any{},
		},
		projects:             map[string]map[string]any{},
		devicePools:          map[string]map[string]any{},
		uploads:              map[string]map[string]any{},
		runs:                 map[string]map[string]any{},
		jobs:                 map[string]map[string]any{},
		suites:               map[string]map[string]any{},
		tests:                map[string]map[string]any{},
		remoteAccessSessions: map[string]map[string]any{},
		instanceProfiles:     map[string]map[string]any{},
		networkProfiles:      map[string]map[string]any{},
		vpceConfigurations:   map[string]map[string]any{},
		deviceInstances:      map[string]map[string]any{},
		testGridProjects:     map[string]map[string]any{},
		testGridSessions:     map[string]map[string]any{},
		offerings:            map[string]map[string]any{},
		offeringTransactions: map[string]map[string]any{},
		tags:                 map[string]map[string]string{},
	}

	s.seedLocked(now)
	return s
}

func (s *deviceFarmStore) seedLocked(now string) {
	projectARN := dfARN("project", "project-000001")
	project := map[string]any{
		"arn":     projectARN,
		"name":    "stackyard-devicefarm-project",
		"created": now,
		"status":  "ACTIVE",
	}
	s.projects[projectARN] = project

	poolARN := dfARN("devicepool", "pool-000001")
	s.devicePools[poolARN] = map[string]any{
		"arn":         poolARN,
		"name":        "stackyard-device-pool",
		"description": "Default Stackyard device pool",
		"type":        "CURATED",
		"projectArn":  projectARN,
		"created":     now,
		"status":      "ACTIVE",
	}

	uploadARN := dfARN("upload", "upload-000001")
	s.uploads[uploadARN] = map[string]any{
		"arn":        uploadARN,
		"name":       "stackyard-upload.apk",
		"type":       "ANDROID_APP",
		"url":        "https://example.com/devicefarm/uploads/upload-000001",
		"projectArn": projectARN,
		"created":    now,
		"status":     "SUCCEEDED",
	}

	runARN := dfARN("run", "run-000001")
	s.runs[runARN] = map[string]any{
		"arn":           runARN,
		"name":          "stackyard-run",
		"projectArn":    projectARN,
		"devicePoolArn": poolARN,
		"appUploadArn":  uploadARN,
		"status":        "COMPLETED",
		"result":        "PASSED",
		"created":       now,
	}

	suiteARN := dfARN("suite", "suite-000001")
	s.suites[suiteARN] = map[string]any{
		"arn":     suiteARN,
		"name":    "stackyard-suite",
		"runArn":  runARN,
		"status":  "COMPLETED",
		"result":  "PASSED",
		"created": now,
	}

	jobARN := dfARN("job", "job-000001")
	s.jobs[jobARN] = map[string]any{
		"arn":      jobARN,
		"name":     "stackyard-job",
		"runArn":   runARN,
		"suiteArn": suiteARN,
		"status":   "COMPLETED",
		"result":   "PASSED",
		"created":  now,
	}

	testARN := dfARN("test", "test-000001")
	s.tests[testARN] = map[string]any{
		"arn":      testARN,
		"name":     "stackyard-test",
		"runArn":   runARN,
		"suiteArn": suiteARN,
		"jobArn":   jobARN,
		"status":   "COMPLETED",
		"result":   "PASSED",
		"created":  now,
	}

	instanceProfileARN := dfARN("instanceprofile", "ip-000001")
	s.instanceProfiles[instanceProfileARN] = map[string]any{
		"arn":                           instanceProfileARN,
		"name":                          "stackyard-instance-profile",
		"description":                   "Default Stackyard instance profile",
		"packageCleanup":                true,
		"excludeAppPackagesFromCleanup": []any{},
		"created":                       now,
	}

	networkProfileARN := dfARN("networkprofile", "np-000001")
	s.networkProfiles[networkProfileARN] = map[string]any{
		"arn":                   networkProfileARN,
		"name":                  "stackyard-network-profile",
		"description":           "Default Stackyard network profile",
		"type":                  "PRIVATE",
		"uplinkBandwidthBits":   10000000,
		"downlinkBandwidthBits": 10000000,
		"created":               now,
	}

	vpceARN := dfARN("vpceconfiguration", "vpce-000001")
	s.vpceConfigurations[vpceARN] = map[string]any{
		"arn":             vpceARN,
		"name":            "stackyard-vpce",
		"description":     "Default Stackyard VPC endpoint configuration",
		"serviceDnsName":  "devicefarm.us-east-1.amazonaws.com",
		"vpceServiceName": "com.amazonaws.vpce.us-east-1.vpce-svc-000001",
		"created":         now,
	}

	deviceInstanceARN := dfARN("deviceinstance", "di-000001")
	s.deviceInstances[deviceInstanceARN] = map[string]any{
		"arn":                deviceInstanceARN,
		"deviceArn":          dfARN("device", "device-000001"),
		"instanceProfileArn": instanceProfileARN,
		"status":             "AVAILABLE",
		"udid":               "device-udid-000001",
		"created":            now,
	}

	testGridProjectARN := dfARN("testgrid-project", "tgp-000001")
	s.testGridProjects[testGridProjectARN] = map[string]any{
		"arn":         testGridProjectARN,
		"name":        "stackyard-testgrid-project",
		"description": "Default Stackyard TestGrid project",
		"created":     now,
	}

	sessionExpiry := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
	testGridSessionARN := dfARN("testgrid-session", "tgs-000001")
	s.testGridSessions[testGridSessionARN] = map[string]any{
		"arn":                testGridSessionARN,
		"projectArn":         testGridProjectARN,
		"status":             "RUNNING",
		"created":            now,
		"endTime":            sessionExpiry,
		"billingMinutes":     1,
		"browser":            "chrome",
		"seleniumProperties": map[string]any{},
	}

	offeringID := "offering-000001"
	s.offerings[offeringID] = map[string]any{
		"id":          offeringID,
		"type":        "RECURRING",
		"description": "Default Stackyard offering",
		"platform":    "ANDROID",
		"recurringCharges": []any{
			map[string]any{
				"cost": map[string]any{
					"amount":       1.0,
					"currencyCode": "USD",
				},
				"frequency": "MONTHLY",
			},
		},
	}

	s.tags[projectARN] = map[string]string{"stackyard": "true"}
	s.tags[poolARN] = map[string]string{"stackyard": "true"}
	s.tags[uploadARN] = map[string]string{"stackyard": "true"}
	s.tags[runARN] = map[string]string{"stackyard": "true"}
}

func (s *deviceFarmStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateProject":
		id := s.nextIDLocked("project")
		arn := dfARN("project", id)
		project := map[string]any{
			"arn":     arn,
			"name":    dfString(payload, []string{"name", "Name"}, "stackyard-devicefarm-project"),
			"created": now,
			"status":  "ACTIVE",
		}
		s.projects[arn] = project
		s.ensureTagsLocked(arn)["stackyard"] = "true"
		return map[string]any{"project": dfCloneMap(project)}

	case "GetProject":
		arn := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		return map[string]any{"project": dfCloneMap(s.ensureProjectLocked(arn, now))}

	case "UpdateProject":
		arn := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		project := s.ensureProjectLocked(arn, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			project["name"] = name
		}
		project["status"] = "ACTIVE"
		return map[string]any{"project": dfCloneMap(project)}

	case "DeleteProject":
		arn := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		project := s.ensureProjectLocked(arn, now)
		project["status"] = "DELETING"
		for key, value := range s.devicePools {
			if dfString(value, []string{"projectArn"}, "") == arn {
				delete(s.devicePools, key)
			}
		}
		for key, value := range s.uploads {
			if dfString(value, []string{"projectArn"}, "") == arn {
				delete(s.uploads, key)
			}
		}
		for key, value := range s.runs {
			if dfString(value, []string{"projectArn"}, "") == arn {
				delete(s.runs, key)
			}
		}
		delete(s.projects, arn)
		delete(s.tags, arn)
		return map[string]any{"project": dfCloneMap(project)}

	case "ListProjects":
		return map[string]any{"projects": s.listResourcesLocked(s.projects), "nextToken": ""}

	case "CreateDevicePool":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "arn", "Arn"}, ""))
		id := s.nextIDLocked("pool")
		arn := dfARN("devicepool", id)
		pool := map[string]any{
			"arn":         arn,
			"name":        dfString(payload, []string{"name", "Name"}, "stackyard-device-pool"),
			"description": dfString(payload, []string{"description", "Description"}, "Stackyard device pool"),
			"type":        dfString(payload, []string{"type", "Type"}, "CURATED"),
			"projectArn":  projectARN,
			"rules":       []any{},
			"created":     now,
			"status":      "ACTIVE",
		}
		s.devicePools[arn] = pool
		s.ensureTagsLocked(arn)["stackyard"] = "true"
		return map[string]any{"devicePool": dfCloneMap(pool)}

	case "GetDevicePool":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveDevicePoolARNLocked(dfString(payload, []string{"arn", "Arn", "devicePoolArn", "DevicePoolArn"}, ""), projectARN)
		return map[string]any{"devicePool": dfCloneMap(s.ensureDevicePoolLocked(arn, projectARN, now))}

	case "UpdateDevicePool":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveDevicePoolARNLocked(dfString(payload, []string{"arn", "Arn", "devicePoolArn", "DevicePoolArn"}, ""), projectARN)
		pool := s.ensureDevicePoolLocked(arn, projectARN, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			pool["name"] = name
		}
		if description := dfString(payload, []string{"description", "Description"}, ""); description != "" {
			pool["description"] = description
		}
		pool["status"] = "ACTIVE"
		return map[string]any{"devicePool": dfCloneMap(pool)}

	case "DeleteDevicePool":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveDevicePoolARNLocked(dfString(payload, []string{"arn", "Arn", "devicePoolArn", "DevicePoolArn"}, ""), projectARN)
		pool := s.ensureDevicePoolLocked(arn, projectARN, now)
		pool["status"] = "DELETING"
		delete(s.devicePools, arn)
		delete(s.tags, arn)
		return map[string]any{"devicePool": dfCloneMap(pool)}

	case "ListDevicePools":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		return map[string]any{"devicePools": s.listByProjectLocked(s.devicePools, projectARN), "nextToken": ""}

	case "GetDevicePoolCompatibility":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		poolARN := s.resolveDevicePoolARNLocked(dfString(payload, []string{"devicePoolArn", "DevicePoolArn", "arn", "Arn"}, ""), projectARN)
		_ = s.ensureDevicePoolLocked(poolARN, projectARN, now)
		device := s.devicePayload(dfString(payload, []string{"deviceArn", "DeviceArn"}, ""))
		return map[string]any{
			"compatibleDevices": []any{
				map[string]any{
					"device":                  device,
					"compatible":              true,
					"incompatibilityMessages": []any{},
				},
			},
			"incompatibleDevices": []any{},
		}

	case "CreateUpload":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "arn", "Arn"}, ""))
		id := s.nextIDLocked("upload")
		arn := dfARN("upload", id)
		upload := map[string]any{
			"arn":        arn,
			"name":       dfString(payload, []string{"name", "Name"}, "stackyard-upload.apk"),
			"type":       dfString(payload, []string{"type", "Type"}, "ANDROID_APP"),
			"url":        "https://example.com/devicefarm/uploads/" + id,
			"projectArn": projectARN,
			"created":    now,
			"status":     "INITIALIZED",
		}
		s.uploads[arn] = upload
		s.ensureTagsLocked(arn)["stackyard"] = "true"
		return map[string]any{"upload": dfCloneMap(upload)}

	case "GetUpload":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveUploadARNLocked(dfString(payload, []string{"arn", "Arn", "uploadArn", "UploadArn"}, ""), projectARN)
		return map[string]any{"upload": dfCloneMap(s.ensureUploadLocked(arn, projectARN, now))}

	case "UpdateUpload":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveUploadARNLocked(dfString(payload, []string{"arn", "Arn", "uploadArn", "UploadArn"}, ""), projectARN)
		upload := s.ensureUploadLocked(arn, projectARN, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			upload["name"] = name
		}
		if status := dfString(payload, []string{"status", "Status"}, ""); status != "" {
			upload["status"] = status
		} else {
			upload["status"] = "SUCCEEDED"
		}
		return map[string]any{"upload": dfCloneMap(upload)}

	case "DeleteUpload":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveUploadARNLocked(dfString(payload, []string{"arn", "Arn", "uploadArn", "UploadArn"}, ""), projectARN)
		upload := s.ensureUploadLocked(arn, projectARN, now)
		upload["status"] = "DELETING"
		delete(s.uploads, arn)
		delete(s.tags, arn)
		return map[string]any{"upload": dfCloneMap(upload)}

	case "ListUploads":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		return map[string]any{"uploads": s.listByProjectLocked(s.uploads, projectARN), "nextToken": ""}

	case "ScheduleRun":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "arn", "Arn"}, ""))
		runID := s.nextIDLocked("run")
		runARN := dfARN("run", runID)
		devicePoolARN := s.resolveDevicePoolARNLocked(dfString(payload, []string{"devicePoolArn", "DevicePoolArn"}, ""), projectARN)
		appUploadARN := s.resolveUploadARNLocked(dfString(payload, []string{"appArn", "appUploadArn", "UploadArn"}, ""), projectARN)
		run := map[string]any{
			"arn":           runARN,
			"name":          dfString(payload, []string{"name", "Name"}, "stackyard-run"),
			"projectArn":    projectARN,
			"devicePoolArn": devicePoolARN,
			"appUploadArn":  appUploadARN,
			"status":        "RUNNING",
			"result":        "PENDING",
			"created":       now,
		}
		s.runs[runARN] = run
		s.ensureTagsLocked(runARN)["stackyard"] = "true"

		suiteARN := dfARN("suite", s.nextIDLocked("suite"))
		s.suites[suiteARN] = map[string]any{
			"arn":     suiteARN,
			"name":    "stackyard-suite",
			"runArn":  runARN,
			"status":  "RUNNING",
			"result":  "PENDING",
			"created": now,
		}

		jobARN := dfARN("job", s.nextIDLocked("job"))
		s.jobs[jobARN] = map[string]any{
			"arn":      jobARN,
			"name":     "stackyard-job",
			"runArn":   runARN,
			"suiteArn": suiteARN,
			"status":   "RUNNING",
			"result":   "PENDING",
			"created":  now,
		}

		testARN := dfARN("test", s.nextIDLocked("test"))
		s.tests[testARN] = map[string]any{
			"arn":      testARN,
			"name":     "stackyard-test",
			"runArn":   runARN,
			"suiteArn": suiteARN,
			"jobArn":   jobARN,
			"status":   "RUNNING",
			"result":   "PENDING",
			"created":  now,
		}

		return map[string]any{"run": dfCloneMap(run)}

	case "GetRun":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), projectARN)
		return map[string]any{"run": dfCloneMap(s.ensureRunLocked(runARN, projectARN, now))}

	case "ListRuns":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		return map[string]any{"runs": s.listByProjectLocked(s.runs, projectARN), "nextToken": ""}

	case "StopRun":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), projectARN)
		run := s.ensureRunLocked(runARN, projectARN, now)
		run["status"] = "STOPPED"
		run["result"] = "STOPPED"
		return map[string]any{"run": dfCloneMap(run)}

	case "DeleteRun":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), projectARN)
		run := s.ensureRunLocked(runARN, projectARN, now)
		run["status"] = "DELETING"
		delete(s.runs, runARN)
		return map[string]any{"run": dfCloneMap(run)}

	case "ListJobs":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		return map[string]any{"jobs": s.listByRunLocked(s.jobs, runARN), "nextToken": ""}

	case "GetJob":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		jobARN := s.resolveJobARNLocked(dfString(payload, []string{"arn", "Arn", "jobArn", "JobArn"}, ""), runARN)
		return map[string]any{"job": dfCloneMap(s.ensureJobLocked(jobARN, runARN, now))}

	case "StopJob":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		jobARN := s.resolveJobARNLocked(dfString(payload, []string{"arn", "Arn", "jobArn", "JobArn"}, ""), runARN)
		job := s.ensureJobLocked(jobARN, runARN, now)
		job["status"] = "STOPPED"
		job["result"] = "STOPPED"
		return map[string]any{"job": dfCloneMap(job)}

	case "ListSuites":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		return map[string]any{"suites": s.listByRunLocked(s.suites, runARN), "nextToken": ""}

	case "GetSuite":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		suiteARN := s.resolveSuiteARNLocked(dfString(payload, []string{"arn", "Arn", "suiteArn", "SuiteArn"}, ""), runARN)
		return map[string]any{"suite": dfCloneMap(s.ensureSuiteLocked(suiteARN, runARN, now))}

	case "ListTests":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"arn", "Arn", "runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		return map[string]any{"tests": s.listByRunLocked(s.tests, runARN), "nextToken": ""}

	case "GetTest":
		runARN := s.resolveRunARNLocked(dfString(payload, []string{"runArn", "RunArn"}, ""), s.firstProjectARNLocked())
		testARN := s.resolveTestARNLocked(dfString(payload, []string{"arn", "Arn", "testArn", "TestArn"}, ""), runARN)
		return map[string]any{"test": dfCloneMap(s.ensureTestLocked(testARN, runARN, now))}

	case "ListArtifacts":
		artifactARN := dfString(payload, []string{"arn", "Arn"}, s.resolveRunARNLocked("", s.firstProjectARNLocked()))
		return map[string]any{
			"artifacts": []any{
				map[string]any{
					"arn":       dfARN("artifact", s.nextIDLocked("artifact")),
					"name":      "artifact-logcat.txt",
					"type":      "FILE",
					"extension": "txt",
					"url":       "https://example.com/devicefarm/artifacts/" + dfIDFromARN(artifactARN, "artifact"),
				},
			},
			"nextToken": "",
		}

	case "ListSamples":
		return map[string]any{
			"samples": []any{
				map[string]any{
					"arn":        dfARN("sample", "sample-000001"),
					"url":        "https://example.com/devicefarm/samples/sample-000001.jpg",
					"metadata":   map[string]any{},
					"sampleType": "SCREENSHOT",
				},
			},
			"nextToken": "",
		}

	case "ListUniqueProblems":
		return map[string]any{
			"uniqueProblems": map[string]any{
				"errored": []any{},
				"failed":  []any{},
				"passed":  []any{},
				"skipped": []any{},
				"warned":  []any{},
			},
			"nextToken": "",
		}

	case "CreateRemoteAccessSession":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "arn", "Arn"}, ""))
		id := s.nextIDLocked("session")
		arn := dfARN("session", id)
		session := map[string]any{
			"arn":        arn,
			"projectArn": projectARN,
			"deviceArn":  dfString(payload, []string{"deviceArn", "DeviceArn"}, dfARN("device", "device-000001")),
			"status":     "RUNNING",
			"created":    now,
		}
		s.remoteAccessSessions[arn] = session
		return map[string]any{"remoteAccessSession": dfCloneMap(session)}

	case "GetRemoteAccessSession":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveRemoteAccessSessionARNLocked(dfString(payload, []string{"arn", "Arn", "remoteAccessSessionArn", "RemoteAccessSessionArn"}, ""), projectARN)
		return map[string]any{"remoteAccessSession": dfCloneMap(s.ensureRemoteAccessSessionLocked(arn, projectARN, now))}

	case "ListRemoteAccessSessions":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"arn", "Arn", "projectArn", "ProjectArn"}, ""))
		return map[string]any{"remoteAccessSessions": s.listByProjectLocked(s.remoteAccessSessions, projectARN), "nextToken": ""}

	case "StopRemoteAccessSession":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveRemoteAccessSessionARNLocked(dfString(payload, []string{"arn", "Arn", "remoteAccessSessionArn", "RemoteAccessSessionArn"}, ""), projectARN)
		session := s.ensureRemoteAccessSessionLocked(arn, projectARN, now)
		session["status"] = "STOPPED"
		return map[string]any{"remoteAccessSession": dfCloneMap(session)}

	case "DeleteRemoteAccessSession":
		projectARN := s.resolveProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn"}, ""))
		arn := s.resolveRemoteAccessSessionARNLocked(dfString(payload, []string{"arn", "Arn", "remoteAccessSessionArn", "RemoteAccessSessionArn"}, ""), projectARN)
		session := s.ensureRemoteAccessSessionLocked(arn, projectARN, now)
		session["status"] = "DELETING"
		delete(s.remoteAccessSessions, arn)
		return map[string]any{"remoteAccessSession": dfCloneMap(session)}

	case "InstallToRemoteAccessSession":
		return map[string]any{"success": true}

	case "CreateInstanceProfile":
		id := s.nextIDLocked("ip")
		arn := dfARN("instanceprofile", id)
		profile := map[string]any{
			"arn":                           arn,
			"name":                          dfString(payload, []string{"name", "Name"}, "stackyard-instance-profile"),
			"description":                   dfString(payload, []string{"description", "Description"}, "Stackyard instance profile"),
			"packageCleanup":                true,
			"excludeAppPackagesFromCleanup": []any{},
			"created":                       now,
		}
		s.instanceProfiles[arn] = profile
		return map[string]any{"instanceProfile": dfCloneMap(profile)}

	case "GetInstanceProfile":
		arn := s.resolveInstanceProfileARNLocked(dfString(payload, []string{"arn", "Arn", "instanceProfileArn", "InstanceProfileArn"}, ""))
		return map[string]any{"instanceProfile": dfCloneMap(s.ensureInstanceProfileLocked(arn, now))}

	case "ListInstanceProfiles":
		return map[string]any{"instanceProfiles": s.listResourcesLocked(s.instanceProfiles), "nextToken": ""}

	case "UpdateInstanceProfile":
		arn := s.resolveInstanceProfileARNLocked(dfString(payload, []string{"arn", "Arn", "instanceProfileArn", "InstanceProfileArn"}, ""))
		profile := s.ensureInstanceProfileLocked(arn, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			profile["name"] = name
		}
		if description := dfString(payload, []string{"description", "Description"}, ""); description != "" {
			profile["description"] = description
		}
		return map[string]any{"instanceProfile": dfCloneMap(profile)}

	case "DeleteInstanceProfile":
		arn := s.resolveInstanceProfileARNLocked(dfString(payload, []string{"arn", "Arn", "instanceProfileArn", "InstanceProfileArn"}, ""))
		profile := s.ensureInstanceProfileLocked(arn, now)
		delete(s.instanceProfiles, arn)
		return map[string]any{"instanceProfile": dfCloneMap(profile)}

	case "CreateNetworkProfile":
		id := s.nextIDLocked("np")
		arn := dfARN("networkprofile", id)
		profile := map[string]any{
			"arn":                   arn,
			"name":                  dfString(payload, []string{"name", "Name"}, "stackyard-network-profile"),
			"description":           dfString(payload, []string{"description", "Description"}, "Stackyard network profile"),
			"type":                  dfString(payload, []string{"type", "Type"}, "PRIVATE"),
			"uplinkBandwidthBits":   10000000,
			"downlinkBandwidthBits": 10000000,
			"created":               now,
		}
		s.networkProfiles[arn] = profile
		return map[string]any{"networkProfile": dfCloneMap(profile)}

	case "GetNetworkProfile":
		arn := s.resolveNetworkProfileARNLocked(dfString(payload, []string{"arn", "Arn", "networkProfileArn", "NetworkProfileArn"}, ""))
		return map[string]any{"networkProfile": dfCloneMap(s.ensureNetworkProfileLocked(arn, now))}

	case "ListNetworkProfiles":
		return map[string]any{"networkProfiles": s.listResourcesLocked(s.networkProfiles), "nextToken": ""}

	case "UpdateNetworkProfile":
		arn := s.resolveNetworkProfileARNLocked(dfString(payload, []string{"arn", "Arn", "networkProfileArn", "NetworkProfileArn"}, ""))
		profile := s.ensureNetworkProfileLocked(arn, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			profile["name"] = name
		}
		if description := dfString(payload, []string{"description", "Description"}, ""); description != "" {
			profile["description"] = description
		}
		return map[string]any{"networkProfile": dfCloneMap(profile)}

	case "DeleteNetworkProfile":
		arn := s.resolveNetworkProfileARNLocked(dfString(payload, []string{"arn", "Arn", "networkProfileArn", "NetworkProfileArn"}, ""))
		profile := s.ensureNetworkProfileLocked(arn, now)
		delete(s.networkProfiles, arn)
		return map[string]any{"networkProfile": dfCloneMap(profile)}

	case "CreateVPCEConfiguration":
		id := s.nextIDLocked("vpce")
		arn := dfARN("vpceconfiguration", id)
		configuration := map[string]any{
			"arn":             arn,
			"name":            dfString(payload, []string{"name", "Name"}, "stackyard-vpce"),
			"description":     dfString(payload, []string{"description", "Description"}, "Stackyard VPCE configuration"),
			"serviceDnsName":  dfString(payload, []string{"serviceDnsName", "ServiceDnsName"}, "devicefarm.us-east-1.amazonaws.com"),
			"vpceServiceName": dfString(payload, []string{"vpceServiceName", "VpceServiceName"}, "com.amazonaws.vpce.us-east-1.vpce-svc-000001"),
			"created":         now,
		}
		s.vpceConfigurations[arn] = configuration
		return map[string]any{"vpceConfiguration": dfCloneMap(configuration)}

	case "GetVPCEConfiguration":
		arn := s.resolveVPCEConfigurationARNLocked(dfString(payload, []string{"arn", "Arn", "vpceConfigurationArn", "VPCEConfigurationArn"}, ""))
		return map[string]any{"vpceConfiguration": dfCloneMap(s.ensureVPCEConfigurationLocked(arn, now))}

	case "ListVPCEConfigurations":
		return map[string]any{"vpceConfigurations": s.listResourcesLocked(s.vpceConfigurations), "nextToken": ""}

	case "UpdateVPCEConfiguration":
		arn := s.resolveVPCEConfigurationARNLocked(dfString(payload, []string{"arn", "Arn", "vpceConfigurationArn", "VPCEConfigurationArn"}, ""))
		configuration := s.ensureVPCEConfigurationLocked(arn, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			configuration["name"] = name
		}
		if description := dfString(payload, []string{"description", "Description"}, ""); description != "" {
			configuration["description"] = description
		}
		return map[string]any{"vpceConfiguration": dfCloneMap(configuration)}

	case "DeleteVPCEConfiguration":
		arn := s.resolveVPCEConfigurationARNLocked(dfString(payload, []string{"arn", "Arn", "vpceConfigurationArn", "VPCEConfigurationArn"}, ""))
		configuration := s.ensureVPCEConfigurationLocked(arn, now)
		delete(s.vpceConfigurations, arn)
		return map[string]any{"vpceConfiguration": dfCloneMap(configuration)}

	case "GetAccountSettings":
		return map[string]any{"accountSettings": dfCloneMap(s.accountSettings)}

	case "GetDevice":
		return map[string]any{"device": s.devicePayload(dfString(payload, []string{"arn", "Arn", "deviceArn", "DeviceArn"}, ""))}

	case "ListDevices":
		return map[string]any{
			"devices": []any{
				s.devicePayload(dfARN("device", "device-000001")),
				s.devicePayload(dfARN("device", "device-000002")),
			},
			"nextToken": "",
		}

	case "GetDeviceInstance":
		arn := s.resolveDeviceInstanceARNLocked(dfString(payload, []string{"arn", "Arn", "deviceInstanceArn", "DeviceInstanceArn"}, ""))
		return map[string]any{"deviceInstance": dfCloneMap(s.ensureDeviceInstanceLocked(arn, now))}

	case "ListDeviceInstances":
		return map[string]any{"deviceInstances": s.listResourcesLocked(s.deviceInstances), "nextToken": ""}

	case "UpdateDeviceInstance":
		arn := s.resolveDeviceInstanceARNLocked(dfString(payload, []string{"arn", "Arn", "deviceInstanceArn", "DeviceInstanceArn"}, ""))
		instance := s.ensureDeviceInstanceLocked(arn, now)
		if status := dfString(payload, []string{"status", "Status"}, ""); status != "" {
			instance["status"] = status
		} else {
			instance["status"] = "AVAILABLE"
		}
		if profileARN := dfString(payload, []string{"instanceProfileArn", "InstanceProfileArn"}, ""); profileARN != "" {
			instance["instanceProfileArn"] = profileARN
		}
		return map[string]any{"deviceInstance": dfCloneMap(instance)}

	case "CreateTestGridProject":
		id := s.nextIDLocked("tgp")
		arn := dfARN("testgrid-project", id)
		project := map[string]any{
			"arn":         arn,
			"name":        dfString(payload, []string{"name", "Name"}, "stackyard-testgrid-project"),
			"description": dfString(payload, []string{"description", "Description"}, "Stackyard TestGrid project"),
			"created":     now,
		}
		s.testGridProjects[arn] = project
		return map[string]any{"testGridProject": dfCloneMap(project)}

	case "GetTestGridProject":
		arn := s.resolveTestGridProjectARNLocked(dfString(payload, []string{"arn", "Arn", "testGridProjectArn", "TestGridProjectArn"}, ""))
		return map[string]any{"testGridProject": dfCloneMap(s.ensureTestGridProjectLocked(arn, now))}

	case "ListTestGridProjects":
		return map[string]any{"testGridProjects": s.listResourcesLocked(s.testGridProjects), "nextToken": ""}

	case "UpdateTestGridProject":
		arn := s.resolveTestGridProjectARNLocked(dfString(payload, []string{"arn", "Arn", "testGridProjectArn", "TestGridProjectArn"}, ""))
		project := s.ensureTestGridProjectLocked(arn, now)
		if name := dfString(payload, []string{"name", "Name"}, ""); name != "" {
			project["name"] = name
		}
		if description := dfString(payload, []string{"description", "Description"}, ""); description != "" {
			project["description"] = description
		}
		return map[string]any{"testGridProject": dfCloneMap(project)}

	case "DeleteTestGridProject":
		arn := s.resolveTestGridProjectARNLocked(dfString(payload, []string{"arn", "Arn", "testGridProjectArn", "TestGridProjectArn"}, ""))
		project := s.ensureTestGridProjectLocked(arn, now)
		delete(s.testGridProjects, arn)
		return map[string]any{"testGridProject": dfCloneMap(project)}

	case "CreateTestGridUrl":
		projectARN := s.resolveTestGridProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "testGridProjectArn", "TestGridProjectArn"}, ""))
		sessionID := s.nextIDLocked("tgs")
		sessionARN := dfARN("testgrid-session", sessionID)
		expires := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
		session := map[string]any{
			"arn":            sessionARN,
			"projectArn":     projectARN,
			"status":         "RUNNING",
			"created":        now,
			"endTime":        expires,
			"billingMinutes": 1,
		}
		s.testGridSessions[sessionARN] = session
		return map[string]any{
			"url":                "https://example.com/devicefarm/testgrid/" + sessionID,
			"expires":            expires,
			"testGridSessionArn": sessionARN,
		}

	case "GetTestGridSession":
		arn := s.resolveTestGridSessionARNLocked(dfString(payload, []string{"arn", "Arn", "testGridSessionArn", "TestGridSessionArn"}, ""))
		return map[string]any{"testGridSession": dfCloneMap(s.ensureTestGridSessionLocked(arn, now))}

	case "ListTestGridSessions":
		projectARN := s.resolveTestGridProjectARNLocked(dfString(payload, []string{"projectArn", "ProjectArn", "testGridProjectArn", "TestGridProjectArn"}, ""))
		return map[string]any{"testGridSessions": s.listByProjectLocked(s.testGridSessions, projectARN), "nextToken": ""}

	case "ListTestGridSessionActions":
		return map[string]any{
			"actions": []any{
				map[string]any{
					"action":      "CLICK",
					"started":     now,
					"duration":    1,
					"description": "Synthetic action",
				},
			},
			"nextToken": "",
		}

	case "ListTestGridSessionArtifacts":
		return map[string]any{
			"artifacts": []any{
				map[string]any{
					"filename": "video.mp4",
					"type":     "VIDEO",
					"url":      "https://example.com/devicefarm/testgrid/artifacts/video.mp4",
				},
			},
			"nextToken": "",
		}

	case "ListOfferings":
		return map[string]any{"offerings": s.listOfferingsLocked(), "nextToken": ""}

	case "GetOfferingStatus":
		offeringID := dfString(payload, []string{"offeringId", "OfferingId"}, "")
		if offeringID == "" {
			offeringID = s.firstOfferingIDLocked()
		}
		offering := s.ensureOfferingLocked(offeringID)
		return map[string]any{
			"current": map[string]any{
				"type":        "RECURRING",
				"offering":    dfCloneMap(offering),
				"quantity":    1,
				"effectiveOn": now,
			},
			"nextPeriod": map[string]any{
				"type":        "RECURRING",
				"offering":    dfCloneMap(offering),
				"quantity":    1,
				"effectiveOn": now,
			},
		}

	case "PurchaseOffering", "RenewOffering":
		offeringID := dfString(payload, []string{"offeringId", "OfferingId"}, "")
		if offeringID == "" {
			offeringID = s.firstOfferingIDLocked()
		}
		_ = s.ensureOfferingLocked(offeringID)
		id := s.nextIDLocked("offeringtxn")
		transaction := map[string]any{
			"offeringTransactionId": id,
			"offeringId":            offeringID,
			"createdOn":             now,
			"status":                "SUCCESS",
			"cost": map[string]any{
				"amount":       1.0,
				"currencyCode": "USD",
			},
		}
		s.offeringTransactions[id] = transaction
		return map[string]any{"offeringTransaction": dfCloneMap(transaction)}

	case "ListOfferingTransactions":
		return map[string]any{"offeringTransactions": s.listResourcesLocked(s.offeringTransactions), "nextToken": ""}

	case "ListOfferingPromotions":
		return map[string]any{
			"offeringPromotions": []any{
				map[string]any{
					"id":          "promotion-000001",
					"description": "Stackyard sample promotion",
				},
			},
			"nextToken": "",
		}

	case "TagResource":
		resourceARN := dfString(payload, []string{"resourceARN", "resourceArn", "ResourceARN", "ResourceArn", "arn", "Arn"}, "")
		if resourceARN == "" {
			resourceARN = s.firstProjectARNLocked()
		}
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range dfTags(payload) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := dfString(payload, []string{"resourceARN", "resourceArn", "ResourceARN", "ResourceArn", "arn", "Arn"}, "")
		if resourceARN == "" {
			resourceARN = s.firstProjectARNLocked()
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range dfStringSlice(payload, []string{"tagKeys", "TagKeys"}) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := dfString(payload, []string{"resourceARN", "resourceArn", "ResourceARN", "ResourceArn", "arn", "Arn"}, "")
		if resourceARN == "" {
			resourceARN = s.firstProjectARNLocked()
		}
		return map[string]any{"tags": dfCloneStringMap(s.ensureTagsLocked(resourceARN))}

	default:
		return s.genericResponseLocked(action, payload, now)
	}
}

func (s *deviceFarmStore) genericResponseLocked(action string, payload map[string]any, now string) map[string]any {
	switch action {
	case "GetVPCEConfiguration":
		return map[string]any{"vpceConfiguration": dfCloneMap(s.ensureVPCEConfigurationLocked("", now))}
	case "GetNetworkProfile":
		return map[string]any{"networkProfile": dfCloneMap(s.ensureNetworkProfileLocked("", now))}
	case "GetInstanceProfile":
		return map[string]any{"instanceProfile": dfCloneMap(s.ensureInstanceProfileLocked("", now))}
	}

	// Every known action should map to a deterministic successful response.
	return map[string]any{
		"action": action,
		"status": "ok",
	}
}

func (s *deviceFarmStore) ensureProjectLocked(arn, now string) map[string]any {
	arn = s.resolveProjectARNLocked(arn)
	if project, ok := s.projects[arn]; ok {
		return project
	}
	project := map[string]any{
		"arn":     arn,
		"name":    dfIDFromARN(arn, "project-000001"),
		"created": now,
		"status":  "ACTIVE",
	}
	s.projects[arn] = project
	return project
}

func (s *deviceFarmStore) ensureDevicePoolLocked(arn, projectARN, now string) map[string]any {
	if arn == "" {
		for key, value := range s.devicePools {
			if projectARN == "" || dfString(value, []string{"projectArn"}, "") == projectARN {
				return s.devicePools[key]
			}
		}
	}
	if arn == "" {
		id := s.nextIDLocked("pool")
		arn = dfARN("devicepool", id)
	}
	if pool, ok := s.devicePools[arn]; ok {
		return pool
	}
	pool := map[string]any{
		"arn":         arn,
		"name":        "stackyard-device-pool",
		"description": "Stackyard device pool",
		"type":        "CURATED",
		"projectArn":  s.resolveProjectARNLocked(projectARN),
		"rules":       []any{},
		"created":     now,
		"status":      "ACTIVE",
	}
	s.devicePools[arn] = pool
	return pool
}

func (s *deviceFarmStore) ensureUploadLocked(arn, projectARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.uploads {
			if projectARN == "" || dfString(value, []string{"projectArn"}, "") == projectARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("upload", s.nextIDLocked("upload"))
	}
	if upload, ok := s.uploads[arn]; ok {
		return upload
	}
	upload := map[string]any{
		"arn":        arn,
		"name":       "stackyard-upload.apk",
		"type":       "ANDROID_APP",
		"url":        "https://example.com/devicefarm/uploads/" + dfIDFromARN(arn, "upload-000001"),
		"projectArn": s.resolveProjectARNLocked(projectARN),
		"created":    now,
		"status":     "SUCCEEDED",
	}
	s.uploads[arn] = upload
	return upload
}

func (s *deviceFarmStore) ensureRunLocked(arn, projectARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.runs {
			if projectARN == "" || dfString(value, []string{"projectArn"}, "") == projectARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("run", s.nextIDLocked("run"))
	}
	if run, ok := s.runs[arn]; ok {
		return run
	}
	run := map[string]any{
		"arn":        arn,
		"name":       "stackyard-run",
		"projectArn": s.resolveProjectARNLocked(projectARN),
		"status":     "COMPLETED",
		"result":     "PASSED",
		"created":    now,
	}
	s.runs[arn] = run
	return run
}

func (s *deviceFarmStore) ensureSuiteLocked(arn, runARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.suites {
			if runARN == "" || dfString(value, []string{"runArn"}, "") == runARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("suite", s.nextIDLocked("suite"))
	}
	if suite, ok := s.suites[arn]; ok {
		return suite
	}
	suite := map[string]any{
		"arn":     arn,
		"name":    "stackyard-suite",
		"runArn":  runARN,
		"status":  "COMPLETED",
		"result":  "PASSED",
		"created": now,
	}
	s.suites[arn] = suite
	return suite
}

func (s *deviceFarmStore) ensureJobLocked(arn, runARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.jobs {
			if runARN == "" || dfString(value, []string{"runArn"}, "") == runARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("job", s.nextIDLocked("job"))
	}
	if job, ok := s.jobs[arn]; ok {
		return job
	}
	job := map[string]any{
		"arn":     arn,
		"name":    "stackyard-job",
		"runArn":  runARN,
		"status":  "COMPLETED",
		"result":  "PASSED",
		"created": now,
	}
	s.jobs[arn] = job
	return job
}

func (s *deviceFarmStore) ensureTestLocked(arn, runARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.tests {
			if runARN == "" || dfString(value, []string{"runArn"}, "") == runARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("test", s.nextIDLocked("test"))
	}
	if test := s.tests[arn]; test != nil {
		return test
	}
	test := map[string]any{
		"arn":     arn,
		"name":    "stackyard-test",
		"runArn":  runARN,
		"status":  "COMPLETED",
		"result":  "PASSED",
		"created": now,
	}
	s.tests[arn] = test
	return test
}

func (s *deviceFarmStore) ensureRemoteAccessSessionLocked(arn, projectARN, now string) map[string]any {
	if arn == "" {
		for _, value := range s.remoteAccessSessions {
			if projectARN == "" || dfString(value, []string{"projectArn"}, "") == projectARN {
				return value
			}
		}
	}
	if arn == "" {
		arn = dfARN("session", s.nextIDLocked("session"))
	}
	if session, ok := s.remoteAccessSessions[arn]; ok {
		return session
	}
	session := map[string]any{
		"arn":        arn,
		"projectArn": s.resolveProjectARNLocked(projectARN),
		"deviceArn":  dfARN("device", "device-000001"),
		"status":     "RUNNING",
		"created":    now,
	}
	s.remoteAccessSessions[arn] = session
	return session
}

func (s *deviceFarmStore) ensureInstanceProfileLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.instanceProfiles {
			return value
		}
		arn = dfARN("instanceprofile", s.nextIDLocked("ip"))
	}
	if profile, ok := s.instanceProfiles[arn]; ok {
		return profile
	}
	profile := map[string]any{
		"arn":                           arn,
		"name":                          "stackyard-instance-profile",
		"description":                   "Stackyard instance profile",
		"packageCleanup":                true,
		"excludeAppPackagesFromCleanup": []any{},
		"created":                       now,
	}
	s.instanceProfiles[arn] = profile
	return profile
}

func (s *deviceFarmStore) ensureNetworkProfileLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.networkProfiles {
			return value
		}
		arn = dfARN("networkprofile", s.nextIDLocked("np"))
	}
	if profile, ok := s.networkProfiles[arn]; ok {
		return profile
	}
	profile := map[string]any{
		"arn":                   arn,
		"name":                  "stackyard-network-profile",
		"description":           "Stackyard network profile",
		"type":                  "PRIVATE",
		"uplinkBandwidthBits":   10000000,
		"downlinkBandwidthBits": 10000000,
		"created":               now,
	}
	s.networkProfiles[arn] = profile
	return profile
}

func (s *deviceFarmStore) ensureVPCEConfigurationLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.vpceConfigurations {
			return value
		}
		arn = dfARN("vpceconfiguration", s.nextIDLocked("vpce"))
	}
	if config, ok := s.vpceConfigurations[arn]; ok {
		return config
	}
	config := map[string]any{
		"arn":             arn,
		"name":            "stackyard-vpce",
		"description":     "Stackyard VPCE configuration",
		"serviceDnsName":  "devicefarm.us-east-1.amazonaws.com",
		"vpceServiceName": "com.amazonaws.vpce.us-east-1.vpce-svc-000001",
		"created":         now,
	}
	s.vpceConfigurations[arn] = config
	return config
}

func (s *deviceFarmStore) ensureDeviceInstanceLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.deviceInstances {
			return value
		}
		arn = dfARN("deviceinstance", s.nextIDLocked("di"))
	}
	if instance, ok := s.deviceInstances[arn]; ok {
		return instance
	}
	instance := map[string]any{
		"arn":                arn,
		"deviceArn":          dfARN("device", "device-000001"),
		"instanceProfileArn": s.resolveInstanceProfileARNLocked(""),
		"status":             "AVAILABLE",
		"udid":               "device-udid-000001",
		"created":            now,
	}
	s.deviceInstances[arn] = instance
	return instance
}

func (s *deviceFarmStore) ensureTestGridProjectLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.testGridProjects {
			return value
		}
		arn = dfARN("testgrid-project", s.nextIDLocked("tgp"))
	}
	if project, ok := s.testGridProjects[arn]; ok {
		return project
	}
	project := map[string]any{
		"arn":         arn,
		"name":        "stackyard-testgrid-project",
		"description": "Stackyard TestGrid project",
		"created":     now,
	}
	s.testGridProjects[arn] = project
	return project
}

func (s *deviceFarmStore) ensureTestGridSessionLocked(arn, now string) map[string]any {
	if arn == "" {
		for _, value := range s.testGridSessions {
			return value
		}
		arn = dfARN("testgrid-session", s.nextIDLocked("tgs"))
	}
	if session, ok := s.testGridSessions[arn]; ok {
		return session
	}
	session := map[string]any{
		"arn":            arn,
		"projectArn":     s.resolveTestGridProjectARNLocked(""),
		"status":         "RUNNING",
		"created":        now,
		"endTime":        time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		"billingMinutes": 1,
	}
	s.testGridSessions[arn] = session
	return session
}

func (s *deviceFarmStore) ensureOfferingLocked(offeringID string) map[string]any {
	if offeringID == "" {
		offeringID = s.firstOfferingIDLocked()
	}
	if offering, ok := s.offerings[offeringID]; ok {
		return offering
	}
	offering := map[string]any{
		"id":          offeringID,
		"type":        "RECURRING",
		"description": "Stackyard offering",
		"platform":    "ANDROID",
		"recurringCharges": []any{
			map[string]any{
				"cost":      map[string]any{"amount": 1.0, "currencyCode": "USD"},
				"frequency": "MONTHLY",
			},
		},
	}
	s.offerings[offeringID] = offering
	return offering
}

func (s *deviceFarmStore) resolveProjectARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return s.firstProjectARNLocked()
	}
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "project-") {
		return dfARN("project", value)
	}
	return s.firstProjectARNLocked()
}

func (s *deviceFarmStore) resolveDevicePoolARNLocked(value, projectARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "pool-") {
		return dfARN("devicepool", value)
	}
	for key, item := range s.devicePools {
		if projectARN == "" || dfString(item, []string{"projectArn"}, "") == projectARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveUploadARNLocked(value, projectARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "upload-") {
		return dfARN("upload", value)
	}
	for key, item := range s.uploads {
		if projectARN == "" || dfString(item, []string{"projectArn"}, "") == projectARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveRunARNLocked(value, projectARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "run-") {
		return dfARN("run", value)
	}
	for key, item := range s.runs {
		if projectARN == "" || dfString(item, []string{"projectArn"}, "") == projectARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveJobARNLocked(value, runARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "job-") {
		return dfARN("job", value)
	}
	for key, item := range s.jobs {
		if runARN == "" || dfString(item, []string{"runArn"}, "") == runARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveSuiteARNLocked(value, runARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "suite-") {
		return dfARN("suite", value)
	}
	for key, item := range s.suites {
		if runARN == "" || dfString(item, []string{"runArn"}, "") == runARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveTestARNLocked(value, runARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "test-") {
		return dfARN("test", value)
	}
	for key, item := range s.tests {
		if runARN == "" || dfString(item, []string{"runArn"}, "") == runARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveRemoteAccessSessionARNLocked(value, projectARN string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "session-") {
		return dfARN("session", value)
	}
	for key, item := range s.remoteAccessSessions {
		if projectARN == "" || dfString(item, []string{"projectArn"}, "") == projectARN {
			return key
		}
	}
	return ""
}

func (s *deviceFarmStore) resolveInstanceProfileARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "ip-") {
		return dfARN("instanceprofile", value)
	}
	for key := range s.instanceProfiles {
		return key
	}
	return ""
}

func (s *deviceFarmStore) resolveNetworkProfileARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "np-") {
		return dfARN("networkprofile", value)
	}
	for key := range s.networkProfiles {
		return key
	}
	return ""
}

func (s *deviceFarmStore) resolveVPCEConfigurationARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "vpce-") {
		return dfARN("vpceconfiguration", value)
	}
	for key := range s.vpceConfigurations {
		return key
	}
	return ""
}

func (s *deviceFarmStore) resolveDeviceInstanceARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "di-") {
		return dfARN("deviceinstance", value)
	}
	for key := range s.deviceInstances {
		return key
	}
	return ""
}

func (s *deviceFarmStore) resolveTestGridProjectARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "tgp-") {
		return dfARN("testgrid-project", value)
	}
	for key := range s.testGridProjects {
		return key
	}
	return ""
}

func (s *deviceFarmStore) resolveTestGridSessionARNLocked(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:aws:devicefarm:") {
		return value
	}
	if strings.HasPrefix(value, "tgs-") {
		return dfARN("testgrid-session", value)
	}
	for key := range s.testGridSessions {
		return key
	}
	return ""
}

func (s *deviceFarmStore) firstProjectARNLocked() string {
	keys := make([]string, 0, len(s.projects))
	for key := range s.projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		now := time.Now().UTC().Format(time.RFC3339)
		project := s.ensureProjectLocked(dfARN("project", "project-000001"), now)
		return dfString(project, []string{"arn"}, dfARN("project", "project-000001"))
	}
	return keys[0]
}

func (s *deviceFarmStore) firstOfferingIDLocked() string {
	keys := make([]string, 0, len(s.offerings))
	for key := range s.offerings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		s.offerings["offering-000001"] = map[string]any{
			"id":          "offering-000001",
			"type":        "RECURRING",
			"description": "Stackyard offering",
		}
		return "offering-000001"
	}
	return keys[0]
}

func (s *deviceFarmStore) nextIDLocked(prefix string) string {
	id := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return id
}

func (s *deviceFarmStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = s.firstProjectARNLocked()
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *deviceFarmStore) listResourcesLocked(resources map[string]map[string]any) []any {
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, dfCloneMap(resources[key]))
	}
	return out
}

func (s *deviceFarmStore) listByProjectLocked(resources map[string]map[string]any, projectARN string) []any {
	projectARN = strings.TrimSpace(projectARN)
	out := make([]any, 0, len(resources))
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := resources[key]
		itemProjectARN := strings.TrimSpace(dfString(value, []string{"projectArn"}, ""))
		if projectARN != "" && projectARN != itemProjectARN {
			continue
		}
		out = append(out, dfCloneMap(value))
	}
	return out
}

func (s *deviceFarmStore) listByRunLocked(resources map[string]map[string]any, runARN string) []any {
	runARN = strings.TrimSpace(runARN)
	out := make([]any, 0, len(resources))
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := resources[key]
		itemRunARN := strings.TrimSpace(dfString(value, []string{"runArn"}, ""))
		if runARN != "" && runARN != itemRunARN {
			continue
		}
		out = append(out, dfCloneMap(value))
	}
	return out
}

func (s *deviceFarmStore) listOfferingsLocked() []any {
	keys := make([]string, 0, len(s.offerings))
	for key := range s.offerings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, dfCloneMap(s.offerings[key]))
	}
	return out
}

func (s *deviceFarmStore) devicePayload(deviceARN string) map[string]any {
	deviceARN = strings.TrimSpace(deviceARN)
	if deviceARN == "" {
		deviceARN = dfARN("device", "device-000001")
	}
	return map[string]any{
		"arn":          deviceARN,
		"name":         "stackyard-device",
		"manufacturer": "Stackyard",
		"model":        "stackyard-model",
		"os":           "14",
		"platform":     "ANDROID",
		"formFactor":   "PHONE",
		"availability": "AVAILABLE",
	}
}

func dfARN(resource, id string) string {
	resource = strings.TrimSpace(resource)
	id = strings.TrimSpace(id)
	if resource == "" {
		resource = "resource"
	}
	if id == "" {
		id = "resource-000001"
	}
	return fmt.Sprintf("arn:aws:devicefarm:%s:%s:%s:%s", deviceFarmDefaultRegion, deviceFarmDefaultAccountID, resource, id)
}

func dfIDFromARN(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	if !strings.HasPrefix(value, "arn:") {
		return value
	}
	if idx := strings.LastIndex(value, ":"); idx >= 0 && idx+1 < len(value) {
		return strings.TrimSpace(value[idx+1:])
	}
	return def
}

func dfString(payload map[string]any, keys []string, def string) string {
	if payload == nil {
		return def
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if s := strings.TrimSpace(typed); s != "" {
				return s
			}
		case fmt.Stringer:
			if s := strings.TrimSpace(typed.String()); s != "" {
				return s
			}
		}
	}
	return def
}

func dfStringSlice(payload map[string]any, keys []string) []string {
	if payload == nil {
		return nil
	}
	out := []string{}
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []string:
			for _, item := range typed {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
		case []any:
			for _, item := range typed {
				if str, ok := item.(string); ok {
					str = strings.TrimSpace(str)
					if str != "" {
						out = append(out, str)
					}
				}
			}
		case string:
			typed = strings.TrimSpace(typed)
			if typed != "" {
				out = append(out, typed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}

func dfTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	addFromMap := func(m map[string]any) {
		for key, value := range m {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	addFromSlice := func(items []any) {
		for _, item := range items {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := dfString(tag, []string{"Key", "key"}, "")
			if key == "" {
				continue
			}
			out[key] = dfString(tag, []string{"Value", "value"}, "")
		}
	}

	for _, key := range []string{"tags", "Tags"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			addFromMap(typed)
		case map[string]string:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k != "" {
					out[k] = strings.TrimSpace(v)
				}
			}
		case []any:
			addFromSlice(typed)
		}
	}
	return out
}

func dfCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func dfCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
