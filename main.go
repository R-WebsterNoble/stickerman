package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
)

var db *sql.DB

func init() {
	connStr := os.Getenv("pgDBConnectionString")
	var err error
	db, err = sql.Open("postgres", connStr)
	checkErr(err)
}

func main() {
	telegramBotApiKey := os.Getenv("TelegramBotApiKey")
	telegramBotApiKey = strings.Replace(telegramBotApiKey, ":", "", -1)
	http.HandleFunc("/"+telegramBotApiKey, handler)
	log.Fatal(http.ListenAndServe(":8085", middleware(http.DefaultServeMux)))
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

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

//func logRequest(handler http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		//buf := new(bytes.Buffer)
//		//buf.ReadFrom(r.Body)
//		//requestBody := buf.String() // Does a complete copy of the bytes in the buffer.
//		//log.Printf("RemoteAddr:%s Method:%s URL:%s Body:%s", r.RemoteAddr, r.Method, r.URL, requestBody)
//
//
//		handler.ServeHTTP(w, r)
//	})
//}