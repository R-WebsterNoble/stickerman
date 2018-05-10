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
	assert.Equal(t, "", response.Body)
}

func TestHandler_HandlesInvalidJson(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "!"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "", response.Body)
}

func TestHandler_HandlesMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211650,\"message\":{\"message_id\":65,\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":212760070,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"type\":\"private\"},\"date\":1524692383,\"text\":\"/start\",\"entities\":[{\"offset\":0,\"length\":6,\"type\":\"bot_command\"}]}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":212760070,\"text\":\"You said /start\"}"
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

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":212760070,\"is_bot\":false,\"first_name\":\"Didassi\",\"username\":\"Didassi\",\"language_code\":\"en-GB\"},\"query\":\"boop\",\"offset\":\"\"}}"}

	response, err := main.Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAQADHgADVwb9BrrlpxhL2ltZAg\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAQADjgsAAiPdEAaKEpsdVb8xOAI\"},{\"type\":\"sticker\",\"id\":\"2\",\"sticker_file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\"}]}"
	assert.Equal(t, expected, response.Body)
}
