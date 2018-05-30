package main

import (
	"log"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"strconv"
)

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
		inlineQueryResults := getAllStickerIdsForKeywords(update.InlineQuery.Query)
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
