// reference(s):
// https://github.com/jesperfj/heroku-log-drain-sample
// https://github.com/heroku/sql-drain

package main

import (
	"fmt"
	"net/http"
	"os"
)

const AuthUserEnv string = "AUTH_USER"
const AuthSecretEnv string = "AUTH_SECRET"
const PortEnv string = "PORT"
const SourcesEnv string = "SOURCES"

func main() {

	http.HandleFunc("/log", checkAuth(os.Getenv(AuthUserEnv), os.Getenv(AuthSecretEnv), processLogs))
	fmt.Println("listening...")
	err := http.ListenAndServe(":"+os.Getenv(PortEnv), nil)
	if err != nil {
		panic(err)
	}
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
