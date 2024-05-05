// reference(s):
// 	https://github.com/heroku/sql-drain
// 	debug test bug - https://github.com/golang/vscode-go/issues/2953

package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const MetricsDbUrlEnv string = "PGWATCH2_URL"

var db *sql.DB
var cpuLoadInsertStmt *sql.Stmt
var herokuPgStatsInsertStmt *sql.Stmt
var initMetricsTableAndPartitionsSelectStmt *sql.Stmt

func herokuPgStatsInsert(time time.Time, dbname string, data []byte) error {
	_, err := herokuPgStatsInsertStmt.Exec(
		time,
		dbname,
		data)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
	}

	return err
}

func cpuLoadInsert(time time.Time, dbname string, data []byte) error {
	_, err := cpuLoadInsertStmt.Exec(
		time,
		dbname,
		data)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
	}

	return err
}

func initMetricsTableAndPartitions(metricname string, time time.Time) error {
	_, err := initMetricsTableAndPartitionsSelectStmt.Exec(
		metricname,
		time)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
	}

	return err
}

func init() {
	// Connect to postgresql
	var err error

	dburl := os.Getenv(MetricsDbUrlEnv) + "?sslmode=require"

	u, err := url.Parse(dburl)
	if err != nil {
		fmt.Printf("Invalid metrics DB URL: %v\n", err)
	}

	if isEnv(DebugEnv) {
		fmt.Printf("[db.go:init] metrics db url %v\n", u.Redacted())
	}

	db, err = sql.Open("postgres", dburl)
	if err != nil {
		fmt.Printf("Open DB error: %v\n", err)
	}

	err = db.Ping()
	if err != nil {
		fmt.Printf("Unable to ping DB: %v\n", err)
	}

	cpuLoadInsertStmt, err = db.Prepare("INSERT into cpu_load(time, dbname, data) VALUES ($1, $2, $3);")
	if err != nil {
		fmt.Printf("Unable to create prepared stmt: %v\n", err)
	}

	herokuPgStatsInsertStmt, err = db.Prepare("INSERT into heroku_pg_stats(time, dbname, data) VALUES ($1, $2, $3);")
	if err != nil {
		fmt.Printf("Unable to create prepared stmt: %v\n", err)
	}

	// The 3rd param ensures that there are always 2 partitions and that a new partition is created when the metric time is within the last partition
	// for example, these are the current partitions:
	// Partitions: subpartitions.cpu_load_y2024w17 FOR VALUES FROM ('2024-04-22 00:00:00+00') TO ('2024-04-29 00:00:00+00'),
	// 			   subpartitions.cpu_load_y2024w18 FOR VALUES FROM ('2024-04-29 00:00:00+00') TO ('2024-05-06 00:00:00+00')
	//
	// it's 2024-04-30 and calling the admin.ensure_partition_metric_time() it will create the following partition
	//
	// Partitions: subpartitions.cpu_load_y2024w17 FOR VALUES FROM ('2024-04-22 00:00:00+00') TO ('2024-04-29 00:00:00+00'),
	// 			   subpartitions.cpu_load_y2024w18 FOR VALUES FROM ('2024-04-29 00:00:00+00') TO ('2024-05-06 00:00:00+00'),
	// 			   subpartitions.cpu_load_y2024w19 FOR VALUES FROM ('2024-05-06 00:00:00+00') TO ('2024-05-13 00:00:00+00')
	//
	// select * from admin.ensure_partition_metric_time($1 ==> metric name 'cpu_load', $2 ==> now(), $3 ==> 1)
	initMetricsTableAndPartitionsSelectStmt, err = db.Prepare("select * from admin.ensure_partition_metric_time($1,$2, 1);")
	if err != nil {
		fmt.Printf("Unable to create prepared stmt: %v\n", err)
	}
}
