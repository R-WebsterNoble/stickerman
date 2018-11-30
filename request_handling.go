package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"log"
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
		groupId := getOrCreateUserGroup(int64(update.InlineQuery.From.ID))
		var offset int
		if update.InlineQuery.Offset != "" {
			var err error
			offset, err = strconv.Atoi(update.InlineQuery.Offset)
			checkErr(err)
		}
		inlineQueryResults, nextOffset := GetAllStickerIdsForKeywords(update.InlineQuery.Query, groupId, offset)
		return inlineQueryResponse(update.InlineQuery.ID, inlineQueryResults, nextOffset)
	}

	errorMessage := "unable to process request: neither message nor update found"
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
	CacheTime     int                              `json:"cache_time"`
	IsPersonal    bool                             `json:"is_personal"`
	NextOffset    string                           `json:"next_offset"`
}

func inlineQueryResponse(inlineQueryID string, queryResultStickerIds []string, nextOffset int) events.APIGatewayProxyResponse {
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

	response := AnswerCallbackQuery{
		"answerInlineQuery",
		inlineQueryID,
		queryResultStickers[:],
		0,
		true,
		nextOffsetString,
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

func textMessageResponse(chatId int64, text string) events.APIGatewayProxyResponse {
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
