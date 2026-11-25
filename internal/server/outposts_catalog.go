package server

import (
	"math"
	"strings"
	"time"
)

// The Outposts hardware catalog and the EC2 instance types it carries.
// ponytail: a fixed five-item catalog; enough to quote, order and size
// capacity tasks against real families.

type opstCatalogItem struct {
	id, class, family string
	powerKva          float64
	weightLbs         int
	uplinks           []int
	storage           []string
	monthly           float64 // list price per unit per month, USD
	vcpus             int     // per unit
}

var opstCatalog = []opstCatalogItem{
	{"OR-RACKM5A", "RACK", "m5", 5, 1400, []int{10, 40, 100}, []string{"EBS"}, 7000, 96},
	{"OR-RACKC5A", "RACK", "c5", 5, 1400, []int{10, 40, 100}, []string{"EBS"}, 6500, 96},
	{"OR-RACKR5S", "RACK", "r5", 10, 1800, []int{10, 40, 100}, []string{"EBS", "S3"}, 9000, 96},
	{"OR-SRVC6GD", "SERVER", "c6gd", 0.6, 30, []int{10}, nil, 400, 64},
	{"OR-SRVC6ID", "SERVER", "c6id", 1, 50, []int{10}, nil, 600, 128},
}

func opstCatalogItemByID(id string) *opstCatalogItem {
	for i := range opstCatalog {
		if opstCatalog[i].id == id {
			return &opstCatalog[i]
		}
	}
	return nil
}

func opstCatalogByFamily(family string) *opstCatalogItem {
	for i := range opstCatalog {
		if opstCatalog[i].family == family {
			return &opstCatalog[i]
		}
	}
	return nil
}

func (c *opstCatalogItem) ec2Capacities() []any {
	return []any{map[string]any{"Family": c.family, "MaxSize": "4xlarge", "Quantity": "1"}}
}

func (c *opstCatalogItem) out() map[string]any {
	uplinks := make([]any, len(c.uplinks))
	for i, u := range c.uplinks {
		uplinks[i] = u
	}
	storage := make([]any, len(c.storage))
	for i, s := range c.storage {
		storage[i] = s
	}
	return map[string]any{
		"CatalogItemId": c.id, "ItemStatus": "AVAILABLE", "EC2Capacities": c.ec2Capacities(), "PowerKva": c.powerKva,
		"WeightLbs": c.weightLbs, "SupportedUplinkGbps": uplinks, "SupportedStorage": storage, "RackScalingType": "SINGLE_RACK",
	}
}

func opstGetCatalogItem(_ *opstStore, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	c := opstCatalogItemByID(budStr(in, "CatalogItemId"))
	if c == nil {
		return nil, opstNotFound("Catalog item", budStr(in, "CatalogItemId"))
	}
	return map[string]any{"CatalogItem": c.out()}, nil
}

func opstListCatalogItems(_ *opstStore, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	classes, storage, families := opstStrs(in, "ItemClassFilter"), opstStrs(in, "SupportedStorageFilter"), opstStrs(in, "EC2FamilyFilter")
	var match []map[string]any
	for i := range opstCatalog {
		c := &opstCatalog[i]
		if !opstMatches(classes, c.class) || !opstMatches(families, c.family) {
			continue
		}
		if len(storage) > 0 {
			ok := false
			for _, s := range c.storage {
				ok = ok || opstMatches(storage, s)
			}
			if !ok {
				continue
			}
		}
		match = append(match, c.out())
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, m := range match[start:end] {
		list = append(list, m)
	}
	return map[string]any{"CatalogItems": list, "NextToken": next}, nil
}

type opstInstanceType struct {
	name, family  string
	vcpus, memMiB int
}

var opstSizes = []struct {
	size  string
	vcpus int
}{{"large", 2}, {"xlarge", 4}, {"2xlarge", 8}, {"4xlarge", 16}}

// opstInstanceTypes lists every instance type the catalog's families offer.
var opstInstanceTypes = func() []opstInstanceType {
	memPerVCPU := map[string]int{"m5": 4096, "c5": 2048, "r5": 8192, "c6gd": 2048, "c6id": 2048}
	var out []opstInstanceType
	for _, c := range opstCatalog {
		for _, s := range opstSizes {
			out = append(out, opstInstanceType{c.family + "." + s.size, c.family, s.vcpus, s.vcpus * memPerVCPU[c.family]})
		}
	}
	return out
}()

func opstInstanceTypeByName(name string) *opstInstanceType {
	for i := range opstInstanceTypes {
		if opstInstanceTypes[i].name == name {
			return &opstInstanceTypes[i]
		}
	}
	return nil
}

// opstTypesOf returns {InstanceType, VCPUs} for every type of the families.
func opstTypesOf(families map[string]bool) []any {
	out := []any{}
	for _, t := range opstInstanceTypes {
		if families[t.family] {
			out = append(out, map[string]any{"InstanceType": t.name, "VCPUs": t.vcpus})
		}
	}
	return out
}

func opstListOrderableInstanceTypes(_ *opstStore, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	gen := budStr(in, "OutpostGenerationFilter")
	var match []any
	for _, t := range opstInstanceTypes {
		c := opstCatalogByFamily(t.family)
		var forms []any
		for _, g := range []string{"GENERATION_1", "GENERATION_2"} {
			if (gen != "" && g != gen) || (c.class == "SERVER" && g == "GENERATION_2") {
				continue
			}
			forms = append(forms, map[string]any{"FormFactor": c.class, "OutpostGeneration": g})
		}
		if len(forms) == 0 {
			continue
		}
		match = append(match, map[string]any{
			"InstanceType": t.name, "VCPUs": t.vcpus, "MemoryInMib": t.memMiB,
			"NetworkPerformance": "Up to 10 Gigabit", "FormFactorConfigs": forms,
		})
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	return map[string]any{"InstanceTypes": append([]any{}, match[start:end]...), "NextToken": next}, nil
}

// ---- pricing ----

var opstTermMonths = map[string]int{"ONE_YEAR": 12, "THREE_YEARS": 36, "FIVE_YEARS": 60}

var opstTermDiscount = map[string]float64{"ONE_YEAR": 1, "THREE_YEARS": 0.85, "FIVE_YEARS": 0.75}

func opstCents(v float64) float64 { return math.Round(v*100) / 100 }

// opstPrice splits a subscription over a monthly list price into upfront
// and monthly parts for a payment option and term.
func opstPrice(monthlyList float64, option, term string) (upfront, monthly float64) {
	months := float64(opstTermMonths[term])
	total := monthlyList * months * opstTermDiscount[term]
	switch option {
	case "ALL_UPFRONT":
		return opstCents(total), 0
	case "PARTIAL_UPFRONT":
		return opstCents(total / 2), opstCents(total / 2 / months)
	}
	return 0, opstCents(total / months)
}

func opstPricing(monthlyList float64, option, term string) map[string]any {
	up, mo := opstPrice(monthlyList, option, term)
	return map[string]any{"PaymentOption": option, "PaymentTerm": term, "UpfrontPrice": up, "MonthlyRecurringPrice": mo, "Currency": "USD"}
}

// opstFamilyOf maps a capacity unit ("m5", "m5.xlarge") to a catalog family.
func opstFamilyOf(unit string) string {
	f, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(unit)), ".")
	return f
}
