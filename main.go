package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"fmt"
	"runtime/debug"
	"os"
	"database/sql"
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

	fmt.Println(`{"request_body":` + strings.Replace(request.Body, "\n", "", -1) + `}`)

	response = ProcessRequest(request)

	fmt.Println(`{"response_body":` + strings.Replace(response.Body, "\n", "", -1) + `}`)

	return response, nil
}