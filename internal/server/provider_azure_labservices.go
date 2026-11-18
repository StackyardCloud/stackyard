package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Microsoft.LabServices: lab plans, which hold lab defaults and the images
// labs can use, and labs with their users, schedules and virtual machines.
// Publishing a lab creates a virtual machine per seat. Nothing runs: the
// machines only change state, and invitations are never emailed.
const azureLabNamespace = "Microsoft.LabServices"

// ponytail: every lab plan offers the same two marketplace images; a
// catalog per region is the upgrade if tests need more.
var azureLabMarketplaceImages = []map[string]any{
	{"name": "microsoftwindowsserver.windowsserver.2022-datacenter-g2", "displayName": "Windows Server 2022 Datacenter",
		"publisher": "MicrosoftWindowsServer", "offer": "WindowsServer", "sku": "2022-datacenter-g2", "osType": "Windows"},
	{"name": "canonical.0001-com-ubuntu-server-jammy.22_04-lts-gen2", "displayName": "Ubuntu Server 22.04 LTS",
		"publisher": "canonical", "offer": "0001-com-ubuntu-server-jammy", "sku": "22_04-lts-gen2", "osType": "Linux"},
}

func init() {
	images := azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labPlans/images", Proxy: true, NoDelete: true,
		ValidateName: azureLabValidateName,
		Validate:     azureLabImageValidate,
		Linked:       azureLabImageLinked,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, azureLabImageCheck, "enabledState", "availableRegions")
		},
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labPlans",
		ValidateName:    azureLabValidateName,
		Validate:        azureLabPlanValidate,
		PatchBody:       azureLabPlanPatch,
		DefaultChildren: func(plan *azureARMResource) []*azureARMResource { return azureLabPlanImages(plan, images) },
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"saveimage": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
				return s.azureLabSaveImage(rec, payload, images)
			},
		},
	})
	registerAzureARMResourceType(images)

	vms := azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labs/virtualMachines", Proxy: true, NoDelete: true, NoPatch: true,
		Validate: func(map[string]any) string {
			return "Lab virtual machines are created by publishing the lab."
		},
		StateProperties: map[string]any{"state": "Stopped"},
		Transitions: map[string]map[string]any{
			"start": {"state": "Running"}, "stop": {"state": "Stopped"},
			// A reimaged machine is rebuilt from the template, stopped;
			// a redeployed one moves hosts and keeps its state.
			"reimage": {"state": "Stopped"}, "redeploy": {},
		},
		Actions: map[string]azureARMAction{"resetpassword": azureLabResetPassword},
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labs",
		ValidateName:        azureLabValidateName,
		Validate:            azureLabValidate,
		WriteOnlyProperties: []string{"virtualMachineProfile.adminUser.password", "virtualMachineProfile.nonAdminUser.password"},
		StateProperties:     map[string]any{"state": "Draft"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"securityProfile": map[string]any{"registrationCode": azureLabRegistrationCode(rec.ID)}}
		},
		Linked: func(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
			return s.azureLabLinked(rec, vms)
		},
		PatchBody: azureLabPatch,
		ValidatePatch: func(s *Server, _ *azureARMResource, payload map[string]any) string {
			if planID, present := azureARMObject(payload["properties"])["labPlanId"]; present {
				return s.azureLabPlanMissingLocked(azureARMString(planID))
			}
			return ""
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"publish": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
				return s.azureLabPublish(rec, vms)
			},
		},
		Actions: map[string]azureARMAction{"syncgroup": azureLabSyncGroup},
	})
	registerAzureARMResourceType(vms)
	azureLabRegisterMembers()
}

// azureLabPlanImages lists the marketplace images a new lab plan offers,
// disabled until a lab creator enables them.
func azureLabPlanImages(plan *azureARMResource, images azureARMResourceType) []*azureARMResource {
	out := make([]*azureARMResource, 0, len(azureLabMarketplaceImages))
	for _, image := range azureLabMarketplaceImages {
		props := azureARMMerge(image, map[string]any{
			"enabledState": "Disabled", "termsStatus": "Enabled", "osState": "Generalized", "version": "latest",
			"availableRegions": []any{plan.Location},
			"sharedGalleryId": "/subscriptions/" + plan.SubscriptionID + "/Providers/Microsoft.Compute/Locations/" + plan.Location +
				"/Publishers/" + azureARMString(image["publisher"]) + "/ArtifactTypes/VMImage/Offers/" + azureARMString(image["offer"]) +
				"/Skus/" + azureARMString(image["sku"]) + "/Versions/latest",
		})
		delete(props, "name")
		out = append(out, azureARMChild(plan, images, azureARMString(image["name"]), props))
	}
	return out
}

