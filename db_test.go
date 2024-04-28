// ENV VAR for tests https://medium.com/the-try-and-catch/how-to-debug-go-with-vs-code-412f8686ebdf

package main

import (
	"testing"
	"time"
)

func TestInsertCpuLoadMetrics(t *testing.T) {

	var hpglog herokuPostgresLog
	hpglog.loadavg1m = 0.54
	hpglog.loadavg5m = 2.4
	hpglog.loadavg15m = 5.4

	insertCpuLoadMetrics(&hpglog, time.Now(), "PGWATCH2_MONITOREDDB_MYTARGETDB_URL")
}
