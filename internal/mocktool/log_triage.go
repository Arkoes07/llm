package mocktool

import (
	"fmt"
	"sort"
	"strings"
)

// =====================================================================
// THE FAKE WORLD
//
// Reported symptom : checkout p99 spiked at 14:03.
// Truth            : checkout is FINE. Its DB is healthy, traffic is normal.
//                    checkout calls payment. payment's max_pool_size was
//                    lowered 20 -> 5 by a config reload at 13:58, so payment
//                    saturated its pool and started timing out under normal
//                    load. checkout is a victim, not the cause.
//
// To get this right the model must:
//   1. read checkout logs      -> sees payment.Authorize timeouts
//   2. check checkout db_pool  -> healthy, rules out checkout's own DB
//   3. check checkout req_rate -> normal, rules out a traffic spike
//   4. HOP to payment logs     -> "pool exhausted (max=5)" + the 13:58 config line
//   5. check payment db_pool   -> saturated
//   6. search the runbook      -> pool exhaustion
//
// Two traps are built in on purpose:
//   - Stopping at checkout leads to the "upstream timeout" runbook, whose fix
//     ("tune the client timeout") is plausible and WRONG.
//   - The smoking-gun config line is at 13:58, BEFORE the incident window.
//     A narrow time_range like "14:00-14:05" misses it entirely.
// =====================================================================

type logLine struct {
	ts   string // "14:03" — zero-padded, so string compare == time compare
	text string
}

var logsByService = map[string][]logLine{
	"checkout": {
		{"14:01", "checkout INFO  request completed 58ms"},
		{"14:02", "checkout INFO  request completed 61ms"},
		{"14:03", "checkout WARN  payment.Authorize slow: 900ms"},
		{"14:03", "checkout ERROR payment.Authorize timeout after 1000ms"},
		{"14:04", "checkout WARN  payment.Authorize slow: 1240ms"},
		{"14:04", "checkout ERROR payment.Authorize timeout after 1000ms"},
		{"14:07", "checkout INFO  request completed 64ms"},
	},
	"payment": {
		{"13:58", "payment INFO  config reloaded: db.max_pool_size 20 -> 5"},
		{"14:01", "payment INFO  charge authorized 118ms"},
		{"14:02", "payment WARN  db acquire wait 380ms"},
		{"14:03", "payment WARN  db acquire wait 910ms"},
		{"14:03", "payment ERROR pool exhausted (max=5)"},
		{"14:04", "payment WARN  db acquire wait 1250ms"},
		{"14:04", "payment ERROR pool exhausted (max=5)"},
		{"14:07", "payment INFO  charge authorized 131ms"},
	},
	"inventory": {
		{"14:00", "inventory INFO reservation ok 12ms"},
		{"14:03", "inventory INFO reservation ok 14ms"},
		{"14:04", "inventory INFO reservation ok 15ms"},
	},
}

var metricsByService = map[string]map[string]string{
	"checkout": {
		"db_pool":      "db_pool_active: 4/20 in use (HEALTHY). max_pool_size=20",
		"latency":      "p99: 1310ms (ELEVATED — 7d baseline ~60ms)",
		"error_rate":   "3.4% (ELEVATED — 7d baseline <0.1%)",
		"request_rate": "1180 rps (NORMAL — 7d baseline 1100–1300 rps)",
	},
	"payment": {
		"db_pool":      "db_pool_active: 5/5 in use (SATURATED). max_pool_size=5",
		"latency":      "p99: 1180ms (ELEVATED — 7d baseline ~130ms)",
		"error_rate":   "6.2% (ELEVATED — 7d baseline <0.1%)",
		"request_rate": "415 rps (NORMAL — 7d baseline 380–450 rps)",
	},
	"inventory": {
		"db_pool":      "db_pool_active: 2/20 in use (HEALTHY). max_pool_size=20",
		"latency":      "p99: 18ms (NORMAL)",
		"error_rate":   "0.01% (NORMAL)",
		"request_rate": "900 rps (NORMAL)",
	},
}

