package main

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"fmt"
	"runtime/debug"
	"os"
	"database/sql"
)

var db *sql.DB

func init() {
	connStr := os.Getenv("pgDBConnectionString")
	var err error
	db, err = sql.Open("postgres", connStr)
	checkErr(err)
}

func Shutdown() {
	CloseDb()
}

func CloseDb() {
	err := db.Close()
	checkErr(err)
}

func main() {
	lambda.Start(Handler)
}

func Handler(request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			//fmt.Fprintf(os.Stderr, "Panic: %s, StackTrace: %s", r, debug.Stack())
			fmt.Printf("Panic: %s, StackTrace: %s", r, debug.Stack())
			response, err = events.APIGatewayProxyResponse{StatusCode: 200}, nil
		}
	}()

	log.Println("Request Body: ", request.Body)

	response = ProcessRequest(request)

	log.Println("Responce Body: ", response.Body)

	return response, nil
}