package server

import (
	"net/http"
	"testing"
)

func TestAzureComputeLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-compute"
	location := "eastus"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	computeBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute"

	for _, path := range []string{
		"/azure/providers/Microsoft.Compute/operations?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/skus?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/usages?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/vmSizes?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/publishers?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/publishers/Canonical/artifactTypes/vmImage/offers?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/publishers/Canonical/artifactTypes/vmImage/offers/0001-com-ubuntu-server-jammy/skus?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/publishers/Canonical/artifactTypes/vmImage/offers/0001-com-ubuntu-server-jammy/skus/22_04-lts-gen2/versions?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/communityGalleries?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/locations/" + location + "/communityGalleries/community-gallery-a?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/sharedGalleries?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/sharedGalleries/shared-gallery-a?api-version=" + azureComputeAPIVersion,
	} {
		resp := providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for GET %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	availabilitySetBase := computeBase + "/availabilitySets/aset-a"
	resp := providerContractRequest(t, ts, http.MethodPut, availabilitySetBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"platformFaultDomainCount":2,"platformUpdateDomainCount":5}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating availability set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		availabilitySetBase + "?api-version=" + azureComputeAPIVersion,
		computeBase + "/availabilitySets?api-version=" + azureComputeAPIVersion,
		subscriptionBase + "/providers/Microsoft.Compute/availabilitySets?api-version=" + azureComputeAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for GET %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
	resp = providerContractRequest(t, ts, http.MethodHead, availabilitySetBase+"?api-version="+azureComputeAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for availability set HEAD, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/capacityReservationGroups/crg-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"sharingProfile":{"subscriptionIds":["00000000-0000-0000-0000-000000000000"]}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating capacity reservation group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/capacityReservationGroups/crg-a/capacityReservations/cr-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"sku":{"name":"Standard_D2s_v3"},"capacity":1}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating capacity reservation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/hostGroups/hg-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"zones":["1"],
		"properties":{"platformFaultDomainCount":1}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating host group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/hostGroups/hg-a/hosts/host-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"sku":{"name":"DSv3-Type1"},"platformFaultDomain":1}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating host, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/disks/disk-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"creationData":{"createOption":"Empty"},"diskSizeGB":64}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating disk, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/snapshots/snap-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"creationData":{"createOption":"Copy","sourceResourceId":"`+computeBase+`/disks/disk-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating snapshot, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/images/img-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"storageProfile":{"osDisk":{"osType":"Linux","snapshot":{"id":"`+computeBase+`/snapshots/snap-a"}}}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	galleryBase := computeBase + "/galleries/gallery-a"
	resp = providerContractRequest(t, ts, http.MethodPut, galleryBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"description":"gallery"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating gallery, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, galleryBase+"/images/image-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"identifier":{"publisher":"stackyard","offer":"demo","sku":"image-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating gallery image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, galleryBase+"/images/image-a/versions/1.0.0?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"storageProfile":{"source":{"id":"`+computeBase+`/images/img-a"}}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating gallery image version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	vmBase := computeBase + "/virtualMachines/vm-a"
	resp = providerContractRequest(t, ts, http.MethodPut, vmBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_DS1_v2"},
			"storageProfile":{"imageReference":{"id":"`+computeBase+`/images/img-a"}},
			"osProfile":{"computerName":"vm-a","adminUsername":"stackyard"},
			"networkProfile":{"networkInterfaces":[]}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VM, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, vmBase+"/start?api-version="+azureComputeAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 starting VM, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, vmBase+"/instanceView?api-version="+azureComputeAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting VM instance view, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, vmBase+"/extensions/ext-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"publisher":"Contoso","type":"Diagnostics","typeHandlerVersion":"1.0"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VM extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, vmBase+"/runCommands/rc-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"source":{"script":"echo stackyard"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VM run command, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, vmBase+"/runCommands?api-version="+azureComputeAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing VM run commands, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	vmssBase := computeBase + "/virtualMachineScaleSets/vmss-a"
	resp = providerContractRequest(t, ts, http.MethodPut, vmssBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{
			"sku":{"name":"Standard_DS1_v2","capacity":2},
			"upgradePolicy":{"mode":"Manual"},
			"virtualMachineProfile":{"storageProfile":{"imageReference":{"id":"`+computeBase+`/images/img-a"}}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VMSS, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, vmssBase+"/virtualMachines/0?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"latestModelApplied":true}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VMSS VM, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, vmssBase+"/virtualMachines/0/extensions/ext-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"publisher":"Contoso","type":"Diagnostics","typeHandlerVersion":"1.0"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating VMSS VM extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, vmssBase+"/virtualMachines?api-version="+azureComputeAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing VMSS VMs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, computeBase+"/cloudServices/cs-a?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"configuration":"<xml />","packageUrl":"https://example.invalid/service.cspkg"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating cloud service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	rpcBase := computeBase + "/restorePointCollections/rpc-a"
	resp = providerContractRequest(t, ts, http.MethodPut, rpcBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"source":{"id":"`+vmBase+`"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating restore point collection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, rpcBase+"/restorePoints/rp-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"consistencyMode":"CrashConsistent"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating restore point, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, vmBase+"?api-version="+azureComputeAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"diagnosticsProfile":{"bootDiagnostics":{"enabled":true}}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching VM, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureComputeValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-compute-validation"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute"

	resp := providerContractRequest(t, ts, http.MethodPut, base+"/virtualMachines/vm-invalid?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"hardwareProfile":{"vmSize":"Standard_DS1_v2"}}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for VM without location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/hostGroups/hg-a/hosts/host-a?api-version="+azureComputeAPIVersion, []byte(`{
		"properties":{"sku":{"name":"DSv3-Type1"}}
	}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for host without host group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/disks/missing?api-version="+azureComputeAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing disk, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