type runbook struct {
	title    string
	keywords []string
	body     string
}

var runbooks = []runbook{
	{
		title:    "DB connection pool exhaustion",
		keywords: []string{"pool", "exhausted", "acquire", "saturated", "max_pool_size"},
		body: `Symptom: "pool exhausted" errors plus rising db acquire wait times,
while request rate is normal.
Cause: max_pool_size too low for steady-state concurrency. Most often follows a
config change — check for a recent config reload that lowered max_pool_size.
Fix: restore max_pool_size to at least 2x steady-state concurrency, redeploy.`,
	},
	{
		title:    "Upstream service timeout",
		keywords: []string{"timeout", "upstream", "retry", "downstream", "slow call"},
		body: `Symptom: timeouts calling a downstream dependency.
Cause: the downstream service is degraded.
Fix: tune the client timeout and add a circuit breaker.
NOTE: this treats the symptom only. If the downstream service is itself
unhealthy, investigate THAT service before changing timeouts here.`,
	},
	{
		title:    "Slow database queries",
		keywords: []string{"slow query", "query plan", "index", "seq scan", "p99 query"},
		body: `Symptom: elevated p99 with db_query_p99 also elevated.
Cause: missing index or a query plan regression.
Fix: identify the slow query via pg_stat_statements and add an index.`,
	},
}

type queryLogsReq struct {
	Service   string `json:"service"`
	TimeRange string `json:"time_range"`
}

func queryLogs(req queryLogsReq) (string, error) {
	lines, ok := logsByService[req.Service]
	if !ok {
		return fmt.Sprintf("unknown service %q. known services: %s", req.Service, strings.Join(knownServices(), ", ")), nil
	}

	from, to, err := parseRange(req.TimeRange)
	if err != nil {
		return "", err
	}

	var out []string
	for _, l := range lines {
		if l.ts >= from && l.ts <= to {
			out = append(out, l.ts+" "+l.text)
		}
	}
	if len(out) == 0 {
		return fmt.Sprintf("no log lines for %s between %s", req.Service, req.TimeRange), nil
	}
	return strings.Join(out, "\n"), nil
}

type getMetricsReq struct {
	Service string `json:"service"`
	Metric  string `json:"metric"`
}

func getMetrics(req getMetricsReq) (string, error) {
	m, ok := metricsByService[req.Service]
	if !ok {
		return fmt.Sprintf("unknown service %q. known services: %s", req.Service, strings.Join(knownServices(), ", ")), nil
	}

	v, ok := m[req.Metric]
	if !ok {
		return fmt.Sprintf("unknown metric %q for %s. available: db_pool, latency, error_rate, request_rate", req.Metric, req.Service), nil
	}

	return fmt.Sprintf("%s.%s = %s", req.Service, req.Metric, v), nil
}

type searchRunbookReq struct {
	Query string `json:"query"`
}

func searchRunbook(req searchRunbookReq) (string, error) {
	q := strings.ToLower(req.Query)
	best, bestScore := -1, 0
	for i, rb := range runbooks {
		score := 0
		for _, kw := range rb.keywords {
			if strings.Contains(q, kw) {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}

	if best == -1 {
		return fmt.Sprintf("no runbook matched %q. try searching the error text, e.g. \"pool exhausted\"", req.Query), nil
	}

	rb := runbooks[best]
	return "Runbook: " + rb.title + "\n" + rb.body, nil
}

func parseRange(r string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(r), "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid time_range %q, expected HH:MM-HH:MM", r)
	}

	from, to := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if len(from) != 5 || len(to) != 5 {
		return "", "", fmt.Errorf("invalid time_range %q, expected HH:MM-HH:MM", r)
	}

	return from, to, nil
}

func knownServices() []string {
	var s []string
	for k := range logsByService {
		s = append(s, k)
	}

	sort.Strings(s)
	return s
}
