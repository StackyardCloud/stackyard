package server

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apigeeconnectpb "cloud.google.com/go/apigeeconnect/apiv1/apigeeconnectpb"
	cloudprofilerpb "cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	cloudquotaspb "cloud.google.com/go/cloudquotas/apiv1/cloudquotaspb"
	procurementpb "cloud.google.com/go/commerce/consumer/procurement/apiv1/procurementpb"
	configpb "cloud.google.com/go/config/apiv1/configpb"
	configdeliverypb "cloud.google.com/go/configdelivery/apiv1/configdeliverypb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	mediatranslationpb "cloud.google.com/go/mediatranslation/apiv1beta1/mediatranslationpb"
	rapidmigrationassessmentpb "cloud.google.com/go/rapidmigrationassessment/apiv1/rapidmigrationassessmentpb"
	recaptchaenterprisepb "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
	recommendationenginepb "cloud.google.com/go/recommendationengine/apiv1beta1/recommendationenginepb"
	recommenderpb "cloud.google.com/go/recommender/apiv1/recommenderpb"
	redispb "cloud.google.com/go/redis/apiv1/redispb"
	clusterpb "cloud.google.com/go/redis/cluster/apiv1/clusterpb"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv2/resourcemanagerpb"
	resourcemanagerv3pb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	retailpb "cloud.google.com/go/retail/apiv2/retailpb"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	schedulerpb "cloud.google.com/go/scheduler/apiv1/schedulerpb"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	securesourcemanagerpb "cloud.google.com/go/securesourcemanager/apiv1/securesourcemanagerpb"
	privatecapb "cloud.google.com/go/security/privateca/apiv1/privatecapb"
	publiccapb "cloud.google.com/go/security/publicca/apiv1/publiccapb"
	securitycenterpb "cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	securitycenterv2pb "cloud.google.com/go/securitycenter/apiv2/securitycenterpb"
	securitycentermanagementpb "cloud.google.com/go/securitycentermanagement/apiv1/securitycentermanagementpb"
	securityposturepb "cloud.google.com/go/securityposture/apiv1/securityposturepb"
	servicecontrolpb "cloud.google.com/go/servicecontrol/apiv1/servicecontrolpb"
	servicedirectorypb "cloud.google.com/go/servicedirectory/apiv1/servicedirectorypb"
	servicehealthpb "cloud.google.com/go/servicehealth/apiv1/servicehealthpb"
	servicemanagementpb "cloud.google.com/go/servicemanagement/apiv1/servicemanagementpb"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	shellpb "cloud.google.com/go/shell/apiv1/shellpb"
	shoppingcsspb "cloud.google.com/go/shopping/css/apiv1/csspb"
	accountspb "cloud.google.com/go/shopping/merchant/accounts/apiv1/accountspb"
	conversionspb "cloud.google.com/go/shopping/merchant/conversions/apiv1/conversionspb"
	datasourcespb "cloud.google.com/go/shopping/merchant/datasources/apiv1/datasourcespb"
	inventoriespb "cloud.google.com/go/shopping/merchant/inventories/apiv1/inventoriespb"
	issueresolutionpb "cloud.google.com/go/shopping/merchant/issueresolution/apiv1/issueresolutionpb"
	lfppb "cloud.google.com/go/shopping/merchant/lfp/apiv1/lfppb"
	notificationspb "cloud.google.com/go/shopping/merchant/notifications/apiv1/notificationspb"
	ordertrackingpb "cloud.google.com/go/shopping/merchant/ordertracking/apiv1/ordertrackingpb"
	productspb "cloud.google.com/go/shopping/merchant/products/apiv1/productspb"
	productstudiopb "cloud.google.com/go/shopping/merchant/productstudio/apiv1alpha/productstudiopb"
	promotionspb "cloud.google.com/go/shopping/merchant/promotions/apiv1/promotionspb"
	quotapb "cloud.google.com/go/shopping/merchant/quota/apiv1/quotapb"
	reportspb "cloud.google.com/go/shopping/merchant/reports/apiv1/reportspb"
	reviewspb "cloud.google.com/go/shopping/merchant/reviews/apiv1beta/reviewspb"
	shoppingtypepb "cloud.google.com/go/shopping/type/typepb"
	adapterpb "cloud.google.com/go/spanner/adapter/apiv1/adapterpb"
	spanneradminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	spanneradmininstancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	spannerpb "cloud.google.com/go/spanner/apiv1/spannerpb"
	executorpb "cloud.google.com/go/spanner/executor/apiv1/executorpb"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	speechv2pb "cloud.google.com/go/speech/apiv2/speechpb"
	storagebatchoperationspb "cloud.google.com/go/storagebatchoperations/apiv1/storagebatchoperationspb"
	storageinsightspb "cloud.google.com/go/storageinsights/apiv1/storageinsightspb"
	storagetransferpb "cloud.google.com/go/storagetransfer/apiv1/storagetransferpb"
	publishpb "cloud.google.com/go/streetview/publish/apiv1/publishpb"
	supportpb "cloud.google.com/go/support/apiv2/supportpb"
	talentpb "cloud.google.com/go/talent/apiv4/talentpb"
	vmmigrationpb "cloud.google.com/go/vmmigration/apiv1/vmmigrationpb"
	vmwareenginepb "cloud.google.com/go/vmwareengine/apiv1/vmwareenginepb"
	vpcaccesspb "cloud.google.com/go/vpcaccess/apiv1/vpcaccesspb"
	webriskpb "cloud.google.com/go/webrisk/apiv1/webriskpb"
	websecurityscannerpb "cloud.google.com/go/websecurityscanner/apiv1/websecurityscannerpb"
	serviceconfigpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	datetimepb "google.golang.org/genproto/googleapis/type/datetime"
	intervalpb "google.golang.org/genproto/googleapis/type/interval"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGCPStage4GRPCParity_ApigeeConnect(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/endpoints/local/connections?pageSize=1", nil, nil)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest apigeeconnect list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["connections"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected connections list in rest payload, got %#v", restBody["connections"])
	}
	restConnection, _ := restItems[0].(map[string]any)
	restEndpointAny, _ := restConnection["endpoint"].(map[string]any)
	restEndpoint, _ := restEndpointAny["name"].(string)

	successReq := &apigeeconnectpb.ListConnectionsRequest{
		Parent:   "projects/stackyard/endpoints/local",
		PageSize: 1,
	}
	var successResp apigeeconnectpb.ListConnectionsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpApigeeConnectListConnectionsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetConnections()) != 1 {
		t.Fatalf("expected one grpc connection, got %d", len(successResp.GetConnections()))
	}
	if successResp.GetConnections()[0].GetEndpoint() != restEndpoint {
		t.Fatalf("expected grpc connection endpoint %q to match rest %q", successResp.GetConnections()[0].GetEndpoint(), restEndpoint)
	}

	invalidReq := &apigeeconnectpb.ListConnectionsRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpApigeeConnectListConnectionsMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for apigeeconnect list, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SpannerAdminDatabaseDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &spanneradminpb.ListDatabasesRequest{Parent: "projects/stackyard/instances/stackyard-instance"}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSpannerAdminDatabaseListDatabasesMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch spanner admin database list databases")
	}
}

func TestGCPStage4GRPCParity_SpannerAdminInstanceDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &spanneradmininstancepb.ListInstancesRequest{Parent: "projects/stackyard"}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSpannerAdminInstanceListInstancesMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch spanner admin instance list instances")
	}
}

func TestGCPStage4GRPCParity_SpannerExecutorDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &executorpb.SpannerAsyncActionRequest{
		ActionId: 1,
		Action: &executorpb.SpannerAction{
			DatabasePath: "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			Action: &executorpb.SpannerAction_Query{
				Query: &executorpb.QueryAction{
					Sql: "SELECT 1",
				},
			},
		},
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSpannerExecutorExecuteActionAsyncMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch spanner executor execute action async")
	}
}

func TestGCPStage4GRPCParity_VMwareEngineDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &vmwareenginepb.ListPrivateCloudsRequest{
		Parent: "projects/stackyard/locations/us-central1",
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpVMwareEngineListPrivateCloudsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch vmwareengine list private clouds")
	}
}

func TestGCPStage4GRPCParity_VPCAccessDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &vpcaccesspb.ListConnectorsRequest{
		Parent: "projects/stackyard/locations/us-central1",
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpVPCAccessListConnectorsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch vpcaccess list connectors")
	}
}

func TestGCPStage4GRPCParity_WebSecurityScannerDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &websecurityscannerpb.ListScanConfigsRequest{
		Parent: "projects/stackyard",
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpWebSecurityScannerListScanConfigsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch websecurityscanner list scan configs")
	}
}

func TestGCPStage4GRPCParity_SpeechDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			LanguageCode: "en-US",
			Encoding:     speechpb.RecognitionConfig_LINEAR16,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: []byte("stackyard"),
			},
		},
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSpeechRecognizeMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch speech recognize")
	}
}

func TestGCPStage4GRPCParity_SpeechV2DispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &speechv2pb.RecognizeRequest{
		Recognizer: "projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		AudioSource: &speechv2pb.RecognizeRequest_Content{
			Content: []byte("stackyard"),
		},
		Config: &speechv2pb.RecognitionConfig{
			LanguageCodes: []string{"en-US"},
		},
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSpeechV2RecognizeMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch speech v2 recognize")
	}
}

func TestGCPStage4GRPCParity_StorageBatchOperationsDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &storagebatchoperationspb.ListJobsRequest{
		Parent: "projects/stackyard/locations/global",
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpStorageBatchOperationsListJobsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch storagebatchoperations list jobs")
	}
}

func TestGCPStage4GRPCParity_StorageInsightsDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &storageinsightspb.ListReportConfigsRequest{
		Parent: "projects/stackyard/locations/us-central1",
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpStorageInsightsListReportConfigsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch storageinsights list report configs")
	}
}

func TestGCPStage4GRPCParity_StorageTransferDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &storagetransferpb.ListTransferJobsRequest{
		Filter: `{"projectId":"stackyard"}`,
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpStorageTransferListTransferJobsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch storagetransfer list transfer jobs")
	}
}

func TestGCPStage4GRPCParity_StreetViewPublishDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &publishpb.ListPhotosRequest{
		View: publishpb.PhotoView_BASIC,
	}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpStreetViewPublishListPhotosMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch streetview publish list photos")
	}
}

