package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

const nmTestPolicy = `{"version":"2021.12",
 "core-network-configuration":{"asn-ranges":["64512-64555"],"edge-locations":[{"location":"us-east-1"},{"location":"us-west-2"}]},
 "segments":[{"name":"prod"},{"name":"dev","require-attachment-acceptance":true}],
 "segment-actions":[{"action":"share","segment":"prod","share-with":["dev"]}],
 "attachment-policies":[{"rule-number":100,"conditions":[{"type":"tag-value","key":"env","value":"dev"}],"action":{"association-method":"constant","segment":"dev"}},
  {"rule-number":200,"conditions":[{"type":"any"}],"action":{"association-method":"constant","segment":"prod"}}]}`

func nmQuote(s string) string { return fmt.Sprintf("%q", s) }

func TestNetworkManagerStage1GlobalNetworkInventory(t *testing.T) {
	_, ts := newNetworkManagerTestServer(t)
	gn := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks", `{"Description":"corp","Tags":[{"Key":"team","Value":"net"}]}`)["GlobalNetwork"].(map[string]any)
	gnID := gn["GlobalNetworkId"].(string)
	if gn["State"] != "PENDING" || !strings.HasPrefix(gnID, "global-network-") {
		t.Fatalf("create: %v", gn)
	}
	list := nmCall(t, ts, http.StatusOK, http.MethodGet, "/global-networks?globalNetworkIds="+gnID, "")["GlobalNetworks"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["State"] != "AVAILABLE" {
		t.Fatalf("describe settles to AVAILABLE: %v", list)
	}
	base := "/global-networks/" + gnID
	site := nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/sites", `{"Description":"hq","Location":{"Address":"Main St"}}`)["Site"].(map[string]any)["SiteId"].(string)
	if code := nmErrCode(t, ts, http.MethodPost, base+"/devices", `{"SiteId":"site-00000000000000000"}`); code != "ResourceNotFoundException" {
		t.Fatalf("device on a missing site: %s", code)
	}
	dev1 := nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/devices", `{"SiteId":`+nmQuote(site)+`,"Vendor":"acme"}`)["Device"].(map[string]any)["DeviceId"].(string)
	dev2 := nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/devices", `{"Vendor":"acme"}`)["Device"].(map[string]any)["DeviceId"].(string)
	if code := nmErrCode(t, ts, http.MethodPost, base+"/links", `{"SiteId":`+nmQuote(site)+`}`); code != "ValidationException" {
		t.Fatalf("link without bandwidth: %s", code)
	}
	link := nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/links", `{"SiteId":`+nmQuote(site)+`,"Bandwidth":{"UploadSpeed":50,"DownloadSpeed":100},"Provider":"isp"}`)["Link"].(map[string]any)["LinkId"].(string)
	if got := nmCall(t, ts, http.StatusOK, http.MethodGet, base+"/links?provider=isp", "")["Links"].([]any); len(got) != 1 {
		t.Fatalf("links by provider: %v", got)
	}
	if code := nmErrCode(t, ts, http.MethodPost, base+"/connections", `{"DeviceId":`+nmQuote(dev1)+`,"ConnectedDeviceId":`+nmQuote(dev1)+`}`); code != "ValidationException" {
		t.Fatalf("self connection: %s", code)
	}
	conn := nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/connections", `{"DeviceId":`+nmQuote(dev1)+`,"ConnectedDeviceId":`+nmQuote(dev2)+`}`)["Connection"].(map[string]any)["ConnectionId"].(string)
	if got := nmCall(t, ts, http.StatusOK, http.MethodGet, base+"/connections?deviceId="+dev2, "")["Connections"].([]any); len(got) != 1 {
		t.Fatalf("connections by either device: %v", got)
	}

	assoc := `{"DeviceId":` + nmQuote(dev1) + `,"LinkId":` + nmQuote(link) + `}`
	nmCall(t, ts, http.StatusOK, http.MethodPost, base+"/link-associations", assoc)
	if code := nmErrCode(t, ts, http.MethodPost, base+"/link-associations", assoc); code != "ConflictException" {
		t.Fatalf("duplicate link association: %s", code)
	}
	if code := nmErrCode(t, ts, http.MethodDelete, base+"/links/"+link, ""); code != "ConflictException" {
		t.Fatalf("delete an associated link: %s", code)
	}
	if code := nmErrCode(t, ts, http.MethodDelete, base+"/sites/"+site, ""); code != "ConflictException" {
		t.Fatalf("delete a site in use: %s", code)
	}
	if code := nmErrCode(t, ts, http.MethodDelete, "/global-networks/"+gnID, ""); code != "ConflictException" {
		t.Fatalf("delete a global network with sites: %s", code)
	}
	nmCall(t, ts, http.StatusOK, http.MethodDelete, base+"/link-associations?deviceId="+dev1+"&linkId="+link, "")
	nmCall(t, ts, http.StatusOK, http.MethodDelete, base+"/connections/"+conn, "")
	for _, p := range []string{"/links/" + link, "/devices/" + dev1, "/devices/" + dev2, "/sites/" + site} {
		nmCall(t, ts, http.StatusOK, http.MethodDelete, base+p, "")
	}

	arn := url.PathEscape(gn["GlobalNetworkArn"].(string))
	nmCall(t, ts, http.StatusOK, http.MethodPost, "/tags/"+arn, `{"Tags":[{"Key":"env","Value":"prod"}]}`)
	if tags := nmCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["TagList"].([]any); len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	if code := nmErrCode(t, ts, http.MethodPost, "/tags/"+arn, `{"Tags":[{"Key":"aws:x","Value":"y"}]}`); code != "ValidationException" {
		t.Fatalf("aws: tag key: %s", code)
	}
	deleted := nmCall(t, ts, http.StatusOK, http.MethodDelete, "/global-networks/"+gnID, "")["GlobalNetwork"].(map[string]any)
	if deleted["State"] != "DELETING" {
		t.Fatalf("delete: %v", deleted)
	}
	nmCall(t, ts, http.StatusNotFound, http.MethodGet, base+"/sites", "")
}

func TestNetworkManagerStage1CoreNetworkPolicyAndAttachments(t *testing.T) {
	srv, ts := newNetworkManagerTestServer(t)
	gnID := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks", `{}`)["GlobalNetwork"].(map[string]any)["GlobalNetworkId"].(string)

	bad := nmCall(t, ts, http.StatusBadRequest, http.MethodPost, "/core-networks", `{"GlobalNetworkId":`+nmQuote(gnID)+`,"PolicyDocument":"{\"version\":\"2021.12\",\"segments\":[]}"}`)
	if errs, _ := bad["Errors"].([]any); len(errs) == 0 {
		t.Fatalf("invalid policy should list errors: %v", bad)
	}
	core := nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks", `{"GlobalNetworkId":`+nmQuote(gnID)+`,"ClientToken":"tok-1"}`)["CoreNetwork"].(map[string]any)
	coreID := core["CoreNetworkId"].(string)
	again := nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks", `{"GlobalNetworkId":`+nmQuote(gnID)+`,"ClientToken":"tok-1"}`)["CoreNetwork"].(map[string]any)
	if again["CoreNetworkId"] != coreID {
		t.Fatalf("client token should return the same core network: %v", again)
	}

	vpc, _ := srv.ec2.CreateVpc("10.20.0.0/16", nil)
	subA, _ := srv.ec2.CreateSubnet(vpc.ID, "10.20.1.0/24", "us-east-1a", nil)
	subB, _ := srv.ec2.CreateSubnet(vpc.ID, "10.20.2.0/24", "us-east-1a", nil)
	vpcArn := "arn:aws:ec2:us-east-1:123456789012:vpc/" + vpc.ID
	subArn := func(id string) string { return "arn:aws:ec2:us-east-1:123456789012:subnet/" + id }
	attach := func(tags string) string {
		return `{"CoreNetworkId":` + nmQuote(coreID) + `,"VpcArn":` + nmQuote(vpcArn) + `,"SubnetArns":[` + nmQuote(subArn(subA.ID)) + `]` + tags + `}`
	}
	if code := nmErrCode(t, ts, http.MethodPost, "/vpc-attachments", attach("")); code != "ValidationException" {
		t.Fatalf("attachment without a LIVE policy: %s", code)
	}

	put := nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks/"+coreID+"/core-network-policy", `{"PolicyDocument":`+nmQuote(nmTestPolicy)+`}`)["CoreNetworkPolicy"].(map[string]any)
	if put["ChangeSetState"] != "PENDING_GENERATION" || put["Alias"] != "LATEST" {
		t.Fatalf("put policy: %v", put)
	}
	if code := nmErrCode(t, ts, http.MethodPost, "/core-networks/"+coreID+"/core-network-policy", `{"PolicyDocument":`+nmQuote(nmTestPolicy)+`,"LatestVersionId":7}`); code != "ConflictException" {
		t.Fatalf("stale LatestVersionId: %s", code)
	}
	changes := nmCall(t, ts, http.StatusOK, http.MethodGet, "/core-networks/"+coreID+"/core-network-change-sets/1", "")["CoreNetworkChanges"].([]any)
	if len(changes) != 4 { // two segments and two edges are added
		t.Fatalf("change set: %v", changes)
	}
	nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks/"+coreID+"/core-network-change-sets/1/execute", "")
	live := nmCall(t, ts, http.StatusOK, http.MethodGet, "/core-networks/"+coreID+"/core-network-policy?alias=LIVE", "")["CoreNetworkPolicy"].(map[string]any)
	if live["PolicyVersionId"] != float64(1) || live["ChangeSetState"] != "EXECUTION_SUCCEEDED" {
		t.Fatalf("live policy: %v", live)
	}
	got := nmCall(t, ts, http.StatusOK, http.MethodGet, "/core-networks/"+coreID, "")["CoreNetwork"].(map[string]any)
	if segs := got["Segments"].([]any); len(segs) != 2 || len(got["Edges"].([]any)) != 2 {
		t.Fatalf("core network segments/edges: %v", got)
	}
	if code := nmErrCode(t, ts, http.MethodDelete, "/core-networks/"+coreID+"/core-network-policy-versions/1", ""); code != "ConflictException" {
		t.Fatalf("delete the LIVE policy version: %s", code)
	}

	// One subnet per Availability Zone.
	dup := `{"CoreNetworkId":` + nmQuote(coreID) + `,"VpcArn":` + nmQuote(vpcArn) + `,"SubnetArns":[` + nmQuote(subArn(subA.ID)) + `,` + nmQuote(subArn(subB.ID)) + `]}`
	if code := nmErrCode(t, ts, http.MethodPost, "/vpc-attachments", dup); code != "ValidationException" {
		t.Fatalf("two subnets in one zone: %s", code)
	}
	missing := `{"CoreNetworkId":` + nmQuote(coreID) + `,"VpcArn":"arn:aws:ec2:us-east-1:123456789012:vpc/vpc-0123456789abcdef0","SubnetArns":[]}`
	if code := nmErrCode(t, ts, http.MethodPost, "/vpc-attachments", missing); code != "ResourceNotFoundException" {
		t.Fatalf("missing VPC: %s", code)
	}

	prod := nmCall(t, ts, http.StatusOK, http.MethodPost, "/vpc-attachments", attach(""))["VpcAttachment"].(map[string]any)["Attachment"].(map[string]any)
	if prod["SegmentName"] != "prod" || prod["State"] != "CREATING" || prod["EdgeLocation"] != "us-east-1" {
		t.Fatalf("prod attachment: %v", prod)
	}
	dev := nmCall(t, ts, http.StatusOK, http.MethodPost, "/vpc-attachments", attach(`,"Tags":[{"Key":"env","Value":"dev"}]`))["VpcAttachment"].(map[string]any)["Attachment"].(map[string]any)
	devID := dev["AttachmentId"].(string)
	if dev["SegmentName"] != "dev" || dev["State"] != "PENDING_ATTACHMENT_ACCEPTANCE" {
		t.Fatalf("dev attachment needs acceptance: %v", dev)
	}
	if code := nmErrCode(t, ts, http.MethodPost, "/attachments/"+prod["AttachmentId"].(string)+"/accept", ""); code != "ConflictException" {
		t.Fatalf("accept an attachment that isn't pending: %s", code)
	}
	accepted := nmCall(t, ts, http.StatusOK, http.MethodPost, "/attachments/"+devID+"/accept", "")["Attachment"].(map[string]any)
	if accepted["State"] != "CREATING" {
		t.Fatalf("accept: %v", accepted)
	}

	// Routes come from AVAILABLE VPC attachments on the segment edge.
	routes := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks/"+gnID+"/network-routes",
		`{"RouteTableIdentifier":{"CoreNetworkSegmentEdge":{"CoreNetworkId":`+nmQuote(coreID)+`,"SegmentName":"prod","EdgeLocation":"us-east-1"}}}`)
	if r := routes["NetworkRoutes"].([]any); len(r) != 1 || r[0].(map[string]any)["DestinationCidrBlock"] != "10.20.0.0/16" {
		t.Fatalf("network routes: %v", routes)
	}
	counts := nmCall(t, ts, http.StatusOK, http.MethodGet, "/global-networks/"+gnID+"/network-resource-count", "")["NetworkResourceCounts"].([]any)
	if len(counts) != 2 {
		t.Fatalf("resource counts (core network, attachments): %v", counts)
	}
	if code := nmErrCode(t, ts, http.MethodDelete, "/core-networks/"+coreID, ""); code != "ConflictException" {
		t.Fatalf("delete a core network with attachments: %s", code)
	}
	for _, a := range []string{prod["AttachmentId"].(string), devID} {
		nmCall(t, ts, http.StatusOK, http.MethodDelete, "/attachments/"+a, "")
	}
	nmCall(t, ts, http.StatusOK, http.MethodDelete, "/core-networks/"+coreID, "")
}