// azureLabImageLinked lets PUT update only an image that exists, keeping
// what the gallery says about it. Caller holds providerStorageMu.
func azureLabImageLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	existing := s.azureARMResources[azureARMKey(rec.ID)]
	if existing == nil {
		return nil, nil, "Images come from the lab plan's galleries and saved lab images; PUT only updates an existing image."
	}
	props := azureARMMerge(existing.Properties, nil)
	for key := range rec.Properties {
		if key == "enabledState" || key == "availableRegions" {
			delete(props, key)
		}
	}
	return props, nil, ""
}

// azureLabSaveImage saves a lab virtual machine as an image in the lab
// plan's shared gallery.
func (s *Server) azureLabSaveImage(rec *azureARMResource, payload map[string]any, images azureARMResourceType) (int, any) {
	name := azureARMString(payload["name"])
	if !azureLabName.MatchString(name) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required: 1 to 100 letters, digits, hyphens, underscores or periods.")
	}
	vmID := azureARMString(payload["labVirtualMachineId"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	plan := s.azureARMResources[azureARMKey(rec.ID)]
	if plan == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	gallery := azureARMString(plan.Properties["sharedGalleryId"])
	if gallery == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Lab plan "+plan.Name+" has no shared image gallery (properties.sharedGalleryId) to save images to.")
	}
	vm := s.azureARMResources[azureARMKey(vmID)]
	if vm == nil || !strings.EqualFold(vm.Type, azureLabNamespace+"/labs/virtualMachines") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "labVirtualMachineId must be the ID of a lab virtual machine.")
	}
	lab := s.azureARMResources[azureARMKey(vm.ID[:strings.LastIndex(strings.ToLower(vm.ID), "/virtualmachines/")])]
	if lab == nil || !strings.EqualFold(azureARMString(lab.Properties["labPlanId"]), plan.ID) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "labVirtualMachineId must be a virtual machine of a lab in lab plan "+plan.Name+".")
	}
	osType := azureARMString(azureARMObject(lab.Properties["virtualMachineProfile"])["osType"])
	image := azureARMChild(plan, images, name, map[string]any{
		"displayName": name, "description": "Saved from lab virtual machine " + vm.ID + ".", "osType": osType, "osState": "Specialized",
		"enabledState": "Disabled", "sharedGalleryId": gallery + "/images/" + name, "availableRegions": []any{plan.Location},
	})
	if s.azureARMResources[azureARMKey(image.ID)] != nil {
		return http.StatusConflict, azureARMErrorBody("Conflict", "Lab plan "+plan.Name+" already has an image named "+name+".")
	}
	image.CreatedAt, image.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	s.azureARMResources[azureARMKey(image.ID)] = image
	return http.StatusOK, nil
}

// azureLabLinked checks the lab plan, keeps the network profile the lab
// was created with (or takes the plan's), derives the OS type and adds the
// template virtual machine of TemplateVM labs. Caller holds
// providerStorageMu.
func (s *Server) azureLabLinked(rec *azureARMResource, vms azureARMResourceType) (map[string]any, []*azureARMResource, string) {
	props := rec.Properties
	out := map[string]any{}
	planID := azureARMString(props["labPlanId"])
	if planID != "" {
		if msg := s.azureLabPlanMissingLocked(planID); msg != "" {
			return nil, nil, msg
		}
	}
	existing := s.azureARMResources[azureARMKey(rec.ID)]
	switch {
	case existing != nil:
		stored := existing.Properties["networkProfile"]
		if sent, present := props["networkProfile"]; present && !reflect.DeepEqual(sent, stored) {
			return nil, nil, "properties.networkProfile can't change after the lab is created."
		}
		if stored != nil {
			out["networkProfile"] = stored
		}
	case planID != "" && props["networkProfile"] == nil:
		plan := s.azureARMResources[azureARMKey(planID)]
		if subnet := azureARMObject(plan.Properties["defaultNetworkProfile"])["subnetId"]; subnet != nil {
			out["networkProfile"] = map[string]any{"subnetId": subnet}
		}
	}
	vmProfile := azureARMObject(props["virtualMachineProfile"])
	out["virtualMachineProfile"] = map[string]any{"osType": s.azureLabOSTypeLocked(azureARMObject(vmProfile["imageReference"]))}
	var linked []*azureARMResource
	if vmProfile["createOption"] == "TemplateVM" {
		stored := s.azureARMResources[azureARMKey(rec.ID+"/virtualMachines/template")]
		linked = append(linked, azureLabRefreshVM(stored, azureLabVM(rec, vms, "template", "Template")))
	}
	return out, linked, ""
}

