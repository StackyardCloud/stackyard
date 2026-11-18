package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureLabTestAPIVersion = "?api-version=2023-06-07"

const azureLabTestBody = `{"location":"eastus2","properties":{
	"autoShutdownProfile":{"shutdownOnDisconnect":"Enabled","disconnectDelay":"PT15M","shutdownOnIdle":"UserAbsence"},
	"connectionProfile":{"clientRdpAccess":"Public","webSshAccess":"None"},
	"securityProfile":{"openAccess":"Disabled"},
	"virtualMachineProfile":{"createOption":"TemplateVM","usageQuota":"PT10H",
		"imageReference":{"publisher":"MicrosoftWindowsServer","offer":"WindowsServer","sku":"2022-datacenter-g2"},
		"sku":{"name":"Standard_Fv2_2_4GB_64_S_SSD","capacity":2},
		"adminUser":{"username":"labadmin","password":"Secret-Pass-1"}},
	"title":"Intro to Networking","labPlanId":"%PLAN%"}}`

func azureLabTestBodyFor(plan string) string {
	return strings.ReplaceAll(azureLabTestBody, "%PLAN%", plan)
}

func TestAzureLabServicesLabPlansAndImages(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-labs")
	ns := sub + "/resourceGroups/rg-labs/providers/Microsoft.LabServices"
	plan := ns + "/labPlans/plan1"
	created := providerContractJSONMap(t, do(http.MethodPut, plan+azureLabTestAPIVersion, `{"location":"eastus2","identity":{"type":"SystemAssigned"},
		"properties":{"allowedRegions":["eastus2"],"defaultConnectionProfile":{"clientRdpAccess":"Public"},
		"defaultNetworkProfile":{"subnetId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s"}}}`))
	if created["properties"].(map[string]any)["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected lab plan: %#v", created)
	}

	// Every plan offers the marketplace images, disabled.
	list := providerContractJSONMap(t, do(http.MethodGet, plan+"/images"+azureLabTestAPIVersion, ""))["value"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected two marketplace images, got %#v", list)
	}
	image := plan + "/images/canonical.0001-com-ubuntu-server-jammy.22_04-lts-gen2"
	enabled := providerContractJSONMap(t, do(http.MethodPut, image+azureLabTestAPIVersion, `{"properties":{"enabledState":"Enabled"}}`))
	ep := enabled["properties"].(map[string]any)
	if ep["enabledState"] != "Enabled" || ep["osType"] != "Linux" || ep["publisher"] != "canonical" || ep["sharedGalleryId"] == nil {
		t.Fatalf("PUT should enable the image and keep its gallery details: %#v", enabled)
	}
	patched := providerContractJSONMap(t, do(http.MethodPatch, image+azureLabTestAPIVersion, `{"properties":{"enabledState":"Disabled"}}`))
	if pp := patched["properties"].(map[string]any); pp["enabledState"] != "Disabled" || pp["displayName"] != "Ubuntu Server 22.04 LTS" {
		t.Fatalf("unexpected patched image: %#v", patched)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, plan+"/images/custom"+azureLabTestAPIVersion, `{"properties":{"enabledState":"Enabled"}}`), "existing image")
	azureSentinelExpectRejected(t, do(http.MethodPut, image+azureLabTestAPIVersion, `{"properties":{}}`), "enabledState is required")
	azureSentinelExpectRejected(t, do(http.MethodPut, image+azureLabTestAPIVersion, `{"properties":{"enabledState":"Enabled","description":"x"}}`), "description can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, image+azureLabTestAPIVersion, `{"properties":{"availableRegions":[1]}}`), "availableRegions")
	azureSentinelExpectRejected(t, do(http.MethodPatch, image+azureLabTestAPIVersion, `{"properties":{"osType":"Windows"}}`), "can't be updated")
	azureResourcesExpectStatus(t, do(http.MethodDelete, image+azureLabTestAPIVersion, ""), http.StatusMethodNotAllowed)

	// PATCH updates the plan's defaults; saving an image needs a gallery.
	azureSentinelExpectRejected(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion, `{"name":"img","labVirtualMachineId":"x"}`), "shared image gallery")
	gallery := "/subscriptions/x/resourceGroups/rg/providers/Microsoft.Compute/galleries/sig"
	azureResourcesExpectStatus(t, do(http.MethodPatch, plan+azureLabTestAPIVersion, `{"properties":{"sharedGalleryId":"`+gallery+`"}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion, `{"name":"img","labVirtualMachineId":"x"}`), "labVirtualMachineId")
	azureSentinelExpectRejected(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion, `{"labVirtualMachineId":"x"}`), "name is required")

	lab := ns + "/labs/lab1"
	azureResourcesExpectStatus(t, do(http.MethodPut, lab+azureLabTestAPIVersion, azureLabTestBodyFor(strings.TrimPrefix(plan, "/azure"))), http.StatusCreated)
	vmID := strings.TrimPrefix(lab, "/azure") + "/virtualMachines/template"
	azureSentinelExpectRejected(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion,
		`{"name":"networking-v1","labVirtualMachineId":"`+strings.TrimPrefix(lab, "/azure")+`"}`), "labVirtualMachineId")
	azureResourcesExpectStatus(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion, `{"name":"networking-v1","labVirtualMachineId":"`+vmID+`"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion, `{"name":"networking-v1","labVirtualMachineId":"`+vmID+`"}`), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion,
		`{"name":"microsoftwindowsserver.windowsserver.2022-datacenter-g2","labVirtualMachineId":"`+vmID+`"}`), http.StatusConflict, "Conflict")
	other := strings.Replace(azureLabTestBodyFor(strings.TrimPrefix(plan, "/azure")), `,"labPlanId":"`+strings.TrimPrefix(plan, "/azure")+`"`, ``, 1)
	azureResourcesExpectStatus(t, do(http.MethodPut, ns+"/labs/planless"+azureLabTestAPIVersion, other), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPost, plan+"/saveImage"+azureLabTestAPIVersion,
		`{"name":"other","labVirtualMachineId":"`+strings.TrimPrefix(ns, "/azure")+`/labs/planless/virtualMachines/template"}`), "lab in lab plan")
	saved := providerContractJSONMap(t, do(http.MethodGet, plan+"/images/networking-v1"+azureLabTestAPIVersion, ""))["properties"].(map[string]any)
	if saved["osType"] != "Windows" || saved["osState"] != "Specialized" || saved["sharedGalleryId"] != gallery+"/images/networking-v1" {
		t.Fatalf("unexpected saved image: %#v", saved)
	}

	for body, want := range map[string]string{
		`{"location":"eastus2","identity":{"type":"UserAssigned"}}`:                                    "identity.type",
		`{"location":"eastus2","properties":{"allowedRegions":"eastus2"}}`:                             "allowedRegions",
		`{"location":"eastus2","properties":{"allowedRegions":[""]}}`:                                  "allowedRegions",
		`{"location":"eastus2","properties":{"defaultConnectionProfile":{"webRdpAccess":"Open"}}}`:     "webRdpAccess",
		`{"location":"eastus2","properties":{"defaultAutoShutdownProfile":{"idleDelay":"5 minutes"}}}`: "idleDelay",
		`{"location":"eastus2","properties":{"sharedGalleryId":"x"}}`:                                  "sharedGalleryId",
		`{"location":"eastus2","properties":{"supportInfo":"call us"}}`:                                "supportInfo",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, ns+"/labPlans/bad"+azureLabTestAPIVersion, body), want)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, plan+azureLabTestAPIVersion, `{"properties":{"provisioningState":"Failed"}}`), "can't be updated")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ns+"/labPlans/bad%20name"+azureLabTestAPIVersion, `{"location":"eastus2"}`), http.StatusBadRequest, "InvalidResourceName")
}

func TestAzureLabServicesLabLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-labs")
	ns := sub + "/resourceGroups/rg-labs/providers/Microsoft.LabServices"
	plan := strings.TrimPrefix(ns, "/azure") + "/labPlans/plan1"
	azureResourcesExpectStatus(t, do(http.MethodPut, ns+"/labPlans/plan1"+azureLabTestAPIVersion,
		`{"location":"eastus2","properties":{"defaultNetworkProfile":{"subnetId":"/subscriptions/x/subnets/s"}}}`), http.StatusCreated)
	lab := ns + "/labs/lab1"
	own := strings.Replace(azureLabTestBodyFor(plan), `"title"`, `"networkProfile":{"subnetId":"/subscriptions/x/subnets/own"},"title"`, 1)
	explicit := providerContractJSONMap(t, do(http.MethodPut, ns+"/labs/own"+azureLabTestAPIVersion, own))["properties"].(map[string]any)
	if explicit["networkProfile"].(map[string]any)["subnetId"] != "/subscriptions/x/subnets/own" {
		t.Fatalf("a lab's own network profile wins over the plan's: %#v", explicit)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, ns+"/labs/wrong"+azureLabTestAPIVersion,
		azureLabTestBodyFor(strings.TrimPrefix(ns, "/azure")+"/labs/own")), "existing lab plan")
	created := providerContractJSONMap(t, do(http.MethodPut, lab+azureLabTestAPIVersion, azureLabTestBodyFor(plan)))
	props := created["properties"].(map[string]any)
	vmProfile := props["virtualMachineProfile"].(map[string]any)
	code := props["securityProfile"].(map[string]any)["registrationCode"]
	if props["state"] != "Draft" || vmProfile["osType"] != "Windows" || len(azureARMString(code)) != 8 ||
		props["networkProfile"].(map[string]any)["subnetId"] != "/subscriptions/x/subnets/s" || props["securityProfile"].(map[string]any)["openAccess"] != "Disabled" {
		t.Fatalf("unexpected lab: %#v", created)
	}
	if _, leaked := vmProfile["adminUser"].(map[string]any)["password"]; leaked {
		t.Fatalf("the admin password must not be returned: %#v", vmProfile)
	}

	// A TemplateVM lab has its template machine before publishing.
	vms := providerContractJSONMap(t, do(http.MethodGet, lab+"/virtualMachines"+azureLabTestAPIVersion, ""))["value"].([]any)
	if len(vms) != 1 {
		t.Fatalf("expected the template machine, got %#v", vms)
	}
	template := vms[0].(map[string]any)["properties"].(map[string]any)
	conn := template["connectionProfile"].(map[string]any)
	if template["vmType"] != "Template" || template["state"] != "Stopped" || conn["adminUsername"] != "labadmin" || conn["rdpAuthority"] == nil || conn["sshInBrowserUrl"] != nil {
		t.Fatalf("unexpected template machine: %#v", template)
	}
	vm := lab + "/virtualMachines/template"
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/start"+azureLabTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/redeploy"+azureLabTestAPIVersion, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodGet, vm+azureLabTestAPIVersion, ""))["properties"].(map[string]any)["state"]; got != "Running" {
		t.Fatalf("a started, redeployed machine should run, got %v", got)
	}
	noPlan := strings.Replace(strings.Replace(azureLabTestBodyFor(plan), `,"labPlanId":"`+plan+`"`, ``, 1), `"labadmin"`, `"newadmin"`, 1)
	kept := providerContractJSONMap(t, do(http.MethodPut, lab+azureLabTestAPIVersion, noPlan))["properties"].(map[string]any)
	if kept["networkProfile"] == nil {
		t.Fatalf("a PUT without the plan should keep the lab's network profile: %#v", kept)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, vm+azureLabTestAPIVersion, ""))["properties"].(map[string]any)["state"]; got != "Running" {
		t.Fatalf("a PUT should keep the template machine's state, got %v", got)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, vm+azureLabTestAPIVersion, ""))["properties"].(map[string]any)["connectionProfile"].(map[string]any)["adminUsername"]; got != "newadmin" {
		t.Fatalf("the template machine should follow the lab's admin user, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/resetPassword"+azureLabTestAPIVersion, `{"username":"newadmin","password":"New-Pass-2"}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPost, vm+"/resetPassword"+azureLabTestAPIVersion, `{"username":"someone","password":"x"}`), "username")
	azureSentinelExpectRejected(t, do(http.MethodPost, vm+"/resetPassword"+azureLabTestAPIVersion, `{"username":"newadmin"}`), "password is required")
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/reimage"+azureLabTestAPIVersion, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodGet, vm+azureLabTestAPIVersion, ""))["properties"].(map[string]any)["state"]; got != "Stopped" {
		t.Fatalf("a reimaged machine should be stopped, got %v", got)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, lab+"/virtualMachines/9"+azureLabTestAPIVersion, `{}`), "publishing")
	azureResourcesExpectStatus(t, do(http.MethodDelete, vm+azureLabTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodPatch, vm+azureLabTestAPIVersion, `{}`), http.StatusMethodNotAllowed)

	// Publishing adds a machine per seat; republishing after scaling down drops the extra one.
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/publish"+azureLabTestAPIVersion, ""), http.StatusOK)
	if got := len(providerContractJSONMap(t, do(http.MethodGet, lab+"/virtualMachines"+azureLabTestAPIVersion, ""))["value"].([]any)); got != 3 {
		t.Fatalf("expected the template and two seats, got %d", got)
	}
	seat := providerContractJSONMap(t, do(http.MethodGet, lab+"/virtualMachines/1"+azureLabTestAPIVersion, ""))["properties"].(map[string]any)
	if seat["vmType"] != "User" || seat["state"] != "Stopped" {
		t.Fatalf("unexpected seat: %#v", seat)
	}
	usages := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.LabServices/locations/eastus2/usages"+azureLabTestAPIVersion, ""))["value"].([]any)
	for _, raw := range usages {
		usage := raw.(map[string]any)
		if usage["name"].(map[string]any)["value"] == "Fv2" && usage["currentValue"] != float64(4) {
			t.Fatalf("both labs' two seats should count against Fv2: %#v", usage)
		}
	}

	patched := providerContractJSONMap(t, do(http.MethodPatch, lab+azureLabTestAPIVersion,
		`{"properties":{"title":"Networking II","virtualMachineProfile":{"osType":"Linux","sku":{"capacity":1}}}}`))
	pp := patched["properties"].(map[string]any)
	pvm := pp["virtualMachineProfile"].(map[string]any)
	if pp["title"] != "Networking II" || pp["state"] != "Published" || pvm["osType"] != "Windows" || pvm["sku"].(map[string]any)["name"] != "Standard_Fv2_2_4GB_64_S_SSD" {
		t.Fatalf("PATCH should change the title and seats only: %#v", patched)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/publish"+azureLabTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, lab+"/virtualMachines/1"+azureLabTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodGet, lab+"/virtualMachines/0"+azureLabTestAPIVersion, ""), http.StatusOK)

	// A PUT keeps the state and network; switching to images drops the template machine.
	image := strings.Replace(azureLabTestBodyFor(plan), `"createOption":"TemplateVM"`, `"createOption":"Image"`, 1)
	replaced := providerContractJSONMap(t, do(http.MethodPut, lab+azureLabTestAPIVersion, image))["properties"].(map[string]any)
	if replaced["state"] != "Published" || replaced["networkProfile"] == nil {
		t.Fatalf("PUT should keep the state and network profile: %#v", replaced)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vm+azureLabTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureSentinelExpectRejected(t, do(http.MethodPut, lab+azureLabTestAPIVersion,
		strings.Replace(image, `"title"`, `"networkProfile":{"subnetId":"/subscriptions/x/subnets/other"},"title"`, 1)), "networkProfile can't change")

	azureResourcesExpectErrorCode(t, do(http.MethodPatch, lab+azureLabTestAPIVersion, `{"properties":{"labPlanId":"/subscriptions/x/labPlans/none"}}`), http.StatusConflict, "Conflict")
	azureResourcesExpectStatus(t, do(http.MethodPatch, lab+azureLabTestAPIVersion, `{"properties":{"labPlanId":"`+plan+`"}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPost, lab+"/syncGroup"+azureLabTestAPIVersion, ""), "activeDirectoryGroupId")
	azureResourcesExpectStatus(t, do(http.MethodPatch, lab+azureLabTestAPIVersion, `{"properties":{"rosterProfile":{"activeDirectoryGroupId":"g1"}}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/syncGroup"+azureLabTestAPIVersion, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, lab+azureLabTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, lab+"/virtualMachines/0"+azureLabTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureLabServicesLabValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-labs")
	lab := sub + "/resourceGroups/rg-labs/providers/Microsoft.LabServices/labs/lab1" + azureLabTestAPIVersion
	valid := azureLabTestBodyFor("/subscriptions/x/resourceGroups/rg/providers/Microsoft.LabServices/labPlans/none")
	for _, tc := range []struct{ from, to, want string }{
		{`"labPlanId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.LabServices/labPlans/none"`, `"labPlanId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.LabServices/labPlans/none"`, "existing lab plan"},
		{`"securityProfile":{"openAccess":"Disabled"},`, ``, "securityProfile is required"},
		{`"createOption":"TemplateVM",`, ``, "createOption is required"},
		{`"usageQuota":"PT10H",`, ``, "usageQuota is required"},
		{`"usageQuota":"PT10H"`, `"usageQuota":"PT10000H"`, "usageQuota"},
		{`"createOption":"TemplateVM"`, `"createOption":"Snapshot"`, "createOption must be one of"},
		{`"publisher":"MicrosoftWindowsServer",`, ``, "imageReference"},
		{`"name":"Standard_Fv2_2_4GB_64_S_SSD",`, ``, "sku.name is required"},
		{`"capacity":2`, `"capacity":401`, "sku.capacity"},
		{`"username":"labadmin",`, ``, "adminUser.username is required"},
		{`,"password":"Secret-Pass-1"`, ``, "adminUser.password is required"},
		{`"clientRdpAccess":"Public"`, `"clientRdpAccess":"Open"`, "clientRdpAccess"},
		{`"shutdownOnIdle":"UserAbsence"`, `"shutdownOnIdle":"Always"`, "shutdownOnIdle"},
		{`"disconnectDelay":"PT15M"`, `"disconnectDelay":"15"`, "disconnectDelay"},
		{`"openAccess":"Disabled"`, `"openAccess":"Yes"`, "openAccess"},
		{`"title":"Intro to Networking"`, `"title":""`, "title"},
		{`"title":"Intro to Networking"`, `"title":"x","description":7`, "description"},
		{`"title":"Intro to Networking"`, `"title":"x","networkProfile":"default"`, "networkProfile must be an object"},
		{`"title":"Intro to Networking"`, `"title":"x","networkProfile":{"subnetId":"x"}`, "networkProfile.subnetId"},
	} {
		body := strings.Replace(valid, tc.from, tc.to, 1)
		if tc.want != "existing lab plan" && body == valid {
			t.Fatalf("replacement %q did not apply", tc.from)
		}
		azureSentinelExpectRejected(t, do(http.MethodPut, lab, body), tc.want)
	}
	noPlan := strings.Replace(valid, `,"labPlanId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.LabServices/labPlans/none"`, ``, 1)
	linux := strings.Replace(noPlan, `"publisher":"MicrosoftWindowsServer","offer":"WindowsServer"`, `"publisher":"canonical","offer":"ubuntu"`, 1)
	created := providerContractJSONMap(t, do(http.MethodPut, lab, linux))["properties"].(map[string]any)
	if created["virtualMachineProfile"].(map[string]any)["osType"] != "Linux" || created["networkProfile"] != nil {
		t.Fatalf("a lab without a plan should have no network and take its OS from the image: %#v", created)
	}
	withNet := strings.Replace(linux, `"title"`, `"networkProfile":{"subnetId":"/subscriptions/x/subnets/s"},"title"`, 1)
	azureSentinelExpectRejected(t, do(http.MethodPut, lab, withNet), "networkProfile can't change")
	for body, want := range map[string]string{
		`{"properties":{"virtualMachineProfile":{"createOption":"Image"}}}`:                              "createOption can't change",
		`{"properties":{"virtualMachineProfile":{"imageReference":{"id":"x"}}}}`:                         "imageReference can't change",
		`{"properties":{"networkProfile":{"subnetId":"/subscriptions/x/subnets/s"}}}`:                    "networkProfile can't be updated",
		`{"properties":{"state":"Published"}}`:                                                           "state can't be updated",
		`{"properties":{"virtualMachineProfile":{"usageQuota":"forever"}}}`:                              "usageQuota",
		`{"properties":{"virtualMachineProfile":{"nonAdminUser":{"username":""}}}}`:                      "nonAdminUser.username",
		`{"properties":{"virtualMachineProfile":{"additionalCapabilities":{"installGpuDrivers":"On"}}}}`: "installGpuDrivers",
		`{"properties":{"virtualMachineProfile":{"useSharedPassword":"On"}}}`:                            "useSharedPassword",
		`{"properties":"x"}`: "properties must be an object",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPatch, lab, body), want)
	}
}

func TestAzureLabServicesUsersAndSchedules(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-labs")
	lab := sub + "/resourceGroups/rg-labs/providers/Microsoft.LabServices/labs/lab1"
	body := strings.Replace(azureLabTestBodyFor(""), `,"labPlanId":""`, ``, 1)
	azureResourcesExpectStatus(t, do(http.MethodPut, lab+azureLabTestAPIVersion, body), http.StatusCreated)

	user := lab + "/users/student1" + azureLabTestAPIVersion
	created := providerContractJSONMap(t, do(http.MethodPut, user, `{"properties":{"email":"student1@contoso.com","additionalUsageQuota":"PT5H"}}`))
	up := created["properties"].(map[string]any)
	if up["invitationState"] != "NotSent" || up["registrationState"] != "NotRegistered" || up["totalUsage"] != "PT0S" || up["invitationSent"] != nil {
		t.Fatalf("unexpected user: %#v", created)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, lab+"/users/student1/invite"+azureLabTestAPIVersion, `{"text":"Welcome"}`), http.StatusOK)
	invited := providerContractJSONMap(t, do(http.MethodGet, user, ""))["properties"].(map[string]any)
	if invited["invitationState"] != "Sent" || invited["invitationSent"] == nil {
		t.Fatalf("invite should mark the invitation sent: %#v", invited)
	}
	again := providerContractJSONMap(t, do(http.MethodPut, user, `{"properties":{"email":"student1@contoso.com","invitationState":"NotSent"}}`))["properties"].(map[string]any)
	if again["invitationState"] != "Sent" || again["invitationSent"] != invited["invitationSent"] {
		t.Fatalf("PUT should keep the invitation: %#v", again)
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, user, `{"properties":{"additionalUsageQuota":"PT8H"}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPatch, user, `{"properties":{"email":"other@contoso.com"}}`), "can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, user, `{"properties":{"additionalUsageQuota":"8 hours"}}`), "additionalUsageQuota")
	azureSentinelExpectRejected(t, do(http.MethodPost, lab+"/users/student1/invite"+azureLabTestAPIVersion, `{"text":5}`), "text must be a string")
	for _, email := range []string{`"a@b"`, `"student.contoso.com"`, `"@contoso.com"`, `"a@b@contoso.com"`, `"student@"`, `7`} {
		azureSentinelExpectRejected(t, do(http.MethodPut, lab+"/users/bad"+azureLabTestAPIVersion, `{"properties":{"email":`+email+`}}`), "properties.email")
	}
	if items := providerContractJSONMap(t, do(http.MethodGet, lab+"/users"+azureLabTestAPIVersion, ""))["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one user, got %#v", items)
	}

	schedule := lab + "/schedules/weekdays" + azureLabTestAPIVersion
	sched := providerContractJSONMap(t, do(http.MethodPut, schedule, `{"properties":{"startAt":"2026-12-01T09:00:00Z","stopAt":"2026-12-01T17:00:00Z",
		"timeZoneId":"America/Los_Angeles","notes":"Class hours","recurrencePattern":{"frequency":"Weekly","interval":1,"weekDays":["Monday","Wednesday"],"expirationDate":"2027-01-31T23:59:59Z"}}}`))
	if sched["properties"].(map[string]any)["timeZoneId"] != "America/Los_Angeles" {
		t.Fatalf("unexpected schedule: %#v", sched)
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, schedule, `{"properties":{"notes":"Moved","timeZoneId":"UTC"}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPatch, schedule, `{"properties":{"provisioningState":"Failed"}}`), "can't be updated")
	for body, want := range map[string]string{
		`{"properties":{"timeZoneId":"UTC"}}`:                                                                                                    "stopAt is required",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z"}}`:                                                                                       "timeZoneId is required",
		`{"properties":{"stopAt":"tomorrow","timeZoneId":"UTC"}}`:                                                                                "stopAt must be",
		`{"properties":{"startAt":"2026-12-01T18:00:00Z","stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC"}}`:                                   "before",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"Pacific Time"}}`:                                                           "timeZoneId",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"America/` + strings.Repeat("x", 50) + `"}}`:                                "timeZoneId",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","notes":7}}`:                                                          "notes",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{}}}`:                                             "frequency is required",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Hourly"}}}`:                         "frequency must be",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Daily","interval":0}}}`:             "interval",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Weekly"}}}`:                         "at least one day",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Weekly","weekDays":["Funday"]}}}`:   "weekDays items",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Daily","weekDays":"Monday"}}}`:      "weekDays must be a list",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":{"frequency":"Daily","expirationDate":"never"}}}`: "expirationDate",
		`{"properties":{"stopAt":"2026-12-01T17:00:00Z","timeZoneId":"UTC","recurrencePattern":"daily"}}`:                                        "recurrencePattern must be an object",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, lab+"/schedules/bad"+azureLabTestAPIVersion, body), want)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, schedule, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, user, ""), http.StatusOK)
}

