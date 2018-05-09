package main

import (
	"log"

	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"errors"
	"fmt"
	"runtime/debug"
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrNameNotProvided is thrown when a name is not provided
	ErrNotMessage = errors.New("Error: can only process Messages")
)

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

	var update Update
	if err := json.Unmarshal([]byte(request.Body), &update);
	err != nil {
		log.Println("error while Unmarshaling: ", err)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	if update.Message != nil {
		responseMessage := processMessage(update)
		return textMessageResponse(update.Message.Chat.ID, responseMessage), nil
	} else {
		error := errors.New("update is not a Message, cant extract chatId")
		log.Println(error)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}
}

func DbStuff() string{

	//dbinfo := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=require",
	//	"devstampsdbinstance.cc0dyoy2bph0.eu-west-2.rds.amazonaws.com", "rds_master", "De8i8tcMIgVX0Hg49auUz+WHvvS5YMLOX1piFLRFlxY=", "devStamps")
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query("SELECT 1;")
	checkErr(err)

	for rows.Next() {
		var num int
		err = rows.Scan(&num)

		checkErr(err)
		fmt.Println("num is ", num)

		return "num is: " + strconv.Itoa(num)
	}

	return "fail"

	//
	//fmt.Println("# Inserting values")
	//
	//var lastInsertId int
	//err = db.QueryRow("INSERT INTO userinfo(username,departname,created) VALUES($1,$2,$3) returning uid;", "astaxie", "研发部门", "2012-12-09").Scan(&lastInsertId)
	//checkErr(err)
	//fmt.Println("last inserted id =", lastInsertId)
	//
	//fmt.Println("# Updating")
	//stmt, err := db.Prepare("update userinfo set username=$1 where uid=$2")
	//checkErr(err)
	//
	//res, err := stmt.Exec("astaxieupdate", lastInsertId)
	//checkErr(err)
	//
	//affect, err := res.RowsAffected()
	//checkErr(err)
	//
	//fmt.Println(affect, "rows changed")
	//
	//fmt.Println("# Querying")
	//rows, err := db.Query("SELECT * FROM userinfo")
	//checkErr(err)
	//
	//for rows.Next() {
	//	var uid int
	//	var username string
	//	var department string
	//	var created time.Time
	//	err = rows.Scan(&uid, &username, &department, &created)
	//	checkErr(err)
	//	fmt.Println("uid | username | department | created ")
	//	fmt.Printf("%3v | %8v | %6v | %6v\n", uid, username, department, created)
	//}
	//
	//fmt.Println("# Deleting")
	//stmt, err = db.Prepare("delete from userinfo where uid=$1")
	//checkErr(err)
	//
	//res, err = stmt.Exec(lastInsertId)
	//checkErr(err)
	//
	//affect, err = res.RowsAffected()
	//checkErr(err)
	//
	//fmt.Println(affect, "rows changed")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func processMessage(update Update) (responseMessage string) {
	var response string

	if update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.Sticker != nil && len(update.Message.Text) != 0{
		return addKeywordToSticker(update)
	}

	if len(update.Message.Text) != 0 {
		response = "You said " + update.Message.Text
	} else if update.Message.Sticker != nil {
		response = "You sent a " + update.Message.Sticker.Emoji + " sticker!"
	}

	return response
}

func addKeywordToSticker(update Update)(responseMessage string){
	stickerFileId := update.Message.ReplyToMessage.Sticker.FileID
	keyword := strings.ToLower(update.Message.Text)

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	transaction, err := db.Begin()
	checkErr(err)
	defer func() {
		err = transaction.Rollback()
		if err != nil && err != sql.ErrTxDone{
			panic(err)
		}
	}()

	insertStickersStatement, err := transaction.Prepare("INSERT INTO stickers(  file_id ) VALUES( $1 ) ON CONFLICT( file_id ) DO UPDATE set file_id=excluded.file_id RETURNING id;")
	checkErr(err)

	insertKeywordsStatement, err := transaction.Prepare("INSERT INTO keywords(  keyword ) VALUES( $1 ) ON CONFLICT( keyword ) DO UPDATE set keyword=excluded.keyword RETURNING id;")
	checkErr(err)

	insertStickersKeywordsStatement, err := transaction.Prepare("INSERT INTO sticker_keywords(  sticker_id, keyword_id ) VALUES( $1, $2 ) ON CONFLICT DO NOTHING;")
	checkErr(err)

	stickerResultRows, err := insertStickersStatement.Query(stickerFileId)
	checkErr(err)

	var stickerId int
	for stickerResultRows.Next(){
		err = stickerResultRows.Scan(&stickerId)
		checkErr(err)
	}
	err = insertStickersStatement.Close()
	checkErr(err)

	keywordsResultRows, err := insertKeywordsStatement.Query(keyword)
	checkErr(err)

	var keywordId int
	for keywordsResultRows.Next(){
		err = keywordsResultRows.Scan(&keywordId)
		checkErr(err)
	}

	stickersKeywordsResult, err := insertStickersKeywordsStatement.Exec(stickerId, keywordId)
	checkErr(err)

	numRowsAffected, err := stickersKeywordsResult.RowsAffected()
	checkErr(err)

	responseMessage = "Added "+ strconv.FormatInt(numRowsAffected, 10) +" keyword(s)."

	err = transaction.Commit()
	checkErr(err)

	return
}

func textMessageResponse(ChatId int64, text string) (events.APIGatewayProxyResponse) {
	response := Response{
		"sendMessage",
		ChatId,
		text,
	}

	json, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            string(json),
		IsBase64Encoded: false,
	}
}
