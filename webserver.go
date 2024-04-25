package main

// reference: https://github.com/jesperfj/heroku-log-drain-sample

import (
	"fmt"
	"net/http"
	"os"
)

func main() {

	http.HandleFunc("/log", checkAuth(os.Getenv("AUTH_USER"), os.Getenv("AUTH_SECRET"), processLogs))
	fmt.Println("listening...")
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
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
