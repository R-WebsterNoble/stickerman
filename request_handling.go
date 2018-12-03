package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
)

func ProcessRequest(responseWriter http.ResponseWriter, request *http.Request) {
	var update Update
	if err := json.NewDecoder(request.Body).Decode(&update);
		err != nil {
		errorMessage := "error while Unmarshaling"
		log.Println(errorMessage, err)
		//return events.APIGatewayProxyResponse{StatusCode: 200, Body: errorMessage}
		http.Error(responseWriter, errorMessage, http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{"update": update}).Info()

	if update.Message != nil {
		response := processMessage(update.Message)
		sendChatResponse(responseWriter, update.Message.Chat.ID, response)
		return
		//return textMessageResponse(update.Message.Chat.ID, responseMessage)
	} else if update.InlineQuery != nil {
		groupId := getOrCreateUserGroup(int64(update.InlineQuery.From.ID))
		var offset int
		if update.InlineQuery.Offset != "" {
			var err error
			offset, err = strconv.Atoi(update.InlineQuery.Offset)
			checkErr(err)
		}
		inlineQueryResults, nextOffset := GetAllStickerIdsForKeywords(update.InlineQuery.Query, groupId, offset)
		inlineQueryResponse(responseWriter, update.InlineQuery.ID, inlineQueryResults, nextOffset)
		return
	} else if update.ChosenInlineResult != nil {
		return
	} else if update.CallbackQuery != nil {
		sendAnswerCallbackQuery(responseWriter, AnswerCallbackQuery{
			CallbackQueryId: update.CallbackQuery.ID,
			Text:            "you clicked: " + update.CallbackQuery.Data,
		})
		return
	}

	errorMessage := "unable to process request: neither message nor update found"
	log.Println(errorMessage)
	http.Error(responseWriter, errorMessage, http.StatusBadRequest)
	return
	//return events.APIGatewayProxyResponse{StatusCode: 200, Body: errorMessage}
}

type InlineQueryResultCachedSticker struct {
	Type          string `json:"type"`
	Id            string `json:"id"`
	StickerFileId string `json:"sticker_file_id"`
}

func inlineQueryResponse(responseWriter http.ResponseWriter, inlineQueryID string, queryResultStickerIds []string, nextOffset int) {
	queryResultStickers := make([]InlineQueryResultCachedSticker, len(queryResultStickerIds))
	for i, stickerId := range queryResultStickerIds {
		queryResultStickers[i] = InlineQueryResultCachedSticker{
			Type:          "sticker",
			Id:            strconv.Itoa(i),
			StickerFileId: stickerId,
		}
	}

	var nextOffsetString string

	if nextOffset > 0 {
		nextOffsetString = strconv.Itoa(nextOffset)
	}

	response := AnswerInlineQuery{
		"answerInlineQuery",
		inlineQueryID,
		queryResultStickers[:],
		0,
		true,
		nextOffsetString,
	}

	sendInlineQueryResponse(responseWriter, inlineQueryID, &response)
	//jsonString := string(jsonResult)
	//return events.APIGatewayProxyResponse{
	//	StatusCode:      200,
	//	Headers:         map[string]string{"Content-Type": "application/json"},
	//	Body:            jsonString,
	//	IsBase64Encoded: false,
	//}
}

func sendChatResponse(responseWriter http.ResponseWriter, chatId int64, response BotResponce) {
	response.SetChatId(chatId)
	jsonResponse := response.ToJson()
	log.WithFields(log.Fields{"response": response}).Info()
	_, err := io.WriteString(responseWriter, jsonResponse)
	checkErr(err)
}

func sendInlineQueryResponse(responseWriter http.ResponseWriter, inlineQueryId string, response BotResponce) {
	response.SetInlineQueryId(inlineQueryId)
	jsonResponse := response.ToJson()
	log.WithFields(log.Fields{"response": response}).Info()
	_, err := io.WriteString(responseWriter, jsonResponse)
	checkErr(err)
}

func sendAnswerCallbackQuery(responseWriter http.ResponseWriter, response AnswerCallbackQuery) {
	response.Method = "answerCallbackQuery"
	jsonResponse := response.ToJson()
	log.WithFields(log.Fields{"response": response}).Info()
	_, err := io.WriteString(responseWriter, jsonResponse)
	checkErr(err)
}
