package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bmizerany/lpx"
	"github.com/kr/logfmt"
)

const DebugEnv string = "HKPG_LOGDRAIN_DEBUG"
const PostgresProcId string = "heroku-postgres"

// This struct and the method below takes care of capturing the data we need
// from each log line. We pass it to Keith Rarick's logfmt parser and it
// handles parsing for us.
type herokuPostgresLog struct {
	source string
	addon  string

	loadavg1m  float64
	loadavg5m  float64
	loadavg15m float64

	readiops  float64
	writeiops float64

	tmpdiskused      int64
	tmpdiskavailable int64

	memorytotal    int64
	memoryfree     int64
	memorycached   int64
	memorypostgres int64

	walpercentageused float64
}

func (r *herokuPostgresLog) HandleLogfmt(key, val []byte) error {
	if string(key) == "source" {
		r.source = string(val)
	} else if string(key) == "addon" {
		r.addon = string(val)
	} else if string(key) == "sample#load-avg-1m" {
		r.loadavg1m, _ = strconv.ParseFloat(string(val), 64)
	} else if string(key) == "sample#load-avg-5m" {
		r.loadavg5m, _ = strconv.ParseFloat(string(val), 64)
	} else if string(key) == "sample#load-avg-15m" {
		r.loadavg15m, _ = strconv.ParseFloat(string(val), 64)
	} else if string(key) == "sample#read-iops" {
		r.readiops, _ = strconv.ParseFloat(string(val), 64)
	} else if string(key) == "sample#write-iops" {
		r.writeiops, _ = strconv.ParseFloat(string(val), 64)
	} else if string(key) == "sample#tmp-disk-used" {
		r.tmpdiskused, _ = strconv.ParseInt(string(val), 10, 64)
	} else if string(key) == "sample#tmp-disk-available" {
		r.tmpdiskavailable, _ = strconv.ParseInt(string(val), 10, 64)
	} else if string(key) == "sample#memory-total" { // kB
		r.memorytotal, _ = strconv.ParseInt(strings.TrimSuffix(string(val), "kB"), 10, 64)
	} else if string(key) == "sample#memory-free" { // kB
		r.memoryfree, _ = strconv.ParseInt(strings.TrimSuffix(string(val), "kB"), 10, 64)
	} else if string(key) == "sample#memory-cached" { // kB
		r.memorycached, _ = strconv.ParseInt(strings.TrimSuffix(string(val), "kB"), 10, 64)
	} else if string(key) == "sample#memory-postgres" { // kB
		r.memorypostgres, _ = strconv.ParseInt(strings.TrimSuffix(string(val), "kB"), 10, 64)
	} else if string(key) == "sample#wal-percentage-used" {
		r.walpercentageused, _ = strconv.ParseFloat(string(val), 64)
	}
	return nil
}

type CpuLoadData struct {
	Load_1min  float64 `json:"load_1min"`
	Load_5min  float64 `json:"load_5min"`
	Load_15min float64 `json:"load_15min"`
}

// HTTP request body (POST)
// 672 <134>1 2024-04-28T00:03:49+00:00 host app heroku-postgres - source=DATABASE addon=postgresql-defined-24903 sample#current_transaction=122163235 sample#db_size=90755887bytes sample#tables=4 sample#active-connections=15 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99997 sample#table-cache-hit-rate=0.99922 sample#load-avg-1m=0.285 sample#load-avg-5m=0.345 sample#load-avg-15m=0.39 sample#read-iops=0 sample#write-iops=2.597 sample#tmp-disk-used=543633408 sample#tmp-disk-available=72435159040 sample#memory-total=3944484kB sample#memory-free=74980kB sample#memory-cached=2984436kB sample#memory-postgres=33960kB sample#wal-percentage-used=0.06650439708481809
// Apr 25 01:09:01 ab-cr-pg-logdrain2pgwatch2 app/web.1 [processLogs] heroku-postgres msg body[source=DATABASE addon=postgresql-defined-24903 sample#current_transaction=12298175 sample#db_size=92451631bytes sample#tables=4 sample#active-connections=15 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99999 sample#table-cache-hit-rate=0.99933 sample#load-avg-1m=0.61 sample#load-avg-5m=0.67 sample#load-avg-15m=0.63 sample#read-iops=0 sample#write-iops=0.41772 sample#tmp-disk-used=543633408 sample#tmp-disk-available=72435159040 sample#memory-total=3944484kB sample#memory-free=841168kB sample#memory-cached=2629476kB sample#memory-postgres=20844kB sample#wal-percentage-used=0.06675254510753698