func TestNetworkManagerStage1PeeringsAndRegistrations(t *testing.T) {
	srv, ts := newNetworkManagerTestServer(t)
	gnID := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks", `{}`)["GlobalNetwork"].(map[string]any)["GlobalNetworkId"].(string)
	tgw, err := srv.ec2.CreateTransitGateway("hub", ec2.TransitGatewayOptionsInput{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tgwArn := "arn:aws:ec2:us-east-1:123456789012:transit-gateway/" + tgw.ID
	missing := "arn:aws:ec2:us-east-1:123456789012:transit-gateway/tgw-0123456789abcdef0"
	if code := nmErrCode(t, ts, http.MethodPost, "/global-networks/"+gnID+"/transit-gateway-registrations", `{"TransitGatewayArn":`+nmQuote(missing)+`}`); code != "ResourceNotFoundException" {
		t.Fatalf("register a missing transit gateway: %s", code)
	}
	reg := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks/"+gnID+"/transit-gateway-registrations", `{"TransitGatewayArn":`+nmQuote(tgwArn)+`}`)
	if reg["TransitGatewayRegistration"].(map[string]any)["State"].(map[string]any)["Code"] != "PENDING" {
		t.Fatalf("registration: %v", reg)
	}
	if code := nmErrCode(t, ts, http.MethodPost, "/global-networks/"+gnID+"/transit-gateway-registrations", `{"TransitGatewayArn":`+nmQuote(tgwArn)+`}`); code != "ConflictException" {
		t.Fatalf("duplicate registration: %s", code)
	}

	coreID := nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks", `{"GlobalNetworkId":`+nmQuote(gnID)+`,"PolicyDocument":`+nmQuote(nmTestPolicy)+`}`)["CoreNetwork"].(map[string]any)["CoreNetworkId"].(string)
	nmCall(t, ts, http.StatusOK, http.MethodGet, "/core-networks/"+coreID+"/core-network-policy?alias=LATEST", "")
	nmCall(t, ts, http.StatusOK, http.MethodPost, "/core-networks/"+coreID+"/core-network-change-sets/1/execute", "")
	peering := nmCall(t, ts, http.StatusOK, http.MethodPost, "/transit-gateway-peerings", `{"CoreNetworkId":`+nmQuote(coreID)+`,"TransitGatewayArn":`+nmQuote(tgwArn)+`}`)
	pid := peering["TransitGatewayPeering"].(map[string]any)["Peering"].(map[string]any)["PeeringId"].(string)
	if code := nmErrCode(t, ts, http.MethodPost, "/transit-gateway-peerings", `{"CoreNetworkId":`+nmQuote(coreID)+`,"TransitGatewayArn":`+nmQuote(tgwArn)+`}`); code != "ConflictException" {
		t.Fatalf("duplicate peering: %s", code)
	}
	nmCall(t, ts, http.StatusOK, http.MethodGet, "/transit-gateway-peerings/"+pid, "")
	rtb := nmCall(t, ts, http.StatusOK, http.MethodPost, "/transit-gateway-route-table-attachments",
		`{"PeeringId":`+nmQuote(pid)+`,"TransitGatewayRouteTableArn":"arn:aws:ec2:us-east-1:123456789012:transit-gateway-route-table/tgw-rtb-0123456789abcdef0"}`)
	rtbID := rtb["TransitGatewayRouteTableAttachment"].(map[string]any)["Attachment"].(map[string]any)["AttachmentId"].(string)
	if code := nmErrCode(t, ts, http.MethodDelete, "/peerings/"+pid, ""); code != "ConflictException" {
		t.Fatalf("delete a peering with a route table attachment: %s", code)
	}
	nmCall(t, ts, http.StatusOK, http.MethodDelete, "/attachments/"+rtbID, "")
	nmCall(t, ts, http.StatusOK, http.MethodDelete, "/peerings/"+pid, "")
	nmCall(t, ts, http.StatusOK, http.MethodDelete, "/global-networks/"+gnID+"/transit-gateway-registrations/"+url.PathEscape(tgwArn), "")
}
