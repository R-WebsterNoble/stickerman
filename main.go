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
	} else if update.InlineQuery != nil {
		//inlineQueryResults := processInlineQuery(update)
		inlineQueryResults := []string{"CAADAQADHgADVwb9BrrlpxhL2ltZAg", "CAADAQADjgsAAiPdEAaKEpsdVb8xOAI", "CAADAQADKgAD5_bHDFnhkQhE_myDAg"}
		return inlineQueryResponse(update.InlineQuery.ID, inlineQueryResults), nil
	} else {
		err = errors.New("update is not a Message, cant extract chatId")
		log.Println(err)
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}
}

type InlineQueryResultCachedSticker struct {
	Type          string `json:"type"`
	Id            string `json:"id"`
	StickerFileId string `json:"sticker_file_id"`
}

type AnswerCallbackQuery struct {
	Method        string                           `json:"method"`
	InlineQueryId string                           `json:"inline_query_id"`
	Results       []InlineQueryResultCachedSticker `json:"results"`
}

func inlineQueryResponse(inlineQueryID string, queryResultStickerIds []string) events.APIGatewayProxyResponse {
	queryResultStickers := make([]InlineQueryResultCachedSticker, len(queryResultStickerIds))
	for i, stickerId := range queryResultStickerIds {
		queryResultStickers[i] = InlineQueryResultCachedSticker{
			Type:          "sticker",
			Id:            strconv.Itoa(i),
			StickerFileId: stickerId,
		}
	}

	response := AnswerCallbackQuery{
		"answerInlineQuery",
		inlineQueryID,
		queryResultStickers[:],
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	jsonString := string(jsonResult)
	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            jsonString,
		IsBase64Encoded: false,
	}
}

func processInlineQuery(update Update) []string {
	return []string{}
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

func textMessageResponse(chatId int64, text string) (events.APIGatewayProxyResponse) {
	response := Response{
		"sendMessage",
		chatId,
		text,
	}

	jsons, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            string(jsons),
		IsBase64Encoded: false,
	}
}