func TestGCPStage4GRPCParity_SpannerExecutor(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 11,
		"action": {
			"databasePath": "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			"read": {
				"table": "Users",
				"column": ["id", "name"],
				"keys": {"all": true}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner executor execute action async, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restActionID, _ := restBody["actionId"].(float64)
	restOutcome, _ := restBody["outcome"].(map[string]any)
	restReadResult, _ := restOutcome["readResult"].(map[string]any)
	restTable, _ := restReadResult["table"].(string)

	successReq := &executorpb.SpannerAsyncActionRequest{
		ActionId: int32(restActionID),
		Action: &executorpb.SpannerAction{
			DatabasePath: "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			Action: &executorpb.SpannerAction_Read{
				Read: &executorpb.ReadAction{
					Table:  "Users",
					Column: []string{"id", "name"},
					Keys:   &executorpb.KeySet{All: true},
				},
			},
		},
	}
	var successResp executorpb.SpannerAsyncActionResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpannerExecutorExecuteActionAsyncMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResp.GetActionId() != int32(restActionID) {
		t.Fatalf("expected grpc action id %d to match rest %.0f", successResp.GetActionId(), restActionID)
	}
	if successResp.GetOutcome() == nil || successResp.GetOutcome().GetReadResult() == nil {
		t.Fatalf("expected grpc read result in outcome, got %#v", successResp.GetOutcome())
	}
	if successResp.GetOutcome().GetReadResult().GetTable() != restTable {
		t.Fatalf("expected grpc read table %q to match rest %q", successResp.GetOutcome().GetReadResult().GetTable(), restTable)
	}

	invalidReq := &executorpb.SpannerAsyncActionRequest{
		ActionId: 0,
		Action: &executorpb.SpannerAction{
			Action: &executorpb.SpannerAction_Query{
				Query: &executorpb.QueryAction{Sql: "SELECT 1"},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerExecutorExecuteActionAsyncMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "action_id-required") {
		t.Fatalf("expected grpc invalid argument for spanner executor execute action async, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_MediaTranslation(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech", []byte(`{
		"streamingConfig":{
			"audioConfig":{
				"audioEncoding":"linear16",
				"sourceLanguageCode":"en-US",
				"targetLanguageCode":"es-ES",
				"sampleRateHertz":16000
			}
		}
	}`), map[string]string{
		"Content-Type": "application/json",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest mediatranslation streaming translate, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restResult, _ := restBody["result"].(map[string]any)
	restTextResult, _ := restResult["textTranslationResult"].(map[string]any)
	restTranslation, _ := restTextResult["translation"].(string)

	successReq := &mediatranslationpb.StreamingTranslateSpeechRequest{
		StreamingRequest: &mediatranslationpb.StreamingTranslateSpeechRequest_StreamingConfig{
			StreamingConfig: &mediatranslationpb.StreamingTranslateSpeechConfig{
				AudioConfig: &mediatranslationpb.TranslateSpeechConfig{
					AudioEncoding:      "linear16",
					SourceLanguageCode: "en-US",
					TargetLanguageCode: "es-ES",
					SampleRateHertz:    16000,
				},
				SingleUtterance: true,
			},
		},
	}
	var successResp mediatranslationpb.StreamingTranslateSpeechResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpMediaTranslationStreamingTranslateSpeechMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResp.GetResult() == nil || successResp.GetResult().GetTextTranslationResult() == nil {
		t.Fatalf("expected text translation result in grpc response, got %#v", successResp.GetResult())
	}
	if successResp.GetResult().GetTextTranslationResult().GetTranslation() != restTranslation {
		t.Fatalf("expected grpc translation %q to match rest %q", successResp.GetResult().GetTextTranslationResult().GetTranslation(), restTranslation)
	}

	invalidReq := &mediatranslationpb.StreamingTranslateSpeechRequest{
		StreamingRequest: &mediatranslationpb.StreamingTranslateSpeechRequest_AudioContent{
			AudioContent: []byte("stackyard"),
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpMediaTranslationStreamingTranslateSpeechMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "streaming_config-required") {
		t.Fatalf("expected grpc invalid argument for mediatranslation streaming translate, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_CloudProfiler(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/profiles?pageSize=1", nil, nil)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest cloudprofiler list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restProfiles, ok := restBody["profiles"].([]any)
	if !ok || len(restProfiles) == 0 {
		t.Fatalf("expected profiles list in rest payload, got %#v", restBody["profiles"])
	}
	restProfile, _ := restProfiles[0].(map[string]any)
	restName, _ := restProfile["name"].(string)

	successReq := &cloudprofilerpb.ListProfilesRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
	}
	var successResp cloudprofilerpb.ListProfilesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpCloudProfilerListProfilesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetProfiles()) != 1 {
		t.Fatalf("expected one grpc profile, got %d", len(successResp.GetProfiles()))
	}
	if successResp.GetProfiles()[0].GetName() != restName {
		t.Fatalf("expected grpc profile name %q to match rest %q", successResp.GetProfiles()[0].GetName(), restName)
	}

	invalidReq := &cloudprofilerpb.CreateProfileRequest{
		Deployment:  &cloudprofilerpb.Deployment{ProjectId: "stackyard", Target: "stackyard-service"},
		ProfileType: []cloudprofilerpb.ProfileType{cloudprofilerpb.ProfileType_CPU},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpCloudProfilerCreateProfileMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for cloudprofiler create, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_CloudQuotas(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/services/compute.googleapis.com/quotaInfos?pageSize=1", nil, nil)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest cloudquotas list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["quotaInfos"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected quotaInfos list in rest payload, got %#v", restBody["quotaInfos"])
	}
	restItem, _ := restItems[0].(map[string]any)
	restName, _ := restItem["name"].(string)

	successReq := &cloudquotaspb.ListQuotaInfosRequest{
		Parent:   "projects/stackyard/locations/global/services/compute.googleapis.com",
		PageSize: 1,
	}
	var successResp cloudquotaspb.ListQuotaInfosResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpCloudQuotasListQuotaInfosMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetQuotaInfos()) != 1 {
		t.Fatalf("expected one grpc quotaInfo, got %d", len(successResp.GetQuotaInfos()))
	}
	if successResp.GetQuotaInfos()[0].GetName() != restName {
		t.Fatalf("expected grpc quotaInfo name %q to match rest %q", successResp.GetQuotaInfos()[0].GetName(), restName)
	}

	invalidReq := &cloudquotaspb.CreateQuotaPreferenceRequest{
		Parent: "projects/stackyard/locations/global",
		QuotaPreference: &cloudquotaspb.QuotaPreference{
			QuotaConfig: &cloudquotaspb.QuotaConfig{PreferredValue: 16},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpCloudQuotasCreateQuotaPreferenceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "service-and-quota_id-required") {
		t.Fatalf("expected grpc invalid argument for cloudquotas create, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Procurement(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/billingAccounts/0123456789/orders?pageSize=1", nil, nil)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest procurement list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["orders"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected orders list in rest payload, got %#v", restBody["orders"])
	}
	restOrder, _ := restItems[0].(map[string]any)
	restName, _ := restOrder["name"].(string)

	successReq := &procurementpb.ListOrdersRequest{
		Parent:   "billingAccounts/0123456789",
		PageSize: 1,
	}
	var successResp procurementpb.ListOrdersResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpProcurementListOrdersMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetOrders()) != 1 {
		t.Fatalf("expected one grpc order, got %d", len(successResp.GetOrders()))
	}
	if successResp.GetOrders()[0].GetName() != restName {
		t.Fatalf("expected grpc order name %q to match rest %q", successResp.GetOrders()[0].GetName(), restName)
	}

	invalidReq := &procurementpb.PlaceOrderRequest{
		Parent: "billingAccounts/0123456789",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpProcurementPlaceOrderMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "display_name-required") {
		t.Fatalf("expected grpc invalid argument for procurement place order, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ConfigDelivery(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/resourceBundles?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "configdelivery",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest configdelivery list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["resourceBundles"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected resourceBundles list in rest payload, got %#v", restBody["resourceBundles"])
	}
	restBundle, _ := restItems[0].(map[string]any)
	restName, _ := restBundle["name"].(string)

	successReq := &configdeliverypb.ListResourceBundlesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp configdeliverypb.ListResourceBundlesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpConfigDeliveryListResourceBundlesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetResourceBundles()) != 1 {
		t.Fatalf("expected one grpc resource bundle, got %d", len(successResp.GetResourceBundles()))
	}
	if successResp.GetResourceBundles()[0].GetName() != restName {
		t.Fatalf("expected grpc resource bundle name %q to match rest %q", successResp.GetResourceBundles()[0].GetName(), restName)
	}

	invalidReq := &configdeliverypb.SuspendRolloutRequest{
		Name: "projects/stackyard/locations/us-central1/fleetPackages/platform-package/rollouts/rollout-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpConfigDeliverySuspendRolloutMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "reason-required") {
		t.Fatalf("expected grpc invalid argument for configdelivery suspend rollout, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Config(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/deployments/platform-foundation", nil, nil)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest config get deployment, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restName, _ := restBody["name"].(string)

	successReq := &configpb.GetDeploymentRequest{
		Name: restName,
	}
	var successResp configpb.Deployment
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpConfigGetDeploymentMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResp.GetName() != restName {
		t.Fatalf("expected grpc deployment name %q to match rest %q", successResp.GetName(), restName)
	}

	invalidReq := &configpb.CreateDeploymentRequest{
		Parent:       "projects/stackyard/locations/us-central1",
		DeploymentId: "platform-foundation",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpConfigCreateDeploymentMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "deployment-required") {
		t.Fatalf("expected grpc invalid argument for config create deployment, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_RapidMigrationAssessment(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/collectors?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest rapidmigrationassessment list collectors, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["collectors"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected collectors list in rest payload, got %#v", restBody["collectors"])
	}
	restCollector, _ := restItems[0].(map[string]any)
	restName, _ := restCollector["name"].(string)

	successReq := &rapidmigrationassessmentpb.ListCollectorsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp rapidmigrationassessmentpb.ListCollectorsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRapidMigrationAssessmentListCollectorsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetCollectors()) != 1 {
		t.Fatalf("expected one grpc collector, got %d", len(successResp.GetCollectors()))
	}
	if successResp.GetCollectors()[0].GetName() != restName {
		t.Fatalf("expected grpc collector name %q to match rest %q", successResp.GetCollectors()[0].GetName(), restName)
	}

	invalidReq := &rapidmigrationassessmentpb.CreateCollectorRequest{
		Parent:      "projects/stackyard/locations/us-central1",
		CollectorId: "collector-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRapidMigrationAssessmentCreateCollectorMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "collector-required") {
		t.Fatalf("expected grpc invalid argument for rapidmigrationassessment create collector, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &rapidmigrationassessmentpb.PauseCollectorRequest{
		Name: "projects/stackyard/locations/us-central1/collectors/collector-1-paused",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRapidMigrationAssessmentPauseCollectorMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "collector-already-paused") {
		t.Fatalf("expected grpc failed precondition for rapidmigrationassessment pause collector, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_VMMigration(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sources?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vmmigration",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vmmigration list sources, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["sources"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected sources list in rest payload, got %#v", restBody["sources"])
	}
	restSource, _ := restItems[0].(map[string]any)
	restName, _ := restSource["name"].(string)

	successReq := &vmmigrationpb.ListSourcesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp vmmigrationpb.ListSourcesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVMMigrationListSourcesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetSources()) != 1 {
		t.Fatalf("expected one grpc source, got %d", len(successResp.GetSources()))
	}
	if successResp.GetSources()[0].GetName() != restName {
		t.Fatalf("expected grpc source name %q to match rest %q", successResp.GetSources()[0].GetName(), restName)
	}

	invalidReq := &vmmigrationpb.CreateSourceRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		SourceId: "source-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVMMigrationCreateSourceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "source-required") {
		t.Fatalf("expected grpc invalid argument for vmmigration create source, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &vmmigrationpb.PauseMigrationRequest{
		MigratingVm: "projects/stackyard/locations/us-central1/sources/source-1/migratingVms/migrating-vm-paused",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVMMigrationPauseMigrationMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "migration-already-paused") {
		t.Fatalf("expected grpc failed precondition for vmmigration pause migration, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_VMwareEngine(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateClouds?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vmwareengine",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vmwareengine list private clouds, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["privateClouds"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected privateClouds list in rest payload, got %#v", restBody["privateClouds"])
	}
	restPrivateCloud, _ := restItems[0].(map[string]any)
	restName, _ := restPrivateCloud["name"].(string)

	successReq := &vmwareenginepb.ListPrivateCloudsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp vmwareenginepb.ListPrivateCloudsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVMwareEngineListPrivateCloudsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetPrivateClouds()) != 1 {
		t.Fatalf("expected one grpc private cloud, got %d", len(successResp.GetPrivateClouds()))
	}
	if successResp.GetPrivateClouds()[0].GetName() != restName {
		t.Fatalf("expected grpc private cloud name %q to match rest %q", successResp.GetPrivateClouds()[0].GetName(), restName)
	}

	invalidReq := &vmwareenginepb.CreatePrivateCloudRequest{
		Parent:         "projects/stackyard/locations/us-central1",
		PrivateCloudId: "private-cloud-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVMwareEngineCreatePrivateCloudMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "private_cloud-required") {
		t.Fatalf("expected grpc invalid argument for vmwareengine create private cloud, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &vmwareenginepb.GetPrivateCloudRequest{
		Name: "projects/stackyard/locations/us-central1/privateClouds/missing-private-cloud",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVMwareEngineGetPrivateCloudMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "private_cloud-not-found") {
		t.Fatalf("expected grpc not found for vmwareengine get private cloud, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_VPCAccess(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vpcaccess list connectors, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["connectors"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected connectors list in rest payload, got %#v", restBody["connectors"])
	}
	restConnector, _ := restItems[0].(map[string]any)
	restName, _ := restConnector["name"].(string)

	successReq := &vpcaccesspb.ListConnectorsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp vpcaccesspb.ListConnectorsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVPCAccessListConnectorsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetConnectors()) != 1 {
		t.Fatalf("expected one grpc connector, got %d", len(successResp.GetConnectors()))
	}
	if successResp.GetConnectors()[0].GetName() != restName {
		t.Fatalf("expected grpc connector name %q to match rest %q", successResp.GetConnectors()[0].GetName(), restName)
	}

	invalidReq := &vpcaccesspb.CreateConnectorRequest{
		Parent:      "projects/stackyard/locations/us-central1",
		ConnectorId: "connector-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVPCAccessCreateConnectorMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "connector-required") {
		t.Fatalf("expected grpc invalid argument for vpcaccess create connector, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &vpcaccesspb.GetConnectorRequest{
		Name: "projects/stackyard/locations/us-central1/connectors/missing-connector",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVPCAccessGetConnectorMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "connector-not-found") {
		t.Fatalf("expected grpc not found for vpcaccess get connector, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_WebRisk(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/uris:search", []byte(`{
		"uri": "http://phish.stackyard.test/path",
		"threatTypes": [2]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest webrisk search uris, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restThreat, ok := restBody["threat"].(map[string]any)
	if !ok {
		t.Fatalf("expected threat object in rest payload, got %#v", restBody["threat"])
	}
	restThreatTypes, ok := restThreat["threatTypes"].([]any)
	if !ok || len(restThreatTypes) == 0 {
		t.Fatalf("expected threatTypes array in rest payload, got %#v", restThreat["threatTypes"])
	}

	successReq := &webriskpb.SearchUrisRequest{
		Uri:         "http://phish.stackyard.test/path",
		ThreatTypes: []webriskpb.ThreatType{webriskpb.ThreatType_SOCIAL_ENGINEERING},
	}
	var successResp webriskpb.SearchUrisResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpWebRiskSearchUrisMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResp.GetThreat() == nil {
		t.Fatalf("expected grpc threat object")
	}
	if len(successResp.GetThreat().GetThreatTypes()) != len(restThreatTypes) {
		t.Fatalf("expected grpc threatTypes length %d to match rest %d", len(successResp.GetThreat().GetThreatTypes()), len(restThreatTypes))
	}

	successSubmitReq := &webriskpb.SubmitUriRequest{
		Parent: "projects/123456789",
		Submission: &webriskpb.Submission{
			Uri: "http://phish.stackyard.test/path",
		},
	}
	var successSubmitResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWebRiskSubmitURIMethod, successSubmitReq, &successSubmitResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for submit uri, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(successSubmitResp.GetName(), "projects/123456789/operations/") {
		t.Fatalf("expected operation name in submit uri response, got %q", successSubmitResp.GetName())
	}

	invalidSearchReq := &webriskpb.SearchHashesRequest{
		HashPrefix:  []byte{0x01, 0x02, 0x03},
		ThreatTypes: []webriskpb.ThreatType{webriskpb.ThreatType_MALWARE},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWebRiskSearchHashesMethod, invalidSearchReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "hash_prefix-length-invalid") {
		t.Fatalf("expected grpc invalid argument for webrisk search hashes, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidSubmitReq := &webriskpb.SubmitUriRequest{
		Parent: "projects/123456789",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWebRiskSubmitURIMethod, invalidSubmitReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "submission-required") {
		t.Fatalf("expected grpc invalid argument for webrisk submit uri, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_WebSecurityScanner(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest websecurityscanner list scan configs, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["scanConfigs"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected scanConfigs list in rest payload, got %#v", restBody["scanConfigs"])
	}
	restScanConfig, _ := restItems[0].(map[string]any)
	restName, _ := restScanConfig["name"].(string)

	successReq := &websecurityscannerpb.ListScanConfigsRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
	}
	var successResp websecurityscannerpb.ListScanConfigsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpWebSecurityScannerListScanConfigsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetScanConfigs()) != 1 {
		t.Fatalf("expected one grpc scan config, got %d", len(successResp.GetScanConfigs()))
	}
	if successResp.GetScanConfigs()[0].GetName() != restName {
		t.Fatalf("expected grpc scan config name %q to match rest %q", successResp.GetScanConfigs()[0].GetName(), restName)
	}

	invalidReq := &websecurityscannerpb.CreateScanConfigRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWebSecurityScannerCreateScanConfigMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "scan_config-required") {
		t.Fatalf("expected grpc invalid argument for websecurityscanner create scan config, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &websecurityscannerpb.GetScanConfigRequest{
		Name: "projects/stackyard/scanConfigs/missing-scan-config",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWebSecurityScannerGetScanConfigMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "scan_config-not-found") {
		t.Fatalf("expected grpc not found for websecurityscanner get scan config, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ResourceManager(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/folders?parent=organizations/123456&pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest resourcemanager list folders, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["folders"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected folders list in rest payload, got %#v", restBody["folders"])
	}
	restFolder, _ := restItems[0].(map[string]any)
	restName, _ := restFolder["name"].(string)

	successReq := &resourcemanagerpb.ListFoldersRequest{
		Parent:   "organizations/123456",
		PageSize: 1,
	}
	var successResp resourcemanagerpb.ListFoldersResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerListFoldersMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetFolders()) != 1 {
		t.Fatalf("expected one grpc folder, got %d", len(successResp.GetFolders()))
	}
	if successResp.GetFolders()[0].GetName() != restName {
		t.Fatalf("expected grpc folder name %q to match rest %q", successResp.GetFolders()[0].GetName(), restName)
	}

	invalidReq := &resourcemanagerpb.CreateFolderRequest{
		Parent: "organizations/123456",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerCreateFolderMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "folder-required") {
		t.Fatalf("expected grpc invalid argument for resourcemanager create folder, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &resourcemanagerpb.UndeleteFolderRequest{
		Name: "folders/active-folder",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerUndeleteFolderMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "folder-not-delete-requested") {
		t.Fatalf("expected grpc failed precondition for resourcemanager undelete folder, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ResourceManagerV3(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects?parent=organizations/123456&pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest resourcemanager v3 list projects, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["projects"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected projects list in rest payload, got %#v", restBody["projects"])
	}
	restProject, _ := restItems[0].(map[string]any)
	restName, _ := restProject["name"].(string)

	successReq := &resourcemanagerv3pb.ListProjectsRequest{
		Parent:   "organizations/123456",
		PageSize: 1,
	}
	var successResp resourcemanagerv3pb.ListProjectsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerV3ListProjectsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetProjects()) != 1 {
		t.Fatalf("expected one grpc project, got %d", len(successResp.GetProjects()))
	}
	if successResp.GetProjects()[0].GetName() != restName {
		t.Fatalf("expected grpc project name %q to match rest %q", successResp.GetProjects()[0].GetName(), restName)
	}

	invalidReq := &resourcemanagerv3pb.CreateProjectRequest{}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerV3CreateProjectMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "project-required") {
		t.Fatalf("expected grpc invalid argument for resourcemanager v3 create project, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &resourcemanagerv3pb.UndeleteProjectRequest{
		Name: "projects/active-project",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpResourceManagerV3UndeleteProjectMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "project-not-delete-requested") {
		t.Fatalf("expected grpc failed precondition for resourcemanager v3 undelete project, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_RedisCluster(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "redis_cluster",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest redis_cluster list clusters, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["clusters"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected clusters list in rest payload, got %#v", restBody["clusters"])
	}
	restCluster, _ := restItems[0].(map[string]any)
	restName, _ := restCluster["name"].(string)

	successReq := &clusterpb.ListClustersRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp clusterpb.ListClustersResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRedisClusterListClustersMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetClusters()) != 1 {
		t.Fatalf("expected one grpc cluster, got %d", len(successResp.GetClusters()))
	}
	if successResp.GetClusters()[0].GetName() != restName {
		t.Fatalf("expected grpc cluster name %q to match rest %q", successResp.GetClusters()[0].GetName(), restName)
	}

	invalidReq := &clusterpb.CreateClusterRequest{
		Parent:    "projects/stackyard/locations/us-central1",
		ClusterId: "cluster-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRedisClusterCreateClusterMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "cluster-required") {
		t.Fatalf("expected grpc invalid argument for redis_cluster create cluster, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &clusterpb.RescheduleClusterMaintenanceRequest{
		Name:           "projects/stackyard/locations/us-central1/clusters/cluster-locked",
		RescheduleType: clusterpb.RescheduleClusterMaintenanceRequest_IMMEDIATE,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRedisClusterRescheduleClusterMaintenanceMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "cluster-maintenance-locked") {
		t.Fatalf("expected grpc failed precondition for redis_cluster reschedule, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Redis(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "redis",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest redis list instances, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["instances"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected instances list in rest payload, got %#v", restBody["instances"])
	}
	restInstance, _ := restItems[0].(map[string]any)
	restName, _ := restInstance["name"].(string)

	successReq := &redispb.ListInstancesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp redispb.ListInstancesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRedisListInstancesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetInstances()) != 1 {
		t.Fatalf("expected one grpc instance, got %d", len(successResp.GetInstances()))
	}
	if successResp.GetInstances()[0].GetName() != restName {
		t.Fatalf("expected grpc instance name %q to match rest %q", successResp.GetInstances()[0].GetName(), restName)
	}

	invalidReq := &redispb.CreateInstanceRequest{
		Parent:     "projects/stackyard/locations/us-central1",
		InstanceId: "redis-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRedisCreateInstanceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "instance-required") {
		t.Fatalf("expected grpc invalid argument for redis create instance, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &redispb.FailoverInstanceRequest{
		Name: "projects/stackyard/locations/us-central1/instances/redis-basic",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRedisFailoverInstanceMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "instance-tier-basic") {
		t.Fatalf("expected grpc failed precondition for redis failover, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_RecaptchaEnterprise(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest recaptchaenterprise list keys, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["keys"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected keys list in rest payload, got %#v", restBody["keys"])
	}
	restKey, _ := restItems[0].(map[string]any)
	restName, _ := restKey["name"].(string)

	successReq := &recaptchaenterprisepb.ListKeysRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
	}
	var successResp recaptchaenterprisepb.ListKeysResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRecaptchaEnterpriseListKeysMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetKeys()) != 1 {
		t.Fatalf("expected one grpc key, got %d", len(successResp.GetKeys()))
	}
	if successResp.GetKeys()[0].GetName() != restName {
		t.Fatalf("expected grpc key name %q to match rest %q", successResp.GetKeys()[0].GetName(), restName)
	}

	invalidReq := &recaptchaenterprisepb.CreateAssessmentRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRecaptchaEnterpriseCreateAssessmentMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "assessment.event-required") {
		t.Fatalf("expected grpc invalid argument for recaptchaenterprise create assessment, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_RecommendationEngine(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/catalogItems?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest recommendationengine list catalog items, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["catalogItems"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected catalogItems list in rest payload, got %#v", restBody["catalogItems"])
	}
	restItem, _ := restItems[0].(map[string]any)
	restID, _ := restItem["id"].(string)

	successReq := &recommendationenginepb.ListCatalogItemsRequest{
		Parent:   "projects/stackyard/locations/global/catalogs/default_catalog",
		PageSize: 1,
	}
	var successResp recommendationenginepb.ListCatalogItemsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRecommendationEngineListCatalogItemsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetCatalogItems()) != 1 {
		t.Fatalf("expected one grpc catalog item, got %d", len(successResp.GetCatalogItems()))
	}
	if successResp.GetCatalogItems()[0].GetId() != restID {
		t.Fatalf("expected grpc catalog item id %q to match rest %q", successResp.GetCatalogItems()[0].GetId(), restID)
	}

	invalidReq := &recommendationenginepb.CreateCatalogItemRequest{
		Parent: "projects/stackyard/locations/global/catalogs/default_catalog",
		CatalogItem: &recommendationenginepb.CatalogItem{
			Id: "",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRecommendationEngineCreateCatalogItemMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "catalog_item.id-and-title-required") {
		t.Fatalf("expected grpc invalid argument for recommendationengine create catalog item, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Recommender(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight/insights?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recommender",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest recommender list insights, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["insights"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected insights list in rest payload, got %#v", restBody["insights"])
	}
	restInsight, _ := restItems[0].(map[string]any)
	restName, _ := restInsight["name"].(string)

	successReq := &recommenderpb.ListInsightsRequest{
		Parent:   "projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight",
		PageSize: 1,
	}
	var successResp recommenderpb.ListInsightsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRecommenderListInsightsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetInsights()) != 1 {
		t.Fatalf("expected one grpc insight, got %d", len(successResp.GetInsights()))
	}
	if successResp.GetInsights()[0].GetName() != restName {
		t.Fatalf("expected grpc insight name %q to match rest %q", successResp.GetInsights()[0].GetName(), restName)
	}

	invalidReq := &recommenderpb.MarkRecommendationDismissedRequest{
		Name: "projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRecommenderMarkRecommendationDismissedMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "etag-required") {
		t.Fatalf("expected grpc invalid argument for recommender mark recommendation dismissed, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &recommenderpb.MarkInsightAcceptedRequest{
		Name: "projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight/insights/insight-accepted",
		Etag: "etag-insight-accepted",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRecommenderMarkInsightAcceptedMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "insight-already-accepted") {
		t.Fatalf("expected grpc failed precondition for recommender mark insight accepted, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Run(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "run",
	}

	restServicesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/services?pageSize=1", nil, headers)
	if restServicesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest run list services, got %d body=%s", restServicesResp.StatusCode, string(providerContractBody(t, restServicesResp)))
	}
	restServicesBody := providerContractJSONMap(t, restServicesResp)
	restServices, ok := restServicesBody["services"].([]any)
	if !ok || len(restServices) == 0 {
		t.Fatalf("expected services list in rest payload, got %#v", restServicesBody["services"])
	}
	restService, _ := restServices[0].(map[string]any)
	restServiceName, _ := restService["name"].(string)

	successListServicesReq := &runpb.ListServicesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListServicesResp runpb.ListServicesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRunListServicesMethod, successListServicesReq, &successListServicesResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListServicesResp.GetServices()) != 1 {
		t.Fatalf("expected one grpc service, got %d", len(successListServicesResp.GetServices()))
	}
	if successListServicesResp.GetServices()[0].GetName() != restServiceName {
		t.Fatalf("expected grpc service name %q to match rest %q", successListServicesResp.GetServices()[0].GetName(), restServiceName)
	}

	restJobsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/jobs?pageSize=1", nil, headers)
	if restJobsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest run list jobs, got %d body=%s", restJobsResp.StatusCode, string(providerContractBody(t, restJobsResp)))
	}
	restJobsBody := providerContractJSONMap(t, restJobsResp)
	restJobs, ok := restJobsBody["jobs"].([]any)
	if !ok || len(restJobs) == 0 {
		t.Fatalf("expected jobs list in rest payload, got %#v", restJobsBody["jobs"])
	}
	restJob, _ := restJobs[0].(map[string]any)
	restJobName, _ := restJob["name"].(string)

	successGetJobReq := &runpb.GetJobRequest{Name: restJobName}
	var successGetJobResp runpb.Job
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunGetJobMethod, successGetJobReq, &successGetJobResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetJobResp.GetName() != restJobName {
		t.Fatalf("expected grpc job name %q to match rest %q", successGetJobResp.GetName(), restJobName)
	}

	successListExecutionsReq := &runpb.ListExecutionsRequest{
		Parent:   "projects/stackyard/locations/us-central1/jobs/job-1",
		PageSize: 1,
	}
	var successListExecutionsResp runpb.ListExecutionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunListExecutionsMethod, successListExecutionsReq, &successListExecutionsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListExecutionsResp.GetExecutions()) != 1 {
		t.Fatalf("expected one grpc execution, got %d", len(successListExecutionsResp.GetExecutions()))
	}

	successGetExecutionReq := &runpb.GetExecutionRequest{
		Name: "projects/stackyard/locations/us-central1/jobs/job-1/executions/execution-1",
	}
	var successGetExecutionResp runpb.Execution
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunGetExecutionMethod, successGetExecutionReq, &successGetExecutionResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetExecutionResp.GetName() == "" {
		t.Fatalf("expected grpc execution name to be set")
	}

	successListTasksReq := &runpb.ListTasksRequest{
		Parent:   "projects/stackyard/locations/us-central1/jobs/job-1/executions/execution-1",
		PageSize: 1,
	}
	var successListTasksResp runpb.ListTasksResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunListTasksMethod, successListTasksReq, &successListTasksResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListTasksResp.GetTasks()) != 1 {
		t.Fatalf("expected one grpc task, got %d", len(successListTasksResp.GetTasks()))
	}

	successGetTaskReq := &runpb.GetTaskRequest{
		Name: "projects/stackyard/locations/us-central1/jobs/job-1/executions/execution-1/tasks/task-1",
	}
	var successGetTaskResp runpb.Task
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunGetTaskMethod, successGetTaskReq, &successGetTaskResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetTaskResp.GetName() == "" {
		t.Fatalf("expected grpc task name to be set")
	}

	successListRevisionsReq := &runpb.ListRevisionsRequest{
		Parent:   "projects/stackyard/locations/us-central1/services/service-1",
		PageSize: 1,
	}
	var successListRevisionsResp runpb.ListRevisionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunListRevisionsMethod, successListRevisionsReq, &successListRevisionsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListRevisionsResp.GetRevisions()) != 1 {
		t.Fatalf("expected one grpc revision, got %d", len(successListRevisionsResp.GetRevisions()))
	}

	successGetRevisionReq := &runpb.GetRevisionRequest{
		Name: "projects/stackyard/locations/us-central1/services/service-1/revisions/service-1-00001",
	}
	var successGetRevisionResp runpb.Revision
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunGetRevisionMethod, successGetRevisionReq, &successGetRevisionResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetRevisionResp.GetName() == "" {
		t.Fatalf("expected grpc revision name to be set")
	}

	invalidCreateServiceReq := &runpb.CreateServiceRequest{
		Parent:    "projects/stackyard/locations/us-central1",
		ServiceId: "service-1",
		Service: &runpb.Service{
			Template: &runpb.RevisionTemplate{},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunCreateServiceMethod, invalidCreateServiceReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "service.template.containers-required") {
		t.Fatalf("expected grpc invalid argument for run create service, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionRunJobReq := &runpb.RunJobRequest{
		Name: "projects/stackyard/locations/us-central1/jobs/job-1",
		Etag: "stale",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRunRunJobMethod, preconditionRunJobReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "etag-mismatch") {
		t.Fatalf("expected grpc failed precondition for run run job, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Scheduler(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "scheduler",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest scheduler list jobs, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["jobs"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected jobs list in rest payload, got %#v", restBody["jobs"])
	}
	restJob, _ := restItems[0].(map[string]any)
	restName, _ := restJob["name"].(string)

	successReq := &schedulerpb.ListJobsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp schedulerpb.ListJobsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSchedulerListJobsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetJobs()) != 1 {
		t.Fatalf("expected one grpc job, got %d", len(successResp.GetJobs()))
	}
	if successResp.GetJobs()[0].GetName() != restName {
		t.Fatalf("expected grpc job name %q to match rest %q", successResp.GetJobs()[0].GetName(), restName)
	}

	getReq := &schedulerpb.GetJobRequest{Name: restName}
	var getResp schedulerpb.Job
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSchedulerGetJobMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restName {
		t.Fatalf("expected grpc get job name %q to match rest %q", getResp.GetName(), restName)
	}

	invalidReq := &schedulerpb.CreateJobRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Job: &schedulerpb.Job{
			TimeZone: "UTC",
			Target: &schedulerpb.Job_HttpTarget{
				HttpTarget: &schedulerpb.HttpTarget{Uri: "https://example.com/hook"},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSchedulerCreateJobMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "job.schedule-required") {
		t.Fatalf("expected grpc invalid argument for scheduler create job, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &schedulerpb.ResumeJobRequest{
		Name: "projects/stackyard/locations/us-central1/jobs/job-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSchedulerResumeJobMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "job-not-paused") {
		t.Fatalf("expected grpc failed precondition for scheduler resume job, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_TalentDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &talentpb.ListTenantsRequest{Parent: "projects/stackyard"}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpTalentListTenantsMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch talent list tenants")
	}
}

func TestGCPStage4GRPCParity_Talent(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "talent",
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v4/projects/stackyard/tenants?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest talent list tenants, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restTenants, ok := restListBody["tenants"].([]any)
	if !ok || len(restTenants) == 0 {
		t.Fatalf("expected tenants list in rest payload, got %#v", restListBody["tenants"])
	}
	restTenant, _ := restTenants[0].(map[string]any)
	restTenantName, _ := restTenant["name"].(string)

	successListReq := &talentpb.ListTenantsRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
	}
	var successListResp talentpb.ListTenantsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTalentListTenantsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetTenants()) != 1 {
		t.Fatalf("expected one grpc tenant, got %d", len(successListResp.GetTenants()))
	}
	if successListResp.GetTenants()[0].GetName() != restTenantName {
		t.Fatalf("expected grpc tenant name %q to match rest %q", successListResp.GetTenants()[0].GetName(), restTenantName)
	}

	successGetReq := &talentpb.GetTenantRequest{Name: restTenantName}
	var successGetResp talentpb.Tenant
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTalentGetTenantMethod, successGetReq, &successGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetResp.GetName() != restTenantName {
		t.Fatalf("expected grpc tenant name %q to match rest %q", successGetResp.GetName(), restTenantName)
	}

	restSearchReqBody := []byte(`{"parent":"projects/stackyard/tenants/tenant-1","requestMetadata":{"domain":"example.com","sessionId":"session-1","userId":"user-1"},"jobQuery":{"query":"Engineer"},"maxPageSize":1}`)
	restSearchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants/tenant-1/jobs:search", restSearchReqBody, map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if restSearchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest talent search jobs, got %d body=%s", restSearchResp.StatusCode, string(providerContractBody(t, restSearchResp)))
	}
	restSearchBody := providerContractJSONMap(t, restSearchResp)
	restMatches, ok := restSearchBody["matchingJobs"].([]any)
	if !ok || len(restMatches) == 0 {
		t.Fatalf("expected matchingJobs in rest payload, got %#v", restSearchBody["matchingJobs"])
	}
	restMatch, _ := restMatches[0].(map[string]any)
	restMatchJob, _ := restMatch["job"].(map[string]any)
	restMatchJobName, _ := restMatchJob["name"].(string)

	successSearchReq := &talentpb.SearchJobsRequest{
		Parent: "projects/stackyard/tenants/tenant-1",
		RequestMetadata: &talentpb.RequestMetadata{
			Domain:    "example.com",
			SessionId: "session-1",
			UserId:    "user-1",
		},
		JobQuery:    &talentpb.JobQuery{Query: "Engineer"},
		MaxPageSize: 1,
	}
	var successSearchResp talentpb.SearchJobsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTalentSearchJobsMethod, successSearchReq, &successSearchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successSearchResp.GetMatchingJobs()) != 1 {
		t.Fatalf("expected one grpc matching job, got %d", len(successSearchResp.GetMatchingJobs()))
	}
	if successSearchResp.GetMatchingJobs()[0].GetJob().GetName() != restMatchJobName {
		t.Fatalf("expected grpc matching job name %q to match rest %q", successSearchResp.GetMatchingJobs()[0].GetJob().GetName(), restMatchJobName)
	}

	successBatchReq := &talentpb.BatchCreateJobsRequest{
		Parent: "projects/stackyard/tenants/tenant-1",
		Jobs: []*talentpb.Job{
			{
				Company:       "projects/stackyard/tenants/tenant-1/companies/company-1",
				RequisitionId: "req-100",
				Title:         "Platform Engineer",
				Description:   "Build and scale platform systems",
			},
		},
	}
	var successBatchResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTalentBatchCreateJobsMethod, successBatchReq, &successBatchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(successBatchResp.GetName(), "/operations/batchCreateJobs-1") {
		t.Fatalf("expected batch create operation name to include /operations/batchCreateJobs-1, got %q", successBatchResp.GetName())
	}

	invalidReq := &talentpb.CreateJobRequest{
		Parent: "projects/stackyard/tenants/tenant-1",
		Job: &talentpb.Job{
			Company: "projects/stackyard/tenants/tenant-1/companies/company-1",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTalentCreateJobMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "job.requisition_id-title-description-required") {
		t.Fatalf("expected grpc invalid argument for talent create job, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_StorageBatchOperations(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest storagebatchoperations list jobs, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["jobs"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected jobs list in rest payload, got %#v", restBody["jobs"])
	}
	restJob, _ := restItems[0].(map[string]any)
	restJobName, _ := restJob["name"].(string)

	successListReq := &storagebatchoperationspb.ListJobsRequest{
		Parent:   "projects/stackyard/locations/global",
		PageSize: 1,
	}
	var successListResp storagebatchoperationspb.ListJobsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsListJobsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetJobs()) != 1 {
		t.Fatalf("expected one grpc job, got %d", len(successListResp.GetJobs()))
	}
	if successListResp.GetJobs()[0].GetName() != restJobName {
		t.Fatalf("expected grpc job name %q to match rest %q", successListResp.GetJobs()[0].GetName(), restJobName)
	}

	successGetReq := &storagebatchoperationspb.GetJobRequest{Name: restJobName}
	var successGetResp storagebatchoperationspb.Job
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsGetJobMethod, successGetReq, &successGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetResp.GetName() != restJobName {
		t.Fatalf("expected grpc get job name %q to match rest %q", successGetResp.GetName(), restJobName)
	}

	successCreateReq := &storagebatchoperationspb.CreateJobRequest{
		Parent: "projects/stackyard/locations/global",
		JobId:  "job-create-1",
		Job: &storagebatchoperationspb.Job{
			Description: "Stackyard create job",
			Source: &storagebatchoperationspb.Job_BucketList{
				BucketList: &storagebatchoperationspb.BucketList{
					Buckets: []*storagebatchoperationspb.BucketList_Bucket{
						{Bucket: "stackyard-source-bucket"},
					},
				},
			},
			Transformation: &storagebatchoperationspb.Job_DeleteObject{
				DeleteObject: &storagebatchoperationspb.DeleteObject{
					PermanentObjectDeletionEnabled: true,
				},
			},
		},
	}
	var successCreateResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsCreateJobMethod, successCreateReq, &successCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(successCreateResp.GetName(), "/operations/createJob.job-create-1") {
		t.Fatalf("expected create operation name to include createJob.job-create-1, got %q", successCreateResp.GetName())
	}

	successGetOperationReq := &longrunningpb.GetOperationRequest{Name: successCreateResp.GetName()}
	var successGetOperationResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpLongrunningGetOpMethod, successGetOperationReq, &successGetOperationResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetOperationResp.GetName() != successCreateResp.GetName() {
		t.Fatalf("expected grpc get operation name %q to match created operation %q", successGetOperationResp.GetName(), successCreateResp.GetName())
	}

	successListOperationsReq := &longrunningpb.ListOperationsRequest{
		Name:     "projects/stackyard/locations/global",
		PageSize: 1,
	}
	var successListOperationsResp longrunningpb.ListOperationsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpLongrunningListOpsMethod, successListOperationsReq, &successListOperationsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListOperationsResp.GetOperations()) != 1 {
		t.Fatalf("expected one grpc listed operation, got %d", len(successListOperationsResp.GetOperations()))
	}

	successListBucketOpsReq := &storagebatchoperationspb.ListBucketOperationsRequest{
		Parent:   "projects/stackyard/locations/global/jobs/job-1",
		PageSize: 1,
	}
	var successListBucketOpsResp storagebatchoperationspb.ListBucketOperationsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsListBucketOperationsMethod, successListBucketOpsReq, &successListBucketOpsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListBucketOpsResp.GetBucketOperations()) != 1 {
		t.Fatalf("expected one grpc bucket operation, got %d", len(successListBucketOpsResp.GetBucketOperations()))
	}

	invalidReq := &storagebatchoperationspb.CreateJobRequest{
		Parent: "projects/stackyard/locations/global",
		JobId:  "job-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsCreateJobMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "job-required") {
		t.Fatalf("expected grpc invalid argument for storagebatchoperations create job, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &storagebatchoperationspb.CancelJobRequest{
		Name: "projects/stackyard/locations/global/jobs/job-succeeded",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageBatchOperationsCancelJobMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "job-terminal-state") {
		t.Fatalf("expected grpc failed precondition for storagebatchoperations cancel job, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_StorageInsights(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storageinsights",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest storageinsights list report configs, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["reportConfigs"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected reportConfigs list in rest payload, got %#v", restBody["reportConfigs"])
	}
	restReportConfig, _ := restItems[0].(map[string]any)
	restReportConfigName, _ := restReportConfig["name"].(string)

	successListReq := &storageinsightspb.ListReportConfigsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListResp storageinsightspb.ListReportConfigsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsListReportConfigsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetReportConfigs()) != 1 {
		t.Fatalf("expected one grpc report config, got %d", len(successListResp.GetReportConfigs()))
	}
	if successListResp.GetReportConfigs()[0].GetName() != restReportConfigName {
		t.Fatalf("expected grpc report config name %q to match rest %q", successListResp.GetReportConfigs()[0].GetName(), restReportConfigName)
	}

	datasetName := "projects/stackyard/locations/us-central1/datasetConfigs/datasetconfig1"
	restDatasetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+datasetName, nil, headers)
	if restDatasetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest storageinsights get dataset config, got %d body=%s", restDatasetResp.StatusCode, string(providerContractBody(t, restDatasetResp)))
	}
	restDatasetBody := providerContractJSONMap(t, restDatasetResp)
	restDatasetName, _ := restDatasetBody["name"].(string)

	successGetDatasetReq := &storageinsightspb.GetDatasetConfigRequest{Name: datasetName}
	var successGetDatasetResp storageinsightspb.DatasetConfig
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsGetDatasetConfigMethod, successGetDatasetReq, &successGetDatasetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetDatasetResp.GetName() != restDatasetName {
		t.Fatalf("expected grpc dataset config name %q to match rest %q", successGetDatasetResp.GetName(), restDatasetName)
	}

	successCreateDatasetReq := &storageinsightspb.CreateDatasetConfigRequest{
		Parent:          "projects/stackyard/locations/us-central1",
		DatasetConfigId: "datasetconfig1",
		RequestId:       "11111111-1111-4111-8111-111111111111",
		DatasetConfig: &storageinsightspb.DatasetConfig{
			Name: datasetName,
			SourceOptions: &storageinsightspb.DatasetConfig_SourceProjects_{
				SourceProjects: &storageinsightspb.DatasetConfig_SourceProjects{
					ProjectNumbers: []int64{123456789},
				},
			},
		},
	}
	var successCreateDatasetResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsCreateDatasetConfigMethod, successCreateDatasetReq, &successCreateDatasetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(successCreateDatasetResp.GetName(), "/operations/createDatasetConfig.datasetconfig1") {
		t.Fatalf("expected create operation name to include createDatasetConfig.datasetconfig1, got %q", successCreateDatasetResp.GetName())
	}

	successGetOperationReq := &longrunningpb.GetOperationRequest{Name: successCreateDatasetResp.GetName()}
	var successGetOperationResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpLongrunningGetOpMethod, successGetOperationReq, &successGetOperationResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetOperationResp.GetName() != successCreateDatasetResp.GetName() {
		t.Fatalf("expected grpc get operation name %q to match created operation %q", successGetOperationResp.GetName(), successCreateDatasetResp.GetName())
	}

	successLinkReq := &storageinsightspb.LinkDatasetRequest{Name: datasetName}
	var successLinkResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsLinkDatasetMethod, successLinkReq, &successLinkResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(successLinkResp.GetName(), "/operations/linkDataset.datasetconfig1") {
		t.Fatalf("expected link operation name to include linkDataset.datasetconfig1, got %q", successLinkResp.GetName())
	}

	invalidReq := &storageinsightspb.CreateDatasetConfigRequest{
		Parent:          "projects/stackyard/locations/us-central1",
		DatasetConfigId: "dataset-config-1",
		DatasetConfig: &storageinsightspb.DatasetConfig{
			SourceOptions: &storageinsightspb.DatasetConfig_SourceProjects_{
				SourceProjects: &storageinsightspb.DatasetConfig_SourceProjects{
					ProjectNumbers: []int64{123456789},
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsCreateDatasetConfigMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "dataset_config_id-invalid") {
		t.Fatalf("expected grpc invalid argument for storageinsights create dataset config, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &storageinsightspb.GetDatasetConfigRequest{
		Name: "projects/stackyard/locations/us-central1/datasetConfigs/missing-datasetconfig",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageInsightsGetDatasetConfigMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "dataset_config-not-found") {
		t.Fatalf("expected grpc not found for storageinsights get dataset config, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_StorageTransfer(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	}

	restGetJobResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs/job-1?projectId=stackyard", nil, headers)
	if restGetJobResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest storagetransfer get transfer job, got %d body=%s", restGetJobResp.StatusCode, string(providerContractBody(t, restGetJobResp)))
	}
	restGetJobBody := providerContractJSONMap(t, restGetJobResp)
	restJobName, _ := restGetJobBody["name"].(string)

	successGetReq := &storagetransferpb.GetTransferJobRequest{
		JobName:   "transferJobs/job-1",
		ProjectId: "stackyard",
	}
	var successGetResp storagetransferpb.TransferJob
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpStorageTransferGetTransferJobMethod, successGetReq, &successGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetResp.GetName() != restJobName {
		t.Fatalf("expected grpc transfer job name %q to match rest %q", successGetResp.GetName(), restJobName)
	}

	restRunResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/transferJobs/job-1:run", []byte(`{"projectId":"stackyard"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if restRunResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest storagetransfer run transfer job, got %d body=%s", restRunResp.StatusCode, string(providerContractBody(t, restRunResp)))
	}
	restRunBody := providerContractJSONMap(t, restRunResp)
	restOperationName, _ := restRunBody["name"].(string)

	successRunReq := &storagetransferpb.RunTransferJobRequest{
		JobName:   "transferJobs/job-1",
		ProjectId: "stackyard",
	}
	var successRunResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageTransferRunTransferJobMethod, successRunReq, &successRunResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successRunResp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", successRunResp.GetName(), restOperationName)
	}
	if successRunResp.GetDone() {
		t.Fatalf("expected grpc run operation done=false, got true")
	}

	invalidReq := &storagetransferpb.GetTransferJobRequest{
		JobName: "transferJobs/job-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageTransferGetTransferJobMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "project_id-required") {
		t.Fatalf("expected grpc invalid argument for storagetransfer get transfer job, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &storagetransferpb.RunTransferJobRequest{
		JobName:   "transferJobs/job-running",
		ProjectId: "stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStorageTransferRunTransferJobMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "transfer_job-active-run-conflict") {
		t.Fatalf("expected grpc failed precondition for storagetransfer run transfer job, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_StreetViewPublish(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "streetview_publish",
	}

	restGetPhotoResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photo/photo-1?view=INCLUDE_DOWNLOAD_URL", nil, headers)
	if restGetPhotoResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest streetview publish get photo, got %d body=%s", restGetPhotoResp.StatusCode, string(providerContractBody(t, restGetPhotoResp)))
	}
	restGetPhotoBody := providerContractJSONMap(t, restGetPhotoResp)
	restPhotoIDAny, _ := restGetPhotoBody["photoId"].(map[string]any)
	restPhotoID, _ := restPhotoIDAny["id"].(string)

	successGetPhotoReq := &publishpb.GetPhotoRequest{
		PhotoId: "photo-1",
		View:    publishpb.PhotoView_INCLUDE_DOWNLOAD_URL,
	}
	var successGetPhotoResp publishpb.Photo
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpStreetViewPublishGetPhotoMethod, successGetPhotoReq, &successGetPhotoResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetPhotoResp.GetPhotoId().GetId() != restPhotoID {
		t.Fatalf("expected grpc photo id %q to match rest %q", successGetPhotoResp.GetPhotoId().GetId(), restPhotoID)
	}

	restGetPhotoSequenceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/photoSequence/sequence-1?filter=published_status%3DPUBLISHED", nil, headers)
	if restGetPhotoSequenceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest streetview publish get photo sequence, got %d body=%s", restGetPhotoSequenceResp.StatusCode, string(providerContractBody(t, restGetPhotoSequenceResp)))
	}
	restGetPhotoSequenceBody := providerContractJSONMap(t, restGetPhotoSequenceResp)
	restOperationName, _ := restGetPhotoSequenceBody["name"].(string)

	successGetPhotoSequenceReq := &publishpb.GetPhotoSequenceRequest{
		SequenceId: "sequence-1",
		Filter:     "published_status=PUBLISHED",
	}
	var successGetPhotoSequenceResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStreetViewPublishGetPhotoSequenceMethod, successGetPhotoSequenceReq, &successGetPhotoSequenceResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetPhotoSequenceResp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", successGetPhotoSequenceResp.GetName(), restOperationName)
	}

	invalidReq := &publishpb.GetPhotoRequest{
		PhotoId: "photo-1",
		View:    publishpb.PhotoView(99),
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStreetViewPublishGetPhotoMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "view-invalid") {
		t.Fatalf("expected grpc invalid argument for streetview publish get photo, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &publishpb.DeletePhotoSequenceRequest{SequenceId: "sequence-processing"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpStreetViewPublishDeletePhotoSequenceMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "photo_sequence-processing") {
		t.Fatalf("expected grpc failed precondition for streetview publish delete photo sequence, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Speech(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restRecognizeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{
		"config":{"languageCode":"en-US","encoding":"LINEAR16","sampleRateHertz":16000},
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if restRecognizeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech recognize, got %d body=%s", restRecognizeResp.StatusCode, string(providerContractBody(t, restRecognizeResp)))
	}
	restRecognizeBody := providerContractJSONMap(t, restRecognizeResp)
	restRecognizeResults, _ := restRecognizeBody["results"].([]any)
	if len(restRecognizeResults) == 0 {
		t.Fatalf("expected speech recognize results in rest payload, got %#v", restRecognizeBody["results"])
	}
	restRecognizeResult, _ := restRecognizeResults[0].(map[string]any)
	restRecognizeAlternatives, _ := restRecognizeResult["alternatives"].([]any)
	if len(restRecognizeAlternatives) == 0 {
		t.Fatalf("expected speech recognize alternatives in rest payload, got %#v", restRecognizeResult["alternatives"])
	}
	restRecognizeAlt, _ := restRecognizeAlternatives[0].(map[string]any)
	restTranscript, _ := restRecognizeAlt["transcript"].(string)

	successRecognizeReq := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			LanguageCode:    "en-US",
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: []byte("stackyard"),
			},
		},
	}
	var successRecognizeResp speechpb.RecognizeResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpeechRecognizeMethod, successRecognizeReq, &successRecognizeResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech recognize, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successRecognizeResp.GetResults()) == 0 || len(successRecognizeResp.GetResults()[0].GetAlternatives()) == 0 {
		t.Fatalf("expected grpc speech recognize results and alternatives, got %#v", &successRecognizeResp)
	}
	if successRecognizeResp.GetResults()[0].GetAlternatives()[0].GetTranscript() != restTranscript {
		t.Fatalf("expected grpc speech transcript %q to match rest %q", successRecognizeResp.GetResults()[0].GetAlternatives()[0].GetTranscript(), restTranscript)
	}

	restListPhraseSetResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListPhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if restListPhraseSetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech list phrase sets, got %d body=%s", restListPhraseSetResp.StatusCode, string(providerContractBody(t, restListPhraseSetResp)))
	}
	restListPhraseSetBody := providerContractJSONMap(t, restListPhraseSetResp)
	restPhraseSets, _ := restListPhraseSetBody["phraseSets"].([]any)
	if len(restPhraseSets) == 0 {
		t.Fatalf("expected phraseSets list in rest payload, got %#v", restListPhraseSetBody["phraseSets"])
	}
	restPhraseSet, _ := restPhraseSets[0].(map[string]any)
	restPhraseSetName, _ := restPhraseSet["name"].(string)

	successListPhraseSetReq := &speechpb.ListPhraseSetRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListPhraseSetResp speechpb.ListPhraseSetResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechListPhraseSetMethod, successListPhraseSetReq, &successListPhraseSetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech list phrase sets, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListPhraseSetResp.GetPhraseSets()) != 1 {
		t.Fatalf("expected one grpc phrase set, got %d", len(successListPhraseSetResp.GetPhraseSets()))
	}
	if successListPhraseSetResp.GetPhraseSets()[0].GetName() != restPhraseSetName {
		t.Fatalf("expected grpc phrase set %q to match rest %q", successListPhraseSetResp.GetPhraseSets()[0].GetName(), restPhraseSetName)
	}

	restStreamingResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/StreamingRecognize", []byte(`{
		"streamingConfig":{"config":{"languageCode":"en-US","sampleRateHertz":16000}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if restStreamingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech streaming recognize, got %d body=%s", restStreamingResp.StatusCode, string(providerContractBody(t, restStreamingResp)))
	}
	restStreamingBody := providerContractJSONMap(t, restStreamingResp)
	restStreamingResults, _ := restStreamingBody["results"].([]any)
	if len(restStreamingResults) == 0 {
		t.Fatalf("expected streaming results in rest payload, got %#v", restStreamingBody["results"])
	}
	restStreamingResult, _ := restStreamingResults[0].(map[string]any)
	restStreamingAlternatives, _ := restStreamingResult["alternatives"].([]any)
	if len(restStreamingAlternatives) == 0 {
		t.Fatalf("expected streaming alternatives in rest payload, got %#v", restStreamingResult["alternatives"])
	}
	restStreamingAlt, _ := restStreamingAlternatives[0].(map[string]any)
	restStreamingTranscript, _ := restStreamingAlt["transcript"].(string)

	successStreamingReq := &speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					LanguageCode:    "en-US",
					SampleRateHertz: 16000,
				},
			},
		},
	}
	var successStreamingResp speechpb.StreamingRecognizeResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechStreamingRecognizeMethod, successStreamingReq, &successStreamingResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech streaming recognize, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successStreamingResp.GetResults()) == 0 || len(successStreamingResp.GetResults()[0].GetAlternatives()) == 0 {
		t.Fatalf("expected grpc speech streaming results and alternatives, got %#v", &successStreamingResp)
	}
	if successStreamingResp.GetResults()[0].GetAlternatives()[0].GetTranscript() != restStreamingTranscript {
		t.Fatalf("expected grpc streaming transcript %q to match rest %q", successStreamingResp.GetResults()[0].GetAlternatives()[0].GetTranscript(), restStreamingTranscript)
	}

	invalidRecognizeReq := &speechpb.RecognizeRequest{
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: []byte("stackyard")},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechRecognizeMethod, invalidRecognizeReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "config-required") {
		t.Fatalf("expected grpc invalid argument for speech recognize, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidListPhraseSetReq := &speechpb.ListPhraseSetRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechListPhraseSetMethod, invalidListPhraseSetReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for speech list phrase sets, got status=%q message=%q", grpcStatus, grpcMessage)
	}

}

