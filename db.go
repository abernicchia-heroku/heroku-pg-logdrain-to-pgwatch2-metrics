// reference(s):
// https://github.com/heroku/sql-drain
// debug test bug - https://github.com/golang/vscode-go/issues/2953

package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const MetricsDbUrlEnv string = "PGWATCH2_URL"

var db *sql.DB
var cpuLoadInsertStmt *sql.Stmt

func cpuLoadInsert(time time.Time, dbname string, data []byte) error {
	//fmt.Printf("CpuLoadInsert INSERT into cpu_load(time, dbname, data) VALUES (%v, %v, %v);\n", time, dbname, string(data))

	_, err := cpuLoadInsertStmt.Exec(
		time,
		dbname,
		data)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
	}

	return err
}

func init() {
	// Connect to postgresql
	var err error

	dburl := os.Getenv(MetricsDbUrlEnv) + "?sslmode=require"

	if isEnv(DebugEnv) {
		fmt.Printf("[db.go:init] metrics db url %v\n", dburl)
	}

	db, err = sql.Open("postgres", dburl)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
	}

	err = db.Ping()
	if err != nil {
		fmt.Printf("Unable to ping DB: %v\n", err)
	}

	cpuLoadInsertStmt, err = db.Prepare("INSERT into cpu_load(time, dbname, data) VALUES ($1, $2, $3);")
	if err != nil {
		fmt.Printf("Unable to create prepared stmt: %v\n", err)
	}
}
