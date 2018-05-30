package main

import (
	"log"

	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"fmt"
	"runtime/debug"
	"database/sql"
	"github.com/lib/pq"
	"os"
	"strconv"
	"strings"
	"github.com/adam-hanna/arrayOperations"
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

	response = ProcessRequest(request)

	log.Println("Responce Body: ", response.Body)

	return response, nil
}

func ProcessRequest(request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse) {
	var update Update
	if err := json.Unmarshal([]byte(request.Body), &update);
		err != nil {
		errorMessage := "error while Unmarshaling"
		log.Println(errorMessage, err)
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: errorMessage}
	}
	if update.Message != nil {
		responseMessage := processMessage(update.Message)
		return textMessageResponse(update.Message.Chat.ID, responseMessage)
	} else if update.InlineQuery != nil {
		inlineQueryResults := processInlineQuery(update.InlineQuery.Query)
		return inlineQueryResponse(update.InlineQuery.ID, inlineQueryResults)
	}

	errorMessage := "unable to process request: neither message or update found"
	log.Println(errorMessage)
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: errorMessage}
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

func cleanKeywords(queryString string) []string {
	if len(queryString) == 0 {
		return []string{}
	}

	queryString = strings.ToLower(queryString)
	queryString = strings.Replace(queryString, ",", " ", -1)
	queryString = strings.Replace(queryString, ":", " ", -1)
	queryString = strings.Replace(queryString, ".", " ", -1)

	return strings.Fields(queryString)
}

func processInlineQuery(queryString string) []string {
	queryString = strings.Join(cleanKeywords(queryString), " ")

	if len(queryString) == 0 {
		return []string{}
	}

	queryString = EscapeSql(queryString)

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query(`SELECT  
  array_agg(s.file_id)
FROM
  keywords k
  JOIN sticker_keywords sk ON sk.keyword_id = k.id
  JOIN stickers s ON sk.sticker_id = s.id
WHERE k.keyword ILIKE ANY (string_to_array($1, ' '))
GROUP BY k.keyword;`, queryString+"%")
	defer rows.Close()
	checkErr(err)

	var allStickerFileIds []string
	if rows.Next() {
		rows.Scan(pq.Array(&allStickerFileIds))
		checkErr(err)
		for rows.Next() {
			var fileIdsForKeyword []string
			rows.Scan(pq.Array(&fileIdsForKeyword))
			checkErr(err)

			v, ok := arrayOperations.Intersect(allStickerFileIds, fileIdsForKeyword)
			if !ok {
				return allStickerFileIds
			}
			allStickerFileIds, ok = v.Interface().([]string)
			if !ok {
				return allStickerFileIds
			}
		}
	}
	checkErr(err)

	return allStickerFileIds
}