func TestGCPStage4GRPCParity_SpeechV2(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restRecognizeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v2.Speech/Recognize", []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"config":{"languageCodes":["en-US"]},
		"content":"c3RhY2t5YXJk"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if restRecognizeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech v2 recognize, got %d body=%s", restRecognizeResp.StatusCode, string(providerContractBody(t, restRecognizeResp)))
	}
	restRecognizeBody := providerContractJSONMap(t, restRecognizeResp)
	restRecognizeResults, _ := restRecognizeBody["results"].([]any)
	if len(restRecognizeResults) == 0 {
		t.Fatalf("expected speech v2 recognize results in rest payload, got %#v", restRecognizeBody["results"])
	}
	restRecognizeResult, _ := restRecognizeResults[0].(map[string]any)
	restRecognizeAlternatives, _ := restRecognizeResult["alternatives"].([]any)
	if len(restRecognizeAlternatives) == 0 {
		t.Fatalf("expected speech v2 recognize alternatives in rest payload, got %#v", restRecognizeResult["alternatives"])
	}
	restRecognizeAlt, _ := restRecognizeAlternatives[0].(map[string]any)
	restTranscript, _ := restRecognizeAlt["transcript"].(string)

	successRecognizeReq := &speechv2pb.RecognizeRequest{
		Recognizer: "projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		Config: &speechv2pb.RecognitionConfig{
			LanguageCodes: []string{"en-US"},
		},
		AudioSource: &speechv2pb.RecognizeRequest_Content{
			Content: []byte("stackyard"),
		},
	}
	var successRecognizeResp speechv2pb.RecognizeResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpeechV2RecognizeMethod, successRecognizeReq, &successRecognizeResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech v2 recognize, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successRecognizeResp.GetResults()) == 0 || len(successRecognizeResp.GetResults()[0].GetAlternatives()) == 0 {
		t.Fatalf("expected grpc speech v2 recognize results and alternatives, got %#v", &successRecognizeResp)
	}
	if successRecognizeResp.GetResults()[0].GetAlternatives()[0].GetTranscript() != restTranscript {
		t.Fatalf("expected grpc speech v2 transcript %q to match rest %q", successRecognizeResp.GetResults()[0].GetAlternatives()[0].GetTranscript(), restTranscript)
	}

	restListRecognizersResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v2.Speech/ListRecognizers", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if restListRecognizersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech v2 list recognizers, got %d body=%s", restListRecognizersResp.StatusCode, string(providerContractBody(t, restListRecognizersResp)))
	}
	restListRecognizersBody := providerContractJSONMap(t, restListRecognizersResp)
	restRecognizers, _ := restListRecognizersBody["recognizers"].([]any)
	if len(restRecognizers) == 0 {
		t.Fatalf("expected recognizers list in rest payload, got %#v", restListRecognizersBody["recognizers"])
	}
	restRecognizer, _ := restRecognizers[0].(map[string]any)
	restRecognizerName, _ := restRecognizer["name"].(string)

	successListRecognizersReq := &speechv2pb.ListRecognizersRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListRecognizersResp speechv2pb.ListRecognizersResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechV2ListRecognizersMethod, successListRecognizersReq, &successListRecognizersResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech v2 list recognizers, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListRecognizersResp.GetRecognizers()) != 1 {
		t.Fatalf("expected one grpc recognizer, got %d", len(successListRecognizersResp.GetRecognizers()))
	}
	if successListRecognizersResp.GetRecognizers()[0].GetName() != restRecognizerName {
		t.Fatalf("expected grpc recognizer %q to match rest %q", successListRecognizersResp.GetRecognizers()[0].GetName(), restRecognizerName)
	}

	restBatchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v2.Speech/BatchRecognize", []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"files":[{"uri":"gs://stackyard/audio-1.wav"}],
		"recognitionOutputConfig":{"inlineResponseConfig":{}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if restBatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest speech v2 batch recognize, got %d body=%s", restBatchResp.StatusCode, string(providerContractBody(t, restBatchResp)))
	}
	restBatchBody := providerContractJSONMap(t, restBatchResp)
	restBatchName, _ := restBatchBody["name"].(string)

	successBatchReq := &speechv2pb.BatchRecognizeRequest{
		Recognizer: "projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		Files: []*speechv2pb.BatchRecognizeFileMetadata{
			{
				AudioSource: &speechv2pb.BatchRecognizeFileMetadata_Uri{
					Uri: "gs://stackyard/audio-1.wav",
				},
			},
		},
		RecognitionOutputConfig: &speechv2pb.RecognitionOutputConfig{
			Output: &speechv2pb.RecognitionOutputConfig_InlineResponseConfig{
				InlineResponseConfig: &speechv2pb.InlineOutputConfig{},
			},
		},
	}
	var successBatchResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechV2BatchRecognizeMethod, successBatchReq, &successBatchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for speech v2 batch recognize, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successBatchResp.GetName() != restBatchName {
		t.Fatalf("expected grpc batch operation name %q to match rest %q", successBatchResp.GetName(), restBatchName)
	}
	if !successBatchResp.GetDone() {
		t.Fatalf("expected grpc batch operation done=true, got %#v", successBatchResp.GetDone())
	}

	invalidRecognizeReq := &speechv2pb.RecognizeRequest{
		AudioSource: &speechv2pb.RecognizeRequest_Content{Content: []byte("stackyard")},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechV2RecognizeMethod, invalidRecognizeReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "recognizer-required") {
		t.Fatalf("expected grpc invalid argument for speech v2 recognize, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidListRecognizersReq := &speechv2pb.ListRecognizersRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpeechV2ListRecognizersMethod, invalidListRecognizersReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for speech v2 list recognizers, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Spanner(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner",
	}

	database := "projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	sessionName := database + "/sessions/s-1"

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+database+"/sessions", []byte(`{"session":{"labels":{"env":"test"}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner create session, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restSessionName, _ := restCreateBody["name"].(string)

	successCreateReq := &spannerpb.CreateSessionRequest{
		Database: database,
		Session:  &spannerpb.Session{},
	}
	var successCreateResp spannerpb.Session
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpannerCreateSessionMethod, successCreateReq, &successCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner create session, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successCreateResp.GetName() != restSessionName {
		t.Fatalf("expected grpc session name %q to match rest %q", successCreateResp.GetName(), restSessionName)
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+database+"/sessions?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner list sessions, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restListItems, ok := restListBody["sessions"].([]any)
	if !ok || len(restListItems) == 0 {
		t.Fatalf("expected sessions list in rest payload, got %#v", restListBody["sessions"])
	}
	restListSession, _ := restListItems[0].(map[string]any)
	restListedSessionName, _ := restListSession["name"].(string)

	successListReq := &spannerpb.ListSessionsRequest{
		Database: database,
		PageSize: 1,
	}
	var successListResp spannerpb.ListSessionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerListSessionsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner list sessions, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetSessions()) != 1 {
		t.Fatalf("expected one grpc session, got %d", len(successListResp.GetSessions()))
	}
	if successListResp.GetSessions()[0].GetName() != restListedSessionName {
		t.Fatalf("expected grpc list session name %q to match rest %q", successListResp.GetSessions()[0].GetName(), restListedSessionName)
	}

	restExecuteResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":executeSql", []byte(`{"sql":"SELECT 1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restExecuteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner execute sql, got %d body=%s", restExecuteResp.StatusCode, string(providerContractBody(t, restExecuteResp)))
	}
	restExecuteBody := providerContractJSONMap(t, restExecuteResp)
	restRows, _ := restExecuteBody["rows"].([]any)
	if len(restRows) == 0 {
		t.Fatalf("expected rows in rest execute sql response, got %#v", restExecuteBody["rows"])
	}
	restFirstRow, _ := restRows[0].([]any)
	restFirstValue, _ := restFirstRow[0].(string)

	successExecuteReq := &spannerpb.ExecuteSqlRequest{
		Session: sessionName,
		Sql:     "SELECT 1",
	}
	var successExecuteResp spannerpb.ResultSet
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerExecuteSQLMethod, successExecuteReq, &successExecuteResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner execute sql, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successExecuteResp.GetRows()) == 0 || len(successExecuteResp.GetRows()[0].GetValues()) == 0 {
		t.Fatalf("expected grpc rows in execute sql response, got %#v", successExecuteResp.GetRows())
	}
	grpcFirstValue := successExecuteResp.GetRows()[0].GetValues()[0].GetStringValue()
	if grpcFirstValue != restFirstValue {
		t.Fatalf("expected grpc first value %q to match rest %q", grpcFirstValue, restFirstValue)
	}

	restBeginResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":beginTransaction", []byte(`{"options":{"readWrite":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restBeginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner begin transaction, got %d body=%s", restBeginResp.StatusCode, string(providerContractBody(t, restBeginResp)))
	}
	restBeginBody := providerContractJSONMap(t, restBeginResp)
	restTxID, _ := restBeginBody["id"].(string)

	successBeginReq := &spannerpb.BeginTransactionRequest{
		Session: sessionName,
		Options: &spannerpb.TransactionOptions{
			Mode: &spannerpb.TransactionOptions_ReadWrite_{
				ReadWrite: &spannerpb.TransactionOptions_ReadWrite{},
			},
		},
	}
	var successBeginResp spannerpb.Transaction
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerBeginTransactionMethod, successBeginReq, &successBeginResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner begin transaction, got %q message=%q", grpcStatus, grpcMessage)
	}
	if base64Tx := string(successBeginResp.GetId()); base64Tx == "" {
		t.Fatalf("expected grpc begin transaction id to be set")
	}
	if restTxID == "" {
		t.Fatalf("expected rest begin transaction id to be set")
	}

	restPartitionResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":partitionQuery", []byte(`{"sql":"SELECT * FROM Users","transaction":{"id":"dHgtcy0x"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restPartitionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner partition query, got %d body=%s", restPartitionResp.StatusCode, string(providerContractBody(t, restPartitionResp)))
	}
	restPartitionBody := providerContractJSONMap(t, restPartitionResp)
	restPartitions, _ := restPartitionBody["partitions"].([]any)

	successPartitionReq := &spannerpb.PartitionQueryRequest{
		Session:     sessionName,
		Sql:         "SELECT * FROM Users",
		Transaction: &spannerpb.TransactionSelector{Selector: &spannerpb.TransactionSelector_Id{Id: []byte("tx-s-1")}},
	}
	var successPartitionResp spannerpb.PartitionResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerPartitionQueryMethod, successPartitionReq, &successPartitionResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner partition query, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successPartitionResp.GetPartitions()) != len(restPartitions) {
		t.Fatalf("expected grpc partitions %d to match rest %d", len(successPartitionResp.GetPartitions()), len(restPartitions))
	}

	restStreamingResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":executeStreamingSql", []byte(`{"sql":"SELECT 1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restStreamingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner execute streaming sql, got %d body=%s", restStreamingResp.StatusCode, string(providerContractBody(t, restStreamingResp)))
	}
	restStreamingBody := providerContractJSONMap(t, restStreamingResp)
	restStreamingValues, _ := restStreamingBody["values"].([]any)
	restStreamingFirst, _ := restStreamingValues[0].(string)

	successStreamingReq := &spannerpb.ExecuteSqlRequest{
		Session: sessionName,
		Sql:     "SELECT 1",
	}
	var successStreamingResp spannerpb.PartialResultSet
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerExecuteStreamingSQLMethod, successStreamingReq, &successStreamingResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner execute streaming sql, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successStreamingResp.GetValues()) == 0 {
		t.Fatalf("expected grpc values in execute streaming sql response")
	}
	if successStreamingResp.GetValues()[0].GetStringValue() != restStreamingFirst {
		t.Fatalf("expected grpc first streaming value %q to match rest %q", successStreamingResp.GetValues()[0].GetStringValue(), restStreamingFirst)
	}

	restBatchWriteResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":batchWrite", []byte(`{"mutationGroups":[{"mutations":[{"insert":{"table":"Users"}}]}]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if restBatchWriteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner batch write, got %d body=%s", restBatchWriteResp.StatusCode, string(providerContractBody(t, restBatchWriteResp)))
	}
	restBatchWriteBody := providerContractJSONMap(t, restBatchWriteResp)
	restIndexes, _ := restBatchWriteBody["indexes"].([]any)

	successBatchWriteReq := &spannerpb.BatchWriteRequest{
		Session: sessionName,
		MutationGroups: []*spannerpb.BatchWriteRequest_MutationGroup{
			{
				Mutations: []*spannerpb.Mutation{
					{
						Operation: &spannerpb.Mutation_Insert{
							Insert: &spannerpb.Mutation_Write{Table: "Users"},
						},
					},
				},
			},
		},
	}
	var successBatchWriteResp spannerpb.BatchWriteResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerBatchWriteMethod, successBatchWriteReq, &successBatchWriteResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner batch write, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successBatchWriteResp.GetIndexes()) != len(restIndexes) {
		t.Fatalf("expected grpc indexes %d to match rest %d", len(successBatchWriteResp.GetIndexes()), len(restIndexes))
	}

	invalidReq := &spannerpb.ListSessionsRequest{
		Database: "projects/stackyard/databases/stackyard-db",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerListSessionsMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "database-required") {
		t.Fatalf("expected grpc invalid argument for spanner list sessions, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &spannerpb.GetSessionRequest{
		Name: "projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/missing-session",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerGetSessionMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "session-not-found") {
		t.Fatalf("expected grpc not found for spanner get session, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	failedPreconditionReq := &spannerpb.CommitRequest{
		Session: sessionName,
		Transaction: &spannerpb.CommitRequest_TransactionId{
			TransactionId: []byte("tx-stale"),
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerCommitMethod, failedPreconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "transaction-stale") {
		t.Fatalf("expected grpc failed precondition for spanner commit, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	abortedReq := &spannerpb.CommitRequest{
		Session: sessionName,
		Transaction: &spannerpb.CommitRequest_TransactionId{
			TransactionId: []byte("tx-abort"),
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerCommitMethod, abortedReq, nil)
	if grpcStatus != "10" || !strings.Contains(grpcMessage, "transaction-aborted") {
		t.Fatalf("expected grpc aborted for spanner commit, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SpannerAdapter(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-adapter",
	}

	database := "projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	sessionName := database + "/sessions/as-1"

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+database+"/sessions:adapter", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner adapter create session, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restSessionName, _ := restCreateBody["name"].(string)

	successCreateReq := &adapterpb.CreateSessionRequest{
		Parent:  database,
		Session: &adapterpb.Session{Name: sessionName},
	}
	var successCreateResp adapterpb.Session
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterCreateSessionMethod, successCreateReq, &successCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner adapter create session, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successCreateResp.GetName() != restSessionName {
		t.Fatalf("expected grpc session name %q to match rest %q", successCreateResp.GetName(), restSessionName)
	}

	restAdaptResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+sessionName+":adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","payload":"aGVsbG8=","attachments":{"trace":"t-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if restAdaptResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner adapter adapt message, got %d body=%s", restAdaptResp.StatusCode, string(providerContractBody(t, restAdaptResp)))
	}
	restAdaptBody := providerContractJSONMap(t, restAdaptResp)
	restPayloadEncoded, _ := restAdaptBody["payload"].(string)
	restPayload, err := base64.StdEncoding.DecodeString(restPayloadEncoded)
	if err != nil {
		t.Fatalf("expected rest payload to be base64, decode err=%v payload=%q", err, restPayloadEncoded)
	}

	successAdaptReq := &adapterpb.AdaptMessageRequest{
		Name:        sessionName,
		Protocol:    "pgwire",
		Payload:     []byte("hello"),
		Attachments: map[string]string{"trace": "t-1"},
	}
	var successAdaptResp adapterpb.AdaptMessageResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterAdaptMessageMethod, successAdaptReq, &successAdaptResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner adapter adapt message, got %q message=%q", grpcStatus, grpcMessage)
	}
	if string(successAdaptResp.GetPayload()) != string(restPayload) {
		t.Fatalf("expected grpc payload %q to match rest %q", string(successAdaptResp.GetPayload()), string(restPayload))
	}
	if !successAdaptResp.GetLast() {
		t.Fatalf("expected grpc adapt message response to be last=true")
	}
	if got := successAdaptResp.GetStateUpdates()["protocol"]; got != "pgwire" {
		t.Fatalf("expected grpc state update protocol=pgwire, got %q", got)
	}

	invalidCreateReq := &adapterpb.CreateSessionRequest{
		Session: &adapterpb.Session{},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterCreateSessionMethod, invalidCreateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for spanner adapter create session, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidAdaptReq := &adapterpb.AdaptMessageRequest{
		Name: sessionName,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterAdaptMessageMethod, invalidAdaptReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "protocol-required") {
		t.Fatalf("expected grpc invalid argument for spanner adapter adapt message, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	failedPreconditionReq := &adapterpb.AdaptMessageRequest{
		Name:     sessionName,
		Protocol: "unsupported",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterAdaptMessageMethod, failedPreconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "protocol-unsupported") {
		t.Fatalf("expected grpc failed precondition for spanner adapter adapt message, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &adapterpb.AdaptMessageRequest{
		Name:     "projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/missing-session",
		Protocol: "pgwire",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdapterAdaptMessageMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "session-not-found") {
		t.Fatalf("expected grpc not found for spanner adapter adapt message, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	restProbeResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner_adapter?stackyard_contract_probe=1&typedSuccess=1", nil, headers)
	if restProbeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner adapter contract probe, got %d body=%s", restProbeResp.StatusCode, string(providerContractBody(t, restProbeResp)))
	}
}

func TestGCPStage4GRPCParity_SpannerAdminDatabase(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	}

	instance := "projects/stackyard/instances/stackyard-instance"
	databaseName := instance + "/databases/stackyard-db"
	backupName := instance + "/backups/backup-1"
	scheduleName := databaseName + "/backupSchedules/daily-full"
	operationName := instance + "/operations/create-database-stackyard-db"

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+instance+"/databases?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin list databases, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restDatabases, ok := restListBody["databases"].([]any)
	if !ok || len(restDatabases) == 0 {
		t.Fatalf("expected databases list in rest payload, got %#v", restListBody["databases"])
	}
	restDatabase, _ := restDatabases[0].(map[string]any)
	restDatabaseName, _ := restDatabase["name"].(string)

	successListReq := &spanneradminpb.ListDatabasesRequest{
		Parent:   instance,
		PageSize: 1,
	}
	var successListResp spanneradminpb.ListDatabasesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseListDatabasesMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin list databases, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetDatabases()) != 1 {
		t.Fatalf("expected one grpc database, got %d", len(successListResp.GetDatabases()))
	}
	if successListResp.GetDatabases()[0].GetName() != restDatabaseName {
		t.Fatalf("expected grpc database name %q to match rest %q", successListResp.GetDatabases()[0].GetName(), restDatabaseName)
	}

	restBackupResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+backupName, nil, headers)
	if restBackupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin get backup, got %d body=%s", restBackupResp.StatusCode, string(providerContractBody(t, restBackupResp)))
	}
	restBackupBody := providerContractJSONMap(t, restBackupResp)
	restBackupName, _ := restBackupBody["name"].(string)

	successBackupReq := &spanneradminpb.GetBackupRequest{Name: backupName}
	var successBackupResp spanneradminpb.Backup
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseGetBackupMethod, successBackupReq, &successBackupResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin get backup, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successBackupResp.GetName() != restBackupName {
		t.Fatalf("expected grpc backup name %q to match rest %q", successBackupResp.GetName(), restBackupName)
	}

	restScheduleResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+scheduleName, nil, headers)
	if restScheduleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin get backup schedule, got %d body=%s", restScheduleResp.StatusCode, string(providerContractBody(t, restScheduleResp)))
	}
	restScheduleBody := providerContractJSONMap(t, restScheduleResp)
	restScheduleName, _ := restScheduleBody["name"].(string)

	successScheduleReq := &spanneradminpb.GetBackupScheduleRequest{Name: scheduleName}
	var successScheduleResp spanneradminpb.BackupSchedule
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseGetBackupScheduleMethod, successScheduleReq, &successScheduleResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin get backup schedule, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successScheduleResp.GetName() != restScheduleName {
		t.Fatalf("expected grpc schedule name %q to match rest %q", successScheduleResp.GetName(), restScheduleName)
	}

	restOperationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+operationName, nil, headers)
	if restOperationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin get operation, got %d body=%s", restOperationResp.StatusCode, string(providerContractBody(t, restOperationResp)))
	}
	restOperationBody := providerContractJSONMap(t, restOperationResp)
	restOpName, _ := restOperationBody["name"].(string)

	successGetOpReq := &longrunningpb.GetOperationRequest{Name: operationName}
	var successGetOpResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseGetOperationMethod, successGetOpReq, &successGetOpResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin get operation, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetOpResp.GetName() != restOpName {
		t.Fatalf("expected grpc operation name %q to match rest %q", successGetOpResp.GetName(), restOpName)
	}

	successPermissionsReq := &iampb.TestIamPermissionsRequest{
		Resource:    databaseName,
		Permissions: []string{"spanner.databases.get", "resourcemanager.projects.get"},
	}
	var successPermissionsResp iampb.TestIamPermissionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseTestIAMPermissionsMethod, successPermissionsReq, &successPermissionsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin test iam permissions, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successPermissionsResp.GetPermissions()) != 1 {
		t.Fatalf("expected grpc filtered permissions length 1, got %d", len(successPermissionsResp.GetPermissions()))
	}

	invalidCreateReq := &spanneradminpb.CreateDatabaseRequest{
		Parent: instance,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseCreateDatabaseMethod, invalidCreateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "create_statement-required") {
		t.Fatalf("expected grpc invalid argument for spanner admin create database, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	alreadyExistsReq := &spanneradminpb.CreateBackupScheduleRequest{
		Parent:           databaseName,
		BackupScheduleId: "existing-schedule",
		BackupSchedule:   &spanneradminpb.BackupSchedule{},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseCreateBackupScheduleMethod, alreadyExistsReq, nil)
	if grpcStatus != "6" || !strings.Contains(grpcMessage, "backup-schedule-already-exists") {
		t.Fatalf("expected grpc already exists for spanner admin create backup schedule, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	failedPreconditionReq := &spanneradminpb.CopyBackupRequest{
		Parent:       instance,
		BackupId:     "backup-copy-1",
		SourceBackup: instance + "/backups/creating-backup",
		ExpireTime:   timestamppb.Now(),
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseCopyBackupMethod, failedPreconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "source-backup-not-ready") {
		t.Fatalf("expected grpc failed precondition for spanner admin copy backup, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &spanneradminpb.GetDatabaseRequest{
		Name: instance + "/databases/missing-db",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminDatabaseGetDatabaseMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "database-not-found") {
		t.Fatalf("expected grpc not found for spanner admin get database, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SpannerAdminInstance(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-instance",
	}

	project := "projects/stackyard"
	instanceName := project + "/instances/stackyard-instance"
	operationName := instanceName + "/operations/create-instance-stackyard-instance"

	restListConfigsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+project+"/instanceConfigs?pageSize=1", nil, headers)
	if restListConfigsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin instance list instance configs, got %d body=%s", restListConfigsResp.StatusCode, string(providerContractBody(t, restListConfigsResp)))
	}
	restListConfigsBody := providerContractJSONMap(t, restListConfigsResp)
	restConfigs, ok := restListConfigsBody["instanceConfigs"].([]any)
	if !ok || len(restConfigs) == 0 {
		t.Fatalf("expected instanceConfigs list in rest payload, got %#v", restListConfigsBody["instanceConfigs"])
	}
	restConfig, _ := restConfigs[0].(map[string]any)
	restConfigName, _ := restConfig["name"].(string)

	successListConfigsReq := &spanneradmininstancepb.ListInstanceConfigsRequest{
		Parent:   project,
		PageSize: 1,
	}
	var successListConfigsResp spanneradmininstancepb.ListInstanceConfigsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceListInstanceConfigsMethod, successListConfigsReq, &successListConfigsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin instance list instance configs, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListConfigsResp.GetInstanceConfigs()) != 1 {
		t.Fatalf("expected one grpc instance config, got %d", len(successListConfigsResp.GetInstanceConfigs()))
	}
	if successListConfigsResp.GetInstanceConfigs()[0].GetName() != restConfigName {
		t.Fatalf("expected grpc instance config name %q to match rest %q", successListConfigsResp.GetInstanceConfigs()[0].GetName(), restConfigName)
	}

	restGetInstanceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+instanceName, nil, headers)
	if restGetInstanceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin instance get instance, got %d body=%s", restGetInstanceResp.StatusCode, string(providerContractBody(t, restGetInstanceResp)))
	}
	restGetInstanceBody := providerContractJSONMap(t, restGetInstanceResp)
	restDisplayName, _ := restGetInstanceBody["displayName"].(string)

	successGetInstanceReq := &spanneradmininstancepb.GetInstanceRequest{Name: instanceName}
	var successGetInstanceResp spanneradmininstancepb.Instance
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceGetInstanceMethod, successGetInstanceReq, &successGetInstanceResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin instance get instance, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetInstanceResp.GetDisplayName() != restDisplayName {
		t.Fatalf("expected grpc displayName %q to match rest %q", successGetInstanceResp.GetDisplayName(), restDisplayName)
	}

	restGetOpResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+operationName, nil, headers)
	if restGetOpResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest spanner admin instance get operation, got %d body=%s", restGetOpResp.StatusCode, string(providerContractBody(t, restGetOpResp)))
	}
	restGetOpBody := providerContractJSONMap(t, restGetOpResp)
	restOpName, _ := restGetOpBody["name"].(string)

	successGetOpReq := &longrunningpb.GetOperationRequest{Name: operationName}
	var successGetOpResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceGetOperationMethod, successGetOpReq, &successGetOpResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin instance get operation, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetOpResp.GetName() != restOpName {
		t.Fatalf("expected grpc operation name %q to match rest %q", successGetOpResp.GetName(), restOpName)
	}

	successPermissionsReq := &iampb.TestIamPermissionsRequest{
		Resource:    instanceName,
		Permissions: []string{"spanner.instances.get", "resourcemanager.projects.get"},
	}
	var successPermissionsResp iampb.TestIamPermissionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceTestIAMPermissionsMethod, successPermissionsReq, &successPermissionsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for spanner admin instance test iam permissions, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successPermissionsResp.GetPermissions()) != 1 {
		t.Fatalf("expected grpc filtered permissions length 1, got %d", len(successPermissionsResp.GetPermissions()))
	}

	invalidCreateReq := &spanneradmininstancepb.CreateInstanceRequest{
		Parent:     project,
		InstanceId: "stackyard-instance-new",
		Instance: &spanneradmininstancepb.Instance{
			Config: "projects/stackyard/instanceConfigs/custom-stackyard-primary",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceCreateInstanceMethod, invalidCreateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "instance.display_name-required") {
		t.Fatalf("expected grpc invalid argument for spanner admin instance create instance, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionMoveReq := &spanneradmininstancepb.MoveInstanceRequest{
		Name:         instanceName,
		TargetConfig: "projects/stackyard/instanceConfigs/custom-stackyard-primary",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceMoveInstanceMethod, preconditionMoveReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "instance-already-uses-target-config") {
		t.Fatalf("expected grpc failed precondition for spanner admin instance move instance, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &spanneradmininstancepb.GetInstanceRequest{
		Name: project + "/instances/missing-instance",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSpannerAdminInstanceGetInstanceMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "instance-not-found") {
		t.Fatalf("expected grpc not found for spanner admin instance get instance, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Shell(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shell",
	}
	environmentName := "users/me/environments/default"
	newKey := "ssh-rsa c3RhY2t5YXJkLW5ldy1rZXk= stackyard@example.com"

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+environmentName, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shell get environment, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restState, _ := restGetBody["state"].(string)

	successGetReq := &shellpb.GetEnvironmentRequest{Name: environmentName}
	var successGetResp shellpb.Environment
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShellGetEnvironmentMethod, successGetReq, &successGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shell get environment, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetResp.GetName() != environmentName {
		t.Fatalf("expected grpc environment name %q, got %q", environmentName, successGetResp.GetName())
	}
	if successGetResp.GetState().String() != restState {
		t.Fatalf("expected grpc environment state %q to match rest %q", successGetResp.GetState().String(), restState)
	}

	restStartResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":start", []byte(`{
		"name":"users/me/environments/default",
		"publicKeys":["`+newKey+`"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if restStartResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shell start environment, got %d body=%s", restStartResp.StatusCode, string(providerContractBody(t, restStartResp)))
	}
	restStartBody := providerContractJSONMap(t, restStartResp)
	restStartOperationName, _ := restStartBody["name"].(string)

	successStartReq := &shellpb.StartEnvironmentRequest{
		Name:       environmentName,
		PublicKeys: []string{newKey},
	}
	var successStartResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShellStartEnvironmentMethod, successStartReq, &successStartResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shell start environment, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successStartResp.GetName() != restStartOperationName {
		t.Fatalf("expected grpc start operation name %q to match rest %q", successStartResp.GetName(), restStartOperationName)
	}

	restAddResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+environmentName+":addPublicKey", []byte(`{
		"environment":"users/me/environments/default",
		"key":"`+newKey+`"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shell",
	})
	if restAddResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shell add public key, got %d body=%s", restAddResp.StatusCode, string(providerContractBody(t, restAddResp)))
	}
	restAddBody := providerContractJSONMap(t, restAddResp)
	restAddOperationName, _ := restAddBody["name"].(string)

	successAddReq := &shellpb.AddPublicKeyRequest{
		Environment: environmentName,
		Key:         newKey,
	}
	var successAddResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShellAddPublicKeyMethod, successAddReq, &successAddResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shell add public key, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successAddResp.GetName() != restAddOperationName {
		t.Fatalf("expected grpc add operation name %q to match rest %q", successAddResp.GetName(), restAddOperationName)
	}

	invalidAuthorizeReq := &shellpb.AuthorizeEnvironmentRequest{
		Name: environmentName,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShellAuthorizeEnvironmentMethod, invalidAuthorizeReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "token-required") {
		t.Fatalf("expected grpc invalid argument for shell authorize, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	duplicateAddReq := &shellpb.AddPublicKeyRequest{
		Environment: environmentName,
		Key:         gcpShellDefaultPublicKey(),
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShellAddPublicKeyMethod, duplicateAddReq, nil)
	if grpcStatus != "6" || !strings.Contains(grpcMessage, "key-already-exists") {
		t.Fatalf("expected grpc already exists for shell add key, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingRemoveReq := &shellpb.RemovePublicKeyRequest{
		Environment: environmentName,
		Key:         "ssh-rsa bWlzc2luZw== stackyard@example.com",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShellRemovePublicKeyMethod, missingRemoveReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "key-not-found") {
		t.Fatalf("expected grpc not found for shell remove key, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingCSS(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-css",
	}
	accountName := "accounts/123456"
	productName := accountName + "/cssProducts/en~US~sku-1"

	restAccountResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+accountName, nil, headers)
	if restAccountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping css get account, got %d body=%s", restAccountResp.StatusCode, string(providerContractBody(t, restAccountResp)))
	}
	restAccountBody := providerContractJSONMap(t, restAccountResp)
	restAccountName, _ := restAccountBody["name"].(string)

	successGetAccountReq := &shoppingcsspb.GetAccountRequest{Name: accountName}
	var successGetAccountResp shoppingcsspb.Account
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSGetAccountMethod, successGetAccountReq, &successGetAccountResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shopping css get account, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetAccountResp.GetName() != restAccountName {
		t.Fatalf("expected grpc account name %q to match rest %q", successGetAccountResp.GetName(), restAccountName)
	}

	restLabelsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/labels?pageSize=1", nil, headers)
	if restLabelsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping css list account labels, got %d body=%s", restLabelsResp.StatusCode, string(providerContractBody(t, restLabelsResp)))
	}
	restLabelsBody := providerContractJSONMap(t, restLabelsResp)
	restLabels, ok := restLabelsBody["accountLabels"].([]any)
	if !ok || len(restLabels) == 0 {
		t.Fatalf("expected accountLabels in rest payload, got %#v", restLabelsBody["accountLabels"])
	}
	restFirstLabel, _ := restLabels[0].(map[string]any)
	restFirstLabelName, _ := restFirstLabel["name"].(string)

	successListLabelsReq := &shoppingcsspb.ListAccountLabelsRequest{
		Parent:   accountName,
		PageSize: 1,
	}
	var successListLabelsResp shoppingcsspb.ListAccountLabelsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSListAccountLabelsMethod, successListLabelsReq, &successListLabelsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shopping css list account labels, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListLabelsResp.GetAccountLabels()) != 1 {
		t.Fatalf("expected one grpc account label, got %d", len(successListLabelsResp.GetAccountLabels()))
	}
	if successListLabelsResp.GetAccountLabels()[0].GetName() != restFirstLabelName {
		t.Fatalf("expected grpc account label name %q to match rest %q", successListLabelsResp.GetAccountLabels()[0].GetName(), restFirstLabelName)
	}

	restProductResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+productName, nil, headers)
	if restProductResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping css get css product, got %d body=%s", restProductResp.StatusCode, string(providerContractBody(t, restProductResp)))
	}
	restProductBody := providerContractJSONMap(t, restProductResp)
	restProductName, _ := restProductBody["name"].(string)

	successGetProductReq := &shoppingcsspb.GetCssProductRequest{Name: productName}
	var successGetProductResp shoppingcsspb.CssProduct
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSGetCssProductMethod, successGetProductReq, &successGetProductResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shopping css get css product, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetProductResp.GetName() != restProductName {
		t.Fatalf("expected grpc css product name %q to match rest %q", successGetProductResp.GetName(), restProductName)
	}

	restQuotaResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+accountName+"/quotas?pageSize=1", nil, headers)
	if restQuotaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping css list quota groups, got %d body=%s", restQuotaResp.StatusCode, string(providerContractBody(t, restQuotaResp)))
	}
	restQuotaBody := providerContractJSONMap(t, restQuotaResp)
	restQuotaGroups, ok := restQuotaBody["quotaGroups"].([]any)
	if !ok || len(restQuotaGroups) == 0 {
		t.Fatalf("expected quotaGroups in rest payload, got %#v", restQuotaBody["quotaGroups"])
	}
	restFirstQuotaGroup, _ := restQuotaGroups[0].(map[string]any)
	restFirstQuotaGroupName, _ := restFirstQuotaGroup["name"].(string)

	successListQuotaReq := &shoppingcsspb.ListQuotaGroupsRequest{
		Parent:   accountName,
		PageSize: 1,
	}
	var successListQuotaResp shoppingcsspb.ListQuotaGroupsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSListQuotaGroupsMethod, successListQuotaReq, &successListQuotaResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shopping css list quota groups, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListQuotaResp.GetQuotaGroups()) != 1 {
		t.Fatalf("expected one grpc quota group, got %d", len(successListQuotaResp.GetQuotaGroups()))
	}
	if successListQuotaResp.GetQuotaGroups()[0].GetName() != restFirstQuotaGroupName {
		t.Fatalf("expected grpc quota group name %q to match rest %q", successListQuotaResp.GetQuotaGroups()[0].GetName(), restFirstQuotaGroupName)
	}

	restInsertResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+accountName+"/cssProductInputs:insert", []byte(`{
		"rawProvidedId":"sku-1",
		"contentLanguage":"en",
		"feedLabel":"US",
		"attributes":{"title":"Stackyard Tee"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-css",
	})
	if restInsertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping css insert css product input, got %d body=%s", restInsertResp.StatusCode, string(providerContractBody(t, restInsertResp)))
	}
	restInsertBody := providerContractJSONMap(t, restInsertResp)
	restInsertName, _ := restInsertBody["name"].(string)

	successInsertReq := &shoppingcsspb.InsertCssProductInputRequest{
		Parent: accountName,
		CssProductInput: &shoppingcsspb.CssProductInput{
			RawProvidedId:   "sku-1",
			ContentLanguage: "en",
			FeedLabel:       "US",
			Attributes:      &shoppingcsspb.Attributes{Title: proto.String("Stackyard Tee")},
		},
	}
	var successInsertResp shoppingcsspb.CssProductInput
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSInsertCssProductInputMethod, successInsertReq, &successInsertResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for shopping css insert css product input, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successInsertResp.GetName() != restInsertName {
		t.Fatalf("expected grpc css product input name %q to match rest %q", successInsertResp.GetName(), restInsertName)
	}

	invalidCreateLabelReq := &shoppingcsspb.CreateAccountLabelRequest{
		Parent:       accountName,
		AccountLabel: &shoppingcsspb.AccountLabel{Description: proto.String("missing display")},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSCreateAccountLabelMethod, invalidCreateLabelReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "display_name-required") {
		t.Fatalf("expected grpc invalid argument for shopping css create account label, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingDeleteLabelReq := &shoppingcsspb.DeleteAccountLabelRequest{
		Name: accountName + "/labels/missing-label",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSDeleteAccountLabelMethod, missingDeleteLabelReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "label-not-found") {
		t.Fatalf("expected grpc not found for shopping css delete account label, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	duplicateInsertReq := &shoppingcsspb.InsertCssProductInputRequest{
		Parent: accountName,
		CssProductInput: &shoppingcsspb.CssProductInput{
			RawProvidedId:   "existing-sku",
			ContentLanguage: "en",
			FeedLabel:       "US",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingCSSInsertCssProductInputMethod, duplicateInsertReq, nil)
	if grpcStatus != "6" || !strings.Contains(grpcMessage, "input-already-exists") {
		t.Fatalf("expected grpc already exists for shopping css insert css product input, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantAccounts(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	}
	accountName := "accounts/123456"
	userCollection := accountName + "/users"
	programName := accountName + "/programs/free-listings"

	restAccountResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/"+accountName, nil, headers)
	if restAccountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant accounts get account, got %d body=%s", restAccountResp.StatusCode, string(providerContractBody(t, restAccountResp)))
	}
	restAccountBody := providerContractJSONMap(t, restAccountResp)
	restAccountName, _ := restAccountBody["name"].(string)

	getAccountReq := &accountspb.GetAccountRequest{Name: accountName}
	var getAccountResp accountspb.Account
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, "/google.shopping.merchant.accounts.v1.AccountsService/GetAccount", getAccountReq, &getAccountResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant get account, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getAccountResp.GetName() != restAccountName {
		t.Fatalf("expected grpc account name %q to match rest %q", getAccountResp.GetName(), restAccountName)
	}

	restUsersResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/"+userCollection+"?pageSize=1", nil, headers)
	if restUsersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant accounts list users, got %d body=%s", restUsersResp.StatusCode, string(providerContractBody(t, restUsersResp)))
	}
	restUsersBody := providerContractJSONMap(t, restUsersResp)
	restUsers, ok := restUsersBody["users"].([]any)
	if !ok || len(restUsers) == 0 {
		t.Fatalf("expected users array in rest payload, got %#v", restUsersBody["users"])
	}
	restFirstUser, _ := restUsers[0].(map[string]any)
	restFirstUserName, _ := restFirstUser["name"].(string)

	listUsersReq := &accountspb.ListUsersRequest{Parent: accountName, PageSize: 1}
	var listUsersResp accountspb.ListUsersResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, "/google.shopping.merchant.accounts.v1.UserService/ListUsers", listUsersReq, &listUsersResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant list users, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listUsersResp.GetUsers()) != 1 {
		t.Fatalf("expected one grpc user, got %d", len(listUsersResp.GetUsers()))
	}
	if listUsersResp.GetUsers()[0].GetName() != restFirstUserName {
		t.Fatalf("expected grpc first user name %q to match rest %q", listUsersResp.GetUsers()[0].GetName(), restFirstUserName)
	}

	restEnableResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/accounts/v1/"+programName+":enable", []byte(`{"name":"accounts/123456/programs/free-listings"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	})
	if restEnableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant accounts enable program, got %d body=%s", restEnableResp.StatusCode, string(providerContractBody(t, restEnableResp)))
	}
	restEnableBody := providerContractJSONMap(t, restEnableResp)
	restEnabledName, _ := restEnableBody["name"].(string)

	enableProgramReq := &accountspb.EnableProgramRequest{Name: programName}
	var enableProgramResp accountspb.Program
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, "/google.shopping.merchant.accounts.v1.ProgramsService/EnableProgram", enableProgramReq, &enableProgramResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant enable program, got %q message=%q", grpcStatus, grpcMessage)
	}
	if enableProgramResp.GetName() != restEnabledName {
		t.Fatalf("expected grpc enabled program name %q to match rest %q", enableProgramResp.GetName(), restEnabledName)
	}

	invalidCreateUserReq := &accountspb.CreateUserRequest{
		UserId: "owner@example.com",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, "/google.shopping.merchant.accounts.v1.UserService/CreateUser", invalidCreateUserReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant create user, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantConversions(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	}
	collection := "/gcp/conversions/v1/accounts/123456/conversionSources"

	restListResp := providerContractRequest(t, ts, http.MethodGet, collection+"?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant conversions list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restItems, ok := restListBody["conversionSources"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected conversionSources list in rest payload, got %#v", restListBody["conversionSources"])
	}
	restFirst, _ := restItems[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)

	listReq := &conversionspb.ListConversionSourcesRequest{
		Parent:   "accounts/123456",
		PageSize: 1,
	}
	var listResp conversionspb.ListConversionSourcesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantConversionsListConversionSourcesMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant conversions list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetConversionSources()) != 1 {
		t.Fatalf("expected one grpc conversion source, got %d", len(listResp.GetConversionSources()))
	}
	if listResp.GetConversionSources()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc first conversion source name %q to match rest %q", listResp.GetConversionSources()[0].GetName(), restFirstName)
	}

	getReq := &conversionspb.GetConversionSourceRequest{Name: restFirstName}
	var getResp conversionspb.ConversionSource
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantConversionsGetConversionSourceMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant conversions get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restFirstName {
		t.Fatalf("expected grpc conversion source name %q to match rest %q", getResp.GetName(), restFirstName)
	}

	createReq := &conversionspb.CreateConversionSourceRequest{
		Parent: "accounts/123456",
		ConversionSource: &conversionspb.ConversionSource{
			SourceData: &conversionspb.ConversionSource_MerchantCenterDestination{
				MerchantCenterDestination: &conversionspb.MerchantCenterDestination{
					DisplayName:  "Primary Destination",
					CurrencyCode: "USD",
					AttributionSettings: &conversionspb.AttributionSettings{
						AttributionLookbackWindowDays: 30,
						AttributionModel:              conversionspb.AttributionSettings_CROSS_CHANNEL_LAST_CLICK,
					},
				},
			},
		},
	}
	var createResp conversionspb.ConversionSource
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantConversionsCreateConversionSourceMethod, createReq, &createResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant conversions create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(createResp.GetName(), "/conversionSources/mcdn:") {
		t.Fatalf("expected grpc create name to be mcdn source, got %q", createResp.GetName())
	}

	updateReq := &conversionspb.UpdateConversionSourceRequest{
		ConversionSource: &conversionspb.ConversionSource{
			Name: createResp.GetName(),
			SourceData: &conversionspb.ConversionSource_MerchantCenterDestination{
				MerchantCenterDestination: &conversionspb.MerchantCenterDestination{
					DisplayName: "Primary Destination Updated",
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantConversionsUpdateConversionSourceMethod, updateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "update_mask-required") {
		t.Fatalf("expected grpc invalid argument for merchant conversions update missing update_mask, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantDatasources(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	}
	collection := "/gcp/datasources/v1/accounts/123456/dataSources"

	restListResp := providerContractRequest(t, ts, http.MethodGet, collection+"?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant datasources list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restItems, ok := restListBody["dataSources"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected dataSources list in rest payload, got %#v", restListBody["dataSources"])
	}
	restFirst, _ := restItems[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)

	listReq := &datasourcespb.ListDataSourcesRequest{
		Parent:   "accounts/123456",
		PageSize: 1,
	}
	var listResp datasourcespb.ListDataSourcesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantDatasourcesListDataSourcesMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant datasources list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetDataSources()) != 1 {
		t.Fatalf("expected one grpc data source, got %d", len(listResp.GetDataSources()))
	}
	if listResp.GetDataSources()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc first data source name %q to match rest %q", listResp.GetDataSources()[0].GetName(), restFirstName)
	}

	restUploadResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/datasources/v1/accounts/123456/dataSources/1001/fileUploads/latest", nil, headers)
	if restUploadResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant datasources get file upload, got %d body=%s", restUploadResp.StatusCode, string(providerContractBody(t, restUploadResp)))
	}
	restUploadBody := providerContractJSONMap(t, restUploadResp)
	restUploadName, _ := restUploadBody["name"].(string)

	uploadReq := &datasourcespb.GetFileUploadRequest{Name: restUploadName}
	var uploadResp datasourcespb.FileUpload
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantDatasourcesGetFileUploadMethod, uploadReq, &uploadResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant datasources get file upload, got %q message=%q", grpcStatus, grpcMessage)
	}
	if uploadResp.GetName() != restUploadName {
		t.Fatalf("expected grpc file upload name %q to match rest %q", uploadResp.GetName(), restUploadName)
	}

	createReq := &datasourcespb.CreateDataSourceRequest{
		Parent: "accounts/123456",
		DataSource: &datasourcespb.DataSource{
			DisplayName: "Stackyard Created Data Source",
			Type: &datasourcespb.DataSource_PrimaryProductDataSource{
				PrimaryProductDataSource: &datasourcespb.PrimaryProductDataSource{
					FeedLabel:       proto.String("US"),
					ContentLanguage: proto.String("en"),
					Countries:       []string{"US"},
				},
			},
		},
	}
	var createResp datasourcespb.DataSource
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantDatasourcesCreateDataSourceMethod, createReq, &createResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant datasources create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(createResp.GetName(), "/dataSources/") {
		t.Fatalf("expected grpc create name to include /dataSources/, got %q", createResp.GetName())
	}

	fetchNoFileReq := &datasourcespb.FetchDataSourceRequest{
		Name: "accounts/123456/dataSources/nofile",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantDatasourcesFetchDataSourceMethod, fetchNoFileReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "fetch-requires-file-input") {
		t.Fatalf("expected grpc failed precondition for merchant datasources fetch, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidListReq := &datasourcespb.ListDataSourcesRequest{
		Parent: "accounts",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantDatasourcesListDataSourcesMethod, invalidListReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant datasources list, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantInventories(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	}
	parent := "accounts/123456/products/sku-1001"

	restLocalResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/inventories/v1/"+parent+"/localInventories?pageSize=1", nil, headers)
	if restLocalResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant inventories local list, got %d body=%s", restLocalResp.StatusCode, string(providerContractBody(t, restLocalResp)))
	}
	restLocalBody := providerContractJSONMap(t, restLocalResp)
	restLocalItems, ok := restLocalBody["localInventories"].([]any)
	if !ok || len(restLocalItems) == 0 {
		t.Fatalf("expected localInventories list in rest payload, got %#v", restLocalBody["localInventories"])
	}
	restFirstLocal, _ := restLocalItems[0].(map[string]any)
	restFirstLocalName, _ := restFirstLocal["name"].(string)

	localListReq := &inventoriespb.ListLocalInventoriesRequest{
		Parent:   parent,
		PageSize: 1,
	}
	var localListResp inventoriespb.ListLocalInventoriesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesListLocalInventoriesMethod, localListReq, &localListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant inventories local list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(localListResp.GetLocalInventories()) != 1 {
		t.Fatalf("expected one grpc local inventory, got %d", len(localListResp.GetLocalInventories()))
	}
	if localListResp.GetLocalInventories()[0].GetName() != restFirstLocalName {
		t.Fatalf("expected grpc first local inventory name %q to match rest %q", localListResp.GetLocalInventories()[0].GetName(), restFirstLocalName)
	}

	localInsertReq := &inventoriespb.InsertLocalInventoryRequest{
		Parent: parent,
		LocalInventory: &inventoriespb.LocalInventory{
			StoreCode: "store-nyc",
			LocalInventoryAttributes: &inventoriespb.LocalInventoryAttributes{
				Quantity: proto.Int64(8),
			},
		},
	}
	var localInsertResp inventoriespb.LocalInventory
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesInsertLocalInventoryMethod, localInsertReq, &localInsertResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant inventories local insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(localInsertResp.GetName(), "/localInventories/store-nyc") {
		t.Fatalf("expected grpc local insert name to include store code, got %q", localInsertResp.GetName())
	}

	localDeleteMissingReq := &inventoriespb.DeleteLocalInventoryRequest{
		Name: "accounts/123456/products/sku-1001/localInventories/missing-store",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesDeleteLocalInventoryMethod, localDeleteMissingReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "local-inventory-not-found") {
		t.Fatalf("expected grpc not found for merchant inventories local delete, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	restRegionalResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/inventories/v1/"+parent+"/regionalInventories?pageSize=1", nil, headers)
	if restRegionalResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant inventories regional list, got %d body=%s", restRegionalResp.StatusCode, string(providerContractBody(t, restRegionalResp)))
	}
	restRegionalBody := providerContractJSONMap(t, restRegionalResp)
	restRegionalItems, ok := restRegionalBody["regionalInventories"].([]any)
	if !ok || len(restRegionalItems) == 0 {
		t.Fatalf("expected regionalInventories list in rest payload, got %#v", restRegionalBody["regionalInventories"])
	}
	restFirstRegional, _ := restRegionalItems[0].(map[string]any)
	restFirstRegionalName, _ := restFirstRegional["name"].(string)

	regionalListReq := &inventoriespb.ListRegionalInventoriesRequest{
		Parent:   parent,
		PageSize: 1,
	}
	var regionalListResp inventoriespb.ListRegionalInventoriesResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesListRegionalInventoriesMethod, regionalListReq, &regionalListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant inventories regional list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(regionalListResp.GetRegionalInventories()) != 1 {
		t.Fatalf("expected one grpc regional inventory, got %d", len(regionalListResp.GetRegionalInventories()))
	}
	if regionalListResp.GetRegionalInventories()[0].GetName() != restFirstRegionalName {
		t.Fatalf("expected grpc first regional inventory name %q to match rest %q", regionalListResp.GetRegionalInventories()[0].GetName(), restFirstRegionalName)
	}

	regionalInsertReq := &inventoriespb.InsertRegionalInventoryRequest{
		Parent: parent,
		RegionalInventory: &inventoriespb.RegionalInventory{
			Region: "us-east1",
		},
	}
	var regionalInsertResp inventoriespb.RegionalInventory
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesInsertRegionalInventoryMethod, regionalInsertReq, &regionalInsertResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant inventories regional insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(regionalInsertResp.GetName(), "/regionalInventories/us-east1") {
		t.Fatalf("expected grpc regional insert name to include region, got %q", regionalInsertResp.GetName())
	}

	invalidRegionalInsertReq := &inventoriespb.InsertRegionalInventoryRequest{
		Parent: parent,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantInventoriesInsertRegionalInventoryMethod, invalidRegionalInsertReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "regional_inventory-required") {
		t.Fatalf("expected grpc invalid argument for merchant inventories regional insert, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantIssueresolution(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	}

	restRenderAccountResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:renderaccountissues", []byte(`{
		"contentOption":"CONTENT_OPTION_UNSPECIFIED"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if restRenderAccountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant issueresolution render account issues, got %d body=%s", restRenderAccountResp.StatusCode, string(providerContractBody(t, restRenderAccountResp)))
	}
	restRenderAccountBody := providerContractJSONMap(t, restRenderAccountResp)
	restRenderedIssues, ok := restRenderAccountBody["renderedIssues"].([]any)
	if !ok || len(restRenderedIssues) == 0 {
		t.Fatalf("expected renderedIssues list in rest payload, got %#v", restRenderAccountBody["renderedIssues"])
	}
	restRenderedIssue, _ := restRenderedIssues[0].(map[string]any)
	restRenderedTitle, _ := restRenderedIssue["title"].(string)

	renderAccountReq := &issueresolutionpb.RenderAccountIssuesRequest{
		Name: "accounts/123456",
	}
	var renderAccountResp issueresolutionpb.RenderAccountIssuesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantIssueresolutionRenderAccountIssuesMethod, renderAccountReq, &renderAccountResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant issueresolution render account issues, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(renderAccountResp.GetRenderedIssues()) != 1 {
		t.Fatalf("expected one grpc rendered account issue, got %d", len(renderAccountResp.GetRenderedIssues()))
	}
	if renderAccountResp.GetRenderedIssues()[0].GetTitle() != restRenderedTitle {
		t.Fatalf("expected grpc rendered account issue title %q to match rest %q", renderAccountResp.GetRenderedIssues()[0].GetTitle(), restRenderedTitle)
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/issueresolution/v1/accounts/123456/aggregateProductStatuses?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant issueresolution list aggregate product statuses, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restStatuses, ok := restListBody["aggregateProductStatuses"].([]any)
	if !ok || len(restStatuses) == 0 {
		t.Fatalf("expected aggregateProductStatuses list in rest payload, got %#v", restListBody["aggregateProductStatuses"])
	}
	restFirstStatus, _ := restStatuses[0].(map[string]any)
	restFirstStatusName, _ := restFirstStatus["name"].(string)

	listReq := &issueresolutionpb.ListAggregateProductStatusesRequest{
		Parent:   "accounts/123456",
		PageSize: 1,
	}
	var listResp issueresolutionpb.ListAggregateProductStatusesResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantIssueresolutionListAggregateProductStatusesMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant issueresolution list aggregate product statuses, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetAggregateProductStatuses()) != 1 {
		t.Fatalf("expected one grpc aggregate product status, got %d", len(listResp.GetAggregateProductStatuses()))
	}
	if listResp.GetAggregateProductStatuses()[0].GetName() != restFirstStatusName {
		t.Fatalf("expected grpc aggregate product status name %q to match rest %q", listResp.GetAggregateProductStatuses()[0].GetName(), restFirstStatusName)
	}

	restTriggerResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/issueresolution/v1/accounts/123456:triggeraction", []byte(`{
		"actionContext":"ctx-account-review",
		"actionInput":{
			"actionFlowId":"flow-review",
			"inputValues":[
				{"inputFieldId":"explanation","textInputValue":{"value":"All issues were fixed and validated."}}
			]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-issueresolution",
	})
	if restTriggerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant issueresolution triggeraction, got %d body=%s", restTriggerResp.StatusCode, string(providerContractBody(t, restTriggerResp)))
	}
	restTriggerBody := providerContractJSONMap(t, restTriggerResp)
	restTriggerMessage, _ := restTriggerBody["message"].(string)

	triggerReq := &issueresolutionpb.TriggerActionRequest{
		Name: "accounts/123456",
		Payload: &issueresolutionpb.TriggerActionPayload{
			ActionContext: "ctx-account-review",
			ActionInput: &issueresolutionpb.ActionInput{
				ActionFlowId: "flow-review",
				InputValues: []*issueresolutionpb.InputValue{
					{
						InputFieldId: "explanation",
						Value: &issueresolutionpb.InputValue_TextInputValue_{
							TextInputValue: &issueresolutionpb.InputValue_TextInputValue{
								Value: "All issues were fixed and validated.",
							},
						},
					},
				},
			},
		},
	}
	var triggerResp issueresolutionpb.TriggerActionResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantIssueresolutionTriggerActionMethod, triggerReq, &triggerResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant issueresolution triggeraction, got %q message=%q", grpcStatus, grpcMessage)
	}
	if triggerResp.GetMessage() != restTriggerMessage {
		t.Fatalf("expected grpc triggeraction message %q to match rest %q", triggerResp.GetMessage(), restTriggerMessage)
	}

	invalidTriggerReq := &issueresolutionpb.TriggerActionRequest{
		Name: "accounts/123456",
		Payload: &issueresolutionpb.TriggerActionPayload{
			ActionContext: "ctx-account-review",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantIssueresolutionTriggerActionMethod, invalidTriggerReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "payload.action_input-required") {
		t.Fatalf("expected grpc invalid argument for merchant issueresolution triggeraction, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionTriggerReq := &issueresolutionpb.TriggerActionRequest{
		Name: "accounts/123456",
		Payload: &issueresolutionpb.TriggerActionPayload{
			ActionContext: "ctx-account-review-locked",
			ActionInput: &issueresolutionpb.ActionInput{
				ActionFlowId: "flow-review",
				InputValues: []*issueresolutionpb.InputValue{
					{
						InputFieldId: "explanation",
						Value: &issueresolutionpb.InputValue_TextInputValue_{
							TextInputValue: &issueresolutionpb.InputValue_TextInputValue{Value: "Locked state"},
						},
					},
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantIssueresolutionTriggerActionMethod, preconditionTriggerReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "action-context-locked") {
		t.Fatalf("expected grpc failed precondition for merchant issueresolution triggeraction, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantLFP(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	}
	parent := "accounts/123456"

	restInsertStoreResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpStores:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"storeAddress":"1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if restInsertStoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant lfp insert store, got %d body=%s", restInsertStoreResp.StatusCode, string(providerContractBody(t, restInsertStoreResp)))
	}
	restInsertStoreBody := providerContractJSONMap(t, restInsertStoreResp)
	restStoreName, _ := restInsertStoreBody["name"].(string)

	insertStoreReq := &lfppb.InsertLfpStoreRequest{
		Parent: parent,
		LfpStore: &lfppb.LfpStore{
			TargetAccount: 567890,
			StoreCode:     "store-nyc",
			StoreAddress:  "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		},
	}
	var insertStoreResp lfppb.LfpStore
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPInsertLfpStoreMethod, insertStoreReq, &insertStoreResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant lfp insert store, got %q message=%q", grpcStatus, grpcMessage)
	}
	if insertStoreResp.GetName() != restStoreName {
		t.Fatalf("expected grpc store name %q to match rest %q", insertStoreResp.GetName(), restStoreName)
	}

	restListStoresResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/"+parent+"/lfpStores?targetAccount=567890&pageSize=1", nil, headers)
	if restListStoresResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant lfp list stores, got %d body=%s", restListStoresResp.StatusCode, string(providerContractBody(t, restListStoresResp)))
	}
	restListStoresBody := providerContractJSONMap(t, restListStoresResp)
	restStores, ok := restListStoresBody["lfpStores"].([]any)
	if !ok || len(restStores) == 0 {
		t.Fatalf("expected lfpStores list in rest payload, got %#v", restListStoresBody["lfpStores"])
	}
	restFirstStore, _ := restStores[0].(map[string]any)
	restFirstStoreName, _ := restFirstStore["name"].(string)

	listStoresReq := &lfppb.ListLfpStoresRequest{
		Parent:        parent,
		TargetAccount: 567890,
		PageSize:      1,
	}
	var listStoresResp lfppb.ListLfpStoresResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPListLfpStoresMethod, listStoresReq, &listStoresResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant lfp list stores, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listStoresResp.GetLfpStores()) != 1 {
		t.Fatalf("expected one grpc lfp store, got %d", len(listStoresResp.GetLfpStores()))
	}
	if listStoresResp.GetLfpStores()[0].GetName() != restFirstStoreName {
		t.Fatalf("expected grpc first lfp store name %q to match rest %q", listStoresResp.GetLfpStores()[0].GetName(), restFirstStoreName)
	}

	restInsertInventoryResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpInventories:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"availability":"in stock",
		"price":{"currencyCode":"USD","amountMicros":"12990000"},
		"quantity":"7"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if restInsertInventoryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant lfp insert inventory, got %d body=%s", restInsertInventoryResp.StatusCode, string(providerContractBody(t, restInsertInventoryResp)))
	}
	restInsertInventoryBody := providerContractJSONMap(t, restInsertInventoryResp)
	restInventoryName, _ := restInsertInventoryBody["name"].(string)

	insertInventoryReq := &lfppb.InsertLfpInventoryRequest{
		Parent: parent,
		LfpInventory: &lfppb.LfpInventory{
			TargetAccount:   567890,
			StoreCode:       "store-nyc",
			OfferId:         "offer-1001",
			RegionCode:      "US",
			ContentLanguage: "en",
			Availability:    "in stock",
			Price: &shoppingtypepb.Price{
				CurrencyCode: proto.String("USD"),
				AmountMicros: proto.Int64(12990000),
			},
			Quantity: proto.Int64(7),
		},
	}
	var insertInventoryResp lfppb.LfpInventory
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPInsertLfpInventoryMethod, insertInventoryReq, &insertInventoryResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant lfp insert inventory, got %q message=%q", grpcStatus, grpcMessage)
	}
	if insertInventoryResp.GetName() != restInventoryName {
		t.Fatalf("expected grpc inventory name %q to match rest %q", insertInventoryResp.GetName(), restInventoryName)
	}

	restInsertSaleResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpSales:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"gtin":"00012345678905",
		"price":{"currencyCode":"USD","amountMicros":"14990000"},
		"quantity":"1",
		"saleTime":"2026-01-01T12:34:56Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if restInsertSaleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant lfp insert sale, got %d body=%s", restInsertSaleResp.StatusCode, string(providerContractBody(t, restInsertSaleResp)))
	}
	restInsertSaleBody := providerContractJSONMap(t, restInsertSaleResp)
	restSaleName, _ := restInsertSaleBody["name"].(string)

	insertSaleReq := &lfppb.InsertLfpSaleRequest{
		Parent: parent,
		LfpSale: &lfppb.LfpSale{
			TargetAccount:   567890,
			StoreCode:       "store-nyc",
			OfferId:         "offer-1001",
			RegionCode:      "US",
			ContentLanguage: "en",
			Gtin:            "00012345678905",
			Price: &shoppingtypepb.Price{
				CurrencyCode: proto.String("USD"),
				AmountMicros: proto.Int64(14990000),
			},
			Quantity: 1,
			SaleTime: timestamppb.New(time.Date(2026, time.January, 1, 12, 34, 56, 0, time.UTC)),
		},
	}
	var insertSaleResp lfppb.LfpSale
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPInsertLfpSaleMethod, insertSaleReq, &insertSaleResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant lfp insert sale, got %q message=%q", grpcStatus, grpcMessage)
	}
	if insertSaleResp.GetName() != restSaleName {
		t.Fatalf("expected grpc sale name %q to match rest %q", insertSaleResp.GetName(), restSaleName)
	}

	restMerchantStateResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/"+parent+"/lfpMerchantStates/567890", nil, headers)
	if restMerchantStateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant lfp get merchant state, got %d body=%s", restMerchantStateResp.StatusCode, string(providerContractBody(t, restMerchantStateResp)))
	}
	restMerchantStateBody := providerContractJSONMap(t, restMerchantStateResp)
	restMerchantStateName, _ := restMerchantStateBody["name"].(string)

	getMerchantStateReq := &lfppb.GetLfpMerchantStateRequest{Name: "accounts/123456/lfpMerchantStates/567890"}
	var getMerchantStateResp lfppb.LfpMerchantState
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPGetLfpMerchantStateMethod, getMerchantStateReq, &getMerchantStateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant lfp get merchant state, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getMerchantStateResp.GetName() != restMerchantStateName {
		t.Fatalf("expected grpc merchant state name %q to match rest %q", getMerchantStateResp.GetName(), restMerchantStateName)
	}

	invalidListReq := &lfppb.ListLfpStoresRequest{Parent: parent}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPListLfpStoresMethod, invalidListReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "target_account-required") {
		t.Fatalf("expected grpc invalid argument for merchant lfp list stores, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingStoreReq := &lfppb.GetLfpStoreRequest{Name: "accounts/123456/lfpStores/567890~missing-store"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantLFPGetLfpStoreMethod, missingStoreReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "lfp-store-not-found") {
		t.Fatalf("expected grpc not found for merchant lfp get store, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantNotifications(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	}
	parent := "accounts/123456"

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/notifications/v1/"+parent+"/notificationsubscriptions", []byte(`{
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant notifications create, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restSubscriptionName, _ := restCreateBody["name"].(string)
	restCallbackURI, _ := restCreateBody["callBackUri"].(string)

	createReq := &notificationspb.CreateNotificationSubscriptionRequest{
		Parent: parent,
		NotificationSubscription: &notificationspb.NotificationSubscription{
			RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
			CallBackUri:     "https://example.com/hooks/merchant-notifications",
			InterestedIn: &notificationspb.NotificationSubscription_AllManagedAccounts{
				AllManagedAccounts: true,
			},
		},
	}
	var createResp notificationspb.NotificationSubscription
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsCreateNotificationSubscriptionMethod, createReq, &createResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createResp.GetName() != restSubscriptionName {
		t.Fatalf("expected grpc created name %q to match rest %q", createResp.GetName(), restSubscriptionName)
	}
	if createResp.GetCallBackUri() != restCallbackURI {
		t.Fatalf("expected grpc callback URI %q to match rest %q", createResp.GetCallBackUri(), restCallbackURI)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/"+restSubscriptionName, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant notifications get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restGetName, _ := restGetBody["name"].(string)

	getReq := &notificationspb.GetNotificationSubscriptionRequest{Name: restSubscriptionName}
	var getResp notificationspb.NotificationSubscription
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsGetNotificationSubscriptionMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restGetName {
		t.Fatalf("expected grpc get name %q to match rest %q", getResp.GetName(), restGetName)
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/"+parent+"/notificationsubscriptions?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant notifications list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restListItems, ok := restListBody["notificationSubscriptions"].([]any)
	if !ok || len(restListItems) == 0 {
		t.Fatalf("expected notificationSubscriptions list in rest payload, got %#v", restListBody["notificationSubscriptions"])
	}
	restFirst, _ := restListItems[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)

	listReq := &notificationspb.ListNotificationSubscriptionsRequest{
		Parent:   parent,
		PageSize: 1,
	}
	var listResp notificationspb.ListNotificationSubscriptionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsListNotificationSubscriptionsMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetNotificationSubscriptions()) != 1 {
		t.Fatalf("expected one grpc notification subscription, got %d", len(listResp.GetNotificationSubscriptions()))
	}
	if listResp.GetNotificationSubscriptions()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc first subscription name %q to match rest %q", listResp.GetNotificationSubscriptions()[0].GetName(), restFirstName)
	}

	restUpdateResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/notifications/v1/"+restSubscriptionName+"?updateMask=callBackUri", []byte(`{
		"name":"`+restSubscriptionName+`",
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications-updated",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if restUpdateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant notifications update, got %d body=%s", restUpdateResp.StatusCode, string(providerContractBody(t, restUpdateResp)))
	}
	restUpdateBody := providerContractJSONMap(t, restUpdateResp)
	restUpdatedCallback, _ := restUpdateBody["callBackUri"].(string)

	updateReq := &notificationspb.UpdateNotificationSubscriptionRequest{
		NotificationSubscription: &notificationspb.NotificationSubscription{
			Name:            restSubscriptionName,
			RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
			CallBackUri:     "https://example.com/hooks/merchant-notifications-updated",
			InterestedIn: &notificationspb.NotificationSubscription_AllManagedAccounts{
				AllManagedAccounts: true,
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"callBackUri"}},
	}
	var updateResp notificationspb.NotificationSubscription
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsUpdateNotificationSubscriptionMethod, updateReq, &updateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications update, got %q message=%q", grpcStatus, grpcMessage)
	}
	if updateResp.GetCallBackUri() != restUpdatedCallback {
		t.Fatalf("expected grpc updated callback URI %q to match rest %q", updateResp.GetCallBackUri(), restUpdatedCallback)
	}

	restHealthResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/"+restSubscriptionName+":getHealth", nil, headers)
	if restHealthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant notifications get health, got %d body=%s", restHealthResp.StatusCode, string(providerContractBody(t, restHealthResp)))
	}
	restHealthBody := providerContractJSONMap(t, restHealthResp)
	restHealthName, _ := restHealthBody["name"].(string)

	healthReq := &notificationspb.GetNotificationSubscriptionHealthMetricsRequest{Name: restSubscriptionName}
	var healthResp notificationspb.NotificationSubscriptionHealthMetrics
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsGetNotificationSubscriptionHealthMetricsMethod, healthReq, &healthResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications get health, got %q message=%q", grpcStatus, grpcMessage)
	}
	if healthResp.GetName() != restHealthName {
		t.Fatalf("expected grpc health name %q to match rest %q", healthResp.GetName(), restHealthName)
	}
	if healthResp.GetAcknowledgedMessagesCount() != 42 || healthResp.GetUndeliveredMessagesCount() != 3 || healthResp.GetOldestUnacknowledgedMessageWaitingTime() != 3600 {
		t.Fatalf("expected deterministic grpc health metrics, got ack=%d undelivered=%d oldest_wait=%d", healthResp.GetAcknowledgedMessagesCount(), healthResp.GetUndeliveredMessagesCount(), healthResp.GetOldestUnacknowledgedMessageWaitingTime())
	}

	deleteReq := &notificationspb.DeleteNotificationSubscriptionRequest{Name: restSubscriptionName}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsDeleteNotificationSubscriptionMethod, deleteReq, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant notifications delete, got %q message=%q", grpcStatus, grpcMessage)
	}

	invalidUpdateReq := &notificationspb.UpdateNotificationSubscriptionRequest{
		NotificationSubscription: &notificationspb.NotificationSubscription{
			Name:            restSubscriptionName,
			RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
			CallBackUri:     "https://example.com/hooks/merchant-notifications-updated",
			InterestedIn: &notificationspb.NotificationSubscription_AllManagedAccounts{
				AllManagedAccounts: true,
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsUpdateNotificationSubscriptionMethod, invalidUpdateReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "update_mask-unsupported") {
		t.Fatalf("expected grpc failed precondition for merchant notifications update unsupported update mask, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidCreateReq := &notificationspb.CreateNotificationSubscriptionRequest{
		Parent: parent,
		NotificationSubscription: &notificationspb.NotificationSubscription{
			RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
			CallBackUri:     "https://example.com/hooks/merchant-notifications",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsCreateNotificationSubscriptionMethod, invalidCreateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "interested_in-required") {
		t.Fatalf("expected grpc invalid argument for merchant notifications create missing interested_in, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingGetReq := &notificationspb.GetNotificationSubscriptionRequest{Name: "accounts/123456/notificationsubscriptions/missing-sub"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantNotificationsGetNotificationSubscriptionMethod, missingGetReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "notification-subscription-not-found") {
		t.Fatalf("expected grpc not found for merchant notifications get missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantOrdertracking(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	}
	parent := "accounts/123456"

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/"+parent+"/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}]
		}
	}`), headers)
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant ordertracking create, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restOrderID, _ := restCreateBody["orderId"].(string)
	restShippingInfo, ok := restCreateBody["shippingInfo"].([]any)
	if !ok || len(restShippingInfo) == 0 {
		t.Fatalf("expected shippingInfo list in rest response, got %#v", restCreateBody["shippingInfo"])
	}
	restShipment, _ := restShippingInfo[0].(map[string]any)
	restShipmentID, _ := restShipment["shipmentId"].(string)

	createReq := &ordertrackingpb.CreateOrderTrackingSignalRequest{
		Parent: parent,
		OrderTrackingSignal: &ordertrackingpb.OrderTrackingSignal{
			OrderCreatedTime: &datetimepb.DateTime{
				Year:    2026,
				Month:   1,
				Day:     2,
				Hours:   3,
				Minutes: 4,
				Seconds: 5,
			},
			OrderId: "ORDER-1001",
			ShippingInfo: []*ordertrackingpb.OrderTrackingSignal_ShippingInfo{
				{
					ShipmentId:       "SHIP-1001",
					TrackingId:       "TRACK-1001",
					Carrier:          "UPS",
					ShippingStatus:   ordertrackingpb.OrderTrackingSignal_ShippingInfo_SHIPPED,
					OriginPostalCode: "94043",
					OriginRegionCode: "US",
				},
			},
			LineItems: []*ordertrackingpb.OrderTrackingSignal_LineItemDetails{
				{
					LineItemId: "line-1",
					ProductId:  "online:en:US:offer-1001",
					Quantity:   2,
				},
			},
		},
	}
	var createResp ordertrackingpb.OrderTrackingSignal
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod, createReq, &createResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant ordertracking create, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createResp.GetOrderId() != restOrderID {
		t.Fatalf("expected grpc order id %q to match rest %q", createResp.GetOrderId(), restOrderID)
	}
	if len(createResp.GetShippingInfo()) != 1 {
		t.Fatalf("expected one grpc shipment, got %d", len(createResp.GetShippingInfo()))
	}
	if createResp.GetShippingInfo()[0].GetShipmentId() != restShipmentID {
		t.Fatalf("expected grpc shipment id %q to match rest %q", createResp.GetShippingInfo()[0].GetShipmentId(), restShipmentID)
	}

	invalidReq := &ordertrackingpb.CreateOrderTrackingSignalRequest{
		Parent: parent,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "order_tracking_signal-required") {
		t.Fatalf("expected grpc invalid argument for merchant ordertracking missing payload, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	duplicateReq := &ordertrackingpb.CreateOrderTrackingSignalRequest{
		Parent: parent,
		OrderTrackingSignal: &ordertrackingpb.OrderTrackingSignal{
			OrderCreatedTime: &datetimepb.DateTime{Year: 2026},
			OrderId:          "duplicate-order",
			ShippingInfo: []*ordertrackingpb.OrderTrackingSignal_ShippingInfo{
				{
					ShipmentId:       "SHIP-DUP",
					TrackingId:       "TRACK-DUP",
					Carrier:          "UPS",
					ShippingStatus:   ordertrackingpb.OrderTrackingSignal_ShippingInfo_SHIPPED,
					OriginPostalCode: "94043",
					OriginRegionCode: "US",
				},
			},
			LineItems: []*ordertrackingpb.OrderTrackingSignal_LineItemDetails{
				{
					LineItemId: "line-dup",
					ProductId:  "online:en:US:offer-dup",
					Quantity:   1,
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod, duplicateReq, nil)
	if grpcStatus != "6" || !strings.Contains(grpcMessage, "order_tracking_signal-already-exists") {
		t.Fatalf("expected grpc already exists for merchant ordertracking duplicate signal, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &ordertrackingpb.CreateOrderTrackingSignalRequest{
		Parent: parent,
		OrderTrackingSignal: &ordertrackingpb.OrderTrackingSignal{
			OrderCreatedTime: &datetimepb.DateTime{Year: 2026},
			OrderId:          "order-precondition",
			ShippingInfo: []*ordertrackingpb.OrderTrackingSignal_ShippingInfo{
				{
					ShipmentId:       "SHIP-ONE",
					TrackingId:       "TRACK-ONE",
					Carrier:          "UPS",
					ShippingStatus:   ordertrackingpb.OrderTrackingSignal_ShippingInfo_SHIPPED,
					OriginPostalCode: "94043",
					OriginRegionCode: "US",
				},
			},
			LineItems: []*ordertrackingpb.OrderTrackingSignal_LineItemDetails{
				{
					LineItemId: "line-one",
					ProductId:  "online:en:US:offer-one",
					Quantity:   1,
				},
			},
			ShipmentLineItemMapping: []*ordertrackingpb.OrderTrackingSignal_ShipmentLineItemMapping{
				{
					ShipmentId: "missing-shipment",
					LineItemId: "line-one",
					Quantity:   1,
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "shipment_line_item_mapping-invalid") {
		t.Fatalf("expected grpc failed precondition for merchant ordertracking mapping mismatch, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantProducts(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	}
	parent := "accounts/123456"
	dataSource := "accounts/123456/dataSources/104628"
	productID := "en~US~sku-1001"
	productName := parent + "/products/" + productID
	productInputName := parent + "/productInputs/" + productID

	restInsertResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/products/v1/"+parent+"/productInputs:insert?dataSource="+dataSource, []byte(`{
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US",
		"productAttributes":{"title":"Stackyard SKU 1001"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if restInsertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant products insert, got %d body=%s", restInsertResp.StatusCode, string(providerContractBody(t, restInsertResp)))
	}
	restInsertBody := providerContractJSONMap(t, restInsertResp)
	restInputName, _ := restInsertBody["name"].(string)

	insertReq := &productspb.InsertProductInputRequest{
		Parent:     parent,
		DataSource: dataSource,
		ProductInput: &productspb.ProductInput{
			OfferId:         "sku-1001",
			ContentLanguage: "en",
			FeedLabel:       "US",
			ProductAttributes: &productspb.ProductAttributes{
				Title: proto.String("Stackyard SKU 1001"),
			},
		},
	}
	var insertResp productspb.ProductInput
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsInsertProductInputMethod, insertReq, &insertResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant products insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if insertResp.GetName() != restInputName {
		t.Fatalf("expected grpc insert name %q to match rest %q", insertResp.GetName(), restInputName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/"+productName, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant products get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restGetName, _ := restGetBody["name"].(string)

	getReq := &productspb.GetProductRequest{Name: productName}
	var getResp productspb.Product
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsGetProductMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant products get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restGetName {
		t.Fatalf("expected grpc product name %q to match rest %q", getResp.GetName(), restGetName)
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/"+parent+"/products?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant products list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restProducts, ok := restListBody["products"].([]any)
	if !ok || len(restProducts) == 0 {
		t.Fatalf("expected rest products list, got %#v", restListBody["products"])
	}
	restFirst, _ := restProducts[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)

	listReq := &productspb.ListProductsRequest{
		Parent:   parent,
		PageSize: 1,
	}
	var listResp productspb.ListProductsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsListProductsMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant products list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetProducts()) != 1 {
		t.Fatalf("expected one grpc product from list, got %d", len(listResp.GetProducts()))
	}
	if listResp.GetProducts()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc first product %q to match rest %q", listResp.GetProducts()[0].GetName(), restFirstName)
	}

	restUpdateResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/products/v1/"+productInputName+"?dataSource="+dataSource+"&updateMask=productAttributes.title", []byte(`{
		"name":"accounts/123456/productInputs/en~US~sku-1001",
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US",
		"productAttributes":{"title":"Stackyard SKU 1001 Updated"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if restUpdateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant products update, got %d body=%s", restUpdateResp.StatusCode, string(providerContractBody(t, restUpdateResp)))
	}
	restUpdateBody := providerContractJSONMap(t, restUpdateResp)
	restAttrs, _ := restUpdateBody["productAttributes"].(map[string]any)
	restUpdatedTitle, _ := restAttrs["title"].(string)

	updateReq := &productspb.UpdateProductInputRequest{
		ProductInput: &productspb.ProductInput{
			Name:            productInputName,
			OfferId:         "sku-1001",
			ContentLanguage: "en",
			FeedLabel:       "US",
			ProductAttributes: &productspb.ProductAttributes{
				Title: proto.String("Stackyard SKU 1001 Updated"),
			},
		},
		DataSource: dataSource,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"productAttributes.title"}},
	}
	var updateResp productspb.ProductInput
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsUpdateProductInputMethod, updateReq, &updateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant products update, got %q message=%q", grpcStatus, grpcMessage)
	}
	if updateResp.GetProductAttributes().GetTitle() != restUpdatedTitle {
		t.Fatalf("expected grpc updated title %q to match rest %q", updateResp.GetProductAttributes().GetTitle(), restUpdatedTitle)
	}

	deleteReq := &productspb.DeleteProductInputRequest{
		Name:       productInputName,
		DataSource: dataSource,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsDeleteProductInputMethod, deleteReq, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant products delete, got %q message=%q", grpcStatus, grpcMessage)
	}

	invalidInsertReq := &productspb.InsertProductInputRequest{
		Parent: parent,
		ProductInput: &productspb.ProductInput{
			OfferId:         "sku-1001",
			ContentLanguage: "en",
			FeedLabel:       "US",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsInsertProductInputMethod, invalidInsertReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "request-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant products insert missing dataSource, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingGetReq := &productspb.GetProductRequest{Name: parent + "/products/en~US~missing-sku"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductsGetProductMethod, missingGetReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "product-not-found") {
		t.Fatalf("expected grpc not found for merchant products missing get, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantPromotions(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	}

	parent := "accounts/123456"
	dataSource := "accounts/123456/dataSources/104628"
	promotionToken := "en~US~promo-1001"
	promotionName := parent + "/promotions/" + promotionToken

	restInsertResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/"+parent+"/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/123456/dataSources/104628",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Stackyard Promotion 1001",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if restInsertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant promotions insert, got %d body=%s", restInsertResp.StatusCode, string(providerContractBody(t, restInsertResp)))
	}
	restInsertBody := providerContractJSONMap(t, restInsertResp)
	restInsertedName, _ := restInsertBody["name"].(string)

	insertReq := &promotionspb.InsertPromotionRequest{
		Parent:     parent,
		DataSource: dataSource,
		Promotion: &promotionspb.Promotion{
			PromotionId:       "promo-1001",
			ContentLanguage:   "en",
			TargetCountry:     "US",
			RedemptionChannel: []promotionspb.RedemptionChannel{promotionspb.RedemptionChannel_ONLINE},
			Attributes: &promotionspb.Attributes{
				ProductApplicability: promotionspb.ProductApplicability_ALL_PRODUCTS,
				OfferType:            promotionspb.OfferType_NO_CODE,
				LongTitle:            "Stackyard Promotion 1001",
				CouponValueType:      promotionspb.CouponValueType_MONEY_OFF,
				PromotionEffectiveTimePeriod: &intervalpb.Interval{
					StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
					EndTime:   timestamppb.New(time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC)),
				},
			},
		},
	}
	var insertResp promotionspb.Promotion
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsInsertPromotionMethod, insertReq, &insertResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant promotions insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if insertResp.GetName() != restInsertedName {
		t.Fatalf("expected grpc insert name %q to match rest %q", insertResp.GetName(), restInsertedName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/"+promotionName, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant promotions get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restGetName, _ := restGetBody["name"].(string)

	getReq := &promotionspb.GetPromotionRequest{Name: promotionName}
	var getResp promotionspb.Promotion
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsGetPromotionMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant promotions get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restGetName {
		t.Fatalf("expected grpc promotion name %q to match rest %q", getResp.GetName(), restGetName)
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/"+parent+"/promotions?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant promotions list, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restPromotions, ok := restListBody["promotions"].([]any)
	if !ok || len(restPromotions) == 0 {
		t.Fatalf("expected rest promotions list, got %#v", restListBody["promotions"])
	}
	restFirst, _ := restPromotions[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)

	listReq := &promotionspb.ListPromotionsRequest{
		Parent:   parent,
		PageSize: 1,
	}
	var listResp promotionspb.ListPromotionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsListPromotionsMethod, listReq, &listResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant promotions list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listResp.GetPromotions()) != 1 {
		t.Fatalf("expected one grpc promotion from list, got %d", len(listResp.GetPromotions()))
	}
	if listResp.GetPromotions()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc first promotion %q to match rest %q", listResp.GetPromotions()[0].GetName(), restFirstName)
	}

	invalidListReq := &promotionspb.ListPromotionsRequest{
		Parent: "accounts",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsListPromotionsMethod, invalidListReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant promotions list invalid parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingGetReq := &promotionspb.GetPromotionRequest{Name: parent + "/promotions/en~US~missing-promo"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsGetPromotionMethod, missingGetReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "promotion-not-found") {
		t.Fatalf("expected grpc not found for merchant promotions get missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionInsertReq := &promotionspb.InsertPromotionRequest{
		Parent:     parent,
		DataSource: "accounts/999999/dataSources/104628",
		Promotion: &promotionspb.Promotion{
			PromotionId:       "promo-1002",
			ContentLanguage:   "en",
			TargetCountry:     "US",
			RedemptionChannel: []promotionspb.RedemptionChannel{promotionspb.RedemptionChannel_ONLINE},
			Attributes: &promotionspb.Attributes{
				ProductApplicability: promotionspb.ProductApplicability_ALL_PRODUCTS,
				OfferType:            promotionspb.OfferType_NO_CODE,
				LongTitle:            "Stackyard Promotion 1002",
				CouponValueType:      promotionspb.CouponValueType_MONEY_OFF,
				PromotionEffectiveTimePeriod: &intervalpb.Interval{
					StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
					EndTime:   timestamppb.New(time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC)),
				},
			},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsInsertPromotionMethod, preconditionInsertReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "request-failed-precondition") {
		t.Fatalf("expected grpc failed precondition for merchant promotions insert account mismatch, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	abortedInsertReq := &promotionspb.InsertPromotionRequest{
		Parent:     parent,
		DataSource: dataSource,
		Promotion: &promotionspb.Promotion{
			PromotionId:       "promo-1003",
			ContentLanguage:   "en",
			TargetCountry:     "US",
			RedemptionChannel: []promotionspb.RedemptionChannel{promotionspb.RedemptionChannel_ONLINE},
			Attributes: &promotionspb.Attributes{
				ProductApplicability: promotionspb.ProductApplicability_ALL_PRODUCTS,
				OfferType:            promotionspb.OfferType_NO_CODE,
				LongTitle:            "Stackyard Promotion 1003",
				CouponValueType:      promotionspb.CouponValueType_MONEY_OFF,
				PromotionEffectiveTimePeriod: &intervalpb.Interval{
					StartTime: timestamppb.New(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
					EndTime:   timestamppb.New(time.Date(2026, time.January, 31, 23, 59, 59, 0, time.UTC)),
				},
			},
			VersionNumber: proto.Int64(1),
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantPromotionsInsertPromotionMethod, abortedInsertReq, nil)
	if grpcStatus != "10" || !strings.Contains(grpcMessage, "version_number-stale") {
		t.Fatalf("expected grpc aborted for merchant promotions insert stale version, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantQuota(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/123456/quotas?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant quota list, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restQuotaGroups, ok := restBody["quotaGroups"].([]any)
	if !ok || len(restQuotaGroups) == 0 {
		t.Fatalf("expected rest quotaGroups list, got %#v", restBody["quotaGroups"])
	}
	restFirst, _ := restQuotaGroups[0].(map[string]any)
	restFirstName, _ := restFirst["name"].(string)
	restFirstMethodDetails, _ := restFirst["methodDetails"].([]any)
	restFirstMethod, _ := restFirstMethodDetails[0].(map[string]any)
	restFirstMethodPath, _ := restFirstMethod["path"].(string)

	successReq := &quotapb.ListQuotaGroupsRequest{
		Parent:   "accounts/123456",
		PageSize: 1,
	}
	var successResp quotapb.ListQuotaGroupsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantQuotaListQuotaGroupsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant quota list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetQuotaGroups()) != 1 {
		t.Fatalf("expected one grpc quota group, got %d", len(successResp.GetQuotaGroups()))
	}
	if successResp.GetQuotaGroups()[0].GetName() != restFirstName {
		t.Fatalf("expected grpc quota group name %q to match rest %q", successResp.GetQuotaGroups()[0].GetName(), restFirstName)
	}
	if len(successResp.GetQuotaGroups()[0].GetMethodDetails()) == 0 {
		t.Fatalf("expected grpc method details for first quota group")
	}
	if successResp.GetQuotaGroups()[0].GetMethodDetails()[0].GetPath() != restFirstMethodPath {
		t.Fatalf("expected grpc method path %q to match rest %q", successResp.GetQuotaGroups()[0].GetMethodDetails()[0].GetPath(), restFirstMethodPath)
	}

	invalidParentReq := &quotapb.ListQuotaGroupsRequest{
		Parent: "accounts",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantQuotaListQuotaGroupsMethod, invalidParentReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant quota invalid parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidTokenReq := &quotapb.ListQuotaGroupsRequest{
		Parent:    "accounts/123456",
		PageToken: "bad",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantQuotaListQuotaGroupsMethod, invalidTokenReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "page_token-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant quota invalid page token, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingAccountReq := &quotapb.ListQuotaGroupsRequest{
		Parent: "accounts/missing",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantQuotaListQuotaGroupsMethod, missingAccountReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "account-not-found") {
		t.Fatalf("expected grpc not found for merchant quota missing account, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantReports(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
		"Content-Type":            "application/json",
	}

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"query":"SELECT product_view.id, product_view.title FROM product_view",
		"pageSize":1
	}`), headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant reports search, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restResults, ok := restBody["results"].([]any)
	if !ok || len(restResults) == 0 {
		t.Fatalf("expected rest results list, got %#v", restBody["results"])
	}
	restFirst, _ := restResults[0].(map[string]any)
	restProductView, _ := restFirst["productView"].(map[string]any)
	restID, _ := restProductView["id"].(string)

	successReq := &reportspb.SearchRequest{
		Parent:   "accounts/123456",
		Query:    "SELECT product_view.id, product_view.title FROM product_view",
		PageSize: 1,
	}
	var successResp reportspb.SearchResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReportsSearchMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant reports search, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetResults()) != 1 {
		t.Fatalf("expected one grpc result row, got %d", len(successResp.GetResults()))
	}
	if successResp.GetResults()[0].GetProductView() == nil {
		t.Fatalf("expected productView in grpc row")
	}
	if successResp.GetResults()[0].GetProductView().GetId() != restID {
		t.Fatalf("expected grpc row productView.id %q to match rest %q", successResp.GetResults()[0].GetProductView().GetId(), restID)
	}

	invalidParentReq := &reportspb.SearchRequest{
		Parent: "accounts",
		Query:  "SELECT product_view.id FROM product_view",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReportsSearchMethod, invalidParentReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant reports invalid parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidQueryReq := &reportspb.SearchRequest{
		Parent: "accounts/123456",
		Query:  "bad query",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReportsSearchMethod, invalidQueryReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "query-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant reports invalid query, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidTokenReq := &reportspb.SearchRequest{
		Parent:    "accounts/123456",
		Query:     "SELECT product_view.id FROM product_view",
		PageToken: "bad",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReportsSearchMethod, invalidTokenReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "page_token-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant reports invalid page token, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingAccountReq := &reportspb.SearchRequest{
		Parent: "accounts/missing",
		Query:  "SELECT product_view.id FROM product_view",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReportsSearchMethod, missingAccountReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "account-not-found") {
		t.Fatalf("expected grpc not found for merchant reports missing account, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantReviews(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
		"Content-Type":            "application/json",
	}

	restMerchantGet := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews/merchant-review-1001", nil, headers)
	if restMerchantGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest merchant reviews get, got %d body=%s", restMerchantGet.StatusCode, string(providerContractBody(t, restMerchantGet)))
	}
	restMerchantBody := providerContractJSONMap(t, restMerchantGet)
	restMerchantName, _ := restMerchantBody["name"].(string)

	getMerchantReq := &reviewspb.GetMerchantReviewRequest{Name: "accounts/123456/merchantReviews/merchant-review-1001"}
	var getMerchantResp reviewspb.MerchantReview
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsGetMerchantReviewMethod, getMerchantReq, &getMerchantResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant reviews get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getMerchantResp.GetName() != restMerchantName {
		t.Fatalf("expected grpc merchant review name %q to match rest %q", getMerchantResp.GetName(), restMerchantName)
	}

	restMerchantList := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews?pageSize=1", nil, headers)
	if restMerchantList.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest merchant reviews list, got %d body=%s", restMerchantList.StatusCode, string(providerContractBody(t, restMerchantList)))
	}
	restMerchantListBody := providerContractJSONMap(t, restMerchantList)
	restMerchantItems, ok := restMerchantListBody["merchantReviews"].([]any)
	if !ok || len(restMerchantItems) == 0 {
		t.Fatalf("expected merchantReviews list in rest payload, got %#v", restMerchantListBody["merchantReviews"])
	}
	restMerchantFirst, _ := restMerchantItems[0].(map[string]any)
	restMerchantFirstName, _ := restMerchantFirst["name"].(string)

	listMerchantReq := &reviewspb.ListMerchantReviewsRequest{Parent: "accounts/123456", PageSize: 1}
	var listMerchantResp reviewspb.ListMerchantReviewsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsListMerchantReviewsMethod, listMerchantReq, &listMerchantResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant reviews list, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listMerchantResp.GetMerchantReviews()) != 1 {
		t.Fatalf("expected one grpc merchant review from list, got %d", len(listMerchantResp.GetMerchantReviews()))
	}
	if listMerchantResp.GetMerchantReviews()[0].GetName() != restMerchantFirstName {
		t.Fatalf("expected grpc first merchant review name %q to match rest %q", listMerchantResp.GetMerchantReviews()[0].GetName(), restMerchantFirstName)
	}

	restProductGet := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/productReviews/product-review-1001", nil, headers)
	if restProductGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest product reviews get, got %d body=%s", restProductGet.StatusCode, string(providerContractBody(t, restProductGet)))
	}
	restProductBody := providerContractJSONMap(t, restProductGet)
	restProductName, _ := restProductBody["name"].(string)

	getProductReq := &reviewspb.GetProductReviewRequest{Name: "accounts/123456/productReviews/product-review-1001"}
	var getProductResp reviewspb.ProductReview
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsGetProductReviewMethod, getProductReq, &getProductResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for product reviews get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getProductResp.GetName() != restProductName {
		t.Fatalf("expected grpc product review name %q to match rest %q", getProductResp.GetName(), restProductName)
	}

	insertMerchantReq := &reviewspb.InsertMerchantReviewRequest{
		Parent:     "accounts/123456",
		DataSource: "accounts/123456/dataSources/104628",
		MerchantReview: &reviewspb.MerchantReview{
			MerchantReviewId: "merchant-review-3001",
			MerchantReviewAttributes: &reviewspb.MerchantReviewAttributes{
				Title:   proto.String("Fast delivery"),
				Content: proto.String("Shipped ahead of schedule."),
			},
		},
	}
	var insertMerchantResp reviewspb.MerchantReview
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsInsertMerchantReviewMethod, insertMerchantReq, &insertMerchantResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant reviews insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(insertMerchantResp.GetName(), "merchant-review-3001") {
		t.Fatalf("expected inserted merchant review name to include merchant-review-3001, got %q", insertMerchantResp.GetName())
	}

	deleteMerchantReq := &reviewspb.DeleteMerchantReviewRequest{Name: "accounts/123456/merchantReviews/merchant-review-3001"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsDeleteMerchantReviewMethod, deleteMerchantReq, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant reviews delete, got %q message=%q", grpcStatus, grpcMessage)
	}

	insertProductReq := &reviewspb.InsertProductReviewRequest{
		Parent:     "accounts/123456",
		DataSource: "accounts/123456/dataSources/104628",
		ProductReview: &reviewspb.ProductReview{
			ProductReviewId: "product-review-3001",
			ProductReviewAttributes: &reviewspb.ProductReviewAttributes{
				Title:   proto.String("Great fit"),
				Content: proto.String("Very comfortable and true to size."),
			},
		},
	}
	var insertProductResp reviewspb.ProductReview
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsInsertProductReviewMethod, insertProductReq, &insertProductResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for product reviews insert, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(insertProductResp.GetName(), "product-review-3001") {
		t.Fatalf("expected inserted product review name to include product-review-3001, got %q", insertProductResp.GetName())
	}

	deleteProductReq := &reviewspb.DeleteProductReviewRequest{Name: "accounts/123456/productReviews/product-review-3001"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsDeleteProductReviewMethod, deleteProductReq, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for product reviews delete, got %q message=%q", grpcStatus, grpcMessage)
	}

	invalidParentReq := &reviewspb.ListMerchantReviewsRequest{Parent: "accounts"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsListMerchantReviewsMethod, invalidParentReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for merchant reviews list invalid parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidTokenReq := &reviewspb.ListProductReviewsRequest{
		Parent:    "accounts/123456",
		PageToken: "bad",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsListProductReviewsMethod, invalidTokenReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "page_token-invalid") {
		t.Fatalf("expected grpc invalid argument for product reviews list invalid page token, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidDataSourceReq := &reviewspb.InsertMerchantReviewRequest{
		Parent:     "accounts/123456",
		DataSource: "bad",
		MerchantReview: &reviewspb.MerchantReview{
			MerchantReviewId: "merchant-review-4001",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsInsertMerchantReviewMethod, invalidDataSourceReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "data_source-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant reviews insert invalid data source, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	mismatchDataSourceReq := &reviewspb.InsertProductReviewRequest{
		Parent:     "accounts/123456",
		DataSource: "accounts/999999/dataSources/104628",
		ProductReview: &reviewspb.ProductReview{
			ProductReviewId: "product-review-4001",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsInsertProductReviewMethod, mismatchDataSourceReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "data_source-account-mismatch") {
		t.Fatalf("expected grpc failed precondition for product reviews insert account mismatch, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	missingReq := &reviewspb.GetMerchantReviewRequest{Name: "accounts/123456/merchantReviews/missing-review"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantReviewsGetMerchantReviewMethod, missingReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "review-not-found") {
		t.Fatalf("expected grpc not found for merchant reviews get missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ShoppingMerchantProductstudio(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	}

	restGenerateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:generateProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"outputConfig":{"returnImageUri":true},
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"},
		"config":{"productDescription":"Stackyard red dress","backgroundDescription":"Clean studio backdrop"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if restGenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant productstudio generate image background, got %d body=%s", restGenerateResp.StatusCode, string(providerContractBody(t, restGenerateResp)))
	}
	restGenerateBody := providerContractJSONMap(t, restGenerateResp)
	restGeneratedImage, _ := restGenerateBody["generatedImage"].(map[string]any)
	restGeneratedName, _ := restGeneratedImage["name"].(string)
	restGeneratedURI, _ := restGeneratedImage["uri"].(string)

	generateReq := &productstudiopb.GenerateProductImageBackgroundRequest{
		Name: "accounts/123456",
		OutputConfig: &productstudiopb.OutputImageConfig{
			ReturnImageUri: true,
		},
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageUri{
				ImageUri: "https://example.com/products/sku-1001.jpg",
			},
		},
		Config: &productstudiopb.GenerateImageBackgroundConfig{
			ProductDescription:    "Stackyard red dress",
			BackgroundDescription: "Clean studio backdrop",
		},
	}
	var generateResp productstudiopb.GenerateProductImageBackgroundResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioGenerateProductImageBackgroundMethod, generateReq, &generateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant productstudio generate background, got %q message=%q", grpcStatus, grpcMessage)
	}
	if generateResp.GetGeneratedImage() == nil {
		t.Fatalf("expected generated image in grpc response")
	}
	if generateResp.GetGeneratedImage().GetName() != restGeneratedName {
		t.Fatalf("expected grpc generated image name %q to match rest %q", generateResp.GetGeneratedImage().GetName(), restGeneratedName)
	}
	if generateResp.GetGeneratedImage().GetUri() != restGeneratedURI {
		t.Fatalf("expected grpc generated image uri %q to match rest %q", generateResp.GetGeneratedImage().GetUri(), restGeneratedURI)
	}

	restRemoveResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:removeProductImageBackground", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageBytes":"aW1hZ2UtYnl0ZXM="}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if restRemoveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant productstudio remove background, got %d body=%s", restRemoveResp.StatusCode, string(providerContractBody(t, restRemoveResp)))
	}
	restRemoveBody := providerContractJSONMap(t, restRemoveResp)
	restRemoveImage, _ := restRemoveBody["generatedImage"].(map[string]any)
	restRemoveName, _ := restRemoveImage["name"].(string)
	restRemoveBytes, _ := restRemoveImage["imageBytes"].(string)

	removeReq := &productstudiopb.RemoveProductImageBackgroundRequest{
		Name: "accounts/123456",
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageBytes{
				ImageBytes: []byte("image-bytes"),
			},
		},
	}
	var removeResp productstudiopb.RemoveProductImageBackgroundResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioRemoveProductImageBackgroundMethod, removeReq, &removeResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant productstudio remove background, got %q message=%q", grpcStatus, grpcMessage)
	}
	if removeResp.GetGeneratedImage() == nil {
		t.Fatalf("expected generated image in remove response")
	}
	if removeResp.GetGeneratedImage().GetName() != restRemoveName {
		t.Fatalf("expected grpc remove generated image name %q to match rest %q", removeResp.GetGeneratedImage().GetName(), restRemoveName)
	}
	if len(removeResp.GetGeneratedImage().GetImageBytes()) == 0 || restRemoveBytes == "" {
		t.Fatalf("expected non-empty imageBytes from remove responses")
	}

	restUpscaleResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456/generatedImages:upscaleProductImage", []byte(`{
		"name":"accounts/123456",
		"inputImage":{"imageUri":"https://example.com/products/sku-1001.jpg"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-productstudio",
	})
	if restUpscaleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant productstudio upscale image, got %d body=%s", restUpscaleResp.StatusCode, string(providerContractBody(t, restUpscaleResp)))
	}
	restUpscaleBody := providerContractJSONMap(t, restUpscaleResp)
	restUpscaleImage, _ := restUpscaleBody["generatedImage"].(map[string]any)
	restUpscaleName, _ := restUpscaleImage["name"].(string)

	upscaleReq := &productstudiopb.UpscaleProductImageRequest{
		Name: "accounts/123456",
		InputImage: &productstudiopb.InputImage{
			Image: &productstudiopb.InputImage_ImageUri{
				ImageUri: "https://example.com/products/sku-1001.jpg",
			},
		},
	}
	var upscaleResp productstudiopb.UpscaleProductImageResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioUpscaleProductImageMethod, upscaleReq, &upscaleResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant productstudio upscale image, got %q message=%q", grpcStatus, grpcMessage)
	}
	if upscaleResp.GetGeneratedImage() == nil {
		t.Fatalf("expected generated image in upscale response")
	}
	if upscaleResp.GetGeneratedImage().GetName() != restUpscaleName {
		t.Fatalf("expected grpc upscale generated image name %q to match rest %q", upscaleResp.GetGeneratedImage().GetName(), restUpscaleName)
	}

	restTextResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/productstudio/v1alpha/accounts/123456:generateProductTextSuggestions", []byte(`{
		"name":"accounts/123456",
		"productInfo":{"productAttributes":{"title":"Red Dress","brand":"Stackyard","color":"red"}},
		"outputSpec":{"workflowId":"title","tone":"playful","targetLanguage":"en"}
	}`), headers)
	if restTextResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest shopping merchant productstudio text suggestions, got %d body=%s", restTextResp.StatusCode, string(providerContractBody(t, restTextResp)))
	}
	restTextBody := providerContractJSONMap(t, restTextResp)
	restTitle, _ := restTextBody["title"].(map[string]any)
	restTitleText, _ := restTitle["text"].(string)
	restAttributes, _ := restTextBody["attributes"].(map[string]any)
	restWorkflow, _ := restAttributes["workflow"].(string)

	textReq := &productstudiopb.GenerateProductTextSuggestionsRequest{
		Name: "accounts/123456",
		ProductInfo: &productstudiopb.ProductInfo{
			ProductAttributes: map[string]string{
				"title": "Red Dress",
				"brand": "Stackyard",
				"color": "red",
			},
		},
		OutputSpec: &productstudiopb.OutputSpec{
			WorkflowId:     proto.String("title"),
			Tone:           proto.String("playful"),
			TargetLanguage: proto.String("en"),
		},
	}
	var textResp productstudiopb.GenerateProductTextSuggestionsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioGenerateProductTextSuggestionsMethod, textReq, &textResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for merchant productstudio text suggestions, got %q message=%q", grpcStatus, grpcMessage)
	}
	if textResp.GetTitle() == nil || textResp.GetDescription() == nil {
		t.Fatalf("expected title and description in grpc text response")
	}
	if textResp.GetTitle().GetText() != restTitleText {
		t.Fatalf("expected grpc title text %q to match rest %q", textResp.GetTitle().GetText(), restTitleText)
	}
	if textResp.GetAttributes()["workflow"] != restWorkflow {
		t.Fatalf("expected grpc workflow attribute %q to match rest %q", textResp.GetAttributes()["workflow"], restWorkflow)
	}

	invalidReq := &productstudiopb.GenerateProductImageBackgroundRequest{
		Name: "accounts/123456",
		Config: &productstudiopb.GenerateImageBackgroundConfig{
			ProductDescription:    "Stackyard red dress",
			BackgroundDescription: "Clean studio backdrop",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioGenerateProductImageBackgroundMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "request-invalid") {
		t.Fatalf("expected grpc invalid argument for merchant productstudio generate background missing inputImage, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	notFoundReq := &productstudiopb.GenerateProductTextSuggestionsRequest{
		Name: "accounts/missing",
		ProductInfo: &productstudiopb.ProductInfo{
			ProductAttributes: map[string]string{"title": "Red Dress"},
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpShoppingMerchantProductstudioGenerateProductTextSuggestionsMethod, notFoundReq, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "resource-not-found") {
		t.Fatalf("expected grpc not found for merchant productstudio missing account, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ServiceHealth(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	}

	restEventsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?pageSize=1&view=EVENT_VIEW_FULL", nil, headers)
	if restEventsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicehealth list events, got %d body=%s", restEventsResp.StatusCode, string(providerContractBody(t, restEventsResp)))
	}
	restEventsBody := providerContractJSONMap(t, restEventsResp)
	restEvents, ok := restEventsBody["events"].([]any)
	if !ok || len(restEvents) == 0 {
		t.Fatalf("expected events list in rest payload, got %#v", restEventsBody["events"])
	}
	restEvent, _ := restEvents[0].(map[string]any)
	restEventName, _ := restEvent["name"].(string)

	successListReq := &servicehealthpb.ListEventsRequest{
		Parent:   "projects/stackyard/locations/global",
		PageSize: 1,
		View:     servicehealthpb.EventView_EVENT_VIEW_FULL,
	}
	var successListResp servicehealthpb.ListEventsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpServiceHealthListEventsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetEvents()) != 1 {
		t.Fatalf("expected one grpc event, got %d", len(successListResp.GetEvents()))
	}
	if successListResp.GetEvents()[0].GetName() != restEventName {
		t.Fatalf("expected grpc event name %q to match rest %q", successListResp.GetEvents()[0].GetName(), restEventName)
	}

	restImpactResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts/impact-1", nil, headers)
	if restImpactResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicehealth get organization impact, got %d body=%s", restImpactResp.StatusCode, string(providerContractBody(t, restImpactResp)))
	}
	restImpactBody := providerContractJSONMap(t, restImpactResp)
	restImpactName, _ := restImpactBody["name"].(string)

	successImpactReq := &servicehealthpb.GetOrganizationImpactRequest{
		Name: "organizations/123456789/locations/global/organizationImpacts/impact-1",
	}
	var successImpactResp servicehealthpb.OrganizationImpact
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceHealthGetOrganizationImpactMethod, successImpactReq, &successImpactResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successImpactResp.GetName() != restImpactName {
		t.Fatalf("expected grpc organization impact name %q to match rest %q", successImpactResp.GetName(), restImpactName)
	}

	invalidListReq := &servicehealthpb.ListEventsRequest{
		Parent:   "projects/stackyard/locations/global",
		PageSize: -1,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceHealthListEventsMethod, invalidListReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "page_size-invalid") {
		t.Fatalf("expected grpc invalid argument for servicehealth list events, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidGetReq := &servicehealthpb.GetOrganizationEventRequest{
		Name: "organizations/123456789/locations/global/organizationEvents",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceHealthGetOrganizationEventMethod, invalidGetReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "name-required") {
		t.Fatalf("expected grpc invalid argument for servicehealth get organization event, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ServiceDirectory(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicedirectory",
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicedirectory list namespaces, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restNamespaces, ok := restListBody["namespaces"].([]any)
	if !ok || len(restNamespaces) == 0 {
		t.Fatalf("expected namespaces list in rest payload, got %#v", restListBody["namespaces"])
	}
	restNamespace, _ := restNamespaces[0].(map[string]any)
	restNamespaceName, _ := restNamespace["name"].(string)

	successListReq := &servicedirectorypb.ListNamespacesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListResp servicedirectorypb.ListNamespacesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpServiceDirectoryListNamespacesMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetNamespaces()) != 1 {
		t.Fatalf("expected one grpc namespace, got %d", len(successListResp.GetNamespaces()))
	}
	if successListResp.GetNamespaces()[0].GetName() != restNamespaceName {
		t.Fatalf("expected grpc namespace name %q to match rest %q", successListResp.GetNamespaces()[0].GetName(), restNamespaceName)
	}

	restResolveResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:resolve", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","maxEndpoints":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if restResolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicedirectory resolve service, got %d body=%s", restResolveResp.StatusCode, string(providerContractBody(t, restResolveResp)))
	}
	restResolveBody := providerContractJSONMap(t, restResolveResp)
	restResolveService, _ := restResolveBody["service"].(map[string]any)
	restResolveServiceName, _ := restResolveService["name"].(string)

	successResolveReq := &servicedirectorypb.ResolveServiceRequest{
		Name:         "projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1",
		MaxEndpoints: 1,
	}
	var successResolveResp servicedirectorypb.ResolveServiceResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceDirectoryResolveServiceMethod, successResolveReq, &successResolveResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResolveResp.GetService().GetName() != restResolveServiceName {
		t.Fatalf("expected grpc resolved service name %q to match rest %q", successResolveResp.GetService().GetName(), restResolveServiceName)
	}
	if len(successResolveResp.GetService().GetEndpoints()) != 1 {
		t.Fatalf("expected one grpc resolved endpoint, got %d", len(successResolveResp.GetService().GetEndpoints()))
	}

	invalidCreateEndpointReq := &servicedirectorypb.CreateEndpointRequest{
		Parent: "projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1",
		Endpoint: &servicedirectorypb.Endpoint{
			Address: "10.10.0.9",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceDirectoryCreateEndpointMethod, invalidCreateEndpointReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "endpoint_id-required") {
		t.Fatalf("expected grpc invalid argument for servicedirectory create endpoint, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidSetPolicyReq := &iampb.SetIamPolicyRequest{
		Resource: "projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceDirectorySetIAMPolicyMethod, invalidSetPolicyReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "policy-required") {
		t.Fatalf("expected grpc invalid argument for servicedirectory set iam policy, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ServiceControl(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	}

	restCheckResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:check", []byte(`{
		"operation": {
			"operationId": "check-op-1",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z"
		}
	}`), headers)
	if restCheckResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicecontrol check, got %d body=%s", restCheckResp.StatusCode, string(providerContractBody(t, restCheckResp)))
	}
	restCheckBody := providerContractJSONMap(t, restCheckResp)
	restCheckOperationID, _ := restCheckBody["operationId"].(string)
	restCheckConfigID, _ := restCheckBody["serviceConfigId"].(string)

	successCheckReq := &servicecontrolpb.CheckRequest{
		ServiceName: "stackyard.googleapis.com",
		Operation: &servicecontrolpb.Operation{
			OperationId: "check-op-1",
			ConsumerId:  "project:stackyard",
			StartTime:   timestamppb.New(gcpStage4ReferenceTime),
		},
	}
	var successCheckResp servicecontrolpb.CheckResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpServiceControlCheckMethod, successCheckReq, &successCheckResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicecontrol check, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successCheckResp.GetOperationId() != restCheckOperationID {
		t.Fatalf("expected grpc check operationId %q to match rest %q", successCheckResp.GetOperationId(), restCheckOperationID)
	}
	if successCheckResp.GetServiceConfigId() != restCheckConfigID {
		t.Fatalf("expected grpc check serviceConfigId %q to match rest %q", successCheckResp.GetServiceConfigId(), restCheckConfigID)
	}

	restReportResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:report", []byte(`{
		"operations": [{
			"operationId": "report-op-1",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z"
		}]
	}`), headers)
	if restReportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicecontrol report, got %d body=%s", restReportResp.StatusCode, string(providerContractBody(t, restReportResp)))
	}
	restReportBody := providerContractJSONMap(t, restReportResp)
	restReportRolloutID, _ := restReportBody["serviceRolloutId"].(string)

	successReportReq := &servicecontrolpb.ReportRequest{
		ServiceName: "stackyard.googleapis.com",
		Operations: []*servicecontrolpb.Operation{
			{
				OperationId: "report-op-1",
				ConsumerId:  "project:stackyard",
				StartTime:   timestamppb.New(gcpStage4ReferenceTime),
			},
		},
	}
	var successReportResp servicecontrolpb.ReportResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceControlReportMethod, successReportReq, &successReportResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicecontrol report, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successReportResp.GetServiceRolloutId() != restReportRolloutID {
		t.Fatalf("expected grpc report serviceRolloutId %q to match rest %q", successReportResp.GetServiceRolloutId(), restReportRolloutID)
	}

	restAllocateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:allocateQuota", []byte(`{
		"allocateOperation": {
			"operationId": "quota-op-1",
			"consumerId": "project:stackyard",
			"methodName": "google.example.v1.Service/Call"
		}
	}`), headers)
	if restAllocateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicecontrol allocateQuota, got %d body=%s", restAllocateResp.StatusCode, string(providerContractBody(t, restAllocateResp)))
	}
	restAllocateBody := providerContractJSONMap(t, restAllocateResp)
	restAllocateOperationID, _ := restAllocateBody["operationId"].(string)

	successAllocateReq := &servicecontrolpb.AllocateQuotaRequest{
		ServiceName: "stackyard.googleapis.com",
		AllocateOperation: &servicecontrolpb.QuotaOperation{
			OperationId: "quota-op-1",
			ConsumerId:  "project:stackyard",
			MethodName:  "google.example.v1.Service/Call",
		},
	}
	var successAllocateResp servicecontrolpb.AllocateQuotaResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceControlAllocateQuotaMethod, successAllocateReq, &successAllocateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicecontrol allocateQuota, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successAllocateResp.GetOperationId() != restAllocateOperationID {
		t.Fatalf("expected grpc allocateQuota operationId %q to match rest %q", successAllocateResp.GetOperationId(), restAllocateOperationID)
	}
	if len(successAllocateResp.GetQuotaMetrics()) == 0 {
		t.Fatalf("expected grpc allocateQuota to include quota metrics")
	}

	invalidCheckReq := &servicecontrolpb.CheckRequest{
		ServiceName: "stackyard.googleapis.com",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceControlCheckMethod, invalidCheckReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "operation-required") {
		t.Fatalf("expected grpc invalid argument for servicecontrol check, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidAllocateReq := &servicecontrolpb.AllocateQuotaRequest{
		ServiceName: "stackyard.googleapis.com",
		AllocateOperation: &servicecontrolpb.QuotaOperation{
			OperationId: "quota-op-1",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceControlAllocateQuotaMethod, invalidAllocateReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "allocate_operation.consumer_id-required") {
		t.Fatalf("expected grpc invalid argument for servicecontrol allocateQuota, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ServiceUsage(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	}

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services?pageSize=1&filter=state:ENABLED", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest serviceusage list services, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restServices, ok := restListBody["services"].([]any)
	if !ok || len(restServices) == 0 {
		t.Fatalf("expected services list in rest payload, got %#v", restListBody["services"])
	}
	restService, _ := restServices[0].(map[string]any)
	restServiceName, _ := restService["name"].(string)
	if restServiceName == "" {
		t.Fatalf("expected rest service name, got %#v", restService["name"])
	}

	successListReq := &serviceusagepb.ListServicesRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
		Filter:   "state:ENABLED",
	}
	var successListResp serviceusagepb.ListServicesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageListServicesMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for serviceusage list services, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetServices()) != 1 {
		t.Fatalf("expected one grpc service, got %d", len(successListResp.GetServices()))
	}
	if successListResp.GetServices()[0].GetName() != restServiceName {
		t.Fatalf("expected grpc service name %q to match rest %q", successListResp.GetServices()[0].GetName(), restServiceName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+restServiceName, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest serviceusage get service, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restState, _ := restGetBody["state"].(string)

	successGetReq := &serviceusagepb.GetServiceRequest{Name: restServiceName}
	var successGetResp serviceusagepb.Service
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageGetServiceMethod, successGetReq, &successGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for serviceusage get service, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetResp.GetState().String() != restState {
		t.Fatalf("expected grpc service state %q to match rest %q", successGetResp.GetState().String(), restState)
	}

	restEnableResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+restServiceName+":enable", []byte(`{
		"name":"`+restServiceName+`"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if restEnableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest serviceusage enable service, got %d body=%s", restEnableResp.StatusCode, string(providerContractBody(t, restEnableResp)))
	}
	restEnableBody := providerContractJSONMap(t, restEnableResp)
	restEnableName, _ := restEnableBody["name"].(string)

	successEnableReq := &serviceusagepb.EnableServiceRequest{Name: restServiceName}
	var successEnableResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageEnableServiceMethod, successEnableReq, &successEnableResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for serviceusage enable service, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successEnableResp.Name != restEnableName {
		t.Fatalf("expected grpc operation name %q to match rest %q", successEnableResp.Name, restEnableName)
	}

	restBatchGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services:batchGet?names="+restServiceName, nil, headers)
	if restBatchGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest serviceusage batch get, got %d body=%s", restBatchGetResp.StatusCode, string(providerContractBody(t, restBatchGetResp)))
	}
	restBatchGetBody := providerContractJSONMap(t, restBatchGetResp)
	restBatchServices, ok := restBatchGetBody["services"].([]any)
	if !ok || len(restBatchServices) == 0 {
		t.Fatalf("expected services in rest batch get payload, got %#v", restBatchGetBody["services"])
	}

	successBatchGetReq := &serviceusagepb.BatchGetServicesRequest{
		Parent: "projects/stackyard",
		Names:  []string{restServiceName},
	}
	var successBatchGetResp serviceusagepb.BatchGetServicesResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageBatchGetServicesMethod, successBatchGetReq, &successBatchGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for serviceusage batch get, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successBatchGetResp.GetServices()) != len(restBatchServices) {
		t.Fatalf("expected grpc batch get services %d to match rest %d", len(successBatchGetResp.GetServices()), len(restBatchServices))
	}
	if len(successBatchGetResp.GetServices()) > 0 && successBatchGetResp.GetServices()[0].GetName() != restServiceName {
		t.Fatalf("expected grpc batch get service name %q to match rest %q", successBatchGetResp.GetServices()[0].GetName(), restServiceName)
	}

	invalidListReq := &serviceusagepb.ListServicesRequest{
		Parent: "projects/stackyard",
		Filter: "state==ENABLED",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageListServicesMethod, invalidListReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "filter-invalid") {
		t.Fatalf("expected grpc invalid argument for serviceusage list services, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	tooManyServiceIDs := make([]string, 21)
	for i := range tooManyServiceIDs {
		tooManyServiceIDs[i] = "serviceusage.googleapis.com"
	}
	invalidBatchEnableReq := &serviceusagepb.BatchEnableServicesRequest{
		Parent:     "projects/stackyard",
		ServiceIds: tooManyServiceIDs,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceUsageBatchEnableServicesMethod, invalidBatchEnableReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "service_ids-too-many") {
		t.Fatalf("expected grpc invalid argument for serviceusage batch enable, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SupportDispatchKnownPath(t *testing.T) {
	t.Parallel()

	req := &supportpb.GetCaseRequest{Name: "projects/stackyard/cases/case-open-1"}
	reqPayload, ok := marshalProtoMessage(req)
	if !ok {
		t.Fatalf("failed to marshal request payload")
	}
	grpcReqBody := make([]byte, 5+len(reqPayload))
	grpcReqBody[0] = 0
	binary.BigEndian.PutUint32(grpcReqBody[1:5], uint32(len(reqPayload)))
	copy(grpcReqBody[5:], reqPayload)

	_, _, _, ok = knownGCPStage4GRPCResponse(gcpSupportGetCaseMethod, grpcReqBody)
	if !ok {
		t.Fatalf("expected knownGCPStage4GRPCResponse to dispatch support get case")
	}
}

func TestGCPStage4GRPCParity_Support(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "support",
	}

	restGetCaseResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1", nil, headers)
	if restGetCaseResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest support get case, got %d body=%s", restGetCaseResp.StatusCode, string(providerContractBody(t, restGetCaseResp)))
	}
	restGetCaseBody := providerContractJSONMap(t, restGetCaseResp)
	restCaseName, _ := restGetCaseBody["name"].(string)
	restCaseDisplayName, _ := restGetCaseBody["displayName"].(string)

	successGetCaseReq := &supportpb.GetCaseRequest{Name: restCaseName}
	var successGetCaseResp supportpb.Case
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSupportGetCaseMethod, successGetCaseReq, &successGetCaseResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for support get case, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetCaseResp.GetName() != restCaseName {
		t.Fatalf("expected grpc case name %q to match rest %q", successGetCaseResp.GetName(), restCaseName)
	}
	if successGetCaseResp.GetDisplayName() != restCaseDisplayName {
		t.Fatalf("expected grpc case displayName %q to match rest %q", successGetCaseResp.GetDisplayName(), restCaseDisplayName)
	}

	restListCommentsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1/comments?pageSize=1", nil, headers)
	if restListCommentsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest support list comments, got %d body=%s", restListCommentsResp.StatusCode, string(providerContractBody(t, restListCommentsResp)))
	}
	restListCommentsBody := providerContractJSONMap(t, restListCommentsResp)
	restComments, ok := restListCommentsBody["comments"].([]any)
	if !ok || len(restComments) == 0 {
		t.Fatalf("expected comments list in rest payload, got %#v", restListCommentsBody["comments"])
	}
	restComment, _ := restComments[0].(map[string]any)
	restCommentName, _ := restComment["name"].(string)

	successListCommentsReq := &supportpb.ListCommentsRequest{
		Parent:   "projects/stackyard/cases/case-open-1",
		PageSize: 1,
	}
	var successListCommentsResp supportpb.ListCommentsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSupportListCommentsMethod, successListCommentsReq, &successListCommentsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for support list comments, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListCommentsResp.GetComments()) != 1 {
		t.Fatalf("expected one grpc comment, got %d", len(successListCommentsResp.GetComments()))
	}
	if successListCommentsResp.GetComments()[0].GetName() != restCommentName {
		t.Fatalf("expected grpc comment name %q to match rest %q", successListCommentsResp.GetComments()[0].GetName(), restCommentName)
	}

	restListAttachmentsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1/attachments?pageSize=1", nil, headers)
	if restListAttachmentsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest support list attachments, got %d body=%s", restListAttachmentsResp.StatusCode, string(providerContractBody(t, restListAttachmentsResp)))
	}
	restListAttachmentsBody := providerContractJSONMap(t, restListAttachmentsResp)
	restAttachments, ok := restListAttachmentsBody["attachments"].([]any)
	if !ok || len(restAttachments) == 0 {
		t.Fatalf("expected attachments list in rest payload, got %#v", restListAttachmentsBody["attachments"])
	}
	restAttachment, _ := restAttachments[0].(map[string]any)
	restAttachmentName, _ := restAttachment["name"].(string)

	successListAttachmentsReq := &supportpb.ListAttachmentsRequest{
		Parent:   "projects/stackyard/cases/case-open-1",
		PageSize: 1,
	}
	var successListAttachmentsResp supportpb.ListAttachmentsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSupportListAttachmentsMethod, successListAttachmentsReq, &successListAttachmentsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for support list attachments, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListAttachmentsResp.GetAttachments()) != 1 {
		t.Fatalf("expected one grpc attachment, got %d", len(successListAttachmentsResp.GetAttachments()))
	}
	if successListAttachmentsResp.GetAttachments()[0].GetName() != restAttachmentName {
		t.Fatalf("expected grpc attachment name %q to match rest %q", successListAttachmentsResp.GetAttachments()[0].GetName(), restAttachmentName)
	}

	restClassificationsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/caseClassifications:search?pageSize=1", nil, headers)
	if restClassificationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest support search classifications, got %d body=%s", restClassificationsResp.StatusCode, string(providerContractBody(t, restClassificationsResp)))
	}
	restClassificationsBody := providerContractJSONMap(t, restClassificationsResp)
	restClassifications, ok := restClassificationsBody["caseClassifications"].([]any)
	if !ok || len(restClassifications) == 0 {
		t.Fatalf("expected caseClassifications in rest payload, got %#v", restClassificationsBody["caseClassifications"])
	}
	restClassification, _ := restClassifications[0].(map[string]any)
	restClassificationID, _ := restClassification["id"].(string)

	successClassificationsReq := &supportpb.SearchCaseClassificationsRequest{PageSize: 1}
	var successClassificationsResp supportpb.SearchCaseClassificationsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSupportSearchCaseClassificationsMethod, successClassificationsReq, &successClassificationsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for support search classifications, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successClassificationsResp.GetCaseClassifications()) != 1 {
		t.Fatalf("expected one grpc case classification, got %d", len(successClassificationsResp.GetCaseClassifications()))
	}
	if successClassificationsResp.GetCaseClassifications()[0].GetId() != restClassificationID {
		t.Fatalf("expected grpc case classification id %q to match rest %q", successClassificationsResp.GetCaseClassifications()[0].GetId(), restClassificationID)
	}

	invalidGetCaseReq := &supportpb.GetCaseRequest{Name: "projects/stackyard/cases/bad*name"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSupportGetCaseMethod, invalidGetCaseReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "name-required") {
		t.Fatalf("expected grpc invalid argument for support get case, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	failedPreconditionReq := &supportpb.CloseCaseRequest{Name: "projects/stackyard/cases/case-closed-1"}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSupportCloseCaseMethod, failedPreconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "case-already-closed") {
		t.Fatalf("expected grpc failed precondition for support close case, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_ServiceManagement(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	}

	restListServicesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services?pageSize=1", nil, headers)
	if restListServicesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicemanagement list services, got %d body=%s", restListServicesResp.StatusCode, string(providerContractBody(t, restListServicesResp)))
	}
	restListServicesBody := providerContractJSONMap(t, restListServicesResp)
	restServices, ok := restListServicesBody["services"].([]any)
	if !ok || len(restServices) == 0 {
		t.Fatalf("expected services list in rest payload, got %#v", restListServicesBody["services"])
	}
	restService, _ := restServices[0].(map[string]any)
	serviceName, _ := restService["serviceName"].(string)
	if serviceName == "" {
		t.Fatalf("expected rest managed service name, got %#v", restService["serviceName"])
	}

	successListServicesReq := &servicemanagementpb.ListServicesRequest{
		PageSize: 1,
	}
	var successListServicesResp servicemanagementpb.ListServicesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementListServicesMethod, successListServicesReq, &successListServicesResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicemanagement list services, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListServicesResp.GetServices()) != 1 {
		t.Fatalf("expected one grpc managed service, got %d", len(successListServicesResp.GetServices()))
	}
	if successListServicesResp.GetServices()[0].GetServiceName() != serviceName {
		t.Fatalf("expected grpc managed service name %q to match rest %q", successListServicesResp.GetServices()[0].GetServiceName(), serviceName)
	}

	restGetConfigResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/"+serviceName+"/configs/2026-01-01r0?view=FULL", nil, headers)
	if restGetConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicemanagement get service config, got %d body=%s", restGetConfigResp.StatusCode, string(providerContractBody(t, restGetConfigResp)))
	}
	restGetConfigBody := providerContractJSONMap(t, restGetConfigResp)
	configID, _ := restGetConfigBody["id"].(string)
	if configID == "" {
		t.Fatalf("expected config id in rest payload, got %#v", restGetConfigBody["id"])
	}

	successGetConfigReq := &servicemanagementpb.GetServiceConfigRequest{
		ServiceName: serviceName,
		ConfigId:    configID,
		View:        servicemanagementpb.GetServiceConfigRequest_FULL,
	}
	var successGetConfigResp serviceconfigpb.Service
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementGetServiceConfigMethod, successGetConfigReq, &successGetConfigResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicemanagement get service config, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successGetConfigResp.GetId() != configID {
		t.Fatalf("expected grpc service config id %q to match rest %q", successGetConfigResp.GetId(), configID)
	}

	restListRolloutsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/"+serviceName+"/rollouts?pageSize=1&filter=status=SUCCESS", nil, headers)
	if restListRolloutsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicemanagement list service rollouts, got %d body=%s", restListRolloutsResp.StatusCode, string(providerContractBody(t, restListRolloutsResp)))
	}
	restListRolloutsBody := providerContractJSONMap(t, restListRolloutsResp)
	restRollouts, ok := restListRolloutsBody["rollouts"].([]any)
	if !ok || len(restRollouts) == 0 {
		t.Fatalf("expected rollouts list in rest payload, got %#v", restListRolloutsBody["rollouts"])
	}
	restRollout, _ := restRollouts[0].(map[string]any)
	restRolloutID, _ := restRollout["rolloutId"].(string)
	if restRolloutID == "" {
		t.Fatalf("expected rolloutId in rest payload, got %#v", restRollout["rolloutId"])
	}

	successListRolloutsReq := &servicemanagementpb.ListServiceRolloutsRequest{
		ServiceName: serviceName,
		PageSize:    1,
		Filter:      "status=SUCCESS",
	}
	var successListRolloutsResp servicemanagementpb.ListServiceRolloutsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementListServiceRolloutsMethod, successListRolloutsReq, &successListRolloutsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicemanagement list rollouts, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListRolloutsResp.GetRollouts()) != 1 {
		t.Fatalf("expected one grpc rollout, got %d", len(successListRolloutsResp.GetRollouts()))
	}
	if successListRolloutsResp.GetRollouts()[0].GetRolloutId() != restRolloutID {
		t.Fatalf("expected grpc rollout id %q to match rest %q", successListRolloutsResp.GetRollouts()[0].GetRolloutId(), restRolloutID)
	}

	restReportResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services:generateConfigReport", []byte(`{
		"newConfig":{"@type":"type.googleapis.com/google.api.Service","name":"stackyard.googleapis.com"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if restReportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest servicemanagement generate config report, got %d body=%s", restReportResp.StatusCode, string(providerContractBody(t, restReportResp)))
	}
	restReportBody := providerContractJSONMap(t, restReportResp)
	restServiceName, _ := restReportBody["serviceName"].(string)

	newConfig, err := anypb.New(&servicemanagementpb.ConfigRef{
		Name: "services/" + serviceName + "/configs/" + configID,
	})
	if err != nil {
		t.Fatalf("failed to construct anypb for servicemanagement report request: %v", err)
	}
	successReportReq := &servicemanagementpb.GenerateConfigReportRequest{
		NewConfig: newConfig,
	}
	var successReportResp servicemanagementpb.GenerateConfigReportResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementGenerateConfigReportMethod, successReportReq, &successReportResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for servicemanagement generate config report, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successReportResp.GetServiceName() != restServiceName {
		t.Fatalf("expected grpc report serviceName %q to match rest %q", successReportResp.GetServiceName(), restServiceName)
	}

	invalidGetServiceReq := &servicemanagementpb.GetServiceRequest{}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementGetServiceMethod, invalidGetServiceReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "service_name-required") {
		t.Fatalf("expected grpc invalid argument for servicemanagement get service, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidListRolloutsReq := &servicemanagementpb.ListServiceRolloutsRequest{
		ServiceName: serviceName,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementListServiceRolloutsMethod, invalidListRolloutsReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "filter-required") {
		t.Fatalf("expected grpc invalid argument for servicemanagement list rollouts, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	invalidGenerateReportReq := &servicemanagementpb.GenerateConfigReportRequest{}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpServiceManagementGenerateConfigReportMethod, invalidGenerateReportReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "new_config-required") {
		t.Fatalf("expected grpc invalid argument for servicemanagement generate config report, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecretManager(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "secretmanager",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/secrets?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest secretmanager list secrets, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["secrets"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected secrets list in rest payload, got %#v", restBody["secrets"])
	}
	restSecret, _ := restItems[0].(map[string]any)
	restSecretName, _ := restSecret["name"].(string)

	successListReq := &secretmanagerpb.ListSecretsRequest{
		Parent:   "projects/stackyard",
		PageSize: 1,
	}
	var successListResp secretmanagerpb.ListSecretsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecretManagerListSecretsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetSecrets()) != 1 {
		t.Fatalf("expected one grpc secret, got %d", len(successListResp.GetSecrets()))
	}
	if successListResp.GetSecrets()[0].GetName() != restSecretName {
		t.Fatalf("expected grpc secret name %q to match rest %q", successListResp.GetSecrets()[0].GetName(), restSecretName)
	}

	versionReq := &secretmanagerpb.AccessSecretVersionRequest{
		Name: "projects/stackyard/secrets/secret-1/versions/1",
	}
	var versionResp secretmanagerpb.AccessSecretVersionResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecretManagerAccessSecretVersionMethod, versionReq, &versionResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(versionResp.GetPayload().GetData()) == 0 {
		t.Fatalf("expected grpc access secret version payload data to be set")
	}

	invalidReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   "projects/stackyard",
		SecretId: "secret-1",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecretManagerCreateSecretMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "secret-required") {
		t.Fatalf("expected grpc invalid argument for secretmanager create secret, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &secretmanagerpb.EnableSecretVersionRequest{
		Name: "projects/stackyard/secrets/secret-1/versions/3",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecretManagerEnableSecretVersionMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "version-destroyed") {
		t.Fatalf("expected grpc failed precondition for secretmanager enable version, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityPrivateCA(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "security-privateca",
	}

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/caPools?pageSize=1", nil, headers)
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest security privateca list ca pools, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["caPools"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected caPools list in rest payload, got %#v", restBody["caPools"])
	}
	restPool, _ := restItems[0].(map[string]any)
	restPoolName, _ := restPool["name"].(string)

	successListReq := &privatecapb.ListCaPoolsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successListResp privatecapb.ListCaPoolsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityPrivateCAListCaPoolsMethod, successListReq, &successListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successListResp.GetCaPools()) != 1 {
		t.Fatalf("expected one grpc ca pool, got %d", len(successListResp.GetCaPools()))
	}
	if successListResp.GetCaPools()[0].GetName() != restPoolName {
		t.Fatalf("expected grpc ca pool name %q to match rest %q", successListResp.GetCaPools()[0].GetName(), restPoolName)
	}

	getReq := &privatecapb.GetCaPoolRequest{Name: restPoolName}
	var getResp privatecapb.CaPool
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPrivateCAGetCaPoolMethod, getReq, &getResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if getResp.GetName() != restPoolName {
		t.Fatalf("expected grpc get ca pool name %q to match rest %q", getResp.GetName(), restPoolName)
	}

	invalidReq := &privatecapb.CreateCaPoolRequest{
		Parent: "projects/stackyard/locations/us-central1",
		CaPool: &privatecapb.CaPool{
			Tier: privatecapb.CaPool_ENTERPRISE,
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPrivateCACreateCaPoolMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "ca_pool_id-required") {
		t.Fatalf("expected grpc invalid argument for security privateca create ca pool, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &privatecapb.RevokeCertificateRequest{
		Name:   "projects/stackyard/locations/us-central1/caPools/pool-1/certificates/cert-revoked",
		Reason: privatecapb.RevocationReason_KEY_COMPROMISE,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPrivateCARevokeCertificateMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "certificate-already-revoked") {
		t.Fatalf("expected grpc failed precondition for security privateca revoke certificate, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityPublicCA(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/externalAccountKeys", []byte(`{"externalAccountKey":{"name":"projects/stackyard/locations/global/externalAccountKeys/eak-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest security publicca create external account key, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restName, _ := restBody["name"].(string)

	successReq := &publiccapb.CreateExternalAccountKeyRequest{
		Parent: "projects/stackyard/locations/global",
		ExternalAccountKey: &publiccapb.ExternalAccountKey{
			Name: restName,
		},
	}
	var successResp publiccapb.ExternalAccountKey
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityPublicCACreateExternalAccountKeyMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if successResp.GetName() != restName {
		t.Fatalf("expected grpc external account key name %q to match rest %q", successResp.GetName(), restName)
	}
	if strings.TrimSpace(successResp.GetKeyId()) == "" {
		t.Fatalf("expected grpc keyId to be set, got %#v", successResp.GetKeyId())
	}

	invalidReq := &publiccapb.CreateExternalAccountKeyRequest{
		Parent: "projects/stackyard/locations/global",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPublicCACreateExternalAccountKeyMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "external_account_key-required") {
		t.Fatalf("expected grpc invalid argument for security publicca create key, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &publiccapb.CreateExternalAccountKeyRequest{
		Parent: "projects/stackyard/locations/us-central1",
		ExternalAccountKey: &publiccapb.ExternalAccountKey{
			Name: "projects/stackyard/locations/us-central1/externalAccountKeys/eak-1",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPublicCACreateExternalAccountKeyMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "location-global-required") {
		t.Fatalf("expected grpc failed precondition for security publicca non-global location, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityCenter(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/sources?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securitycenter",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest securitycenter list sources, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["sources"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected sources list in rest payload, got %#v", restBody["sources"])
	}
	restSource, _ := restItems[0].(map[string]any)
	restName, _ := restSource["name"].(string)

	successReq := &securitycenterpb.ListSourcesRequest{
		Parent:   "organizations/123456",
		PageSize: 1,
	}
	var successResp securitycenterpb.ListSourcesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterListSourcesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetSources()) != 1 {
		t.Fatalf("expected one grpc source, got %d", len(successResp.GetSources()))
	}
	if successResp.GetSources()[0].GetName() != restName {
		t.Fatalf("expected grpc source name %q to match rest %q", successResp.GetSources()[0].GetName(), restName)
	}

	invalidReq := &securitycenterpb.CreateSourceRequest{
		Parent: "organizations/123456",
		Source: &securitycenterpb.Source{},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterCreateSourceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "source.display_name-required") {
		t.Fatalf("expected grpc invalid argument for securitycenter create source, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &securitycenterpb.SetMuteRequest{
		Name: "organizations/123456/sources/source-1/findings/already-muted",
		Mute: securitycenterpb.Finding_MUTED,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterSetMuteMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "finding-already-muted") {
		t.Fatalf("expected grpc failed precondition for securitycenter set mute, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityCenterV2(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/sources?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securitycenter-apiv2",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest securitycenter v2 list sources, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["sources"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected sources list in rest payload, got %#v", restBody["sources"])
	}
	restSource, _ := restItems[0].(map[string]any)
	restName, _ := restSource["name"].(string)

	successReq := &securitycenterv2pb.ListSourcesRequest{
		Parent:   "organizations/123456",
		PageSize: 1,
	}
	var successResp securitycenterv2pb.ListSourcesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterV2ListSourcesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetSources()) != 1 {
		t.Fatalf("expected one grpc source, got %d", len(successResp.GetSources()))
	}
	if successResp.GetSources()[0].GetName() != restName {
		t.Fatalf("expected grpc source name %q to match rest %q", successResp.GetSources()[0].GetName(), restName)
	}

	invalidReq := &securitycenterv2pb.CreateSourceRequest{
		Parent: "organizations/123456",
		Source: &securitycenterv2pb.Source{},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterV2CreateSourceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "source.display_name-required") {
		t.Fatalf("expected grpc invalid argument for securitycenter v2 create source, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &securitycenterv2pb.SetMuteRequest{
		Name: "organizations/123456/sources/source-1/findings/already-muted",
		Mute: securitycenterv2pb.Finding_MUTED,
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterV2SetMuteMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "finding-already-muted") {
		t.Fatalf("expected grpc failed precondition for securitycenter v2 set mute, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecureSourceManager(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest securesourcemanager list repositories, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["repositories"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected repositories list in rest payload, got %#v", restBody["repositories"])
	}
	restRepo, _ := restItems[0].(map[string]any)
	restName, _ := restRepo["name"].(string)

	successReq := &securesourcemanagerpb.ListRepositoriesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp securesourcemanagerpb.ListRepositoriesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecureSourceManagerListRepositoriesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetRepositories()) != 1 {
		t.Fatalf("expected one grpc repository, got %d", len(successResp.GetRepositories()))
	}
	if successResp.GetRepositories()[0].GetName() != restName {
		t.Fatalf("expected grpc repository name %q to match rest %q", successResp.GetRepositories()[0].GetName(), restName)
	}

	invalidReq := &securesourcemanagerpb.ListRepositoriesRequest{
		Parent: "projects/stackyard",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecureSourceManagerListRepositoriesMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for securesourcemanager list repositories, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &securesourcemanagerpb.ClosePullRequestRequest{
		Name: "projects/stackyard/locations/us-central1/repositories/repository-1/pullRequests/pull-request-merged",
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecureSourceManagerClosePullRequestMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "pull_request-open-required") {
		t.Fatalf("expected grpc failed precondition for securesourcemanager close pull request, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_Retail(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "retail",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest retail list products, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["products"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected products list in rest payload, got %#v", restBody["products"])
	}
	restProduct, _ := restItems[0].(map[string]any)
	restName, _ := restProduct["name"].(string)

	successReq := &retailpb.ListProductsRequest{
		Parent:   "projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch",
		PageSize: 1,
	}
	var successResp retailpb.ListProductsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpRetailListProductsMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetProducts()) != 1 {
		t.Fatalf("expected one grpc product, got %d", len(successResp.GetProducts()))
	}
	if successResp.GetProducts()[0].GetName() != restName {
		t.Fatalf("expected grpc product name %q to match rest %q", successResp.GetProducts()[0].GetName(), restName)
	}

	invalidReq := &retailpb.CreateProductRequest{
		Parent: "projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch",
		Product: &retailpb.Product{
			Title: "invalid-missing-product-id",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpRetailCreateProductMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "product_id-required") {
		t.Fatalf("expected grpc invalid argument for retail create product, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityCenterManagement(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/securityCenterServices?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest securitycentermanagement list services, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["securityCenterServices"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected securityCenterServices list in rest payload, got %#v", restBody["securityCenterServices"])
	}
	restService, _ := restItems[0].(map[string]any)
	restName, _ := restService["name"].(string)

	successReq := &securitycentermanagementpb.ListSecurityCenterServicesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp securitycentermanagementpb.ListSecurityCenterServicesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterManagementListServicesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetSecurityCenterServices()) != 1 {
		t.Fatalf("expected one grpc security center service, got %d", len(successResp.GetSecurityCenterServices()))
	}
	if successResp.GetSecurityCenterServices()[0].GetName() != restName {
		t.Fatalf("expected grpc security center service name %q to match rest %q", successResp.GetSecurityCenterServices()[0].GetName(), restName)
	}

	invalidReq := &securitycentermanagementpb.UpdateSecurityCenterServiceRequest{
		SecurityCenterService: &securitycentermanagementpb.SecurityCenterService{
			Name: "projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"bad_field"}},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterManagementUpdateServiceMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "update_mask-invalid") {
		t.Fatalf("expected grpc invalid argument for securitycentermanagement update service, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	preconditionReq := &securitycentermanagementpb.UpdateSecurityCenterServiceRequest{
		SecurityCenterService: &securitycentermanagementpb.SecurityCenterService{
			Name:                    "projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics",
			IntendedEnablementState: securitycentermanagementpb.SecurityCenterService_INGEST_ONLY,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"intended_enablement_state"}},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityCenterManagementUpdateServiceMethod, preconditionReq, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "intended_enablement_state-read-only") {
		t.Fatalf("expected grpc failed precondition for securitycentermanagement update service, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_SecurityPosture(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations/global/postures?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest securityposture list postures, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["postures"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected postures list in rest payload, got %#v", restBody["postures"])
	}
	restPosture, _ := restItems[0].(map[string]any)
	restName, _ := restPosture["name"].(string)

	successReq := &securityposturepb.ListPosturesRequest{
		Parent:   "organizations/123456/locations/global",
		PageSize: 1,
	}
	var successResp securityposturepb.ListPosturesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpSecurityPostureListPosturesMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetPostures()) != 1 {
		t.Fatalf("expected one grpc posture, got %d", len(successResp.GetPostures()))
	}
	if successResp.GetPostures()[0].GetName() != restName {
		t.Fatalf("expected grpc posture name %q to match rest %q", successResp.GetPostures()[0].GetName(), restName)
	}

	invalidReq := &securityposturepb.UpdatePostureRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Posture: &securityposturepb.Posture{
			Name: "organizations/123456/locations/global/postures/posture-1",
		},
	}
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpSecurityPostureUpdatePostureMethod, invalidReq, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "revision_id-required") {
		t.Fatalf("expected grpc invalid argument for securityposture update posture, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func newGCPStage4GRPCContractServer(t *testing.T) *httptest.Server {
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

func assertGCPStage4GRPCUnary(t *testing.T, ts *httptest.Server, methodPath string, req proto.Message, out proto.Message) (grpcStatus, grpcMessage string) {
	t.Helper()

	body := grpcStage4EncodeUnaryRequest(t, req)
	resp := providerContractRequest(t, ts, http.MethodPost, methodPath, body, map[string]string{
		"Content-Type": "application/grpc",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 grpc bridge response for %s, got %d body=%s", methodPath, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload, status, message := grpcStage4DecodeUnaryResponse(t, resp)
	if out != nil && len(payload) > 0 {
		if err := proto.Unmarshal(payload, out); err != nil {
			t.Fatalf("unmarshal grpc payload for %s: %v", methodPath, err)
		}
	}
	return status, message
}

func grpcStage4EncodeUnaryRequest(t *testing.T, msg proto.Message) []byte {
	t.Helper()

	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal grpc request: %v", err)
	}
	frame := make([]byte, 5+len(payload))
	frame[0] = 0
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func grpcStage4DecodeUnaryResponse(t *testing.T, resp *http.Response) (payload []byte, grpcStatus, grpcMessage string) {
	t.Helper()

	data := providerContractBody(t, resp)
	if len(data) < 5 {
		t.Fatalf("expected grpc frame in response, got %d bytes", len(data))
	}
	if data[0] != 0 {
		t.Fatalf("expected uncompressed grpc frame, got compression flag %d", data[0])
	}
	frameLen := int(binary.BigEndian.Uint32(data[1:5]))
	if frameLen != len(data)-5 {
		t.Fatalf("grpc frame length mismatch header=%d actual=%d", frameLen, len(data)-5)
	}
	payload = bytes.Clone(data[5:])
	grpcStatus = strings.TrimSpace(resp.Trailer.Get("Grpc-Status"))
	grpcMessage = strings.TrimSpace(resp.Trailer.Get("Grpc-Message"))
	if grpcStatus == "" {
		grpcStatus = strings.TrimSpace(resp.Header.Get("Grpc-Status"))
	}
	if grpcMessage == "" {
		grpcMessage = strings.TrimSpace(resp.Header.Get("Grpc-Message"))
	}
	return payload, grpcStatus, grpcMessage
}
