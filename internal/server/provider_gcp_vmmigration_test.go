package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVMMigrationRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	source := parent + "/sources/source-1"
	migratingVM := source + "/migratingVms/migrating-vm-1"

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/sources?pageSize=1", nil, "sources")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source, nil, "/sources/source-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/sources?sourceId=source-1", []byte(`{"source":{"description":"source"}}`), "operations/createSources.source-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPatch, source+"?updateMask=description", []byte(`{"source":{"name":"projects/stackyard/locations/us-central1/sources/source-1","description":"updated"}}`), "operations/update.sources.source-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodDelete, source, nil, "operations/delete.sources.source-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+":fetchInventory", nil, "vmwareVms")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+":fetchStorageInventory?pageSize=1", nil, "resources")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/utilizationReports?pageSize=1", nil, "utilizationReports")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/utilizationReports/report-1", nil, "/utilizationReports/report-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/utilizationReports?utilizationReportId=report-1", []byte(`{"utilizationReport":{"displayName":"report"}}`), "operations/createUtilizationReports.report-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/datacenterConnectors?pageSize=1", nil, "datacenterConnectors")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/datacenterConnectors/connector-1", nil, "/datacenterConnectors/connector-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/datacenterConnectors?datacenterConnectorId=connector-1", []byte(`{"datacenterConnector":{"name":"connector-1"}}`), "operations/createDatacenterConnectors.connector-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/datacenterConnectors/connector-1:upgradeAppliance", []byte(`{}`), "operations/upgradeAppliance.connector-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/migratingVms?pageSize=1", nil, "migratingVms")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM, nil, "/migratingVms/migrating-vm-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/migratingVms?migratingVmId=migrating-vm-1", []byte(`{"migratingVm":{"displayName":"vm"}}`), "operations/createMigratingVms.migrating-vm-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPatch, migratingVM+"?updateMask=display_name", []byte(`{"migratingVm":{"name":"projects/stackyard/locations/us-central1/sources/source-1/migratingVms/migrating-vm-1","displayName":"vm"}}`), "operations/update.migratingVms.migrating-vm-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+":startMigration", []byte(`{}`), "operations/startMigration.migrating-vm-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/migratingVms/migrating-vm-paused:resumeMigration", []byte(`{}`), "resumeMigration")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+":pauseMigration", []byte(`{}`), "pauseMigration")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+":finalizeMigration", []byte(`{}`), "finalizeMigration")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+":extendMigration", []byte(`{}`), "extendMigration")

	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+"/cloneJobs?cloneJobId=clone-job-1", []byte(`{"cloneJob":{"name":"clone"}}`), "createCloneJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/cloneJobs?pageSize=1", nil, "cloneJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/cloneJobs/clone-job-1", nil, "/cloneJobs/clone-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+"/cloneJobs/clone-job-1:cancel", []byte(`{}`), "operations/cancel.clone-job-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+"/cutoverJobs?cutoverJobId=cutover-job-1", []byte(`{"cutoverJob":{"name":"cutover"}}`), "createCutoverJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/cutoverJobs?pageSize=1", nil, "cutoverJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/cutoverJobs/cutover-job-1", nil, "/cutoverJobs/cutover-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, migratingVM+"/cutoverJobs/cutover-job-1:cancel", []byte(`{}`), "operations/cancel.cutover-job-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/groups?pageSize=1", nil, "groups")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/groups/group-1", nil, "/groups/group-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/groups?groupId=group-1", []byte(`{"group":{"description":"group"}}`), "createGroups")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPatch, parent+"/groups/group-1?updateMask=description", []byte(`{"group":{"name":"projects/stackyard/locations/us-central1/groups/group-1","description":"group"}}`), "update.groups.group-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/groups/group-1:addGroupMigration", []byte(`{}`), "addGroupMigration")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/groups/group-1:removeGroupMigration", []byte(`{}`), "removeGroupMigration")

	globalParent := "/gcp/v1/projects/stackyard/locations/global"
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, globalParent+"/targetProjects?pageSize=1", nil, "targetProjects")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, globalParent+"/targetProjects/target-project-1", nil, "/targetProjects/target-project-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, globalParent+"/targetProjects?targetProjectId=target-project-1", []byte(`{"targetProject":{"project":"target-project"}}`), "createTargetProjects")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/replicationCycles?pageSize=1", nil, "replicationCycles")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, migratingVM+"/replicationCycles/replication-cycle-1", nil, "/replicationCycles/replication-cycle-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/imageImports?pageSize=1", nil, "imageImports")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/imageImports/image-import-1", nil, "/imageImports/image-import-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/imageImports?imageImportId=image-import-1", []byte(`{"imageImport":{"displayName":"image"}}`), "createImageImports")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/imageImports/image-import-1/imageImportJobs?pageSize=1", nil, "imageImportJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/imageImports/image-import-1/imageImportJobs/image-import-job-1", nil, "/imageImportJobs/image-import-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/imageImports/image-import-1/imageImportJobs/image-import-job-1:cancel", []byte(`{}`), "operations/cancel.image-import-job-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/diskMigrationJobs?pageSize=1", nil, "diskMigrationJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, source+"/diskMigrationJobs/disk-migration-job-1", nil, "/diskMigrationJobs/disk-migration-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/diskMigrationJobs?diskMigrationJobId=disk-migration-job-1", []byte(`{"diskMigrationJob":{"displayName":"disk"}}`), "createDiskMigrationJobs")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPatch, source+"/diskMigrationJobs/disk-migration-job-1?updateMask=display_name", []byte(`{"diskMigrationJob":{"name":"projects/stackyard/locations/us-central1/sources/source-1/diskMigrationJobs/disk-migration-job-1","displayName":"disk"}}`), "update.diskMigrationJobs.disk-migration-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/diskMigrationJobs/disk-migration-job-1:run", []byte(`{}`), "operations/run.disk-migration-job-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, source+"/diskMigrationJobs/disk-migration-job-1:cancel", []byte(`{}`), "operations/cancel.disk-migration-job-1")

	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPVMMigrationSuccess(t, ts, http.MethodGet, parent+"/operations/vmmigration-op-1", nil, "operations/vmmigration-op-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, parent+"/operations/vmmigration-op-1:cancel", []byte(`{}`), `{}`)
	assertGCPVMMigrationSuccess(t, ts, http.MethodDelete, parent+"/operations/vmmigration-op-1", nil, `{}`)

	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmmigration.v1.VmMigration/ListSources", []byte(`{"parent":"projects/stackyard/locations/us-central1","pageSize":1}`), "sources")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmmigration.v1.VmMigration/GetSource", []byte(`{"name":"projects/stackyard/locations/us-central1/sources/source-1"}`), "/sources/source-1")
	assertGCPVMMigrationSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vmmigration.v1.VmMigration/CreateSource", []byte(`{"parent":"projects/stackyard/locations/us-central1","sourceId":"source-1","source":{"description":"source"}}`), "operations/create.sources.source-1")
}