func TestAzureLabServicesSkusAndUsages(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-labs")
	ns := sub + "/providers/Microsoft.LabServices"
	skus := providerContractJSONMap(t, do(http.MethodGet, ns+"/skus"+azureLabTestAPIVersion, ""))["value"].([]any)
	if len(skus) != 4 || skus[0].(map[string]any)["family"] != "Fv2" || skus[0].(map[string]any)["size"] != "Fv2_2_4GB_64_S_SSD" {
		t.Fatalf("unexpected SKUs: %#v", skus)
	}
	body := strings.Replace(strings.Replace(azureLabTestBodyFor(""), `,"labPlanId":""`, ``, 1), `"name":"Standard_Fv2_2_4GB_64_S_SSD","capacity":2`, `"name":"Standard_Dv4_2_8GB_128_S_SSD","capacity":3`, 1)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-labs/providers/Microsoft.LabServices/labs/dv4"+azureLabTestAPIVersion, body), http.StatusCreated)
	for location, want := range map[string]float64{"westus": 0, "eastus2": 3} {
		usages := providerContractJSONMap(t, do(http.MethodGet, ns+"/locations/"+location+"/usages"+azureLabTestAPIVersion, ""))["value"].([]any)
		dv4, fv2 := usages[0].(map[string]any), usages[1].(map[string]any)
		if len(usages) != 2 || dv4["name"].(map[string]any)["value"] != "Dv4" || dv4["currentValue"] != want || fv2["currentValue"] != float64(0) || dv4["limit"] != float64(400) {
			t.Fatalf("unexpected %s usages: %#v", location, usages)
		}
	}
	other := "/azure/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.LabServices/locations/eastus2/usages" + azureLabTestAPIVersion
	if got := providerContractJSONMap(t, do(http.MethodGet, other, ""))["value"].([]any)[0].(map[string]any)["currentValue"]; got != float64(0) {
		t.Fatalf("usages belong to their subscription, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, ns+"/skus"+azureLabTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, ns+"/skus?api-version=", ""), http.StatusBadRequest, "InvalidApiVersionParameter")
}
