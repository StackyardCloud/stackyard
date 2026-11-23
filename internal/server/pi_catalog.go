package server

import "strings"

// Performance Insights catalogs: the counter metrics ListAvailableResourceMetrics
// returns, the dimension groups db.load can be sliced by, and the fixed sample
// dimension keys Stackyard reports for each group.

type piMetricDef struct {
	name, unit, desc string
	scale            float64 // typical magnitude of the synthetic series
}

// ponytail: one counter catalog for every engine; AWS varies db.* counters by
// engine. Split per engine if a caller needs engine-specific names.
var piCounterMetrics = map[string][]piMetricDef{
	"os": {
		{"os.cpuUtilization.user", "Percent", "The percentage of CPU in use by user programs.", 30},
		{"os.cpuUtilization.system", "Percent", "The percentage of CPU in use by the kernel.", 8},
		{"os.cpuUtilization.wait", "Percent", "The percentage of CPU unused while waiting for I/O access.", 4},
		{"os.cpuUtilization.idle", "Percent", "The percentage of CPU that is idle.", 55},
		{"os.cpuUtilization.total", "Percent", "The total percentage of the CPU in use.", 45},
		{"os.loadAverageMinute.one", "Processes", "The number of processes requesting CPU time over the last minute.", 1.5},
		{"os.memory.total", "Kilobytes", "The total amount of memory.", 8_000_000},
		{"os.memory.free", "Kilobytes", "The amount of unassigned memory.", 2_000_000},
		{"os.memory.active", "Kilobytes", "The amount of assigned memory.", 4_500_000},
		{"os.diskIO.rdsdev.readIOsPS", "Requests per second", "The number of read operations per second.", 120},
		{"os.diskIO.rdsdev.writeIOsPS", "Requests per second", "The number of write operations per second.", 90},
		{"os.network.rx", "Bytes per second", "The number of bytes received per second.", 250_000},
		{"os.network.tx", "Bytes per second", "The number of bytes uploaded per second.", 180_000},
	},
	"db": {
		{"db.SQL.Queries", "Queries per second", "The number of statements executed per second.", 400},
		{"db.Transactions.xact_commit", "Commits per second", "The number of transactions committed per second.", 150},
		{"db.Transactions.xact_rollback", "Rollbacks per second", "The number of transactions rolled back per second.", 2},
		{"db.Users.numbackends", "Connections", "The number of backends currently connected.", 40},
		{"db.IO.blks_read", "Blocks per second", "The number of disk blocks read per second.", 300},
		{"db.Cache.blks_hit", "Hits per second", "The number of buffer cache hits per second.", 9000},
		{"db.Concurrency.deadlocks", "Deadlocks per minute", "The number of deadlocks per minute.", 0.2},
		{"db.Checkpoint.checkpoint_sync_latency", "Milliseconds", "The time spent syncing checkpoint files.", 12},
	},
}

// The two load metrics are always available and are the only ones that can be
// grouped by dimension.
var piLoadMetrics = map[string]float64{"db.load": 2.0, "db.sampledload": 2.2}

var piStatistics = map[string]bool{"avg": true, "min": true, "max": true, "sum": true}

// piMetric resolves a query metric ("db.load.avg", "os.memory.free.max") to its
// definition. Load metrics are reported as loadable.
func piMetric(name string) (def piMetricDef, load bool, ok bool) {
	i := strings.LastIndex(name, ".")
	if i < 0 || !piStatistics[name[i+1:]] {
		return piMetricDef{}, false, false
	}
	root := name[:i]
	if scale, isLoad := piLoadMetrics[root]; isLoad {
		return piMetricDef{name: root, unit: "Average active sessions", scale: scale}, true, true
	}
	for _, defs := range piCounterMetrics {
		for _, d := range defs {
			if d.name == root {
				return d, false, true
			}
		}
	}
	return piMetricDef{}, false, false
}

// piGroup is a dimension group with its dimensions and sample keys. Each key
// lists the values of the group's dimensions, in order.
type piGroup struct {
	dims []string
	keys [][]string
}

