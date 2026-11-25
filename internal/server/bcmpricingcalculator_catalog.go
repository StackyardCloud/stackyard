package server

import (
	"math"
	"sort"
	"time"
)

// The Pricing Calculator's built-in price list: a small, fixed set of
// us-east-1 on-demand rates (USD) plus Reserved Instance offerings. Every
// estimate is computed from it, so results are deterministic.

type bcmpcPrice struct {
	unit, location string
	rate           float64
	// hourly units (Hrs, Requests, GB-seconds) accrue amount x hours;
	// monthly units (GB-Mo, metrics) accrue amount x the share of the month.
	hourly bool
	// spEligible usage is covered by Compute Savings Plans.
	spEligible bool
}

const bcmpcUSEast = "US East (N. Virginia)"

var bcmpcPrices = map[string]bcmpcPrice{
	"AmazonEC2|BoxUsage:t3.micro|RunInstances":                  {unit: "Hrs", location: bcmpcUSEast, rate: 0.0104, hourly: true, spEligible: true},
	"AmazonEC2|BoxUsage:t3.small|RunInstances":                  {unit: "Hrs", location: bcmpcUSEast, rate: 0.0208, hourly: true, spEligible: true},
	"AmazonEC2|BoxUsage:m5.large|RunInstances":                  {unit: "Hrs", location: bcmpcUSEast, rate: 0.096, hourly: true, spEligible: true},
	"AmazonEC2|BoxUsage:c5.xlarge|RunInstances":                 {unit: "Hrs", location: bcmpcUSEast, rate: 0.17, hourly: true, spEligible: true},
	"AmazonEC2|EBS:VolumeUsage.gp3|CreateVolume-Gp3":            {unit: "GB-Mo", location: bcmpcUSEast, rate: 0.08},
	"AmazonS3|TimedStorage-ByteHrs|StandardStorage":             {unit: "GB-Mo", location: bcmpcUSEast, rate: 0.023},
	"AmazonS3|Requests-Tier1|PutObject":                         {unit: "Requests", location: bcmpcUSEast, rate: 0.000005, hourly: true},
	"AWSLambda|Request|Invoke":                                  {unit: "Requests", location: bcmpcUSEast, rate: 0.0000002, hourly: true},
	"AWSLambda|Lambda-GB-Second|Invoke":                         {unit: "Lambda-GB-Second", location: bcmpcUSEast, rate: 0.0000166667, hourly: true, spEligible: true},
	"AmazonRDS|InstanceUsage:db.t3.micro|CreateDBInstance:0002": {unit: "Hrs", location: bcmpcUSEast, rate: 0.017, hourly: true},
	"AmazonDynamoDB|TimedStorage-ByteHrs|StandardStorage":       {unit: "GB-Mo", location: bcmpcUSEast, rate: 0.25},
	"AmazonCloudWatch|CW:MetricMonitorUsage|MetricStorage":      {unit: "Metrics", location: bcmpcUSEast, rate: 0.30},
}

func bcmpcLookup(serviceCode, usageType, operation string) (bcmpcPrice, bool) {
	p, ok := bcmpcPrices[serviceCode+"|"+usageType+"|"+operation]
	return p, ok
}

// bcmpcRIOffering is a Reserved Instance offering: it covers instanceCount
// hours of usageType each hour.
type bcmpcRIOffering struct {
	usageType, term, payment string
	upfront, hourly          float64
}

var bcmpcRIOfferings = map[string]bcmpcRIOffering{
	"4e2b8c9f-1a3d-4f6e-9b7a-0c1d2e3f4a5b": {usageType: "BoxUsage:t3.micro", term: "1yr", payment: "No Upfront", hourly: 0.0065},
	"7d1e2f3a-4b5c-4d6e-8f9a-0b1c2d3e4f5a": {usageType: "BoxUsage:m5.large", term: "1yr", payment: "No Upfront", hourly: 0.060},
	"9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d": {usageType: "BoxUsage:m5.large", term: "3yr", payment: "All Upfront", upfront: 1003},
}

// bcmpcSPDiscount is how much less Savings Plans rates are than on-demand.
const bcmpcSPDiscount = 0.28

// bcmpcQuantity turns a usage modification's amounts into a quantity over
// [start, end): each amount holds from its startHour until the next one.
func bcmpcQuantity(amounts []any, p bcmpcPrice, start, end time.Time) float64 {
	type step struct {
		at     time.Time
		amount float64
	}
	steps := make([]step, 0, len(amounts))
	for _, a := range amounts {
		m, _ := a.(map[string]any)
		at, _ := budTime(m["startHour"])
		amount, _ := m["amount"].(float64)
		steps = append(steps, step{at, amount})
	}
	sort.SliceStable(steps, func(i, j int) bool { return steps[i].at.Before(steps[j].at) })
	total := end.Sub(start).Hours()
	q := 0.0
	for i, s := range steps {
		until := end
		if i+1 < len(steps) {
			until = steps[i+1].at
		}
		hours := until.Sub(s.at).Hours()
		if p.hourly {
			q += s.amount * hours
		} else {
			q += s.amount * hours / total
		}
	}
	return bcmpcRound(q)
}

func bcmpcRound(v float64) float64 { return math.Round(v*1e6) / 1e6 }

func bcmpcCost(v float64) map[string]any {
	return map[string]any{"amount": bcmpcRound(v), "currency": "USD"}
}
