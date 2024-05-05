// reference(s):
// 	https://github.com/jesperfj/heroku-log-drain-sample
// 	https://github.com/heroku/sql-drain
// 	https://pkg.go.dev/net/http#Server.Shutdown

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// POST sample
// curl -u u:p -d "672 <134>1 2024-04-28T00:03:49+00:00 host app heroku-postgres - source=DATABASE addon=postgresql-defined-24903 sample#current_transaction=122163235 sample#db_size=90755887bytes sample#tables=4 sample#active-connections=15 sample#waiting-connections=0 sample#index-cache-hit-rate=0.99997 sample#table-cache-hit-rate=0.99922 sample#load-avg-1m=0.285 sample#load-avg-5m=0.345 sample#load-avg-15m=0.39 sample#read-iops=0 sample#write-iops=2.597 sample#tmp-disk-used=543633408 sample#tmp-disk-available=72435159040 sample#memory-total=3944484kB sample#memory-free=74980kB sample#memory-cached=2984436kB sample#memory-postgres=33960kB sample#wal-percentage-used=0.06650439708481809" -H "Content-Type: application/x-www-form-urlencoded" -X POST http://localhost:8080/log

const AuthUserEnv string = "AUTH_USER"
const AuthSecretEnv string = "AUTH_SECRET"
const PortEnv string = "PORT"
const SourcesEnv string = "SOURCES"

func main() {
	var srv http.Server
	srv.Addr = ":" + os.Getenv(PortEnv)

	// Catching signals in a goroutine so that it won't block and wait for all the http Server connections are closed before exiting
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		fmt.Printf("Received SIGINT/SIGTERM, shutting down ...\n")

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			fmt.Printf("Error while shutting down %v\n", err)
		}
		close(idleConnsClosed)
	}()

	http.HandleFunc("/log", checkAuth(os.Getenv(AuthUserEnv), os.Getenv(AuthSecretEnv), processLogs))
	fmt.Printf("listening on PORT[%v] ...\n", os.Getenv(PortEnv))
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}

	<-idleConnsClosed

	fmt.Printf("Exiting ...\n")
}

func checkAuth(correctUser string, correctPass string, pass http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
			return
		}

		username, password, ok := r.BasicAuth()

		if !ok {
			http.Error(w, "authorization required", http.StatusBadRequest)
			return
		}

		if (username != correctUser) || (password != correctPass) {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		pass(w, r)
	}
}
