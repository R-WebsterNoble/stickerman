package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
)

var db *sql.DB

func init() {
	connStr := os.Getenv("pgDBConnectionString")
	var err error
	db, err = sql.Open("postgres", connStr)
	checkErr(err)
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func handler(responseWriter http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			//fmt.Fprintf(os.Stderr, "Panic: %s, StackTrace: %s", r, debug.Stack())
			fmt.Printf("Panic: %s, StackTrace: %s", r, debug.Stack())
			//response, err = events.APIGatewayProxyResponse{StatusCode: 200}, nil
			http.Error(responseWriter, "Something went wrong :(", http.StatusInternalServerError)
			return
		}
	}()

	//fmt.Println(`{"request_body":` + strings.Replace(request.Body., "\n", "", -1) + `}`)
	ProcessRequest(responseWriter, request)

	//fmt.Println(`{"response_body":` + strings.Replace(response.Body, "\n", "", -1) + `}`)
}