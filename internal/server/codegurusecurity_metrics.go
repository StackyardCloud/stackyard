package server

import (
	"sort"
	"time"
)

// Finding metrics. Each finished run adds to the day's bucket the findings it
// opened and closed (with their time to close) and records the account's
// open findings afterwards; ListFindingsMetrics reads the buckets and
// GetMetricsSummary ranks categories and scans by the findings open on a day.

const cgsMaxMetricDays = 366 // days one ListFindingsMetrics range may span

func cgsSeverityValues(v [5]float64) map[string]any {
	return map[string]any{"critical": v[0], "high": v[1], "medium": v[2], "low": v[3], "info": v[4]}
}

func (s *Server) cgsListFindingsMetrics(req cgsRequest) (map[string]any, *cgsErr) {
	from, to := cgsTime(req.in, "startDate"), cgsTime(req.in, "endDate")
	if to.Before(from) || to.Sub(from) > cgsMaxMetricDays*24*time.Hour {
		return nil, cgsValidation("endDate must be on or after startDate and at most %d days later", cgsMaxMetricDays)
	}
	first, last := from.Format("2006-01-02"), to.Format("2006-01-02")
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.advance(time.Now().UTC())
	var keys []string
	for k := range st.days {
		if k >= first && k <= last {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	start, end, next, err := cgsPage(req.in, len(keys), 20)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, k := range keys[start:end] {
		d := st.days[k]
		var mttc [5]float64
		for i := range mttc {
			if d.closedN[i] > 0 {
				mttc[i] = d.closeDays[i] / float64(d.closedN[i])
			}
		}
		out = append(out, map[string]any{
			"date": c9Epoch(cgsDayStart(k)), "newFindings": cgsSeverityValues(d.opened), "closedFindings": cgsSeverityValues(d.closed),
			"openFindings": cgsSeverityValues(d.open), "meanTimeToClose": cgsSeverityValues(mttc),
		})
	}
	return map[string]any{"findingsMetrics": out, "nextToken": next}, nil
}

type cgsRank struct {
	name  string
	count int
}

func cgsTop(counts map[string]int, n int, nameKey, countKey string) []any {
	ranks := make([]cgsRank, 0, len(counts))
	for k, v := range counts {
		ranks = append(ranks, cgsRank{k, v})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].count > ranks[j].count || ranks[i].count == ranks[j].count && ranks[i].name < ranks[j].name
	})
	out := make([]any, 0, n)
	for _, r := range ranks[:min(n, len(ranks))] {
		out = append(out, map[string]any{nameKey: r.name, countKey: float64(r.count)})
	}
	return out
}

func (s *Server) cgsGetMetricsSummary(req cgsRequest) (map[string]any, *cgsErr) {
	date := cgsTime(req.in, "date")
	key := date.Format("2006-01-02")
	endOfDay := cgsDayStart(key).Add(24 * time.Hour)
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.advance(time.Now().UTC())
	// Open findings: the last bucket on or before the day.
	var open [5]float64
	best := ""
	for k := range st.days {
		if k <= key && k > best {
			best = k
		}
	}
	if best != "" {
		open = st.days[best].open
	}
	categories, scans, critical := map[string]int{}, map[string]int{}, map[string]int{}
	for name, sc := range st.scans {
		run := sc.lastDone()
		if run == nil {
			continue
		}
		for _, f := range run.findings {
			openThen := !f.created.After(endOfDay) && (f.status == "Open" || f.updated.After(endOfDay))
			if !openThen {
				continue
			}
			rule := cgsRules[f.rule]
			categories[rule.category]++
			scans[name]++
			if rule.severity == "Critical" {
				critical[name]++
			}
		}
	}
	return map[string]any{"metricsSummary": map[string]any{
		"date": c9Epoch(cgsDayStart(key)), "openFindings": cgsSeverityValues(open),
		"categoriesWithMostFindings":        cgsTop(categories, 5, "categoryName", "findingNumber"),
		"scansWithMostOpenFindings":         cgsTop(scans, 3, "scanName", "findingNumber"),
		"scansWithMostOpenCriticalFindings": cgsTop(critical, 3, "scanName", "findingNumber"),
	}}, nil
}