func TestGCPVMMigrationRouter_ListSourcesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_CreateSourceRequiresSourceBody(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sources?sourceId=source-1", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration create source, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_UpdateSourceRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/sources/source-1", []byte(`{"source":{"name":"projects/stackyard/locations/us-central1/sources/source-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration update source, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_PauseMigrationAlreadyPaused(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sources/source-1/migratingVms/migrating-vm-paused:pauseMigration", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration pause migration, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_TargetProjectsRequireGlobalLocation(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/targetProjects?targetProjectId=target-project-1", []byte(`{"targetProject":{"project":"target-project"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration target projects create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_GRPCBridgeCreateSourceRequiresSource(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vmmigration.v1.VmMigration/CreateSource", []byte(`{"parent":"projects/stackyard/locations/us-central1","sourceId":"source-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vmmigration grpc bridge create source, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVMMigrationRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVMMigrationContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vmmigration",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration list sources, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	sources, ok := listBody["sources"].([]any)
	if !ok || len(sources) == 0 {
		t.Fatalf("expected sources array, got %#v", listBody["sources"])
	}
	source, _ := sources[0].(map[string]any)
	if _, ok := source["name"].(string); !ok {
		t.Fatalf("expected sources[0].name string, got %#v", source["name"])
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources/source-1/migratingVms/migrating-vm-1", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration get migrating vm, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["state"].(string); !ok {
		t.Fatalf("expected migratingVm state string, got %#v", getBody["state"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sources?sourceId=source-1", []byte(`{"source":{"description":"source"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration create source, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
	if _, ok := createBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", createBody["metadata"])
	}

	fetchStorageResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources/source-1:fetchStorageInventory", nil, headers)
	if fetchStorageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration fetch storage inventory, got %d body=%s", fetchStorageResp.StatusCode, string(providerContractBody(t, fetchStorageResp)))
	}
	fetchStorageBody := providerContractJSONMap(t, fetchStorageResp)
	if _, ok := fetchStorageBody["resources"].([]any); !ok {
		t.Fatalf("expected resources array, got %#v", fetchStorageBody["resources"])
	}
}

func TestGCPVMMigrationRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vmmigration?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "vmmigration" {
		t.Fatalf("expected service=vmmigration, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in probe response, got %#v", body["name"])
	}
}

func newGCPVMMigrationContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPVMMigrationSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vmmigration",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vmmigration router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