// This is called every time we receive log lines from an app
func processLogs(w http.ResponseWriter, r *http.Request) {

	if isEnv(DebugEnv) {
		bodyBytes, _ := io.ReadAll(r.Body)
		fmt.Printf("[processLogs] HTTP request received %v\n", string(bodyBytes))
	}

	lp := lpx.NewReader(bufio.NewReader(r.Body))
	// a single request may contain multiple log lines. Loop over each of them
	for lp.Next() {

		if isEnv(DebugEnv) {
			mytimeBucket, _ := timestamp2Bucket(lp.Header().Time)
			fmt.Printf("[processLogs] PrivalVersion[%v] Time[%v] Hostname[%v] Name[%v] Procid[%v] Msgid[%v]\n", string(lp.Header().PrivalVersion), mytimeBucket, string(lp.Header().Hostname), string(lp.Header().Name), string(lp.Header().Procid), string(lp.Header().Msgid))
		}

		// we only care about logs from the heroku-postgres
		if string(lp.Header().Procid) == PostgresProcId {

			fmt.Printf("[processLogs] heroku-postgres msg body[%v]\n", strings.TrimSuffix(string(lp.Bytes()), "\n"))

			rl := new(herokuPostgresLog)
			if err := logfmt.Unmarshal(lp.Bytes(), rl); err != nil {
				fmt.Printf("Error parsing log line: %v\n", err)
			} else {
				/*
					timeBucket, err := timestamp2Bucket(lp.Header().Time)
					if err != nil {
						fmt.Printf("Error parsing time: %v", err)
						continue
					}
				*/

				fmt.Printf("time[%v] source[%v] addon[%v] loadavg1m[%v] loadavg5m[%v] loadavg15m[%v] readiops[%v] writeiops[%v] tmpdiskused[%v] tmpdiskavailable[%v] memorytotal[%v] memoryfree[%v] memorycached[%v] memorypostgres[%v] walpercentageused[%v] \n", /*timeBucket*/
					string(lp.Header().Time), rl.source, rl.addon, rl.loadavg1m, rl.loadavg5m, rl.loadavg15m, rl.readiops, rl.writeiops, rl.tmpdiskused, rl.tmpdiskavailable, rl.memorytotal, rl.memoryfree, rl.memorycached, rl.memorypostgres, rl.walpercentageused)

				t, err := timestamp2Time(lp.Header().Time)
				if err != nil {
					fmt.Printf("could not convert to timestamp: %s\n", err)
					continue
				}

				// retrieve from the config {"DATABASE": "PGWATCH2_MONITOREDDB_MYTARGETDB_URL", "DATABASE_ONYX": "PGWATCH2_MONITOREDDB_2_URL", "DATABASE_GREEN": "PGWATCH2_MONITOREDDB_3_URL"}
				// if source is one of the configured sources then retrieve the related monitored db name used to store metrics
				//
				fmt.Printf("looking for source[%v] in [%v]\n", rl.source, os.Getenv(SourcesEnv))

				sourcesMap := make(map[string]string)
				if err := json.Unmarshal([]byte(os.Getenv(SourcesEnv)), &sourcesMap); err == nil {
					fmt.Printf("unmarshalled JSON %v\n", len(sourcesMap))

					if monitoreddbname, ok := sourcesMap[rl.source]; ok {
						//var monitoreddbname = "PGWATCH2_MONITOREDDB_MYTARGETDB_URL"
						//if isEnv(DebugEnv) {
						fmt.Printf("found source[%v] monitored db name[%v]\n", rl.source, monitoreddbname)
						//}

						_ = insertCpuLoadMetrics(rl, t, monitoreddbname)
					}
				} else {
					fmt.Printf("json.Unmarshal error: %v\n", err)
				}
			}
		}
	}

}

func insertCpuLoadMetrics(rl *herokuPostgresLog, t time.Time, monitoreddbname string) error {
	cld := new(CpuLoadData)
	cld.Load_1min = rl.loadavg1m
	cld.Load_5min = rl.loadavg5m
	cld.Load_15min = rl.loadavg15m

	jsonData, err := json.Marshal(cld)
	if err != nil {
		fmt.Printf("could not marshal json: %s\n", err)
		return err
	}

	cpuLoadInsert(t, monitoreddbname, jsonData)
	return nil
}

// Heroku log lines are formatted according to RFC5424 which is a subset
// of RFC3339 (RFC5424 is more restrictive).
// Reference: https://devcenter.heroku.com/articles/logging#log-format
func timestamp2Bucket(b []byte) (int64, error) {
	t, err := time.Parse(time.RFC3339, string(b))
	if err != nil {
		return 0, err
	}
	return (t.Unix() / 60) * 60, nil
}

func timestamp2Time(b []byte) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, string(b))
	if err != nil {
		return time.Now(), err
	}
	return t, nil
}

func isEnv(key string) bool {
	if _, ok := os.LookupEnv(key); ok {
		return true
	}
	return false
}

// CREATE TABLE public.mycpu_load (
// 	time timestamp with time zone,
// 	dbname text,
// 	data jsonb
//   );
