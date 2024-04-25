package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bmizerany/lpx"
	"github.com/kr/logfmt"
)

// This struct and the method below takes care of capturing the data we need
// from each log line. We pass it to Keith Rarick's logfmt parser and it
// handles parsing for us.
// type routerLog struct {
// 	host string
// }

// func (r *routerLog) HandleLogfmt(key, val []byte) error {
// 	if string(key) == "host" {
// 		r.host = string(val)
// 	}
// 	return nil
// }

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

// Apr 25 01:09:01 ab-cr-pg-logdrain2pgwatch2 app/web.1 [processLogs] heroku-postgres msg body[source=DATABASE addon=postgresql-defined-24903 sample#current_transaction=12298175 sample#db_size=92451631bytes sample#tables=4 sample#active-connections=15 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99999 sample#table-cache-hit-rate=0.99933 sample#load-avg-1m=0.61 sample#load-avg-5m=0.67 sample#load-avg-15m=0.63 sample#read-iops=0 sample#write-iops=0.41772 sample#tmp-disk-used=543633408 sample#tmp-disk-available=72435159040 sample#memory-total=3944484kB sample#memory-free=841168kB sample#memory-cached=2629476kB sample#memory-postgres=20844kB sample#wal-percentage-used=0.06675254510753698
// Apr 25 01:09:01 ab-cr-pg-logdrain2pgwatch2 app/web.1 ]

// This is called every time we receive log lines from an app
func processLogs(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("[processLogs] HTTP request received\n")

	lp := lpx.NewReader(bufio.NewReader(r.Body))
	// a single request may contain multiple log lines. Loop over each of them
	for lp.Next() {
		// we only care about logs from the heroku router
		mytimeBucket, _ := timestamp2Bucket(lp.Header().Time)
		fmt.Printf("[processLogs] PrivalVersion[%v] Time[%v] Hostname[%v] Name[%v] Procid[%v] Msgid[%v]\n", string(lp.Header().PrivalVersion), mytimeBucket, string(lp.Header().Hostname), string(lp.Header().Name), string(lp.Header().Procid), string(lp.Header().Msgid))

		if string(lp.Header().Procid) == "heroku-postgres" {

			fmt.Printf("[processLogs] heroku-postgres msg body[%v]\n", string(lp.Bytes()))

			rl := new(herokuPostgresLog)
			if err := logfmt.Unmarshal(lp.Bytes(), rl); err != nil {
				fmt.Printf("Error parsing log line: %v\n", err)
			} else {
				timeBucket, err := timestamp2Bucket(lp.Header().Time)
				if err != nil {
					fmt.Printf("Error parsing time: %v", err)
					continue
				}

				fmt.Printf("time[%v] source[%v] addon[%v] loadavg1m[%v] loadavg5m[%v] loadavg15m[%v] readiops[%v] writeiops[%v] tmpdiskused[%v] tmpdiskavailable[%v] memorytotal[%v] memoryfree[%v] memorycached[%v] memorypostgres[%v] walpercentageused[%v] \n", timeBucket, rl.source, rl.addon, rl.loadavg1m, rl.loadavg5m, rl.loadavg15m, rl.readiops, rl.writeiops, rl.tmpdiskused, rl.tmpdiskavailable, rl.memorytotal, rl.memoryfree, rl.memorycached, rl.memorypostgres, rl.walpercentageused)
			}
		}
	}

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