func EscapeSql(s string) (result string) {
	result = strings.Replace(s, "\\", "\\\\", -1)
	result = strings.Replace(result, "%", "\\%", -1)
	result = strings.Replace(result, "_", "\\_", -1)
	return result
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func processMessage(message *Message) (responseMessage string) {
	if message.ReplyToMessage != nil && message.ReplyToMessage.Sticker != nil && len(message.Text) != 0 {
		return addKeywordFromStickerReply(message)
	}

	if len(message.Text) != 0 {
		if message.Text[0] == '/' {
			switch message.Text {
			case "/start":
				fallthrough
			case "/help":
				return "This Bot is designed to help you find Stickers.\n" +
					"\n" +
					"Usage:\n" +
					"To search for Stickers in any chat type: @DevStampsBot followed by your search keywords.\n" +
					"\n" +
					"To add new Stickers and keywords to the bot, send the sticker to this chat then reply to the sticker with a message containing the keywords you want to add."
			case "/add":
				SetUserMode(message.Chat.ID, "add")
				return "You are now adding keywords"
			case "/remove":
				SetUserMode(message.Chat.ID, "remove")
				return "You are now removing keywords from the sticker"
			}
		} else {
			return ProcessKeywordMessage(message)
		}
	} else if message.Sticker != nil {
		return ProcessStickerMessage(message)
	}

	return "I don't know how to interpret your message"
}

func ProcessKeywordMessage(message *Message) (responseMessage string) {
	usersStickerId, mode := GetUserState(message.Chat.ID)
	if usersStickerId == "" {
		responseMessage = "Send a sticker to me then I'll be able to add searchable keywords to it"
	}

	switch mode {
	case "add":
		responseMessage = addKeywordsToSticker(usersStickerId, message.Text)
	case "remove":
		return removeKeywordsFromSticker(usersStickerId, message.Text)
	}

	return responseMessage
}

func ProcessStickerMessage(message *Message) (responseMessage string) {
	mode := SetUserStickerAndGetMode(message.Chat.ID, message.Sticker.FileID)
	return "Now you are " + mode + "ing keywords to " + message.Sticker.Emoji + " sticker"
}

func SetUserMode(chatId int64, userMode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	_, err = db.Exec("INSERT INTO sessions (chat_id, mode) VALUES ($1, $2) ON CONFLICT( chat_id ) DO UPDATE set mode=excluded.mode;", chatId, userMode)
	checkErr(err)

	return
}

func SetUserStickerAndGetMode(chatId int64, usersStickerId string) (mode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	err = db.QueryRow("INSERT INTO sessions (chat_id, file_id) VALUES ($1, $2)\nON CONFLICT( chat_id ) DO UPDATE set file_id=excluded.file_id  RETURNING mode;", chatId, usersStickerId).Scan(&mode)
	checkErr(err)

	return
}

func GetUserState(chatId int64) (usersStickerId string, usersMode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	rows, err := db.Query(`SELECT file_id, mode FROM sessions WHERE chat_id = $1`, chatId)
	defer rows.Close()
	checkErr(err)

	for rows.Next() {
		rows.Scan(&usersStickerId, &usersMode)
		checkErr(err)
	}
	checkErr(err)

	return
}

func addKeywordFromStickerReply(message *Message) (responseMessage string) {
	stickerFileId := message.ReplyToMessage.Sticker.FileID
	return addKeywordsToSticker(stickerFileId, message.Text)
}

func addKeywordsToSticker(stickerFileId string, keywordsString string) (responseMessage string) {
	keywords := cleanKeywords(keywordsString)

	if len(keywords) == 0 {
		return "No keywords to add"
	}

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	defer db.Close()
	checkErr(err)

	transaction, err := db.Begin()
	defer func() {
		err = transaction.Rollback()
		if err != nil && err != sql.ErrTxDone {
			panic(err)
		}
	}()
	checkErr(err)

	insertStickersStatement, err := transaction.Prepare("INSERT INTO stickers( file_id ) VALUES( $1 ) ON CONFLICT( file_id ) DO UPDATE set file_id=excluded.file_id RETURNING id;")
	defer insertStickersStatement.Close()
	checkErr(err)

	insertKeywordsStatement, err := transaction.Prepare("INSERT INTO keywords( keyword ) VALUES( $1 ) ON CONFLICT( keyword ) DO UPDATE set keyword=excluded.keyword RETURNING id;")
	defer insertKeywordsStatement.Close()
	checkErr(err)

	insertStickersKeywordsStatement, err := transaction.Prepare("INSERT INTO sticker_keywords( sticker_id, keyword_id ) VALUES( $1, $2 ) ON CONFLICT DO NOTHING;")
	defer insertStickersKeywordsStatement.Close()
	checkErr(err)

	stickerResultRows, err := insertStickersStatement.Query(stickerFileId)
	defer stickerResultRows.Close()
	checkErr(err)

	var stickerId int
	for stickerResultRows.Next() {
		err = stickerResultRows.Scan(&stickerId)
		checkErr(err)
	}
	err = insertStickersStatement.Close()
	checkErr(err)

	var keywordsAdded int64
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		keywordsResultRows, err := insertKeywordsStatement.Query(keyword)
		checkErr(err)

		var keywordId int
		for keywordsResultRows.Next() {
			err = keywordsResultRows.Scan(&keywordId)
			checkErr(err)
		}
		checkErr(err)
		err = keywordsResultRows.Close()
		checkErr(err)

		stickersKeywordsResult, err := insertStickersKeywordsStatement.Exec(stickerId, keywordId)
		checkErr(err)

		numRowsAffected, err := stickersKeywordsResult.RowsAffected()
		checkErr(err)

		keywordsAdded += numRowsAffected
	}

	responseMessage = "Added " + strconv.FormatInt(keywordsAdded, 10) + " keyword(s)."

	err = transaction.Commit()
	checkErr(err)

	return
}

func removeKeywordsFromSticker(stickerFileId string, keywordsString string) string {
	keywords := strings.Join(cleanKeywords(keywordsString), " ")

	if len(keywords) == 0 {
		return "No keywords to remove"
	}

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	result, err := db.Exec("DELETE FROM sticker_keywords sk \nUSING keywords k, stickers s\n    WHERE sk.keyword_id = k.id\n    AND sk.sticker_id = s.id\nand s.file_id = $1\nand k.keyword ILIKE ANY (string_to_array($2, ' '));", stickerFileId, keywords)
	checkErr(err)

	numRows, err := result.RowsAffected()

	return "You have deleted " + strconv.FormatInt(numRows, 10) + " keywords."
}

func textMessageResponse(chatId int64, text string) (events.APIGatewayProxyResponse) {
	response := Response{
		"sendMessage",
		chatId,
		text,
	}

	jsonString, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            string(jsonString),
		IsBase64Encoded: false,
	}
}
