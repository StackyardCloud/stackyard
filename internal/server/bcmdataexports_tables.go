package server

import (
	"regexp"
	"sort"
	"strings"
)

// Data Exports table catalog: the tables AWS offers, their properties and a
// representative subset of their columns.

type bcmdeProperty struct {
	name, def, desc string
	valid           []string // empty: free-form
}

type bcmdeColumn struct {
	name, typ, desc string
	needs           string // a property that must be TRUE for the column to exist
}

type bcmdeTable struct {
	name, desc string
	props      []bcmdeProperty
	columns    []bcmdeColumn
}

var bcmdeBool = []string{"TRUE", "FALSE"}

func bcmdeCols(typ string, names ...string) []bcmdeColumn {
	out := make([]bcmdeColumn, 0, len(names))
	for _, n := range names {
		out = append(out, bcmdeColumn{name: n, typ: typ, desc: strings.ReplaceAll(n, "_", " ")})
	}
	return out
}

func bcmdeJoin(parts ...[]bcmdeColumn) []bcmdeColumn {
	var out []bcmdeColumn
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

var bcmdeTables = []bcmdeTable{
	{
		name: "CARBON_EMISSIONS", desc: "Estimated carbon emissions of your AWS usage.",
		columns: bcmdeJoin(
			bcmdeCols("string", "last_refresh_timestamp", "location", "model_version", "payer_account_id", "product_code",
				"region_code", "total_lbm_emissions_unit", "total_mbm_emissions_unit", "usage_account_id", "usage_period_start", "usage_period_end"),
			bcmdeCols("double", "total_lbm_emissions_value", "total_mbm_emissions_value"),
		),
	},
	{
		name: "COST_AND_USAGE_REPORT", desc: "Cost and Usage Report 2.0: your AWS costs and usage, line by line.",
		props: []bcmdeProperty{
			{name: "BILLING_VIEW_ARN", desc: "The billing view to export; defaults to the primary view."},
			{name: "INCLUDE_MANUAL_DISCOUNT_COMPATIBILITY", def: "FALSE", valid: bcmdeBool, desc: "Include manual discount compatibility columns."},
			{name: "INCLUDE_RESOURCES", def: "FALSE", valid: bcmdeBool, desc: "Include resource IDs as a column."},
			{name: "INCLUDE_SPLIT_COST_ALLOCATION_DATA", def: "FALSE", valid: bcmdeBool, desc: "Include split cost allocation data."},
			{name: "TIME_GRANULARITY", def: "HOURLY", valid: []string{"HOURLY", "DAILY", "MONTHLY"}, desc: "The time granularity of line items."},
		},
		columns: bcmdeJoin(
			bcmdeCols("string", "bill_bill_type", "bill_billing_entity", "bill_invoice_id", "bill_invoicing_entity", "bill_payer_account_id",
				"bill_payer_account_name", "identity_line_item_id", "identity_time_interval", "line_item_currency_code",
				"line_item_legal_entity", "line_item_line_item_description", "line_item_line_item_type", "line_item_operation",
				"line_item_product_code", "line_item_usage_account_id", "line_item_usage_account_name", "line_item_usage_type",
				"pricing_term", "pricing_unit", "product_region_code", "product_servicecode", "product_instance_type"),
			bcmdeCols("timestamp", "bill_billing_period_end_date", "bill_billing_period_start_date", "line_item_usage_end_date", "line_item_usage_start_date"),
			bcmdeCols("double", "line_item_blended_cost", "line_item_blended_rate", "line_item_net_unblended_cost", "line_item_unblended_cost",
				"line_item_unblended_rate", "line_item_usage_amount", "pricing_public_on_demand_cost", "pricing_public_on_demand_rate"),
			bcmdeCols("map<string, string>", "cost_category", "discount", "product", "resource_tags"),
			[]bcmdeColumn{{name: "line_item_resource_id", typ: "string", desc: "line item resource id", needs: "INCLUDE_RESOURCES"}},
		),
	},
	{
		name: "COST_OPTIMIZATION_RECOMMENDATIONS", desc: "Cost Optimization Hub recommendations.",
		props: []bcmdeProperty{
			{name: "FILTER", def: "{}", desc: "A Cost Optimization Hub filter, as JSON."},
			{name: "INCLUDE_ALL_RECOMMENDATIONS", def: "FALSE", valid: bcmdeBool, desc: "Include recommendations that overlap others."},
		},
		columns: bcmdeJoin(
			bcmdeCols("string", "account_id", "action_type", "currency_code", "current_resource_summary", "current_resource_type",
				"implementation_effort", "last_refresh_timestamp", "recommendation_id", "recommendation_source", "recommended_resource_summary",
				"recommended_resource_type", "region", "resource_arn"),
			bcmdeCols("double", "estimated_monthly_cost_after_discount", "estimated_monthly_cost_before_discount",
				"estimated_monthly_savings_after_discount", "estimated_monthly_savings_before_discount", "estimated_savings_percentage_after_discount"),
			bcmdeCols("boolean", "restart_needed", "rollback_possible"),
			bcmdeCols("map<string, string>", "tags"),
		),
	},
	{
		name: "FOCUS_1_0_AWS", desc: "Cost and usage in the FinOps Open Cost and Usage Specification (FOCUS) 1.0.",
		columns: bcmdeJoin(
			bcmdeCols("string", "BillingAccountId", "BillingAccountName", "BillingCurrency", "ChargeCategory", "ChargeDescription",
				"ChargeFrequency", "PricingUnit", "ProviderName", "RegionId", "RegionName", "ResourceId", "ServiceCategory",
				"ServiceName", "SubAccountId", "SubAccountName", "UsageUnit"),
			bcmdeCols("timestamp", "BillingPeriodEnd", "BillingPeriodStart", "ChargePeriodEnd", "ChargePeriodStart"),
			bcmdeCols("decimal", "BilledCost", "ConsumedQuantity", "EffectiveCost", "ListCost", "ListUnitPrice", "PricingQuantity"),
			bcmdeCols("map<string, string>", "Tags"),
		),
	},
}

func bcmdeTableByName(name string) *bcmdeTable {
	for i := range bcmdeTables {
		if bcmdeTables[i].name == name {
			return &bcmdeTables[i]
		}
	}
	return nil
}

// resolveProps checks caller properties against the table and fills defaults.
func (t *bcmdeTable) resolveProps(in map[string]any) (map[string]string, *budErr) {
	out := map[string]string{}
	for _, p := range t.props {
		if p.def != "" {
			out[p.name] = p.def
		}
	}
	for k, v := range in {
		val, _ := v.(string)
		var prop *bcmdeProperty
		for i := range t.props {
			if t.props[i].name == k {
				prop = &t.props[i]
			}
		}
		if prop == nil {
			return nil, bcmdeInvalid("Table %s has no property %s.", t.name, k)
		}
		if len(prop.valid) > 0 && !cohIn(prop.valid, val) {
			return nil, bcmdeInvalid("Property %s must be one of %s.", k, strings.Join(prop.valid, ", "))
		}
		out[k] = val
	}
	return out, nil
}

func (t *bcmdeTable) schema(props map[string]string) []bcmdeColumn {
	out := make([]bcmdeColumn, 0, len(t.columns))
	for _, c := range t.columns {
		if c.needs == "" || props[c.needs] == "TRUE" {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

var bcmdeQueryRE = regexp.MustCompile(`(?is)^\s*SELECT\s+(.+?)\s+FROM\s+([A-Za-z0-9_]+)\s*;?\s*$`)

// bcmdeParseQuery checks a "SELECT col, ... FROM TABLE" statement against the
// catalog and the export's table configuration, returning the table and the
// selected columns.
// ponytail: only plain column lists; WHERE clauses and expressions are refused.
func bcmdeParseQuery(query string, configs map[string]any) (*bcmdeTable, []bcmdeColumn, *budErr) {
	m := bcmdeQueryRE.FindStringSubmatch(query)
	if m == nil {
		return nil, nil, bcmdeInvalid("QueryStatement must be SELECT <columns> FROM <table>.")
	}
	t := bcmdeTableByName(strings.ToUpper(m[2]))
	if t == nil {
		return nil, nil, bcmdeInvalid("Table %s doesn't exist.", m[2])
	}
	for name := range configs {
		if name != t.name {
			return nil, nil, bcmdeInvalid("TableConfigurations names %s, but the query reads %s.", name, t.name)
		}
	}
	props, err := t.resolveProps(budMap(configs, t.name))
	if err != nil {
		return nil, nil, err
	}
	available := t.schema(props)
	if strings.TrimSpace(m[1]) == "*" {
		return t, available, nil
	}
	var cols []bcmdeColumn
	seen := map[string]bool{}
	for _, raw := range strings.Split(m[1], ",") {
		name := strings.TrimSpace(raw)
		var found *bcmdeColumn
		for i := range available {
			if strings.EqualFold(available[i].name, name) {
				found = &available[i]
			}
		}
		if found == nil {
			return nil, nil, bcmdeInvalid("Column %q doesn't exist in %s with this table configuration.", name, t.name)
		}
		if seen[found.name] {
			return nil, nil, bcmdeInvalid("Column %s is selected twice.", found.name)
		}
		seen[found.name] = true
		cols = append(cols, *found)
	}
	return t, cols, nil
}
