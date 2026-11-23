package sesv2

import "testing"

func TestTenantLifecycle(t *testing.T) {
	s := NewService()
	tenant, err := s.CreateTenant("acme", map[string]string{"env": "dev"}, nil, "")
	if err != nil || tenant.ID == "" || tenant.ARN == "" {
		t.Fatalf("create: %+v %v", tenant, err)
	}
	if _, err := s.CreateTenant("acme", nil, nil, ""); err != ErrAlreadyExists {
		t.Fatalf("duplicate create: %v", err)
	}
	identity := "arn:aws:ses:us-east-1:123456789012:identity/example.com"
	if err := s.AssociateTenantResource("acme", identity); err != nil {
		t.Fatalf("associate: %v", err)
	}
	if err := s.AssociateTenantResource("acme", "not-an-arn"); err != ErrInvalidParameter {
		t.Fatalf("associate bad arn: %v", err)
	}
	if res, ok := s.ListTenantResources("acme", "EMAIL_IDENTITY"); !ok || len(res) != 1 || res[0].ResourceType != "EMAIL_IDENTITY" {
		t.Fatalf("tenant resources: %+v %v", res, ok)
	}
	if res, _ := s.ListTenantResources("acme", "CONFIGURATION_SET"); len(res) != 0 {
		t.Fatalf("filter not applied: %+v", res)
	}
	if got := s.ListResourceTenants(identity); len(got) != 1 || got[0].TenantName != "acme" {
		t.Fatalf("resource tenants: %+v", got)
	}
	if err := s.DisassociateTenantResource("acme", identity); err != nil {
		t.Fatalf("disassociate: %v", err)
	}
	if got := s.ListResourceTenants(identity); len(got) != 0 {
		t.Fatalf("still associated: %+v", got)
	}
	if !s.DeleteTenant("acme") || s.DeleteTenant("acme") {
		t.Fatal("delete should succeed once")
	}
}

func TestMultiRegionEndpointLifecycle(t *testing.T) {
	s := NewService()
	if _, err := s.CreateMultiRegionEndpoint("mre", nil, nil); err != ErrInvalidParameter {
		t.Fatalf("create without regions: %v", err)
	}
	e, err := s.CreateMultiRegionEndpoint("mre", []string{"us-west-2"}, nil)
	if err != nil || e.ID == "" {
		t.Fatalf("create: %+v %v", e, err)
	}
	if got := s.ListMultiRegionEndpoints(); len(got) != 1 || got[0].Regions[0] != "us-west-2" {
		t.Fatalf("list: %+v", got)
	}
	if !s.DeleteMultiRegionEndpoint("mre") {
		t.Fatal("delete failed")
	}
	if _, ok := s.GetMultiRegionEndpoint("mre"); ok {
		t.Fatal("endpoint still present")
	}
}
