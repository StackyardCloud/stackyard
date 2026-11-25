package server

import (
	"fmt"
	"math"
	"time"
)

// cohSeed is one entry of the fixed recommendation catalog.
// ponytail: Stackyard has no usage data to analyse, so Cost Optimization Hub
// serves this fixed, deterministic catalog for the enrolled account instead of
// deriving recommendations from the EC2, Lambda or RDS stores.
type cohSeed struct {
	id, resourceID, arn, region, currentType, recommendedType, action, effort, source string
	current, recommended, currentSummary, recommendedSummary                          string
	cost, savings                                                                     float64
	restart, rollback, superseded                                                     bool
	tags                                                                              map[string]string
}

var cohSeeds = []cohSeed{
	{id: "coh-ec2-rightsize-0001", resourceID: "i-0a1b2c3d4e5f60001", arn: "arn:aws:ec2:us-east-1:%s:instance/i-0a1b2c3d4e5f60001",
		region: "us-east-1", currentType: "Ec2Instance", recommendedType: "Ec2Instance", action: "Rightsize", effort: "Medium",
		source: "ComputeOptimizer", current: "m5.2xlarge", recommended: "m5.xlarge", currentSummary: "m5.2xlarge", recommendedSummary: "m5.xlarge",
		cost: 280.32, savings: 140.16, restart: true, rollback: true, tags: map[string]string{"env": "prod", "team": "orders"}},
	{id: "coh-ec2-graviton-0001", resourceID: "i-0a1b2c3d4e5f60001", arn: "arn:aws:ec2:us-east-1:%s:instance/i-0a1b2c3d4e5f60001",
		region: "us-east-1", currentType: "Ec2Instance", recommendedType: "Ec2Instance", action: "MigrateToGraviton", effort: "High",
		source: "ComputeOptimizer", current: "m5.2xlarge", recommended: "m7g.xlarge", currentSummary: "m5.2xlarge", recommendedSummary: "m7g.xlarge",
		cost: 280.32, savings: 168.19, restart: true, rollback: true, superseded: true, tags: map[string]string{"env": "prod", "team": "orders"}},
	{id: "coh-ec2-stop-0003", resourceID: "i-0a1b2c3d4e5f60003", arn: "arn:aws:ec2:eu-west-1:%s:instance/i-0a1b2c3d4e5f60003",
		region: "eu-west-1", currentType: "Ec2Instance", recommendedType: "Ec2Instance", action: "Stop", effort: "Low",
		source: "ComputeOptimizer", current: "t3.medium", recommended: "t3.medium", currentSummary: "t3.medium (idle)", recommendedSummary: "Stopped",
		cost: 30.37, savings: 30.37, restart: false, rollback: true, tags: map[string]string{"env": "dev"}},
	{id: "coh-ebs-upgrade-0002", resourceID: "vol-0a1b2c3d4e5f60002", arn: "arn:aws:ec2:us-east-1:%s:volume/vol-0a1b2c3d4e5f60002",
		region: "us-east-1", currentType: "EbsVolume", recommendedType: "EbsVolume", action: "Upgrade", effort: "VeryLow",
		source: "ComputeOptimizer", current: "gp2", recommended: "gp3", currentSummary: "500 GiB gp2", recommendedSummary: "500 GiB gp3",
		cost: 50, savings: 10, restart: false, rollback: true, tags: map[string]string{"env": "prod"}},
	{id: "coh-lambda-rightsize-0004", resourceID: "orders-api", arn: "arn:aws:lambda:us-east-1:%s:function:orders-api",
		region: "us-east-1", currentType: "LambdaFunction", recommendedType: "LambdaFunction", action: "Rightsize", effort: "Low",
		source: "ComputeOptimizer", current: "3008", recommended: "1024", currentSummary: "3008 MB", recommendedSummary: "1024 MB",
		cost: 42.5, savings: 27.6, restart: false, rollback: true, tags: map[string]string{"team": "orders"}},
	{id: "coh-csp-purchase-0005", resourceID: "%s", arn: "arn:aws:savingsplans::%s:savingsplan/recommendation",
		region: "Global", currentType: "ComputeSavingsPlans", recommendedType: "ComputeSavingsPlans", action: "PurchaseSavingsPlans", effort: "VeryLow",
		source: "CostExplorer", current: "", recommended: "1.20", currentSummary: "On-Demand", recommendedSummary: "$1.20/hour, 1 year, No Upfront",
		cost: 1200, savings: 264, restart: false, rollback: false},
	{id: "coh-rds-ri-purchase-0006", resourceID: "%s", arn: "arn:aws:rds:us-west-2:%s:ri/recommendation",
		region: "us-west-2", currentType: "RdsReservedInstances", recommendedType: "RdsReservedInstances", action: "PurchaseReservedInstances", effort: "VeryLow",
		source: "CostExplorer", current: "", recommended: "db.r6g.large", currentSummary: "On-Demand", recommendedSummary: "2 x db.r6g.large postgres, 1 year",
		cost: 350.4, savings: 113.88, restart: false, rollback: false},
}

func cohRound(v float64) float64 { return math.Round(v*100) / 100 }