// azureLabOSTypeLocked is the OS of a stored lab plan image, or else
// guessed from the marketplace image's names.
// ponytail: the guess knows common Linux publishers only.
func (s *Server) azureLabOSTypeLocked(ref map[string]any) string {
	if image := s.azureARMResources[azureARMKey(azureARMString(ref["id"]))]; image != nil {
		if osType := azureARMString(image.Properties["osType"]); osType != "" {
			return osType
		}
	}
	names := strings.ToLower(fmt.Sprint(ref["id"], ref["publisher"], ref["offer"], ref["sku"]))
	for _, linux := range []string{"canonical", "ubuntu", "debian", "redhat", "rhel", "centos", "suse", "linux", "rocky"} {
		if strings.Contains(names, linux) {
			return "Linux"
		}
	}
	return "Windows"
}

// azureLabVM builds a stopped lab virtual machine with the lab's
// connection details.
func azureLabVM(lab *azureARMResource, vms azureARMResourceType, name, vmType string) *azureARMResource {
	vmProfile := azureARMObject(lab.Properties["virtualMachineProfile"])
	access := azureARMObject(lab.Properties["connectionProfile"])
	host := strings.ToLower(lab.Name) + "-" + name + ".labs.stackyard.invalid"
	ip := 4
	if n, err := strconv.Atoi(name); err == nil {
		ip = 5 + n
	}
	conn := map[string]any{
		"privateIpAddress": fmt.Sprintf("10.0.%d.%d", ip/250, ip%250),
		"adminUsername":    azureARMObject(vmProfile["adminUser"])["username"],
	}
	if user := azureARMObject(vmProfile["nonAdminUser"])["username"]; user != nil {
		conn["nonAdminUsername"] = user
	}
	for _, way := range []struct{ access, key, value string }{
		{"clientRdpAccess", "rdpAuthority", host + ":3389"}, {"clientSshAccess", "sshAuthority", host + ":22"},
		{"webRdpAccess", "rdpInBrowserUrl", "https://" + host + "/rdp"}, {"webSshAccess", "sshInBrowserUrl", "https://" + host + "/ssh"},
	} {
		if level := azureARMString(access[way.access]); level != "" && level != "None" {
			conn[way.key] = way.value
		}
	}
	return azureARMChild(lab, vms, name, map[string]any{"vmType": vmType, "connectionProfile": conn})
}

// azureLabRefreshVM returns fresh, or a copy of the stored machine (its
// state kept) with fresh's connection details, which follow the lab.
func azureLabRefreshVM(stored, fresh *azureARMResource) *azureARMResource {
	if stored == nil {
		return fresh
	}
	updated := *stored
	updated.Properties = azureARMMerge(stored.Properties, nil)
	updated.Properties["connectionProfile"] = fresh.Properties["connectionProfile"]
	return &updated
}

// azureLabPlanMissingLocked rejects a labPlanId that isn't a stored lab
// plan. Caller holds providerStorageMu.
func (s *Server) azureLabPlanMissingLocked(planID string) string {
	if plan := s.azureARMResources[azureARMKey(planID)]; plan == nil || !strings.EqualFold(plan.Type, azureLabNamespace+"/labPlans") {
		return "properties.labPlanId must be the ID of an existing lab plan."
	}
	return ""
}

// azureLabPublish publishes a lab: it has one user virtual machine per
// seat, named by number, and seats removed since the last publish go.
// ponytail: seats follow the capacity only on publish, as a republish
// rescales a lab; changing the capacity alone doesn't.
func (s *Server) azureLabPublish(rec *azureARMResource, vms azureARMResourceType) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	lab := s.azureARMResources[azureARMKey(rec.ID)]
	if lab == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	seats, _ := azureARMObject(azureARMObject(lab.Properties["virtualMachineProfile"])["sku"])["capacity"].(float64)
	for i := 0; i < int(seats); i++ {
		key := azureARMKey(lab.ID + "/virtualMachines/" + strconv.Itoa(i))
		s.azureARMResources[key] = azureLabRefreshVM(s.azureARMResources[key], azureLabVM(lab, vms, strconv.Itoa(i), "User"))
	}
	for _, vm := range s.azureARMInlineChildrenLocked(lab.ID, vms) {
		if n, err := strconv.Atoi(vm.Name); err == nil && n >= int(seats) {
			s.deleteAzureARMTreeLocked(vm.ID)
		}
	}
	updated := *lab
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(lab.Properties, map[string]any{"state": "Published"})
	s.azureARMResources[azureARMKey(lab.ID)] = &updated
	return http.StatusOK, nil
}