var piRDSGroups = map[string]piGroup{
	"db.sql": {dims: []string{"db.sql.id", "db.sql.db_id", "db.sql.statement", "db.sql.tokenized_id"}, keys: [][]string{
		{"A1B2C3D4E5F60718", "pi-3b1f8a20c9d71e5a", "SELECT * FROM orders WHERE customer_id = 4711", "9F2E1D0C8B7A6954"},
		{"B2C3D4E5F6071829", "pi-4c2a9b31dae82f6b", "UPDATE inventory SET quantity = quantity - 1 WHERE sku = 'SKU-1042'", "8E1D0C9B7A695843"},
		{"C3D4E5F60718293A", "pi-5d3bac42ebf93a7c", "INSERT INTO events (kind, payload) VALUES ('login', '{}')", "7D0C9B8A69584732"},
		{"D4E5F60718293A4B", "pi-6e4cbd53fca04b8d", "SELECT count(*) FROM sessions WHERE expires_at > now()", "6C9B8A7958473621"},
		{"E5F60718293A4B5C", "pi-7f5dce64adb15c9e", "DELETE FROM carts WHERE updated_at < now() - interval '7 days'", "5B8A796847362510"},
	}},
	"db.sql_tokenized": {dims: []string{"db.sql_tokenized.id", "db.sql_tokenized.db_id", "db.sql_tokenized.statement"}, keys: [][]string{
		{"9F2E1D0C8B7A6954", "pi-2a0e7910b8c60d49", "SELECT * FROM orders WHERE customer_id = ?"},
		{"8E1D0C9B7A695843", "pi-1b9f6820a7b51c38", "UPDATE inventory SET quantity = quantity - ? WHERE sku = ?"},
		{"7D0C9B8A69584732", "pi-0c8e5731968a0b27", "INSERT INTO events (kind, payload) VALUES (?, ?)"},
	}},
	"db.wait_event": {dims: []string{"db.wait_event.name", "db.wait_event.type"}, keys: [][]string{
		{"CPU", "CPU"}, {"IO:DataFileRead", "IO"}, {"Lock:transactionid", "Lock"}, {"LWLock:BufferMapping", "LWLock"}, {"Client:ClientRead", "Client"},
	}},
	"db.wait_event_type": {dims: []string{"db.wait_event_type.name"}, keys: [][]string{{"CPU"}, {"IO"}, {"Lock"}, {"LWLock"}, {"Client"}}},
	"db.user":            {dims: []string{"db.user.id", "db.user.name"}, keys: [][]string{{"16384", "app"}, {"16385", "reporting"}, {"10", "rdsadmin"}}},
	"db.host":            {dims: []string{"db.host.id", "db.host.name"}, keys: [][]string{{"host-01", "10.0.1.15"}, {"host-02", "10.0.2.27"}, {"host-03", "10.0.3.8"}}},
	"db.application":     {dims: []string{"db.application.name"}, keys: [][]string{{"app-server"}, {"psql"}, {"batch-worker"}}},
	"db.session_type":    {dims: []string{"db.session_type.name"}, keys: [][]string{{"client backend"}, {"autovacuum worker"}}},
	"db.name":            {dims: []string{"db.name.name"}, keys: [][]string{{"appdb"}, {"postgres"}}},
}

var piDocDBGroups = map[string]piGroup{
	"db.query": {dims: []string{"db.query.id", "db.query.db_id", "db.query.statement", "db.query.tokenized_id"}, keys: [][]string{
		{"Q1A2B3C4D5E6F708", "pi-8a6edf75bec26daf", `{"find": "orders", "filter": {"customerId": 4711}}`, "T9F8E7D6C5B4A392"},
		{"Q2B3C4D5E6F70819", "pi-9b7fe086cfd37eb0", `{"update": "inventory", "updates": [{"q": {"sku": "SKU-1042"}}]}`, "T8E7D6C5B4A39281"},
		{"Q3C4D5E6F708192A", "pi-ac80f197d0e48fc1", `{"insert": "events", "documents": [{"kind": "login"}]}`, "T7D6C5B4A3928170"},
	}},
	"db.query_tokenized": {dims: []string{"db.query_tokenized.id", "db.query_tokenized.db_id", "db.query_tokenized.statement"}, keys: [][]string{
		{"T9F8E7D6C5B4A392", "pi-bd91a2a8e1f5a0d2", `{"find": "orders", "filter": {"customerId": "?"}}`},
		{"T8E7D6C5B4A39281", "pi-cea2b3b9f206b1e3", `{"update": "inventory", "updates": [{"q": {"sku": "?"}}]}`},
	}},
	"db.wait_state":  {dims: []string{"db.wait_state.name"}, keys: [][]string{{"CPU"}, {"IO"}, {"Latch"}, {"Lock"}}},
	"db.user":        {dims: []string{"db.user.name"}, keys: [][]string{{"app"}, {"reporting"}}},
	"db.host":        {dims: []string{"db.host.id", "db.host.name"}, keys: [][]string{{"host-01", "10.0.1.15"}, {"host-02", "10.0.2.27"}}},
	"db.application": {dims: []string{"db.application.name"}, keys: [][]string{{"mongosh"}, {"app-server"}}},
	"db.name":        {dims: []string{"db.name.name"}, keys: [][]string{{"appdb"}, {"admin"}}},
}

// piDetailGroups are the groups GetDimensionKeyDetails supports, keyed to the
// dimension that identifies a key and the dimension holding its full text.
var piDetailGroups = map[string][2]string{
	"db.sql":   {"db.sql.id", "db.sql.statement"},
	"db.query": {"db.query.id", "db.query.statement"},
}

func piGroups(serviceType string) map[string]piGroup {
	if serviceType == "DOCDB" {
		return piDocDBGroups
	}
	return piRDSGroups
}
