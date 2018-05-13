package main_test

import (
	"testing"
	"stickerman"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler_HandlesUnknownMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211654,\"edited_message\":{\"message_id\":64,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524691085,\"edit_date\":1524693406,\"text\":\"hig\"}}"}


	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "unable to process request: neither message or update found", response.Body)
}

func TestHandler_HandlesInvalidJson(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "!"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "error while Unmarshaling", response.Body)
}

func TestHandler_HandlesMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211650,\"message\":{\"message_id\":65,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524692383,\"text\":\"/start\",\"entities\":[{\"offset\":0,\"length\":6,\"type\":\"bot_command\"}]}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":212760070,\"text\":\"This Bot is designed to help you find Stickers.\\n\\nUsage:\\nTo search for Stickers in any chat type: @DevStampsBot followed by your search keywords.\\n\\nTo add new Stickers and keywords to the bot, send the sticker to this chat then reply to the sticker with a message containing the keywords you want to add.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesSticker(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211708,\"message\":{\"message_id\":315,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524775382,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":212760070,\"text\":\"You sent a 👉 sticker!\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReply(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":359,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1525458701,\"reply_to_message\":{\"message_id\":321,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524777329,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}},\"text\":\"keyword\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":212760070,\"text\":\"Added 0 keyword(s).\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithMultipleKeywords(t *testing.T) {
	t.SkipNow() // need to setup test db fixtures for this to work more than once
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":359,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1525458701,\"reply_to_message\":{\"message_id\":321,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524777329,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}},\"text\":\"keyword1 keyword2\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":212760070,\"text\":\"Added 2 keyword(s).\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithSQLI(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\" '''\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleResults(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"sB0Umf2MBsk\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgADFQAD2EMzEnvdd9kfrCGwAg\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAgADFgAD2EMzEqA6t2tUdswBAg\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleKeywords(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"degWs89raGY vjmvodk8LG8\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAwADWwEAAm9iOwdJbHljxEZDHgI\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAwADgwEAAm9iOweRXewEFMcJ2gI\"}]}"
	assert.Equal(t, expected, response.Body)
}