func cohPricing(cost float64) map[string]any {
	return map[string]any{
		"estimatedCostBeforeDiscounts": cost, "estimatedNetUnusedAmortizedCommitments": 0.0,
		"estimatedDiscounts":          map[string]any{"savingsPlansDiscount": 0.0, "reservedInstancesDiscount": 0.0, "otherDiscount": 0.0},
		"estimatedCostAfterDiscounts": cost,
	}
}

func cohUsage(usageType, unit string, amount float64) []any {
	return []any{map[string]any{"usageType": usageType, "usageAmount": amount, "unit": unit}}
}

// cohDetails builds the ResourceDetails union for one side of a seed.
func cohDetails(sd cohSeed, config string, cost float64) map[string]any {
	switch sd.currentType {
	case "Ec2Instance":
		return map[string]any{"ec2Instance": map[string]any{
			"configuration":   map[string]any{"instance": map[string]any{"type": config}},
			"costCalculation": map[string]any{"usages": cohUsage("BoxUsage:"+config, "Hrs", 730), "pricing": cohPricing(cost)},
		}}
	case "EbsVolume":
		return map[string]any{"ebsVolume": map[string]any{
			"configuration":   map[string]any{"storage": map[string]any{"type": config, "sizeInGb": 500.0}, "performance": map[string]any{"iops": 3000.0, "throughput": 125.0}, "attachmentState": "attached"},
			"costCalculation": map[string]any{"usages": cohUsage("EBS:VolumeUsage."+config, "GB-Mo", 500), "pricing": cohPricing(cost)},
		}}
	case "LambdaFunction":
		mem := 1024
		if config == "3008" {
			mem = 3008
		}
		return map[string]any{"lambdaFunction": map[string]any{
			"configuration":   map[string]any{"compute": map[string]any{"memorySizeInMB": float64(mem), "architecture": "x86_64", "platform": "Linux"}},
			"costCalculation": map[string]any{"usages": cohUsage("Lambda-GB-Second", "Lambda-GB-Second", float64(mem)*2000), "pricing": cohPricing(cost)},
		}}
	case "ComputeSavingsPlans":
		return map[string]any{"computeSavingsPlans": map[string]any{
			"configuration": map[string]any{"accountScope": "Payer", "term": "1 Year", "paymentOption": "No Upfront", "hourlyCommitment": config},
			"costCalculation": map[string]any{"pricing": map[string]any{
				"monthlySavingsPlansEligibleCost": sd.cost, "estimatedMonthlyCommitment": cohRound(sd.cost - sd.savings),
				"savingsPercentage": cohRound(sd.savings / sd.cost * 100), "estimatedOnDemandCost": sd.cost,
			}},
		}}
	case "RdsReservedInstances":
		return map[string]any{"rdsReservedInstances": map[string]any{
			"configuration": map[string]any{"accountScope": "Payer", "service": "RDS", "term": "1 Year", "paymentOption": "No Upfront",
				"reservedInstancesRegion": sd.region, "instanceType": "db.r6g.large", "instanceFamily": "r6g", "numberOfInstancesToPurchase": "2",
				"databaseEngine": "PostgreSQL", "deploymentOption": "Single-AZ", "licenseModel": "No license required", "sizeFlexEligible": true},
			"costCalculation": map[string]any{"pricing": map[string]any{
				"estimatedOnDemandCost": sd.cost, "monthlyReservationEligibleCost": sd.cost,
				"savingsPercentage": cohRound(sd.savings / sd.cost * 100), "estimatedMonthlyAmortizedReservationCost": cohRound(sd.cost - sd.savings),
			}},
		}}
	}
	return map[string]any{}
}

// cohRecommendation renders a seed in the GetRecommendation shape plus the
// list-only summary fields.
func cohRecommendation(sd cohSeed, account string, refreshed time.Time) map[string]any {
	tags := make([]any, 0, len(sd.tags))
	for _, t := range budTagsOut(sd.tags) {
		m := t.(map[string]any)
		tags = append(tags, map[string]any{"key": m["Key"], "value": m["Value"]})
	}
	resourceID, arn := sd.resourceID, fmt.Sprintf(sd.arn, account)
	if resourceID == "%s" { // commitment purchases are scoped to the account
		resourceID = account
	}
	return map[string]any{
		"recommendationId": sd.id, "resourceId": resourceID, "resourceArn": arn, "accountId": account,
		"currencyCode": "USD", "recommendationLookbackPeriodInDays": 14.0, "costCalculationLookbackPeriodInDays": 14.0,
		"estimatedSavingsPercentage": cohRound(sd.savings / sd.cost * 100), "estimatedSavingsOverCostCalculationLookbackPeriod": cohRound(sd.savings * 14 / 30),
		"currentResourceType": sd.currentType, "recommendedResourceType": sd.recommendedType, "region": sd.region, "source": sd.source,
		"lastRefreshTimestamp": budEpoch(refreshed), "estimatedMonthlySavings": sd.savings, "estimatedMonthlyCost": sd.cost,
		"implementationEffort": sd.effort, "restartNeeded": sd.restart, "actionType": sd.action, "rollbackPossible": sd.rollback,
		"currentResourceDetails":     cohDetails(sd, sd.current, sd.cost),
		"recommendedResourceDetails": cohDetails(sd, sd.recommended, cohRound(sd.cost-sd.savings)),
		"currentResourceSummary":     sd.currentSummary, "recommendedResourceSummary": sd.recommendedSummary,
		"tags": tags,
	}
}
