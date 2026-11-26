package server

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Vector enrichment input and output in Stackyard's S3. Input is checked
// when a job starts (a job over unusable input settles FAILED) and read
// again, then enriched, when it is exported. Both happen outside the lock.

const sgeoMaxVectorRows = 100000

// sgeoVectorColumns names the CSV columns a vector enrichment job reads.
func sgeoVectorColumns(cfg map[string]any) []string {
	if mm := fcObj(cfg, "MapMatchingConfig"); mm != nil {
		return []string{fcStr(mm, "IdAttributeName"), fcStr(mm, "TimestampAttributeName"), fcStr(mm, "XAttributeName"), fcStr(mm, "YAttributeName")}
	}
	rg := fcObj(cfg, "ReverseGeocodingConfig")
	return []string{fcStr(rg, "XAttributeName"), fcStr(rg, "YAttributeName")}
}

// sgeoReadVector reads and parses the CSV at uri, checking that cols exist
// and that the last two (x and y) hold numbers. It returns the rows and the
// column indexes of cols.
const sgeoServicePrincipal = "sagemaker-geospatial.amazonaws.com"

func (s *Server) sgeoReadVector(c s3Caller, uri string, cols []string) ([][]string, []int, string) {
	files, err := s.pszReadS3As(c, sgeoServicePrincipal, uri)
	if err != nil {
		return nil, nil, err.Error()
	}
	var rows [][]string
	for _, f := range files {
		recs, err := csv.NewReader(bytes.NewReader(f.body)).ReadAll()
		if err != nil {
			return nil, nil, fmt.Sprintf("s3://%s isn't valid CSV: %v", f.key, err)
		}
		if len(rows) > 0 && len(recs) > 0 {
			recs = recs[1:] // later files repeat the header
		}
		rows = append(rows, recs...)
		if len(rows) > sgeoMaxVectorRows+1 {
			return nil, nil, fmt.Sprintf("input has more than %d rows", sgeoMaxVectorRows)
		}
	}
	if len(rows) == 0 {
		return nil, nil, "input has no header row"
	}
	idx := make([]int, len(cols))
	for i, c := range cols {
		idx[i] = -1
		for j, h := range rows[0] {
			if strings.TrimSpace(h) == c {
				idx[i] = j
			}
		}
		if idx[i] < 0 {
			return nil, nil, fmt.Sprintf("column %s is not in the input header", c)
		}
	}
	for n, row := range rows[1:] {
		for _, i := range idx[len(idx)-2:] {
			if i >= len(row) {
				return nil, nil, fmt.Sprintf("row %d is short", n+2)
			}
			if _, err := strconv.ParseFloat(strings.TrimSpace(row[i]), 64); err != nil {
				return nil, nil, fmt.Sprintf("row %d: %q is not a coordinate", n+2, row[i])
			}
		}
	}
	return rows, idx, ""
}

func (s *Server) sgeoCheckVectorInput(in map[string]any) string {
	uri := fcStr(fcObj(fcObj(fcObj(in, "InputConfig"), "DataSourceConfig"), "S3Data"), "S3Uri")
	if uri == "" {
		return ""
	}
	_, _, failure := s.sgeoReadVector(s3CallerIn(in), uri, sgeoVectorColumns(fcObj(in, "JobConfig")))
	return failure
}

// vectorSnapshot reads a completed vector job's input settings.
func (st *sgeoStore) vectorSnapshot(arn string) (map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	j, err := st.job("vector-enrichment-job", arn)
	if err != nil || j.doc["Status"] != "COMPLETED" {
		return nil, false
	}
	return map[string]any{"uri": fcStr(fcObj(fcObj(fcObj(j.doc, "InputConfig"), "DataSourceConfig"), "S3Data"), "S3Uri"),
		"config": fcObj(j.doc, "JobConfig")}, true
}

// sgeoPrepareVectorExport enriches the job's input into the output CSV.
// ponytail: placeholder enrichment — reverse geocoding labels each point
// with a stable synthetic place, map matching snaps to a 1e-4 degree grid.
func (s *Server) sgeoPrepareVectorExport(in map[string]any, pre *sgeoPre) {
	snap, ok := s.sageMakerGeospatial.vectorSnapshot(fcStr(in, "Arn"))
	if !ok {
		return // the export reports the job's state
	}
	cfg := fcObj(snap, "config")
	rows, idx, failure := s.sgeoReadVector(s3CallerIn(in), fcStr(snap, "uri"), sgeoVectorColumns(cfg))
	if failure != "" {
		pre.failure = failure
		return
	}
	xi, yi := idx[len(idx)-2], idx[len(idx)-1]
	matching := fcObj(cfg, "MapMatchingConfig") != nil
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := append([]string{}, rows[0]...)
	if matching {
		header = append(header, "matched_x", "matched_y")
	} else {
		header = append(header, "reverse_geo.label", "reverse_geo.municipality", "reverse_geo.region", "reverse_geo.country", "reverse_geo.postal_code")
	}
	_ = w.Write(header)
	for _, row := range rows[1:] {
		x, _ := strconv.ParseFloat(strings.TrimSpace(row[xi]), 64)
		y, _ := strconv.ParseFloat(strings.TrimSpace(row[yi]), 64)
		out := append([]string{}, row...)
		if matching {
			out = append(out, strconv.FormatFloat(math.Round(x*1e4)/1e4, 'f', 4, 64), strconv.FormatFloat(math.Round(y*1e4)/1e4, 'f', 4, 64))
		} else {
			key := fmt.Sprintf("%.3f,%.3f", x, y)
			n := int(amlScore(key) * 9000)
			out = append(out, fmt.Sprintf("%d Main St, Town %d", 100+n%900, n%97), fmt.Sprintf("Town %d", n%97),
				fmt.Sprintf("Region %d", n%13), "USA", fmt.Sprintf("%05d", 10000+n))
		}
		_ = w.Write(out)
	}
	w.Flush()
	bucket, prefix, _ := pszS3Path(fcStr(fcObj(fcObj(in, "OutputConfig"), "S3Data"), "S3Uri"))
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	arn := fcStr(in, "Arn")
	prefix += arn[strings.LastIndex(arn, "/")+1:] + "/"
	pre.writes = append(pre.writes, sgeoWrite{bucket, prefix + "output.csv", "text/csv", buf.Bytes()})
}