func azureLabSyncGroup(rec *azureARMResource, _ map[string]any) (int, any) {
	if azureARMString(azureARMObject(rec.Properties["rosterProfile"])["activeDirectoryGroupId"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent",
			"Lab "+rec.Name+" has no Azure AD group to sync; set properties.rosterProfile.activeDirectoryGroupId.")
	}
	return http.StatusOK, nil
}

// azureLabResetPassword accepts a new password for one of the machine's
// users; the emulator keeps no passwords.
func azureLabResetPassword(rec *azureARMResource, payload map[string]any) (int, any) {
	user, password := azureARMString(payload["username"]), azureARMString(payload["password"])
	conn := azureARMObject(rec.Properties["connectionProfile"])
	if user == "" || (user != conn["adminUsername"] && user != conn["nonAdminUsername"]) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "username must be the virtual machine's admin or non-admin user.")
	}
	if password == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "password is required.")
	}
	return http.StatusOK, nil
}

// azureLabRegistrationCode is the code students use to register, derived
// from the lab ID.
func azureLabRegistrationCode(id string) string {
	code := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(azureARMDerivedKey(id, "registration")))
	return code[:8]
}

// azureLabSKUSizes are the lab sizes Skus - List returns, from its sample.
// ponytail: four sizes in eastus2; the full regional catalog is the upgrade.
var azureLabSKUSizes = []struct {
	name, family                   string
	cpus, memory, storage          string
	windows, inactive, hybridOrLin float64
}{
	{"Standard_Fv2_2_4GB_64_S_SSD", "Fv2", "2", "4", "64", 20.4, 0.7, 20.4},
	{"Standard_Fv2_2_4GB_256_S_SSD", "Fv2", "2", "4", "256", 22.8, 2.7, 22.8},
	{"Standard_Dv4_2_8GB_128_S_SSD", "Dv4", "2", "8", "128", 24.2, 1.4, 24.2},
	{"Standard_Dv4_2_8GB_256_S_SSD", "Dv4", "2", "8", "256", 25.8, 2.7, 25.8},
}

func azureLabSKUs() []map[string]any {
	items := make([]map[string]any, 0, len(azureLabSKUSizes))
	for _, size := range azureLabSKUSizes {
		cost := func(unit string, quantity float64) map[string]any {
			return map[string]any{"meterId": "", "quantity": quantity, "extendedUnit": unit}
		}
		capability := func(name, value string) map[string]any { return map[string]any{"name": name, "value": value} }
		items = append(items, map[string]any{
			"resourceType": "labs", "name": size.name, "tier": "Standard", "size": strings.TrimPrefix(size.name, "Standard_"),
			"family": size.family, "locations": []any{"eastus2"},
			"capacity": map[string]any{"minimum": 0, "maximum": 400, "default": 1, "scaleType": "Automatic"},
			"costs": []any{cost("WindowsHourly", size.windows), cost("HybridBenefitHourly", size.hybridOrLin),
				cost("InactiveHourly", size.inactive), cost("LinuxHourly", size.hybridOrLin)},
			"capabilities": []any{capability("vCPUs", size.cpus), capability("MemoryGB", size.memory), capability("StorageGB", size.storage),
				capability("StorageType", "StandardSSD"), capability("HyperVGenerations", "V1,V2"), capability("IsGpu", "False")},
		})
	}
	return items
}

// azureLabUsages counts, per SKU family, the seats of the subscription's
// labs in the location. scope is .../providers/Microsoft.LabServices/
// locations/{location}.
// ponytail: labs with sizes outside azureLabSKUSizes aren't counted, and
// every family's limit is the SKU maximum.
func azureLabUsages(s *Server, scope string) []map[string]any {
	parts := strings.Split(scope, "/")
	sub, location := parts[2], strings.ReplaceAll(parts[len(parts)-1], " ", "")
	used := map[string]float64{}
	for _, lab := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureLabNamespace+"/labs") && strings.EqualFold(rec.SubscriptionID, sub) &&
			strings.EqualFold(strings.ReplaceAll(rec.Location, " ", ""), location)
	}) {
		sku := azureARMObject(azureARMObject(lab.Properties["virtualMachineProfile"])["sku"])
		for _, size := range azureLabSKUSizes {
			if size.name == sku["name"] {
				seats, _ := sku["capacity"].(float64)
				used[size.family] += seats
			}
		}
	}
	items := []map[string]any{}
	for _, family := range []string{"Dv4", "Fv2"} {
		items = append(items, map[string]any{"id": scope + "/usages/" + family, "name": map[string]any{"value": family},
			"unit": "Count", "currentValue": used[family], "limit": 400})
	}
	return items
}
