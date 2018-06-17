package main

import (
	"testing"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"os"
	"database/sql"
)

func TestMain(m *testing.M) {
	defer Shutdown()

	testDbName := "stampstest"
	adminDb := setupTestDB(testDbName)
	defer tearDownDB(adminDb, testDbName)

	exitCode := m.Run()

	tearDownDB(adminDb, testDbName)
	os.Exit(exitCode)
}

func setupTestDB(dbName string) (adminDb *sql.DB) {
	connStr := os.Getenv("pgAdminDBConnectionString")

	adminDb, err := sql.Open("postgres", connStr)
	checkErr(err)

	_, err = adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)

	_, err = adminDb.Exec("CREATE DATABASE " + dbName)
	checkErr(err)

	SetupDB("schema.sql")

	return adminDb
}

func tearDownDB(adminDb *sql.DB, dbName string) {
	CloseDb()
	defer adminDb.Close()

	_, err := adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)
}


func TestHandler_HandlesUnknownMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211654,\"edited_message\":{\"message_id\":64,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1524691085,\"edit_date\":1524693406,\"text\":\"hig\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "unable to process request: neither message or update found", response.Body)
}

func TestHandler_HandlesInvalidJson(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "!"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "error while Unmarshaling", response.Body)
}

func TestHandler_HandlesMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211650,\"message\":{\"message_id\":65,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1524692383,\"text\":\"/start\",\"entities\":[{\"offset\":0,\"length\":6,\"type\":\"bot_command\"}]}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"This Bot is designed to help you find stickers.\\n\\nUsage:\\nTo search for a stickers in any chat type: @DevStampsBot followed by your search keywords.\\n\\nTo add new sticker and keywords to the bot, first send the sticker to this chat.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesSticker(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211708,\"message\":{\"message_id\":315,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1524775382,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"That's a nice sticker. Send me some keywords and I'll add them to it.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReply(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":359,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1525458701,\"reply_to_message\":{\"message_id\":321,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1524777329,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}},\"text\":\"keyword\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"Added 0 keyword(s).\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithMultipleKeywords(t *testing.T) {
	t.SkipNow() // need to setup test db fixtures for this to work more than once
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":359,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1525458701,\"reply_to_message\":{\"message_id\":321,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1524777329,\"sticker\":{\"width\":512,\"height\":512,\"emoji\":\"👉\",\"set_name\":\"Feroxdoon2\",\"thumb\":{\"file_id\":\"AAQBABOqNQMwAAQ78UrarWIt0iRYAAIC\",\"file_size\":4670,\"width\":128,\"height\":128},\"file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\",\"file_size\":24458}},\"text\":\"keyword1 keyword2\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"Added 2 keyword(s).\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQuery(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"CAADAgADAQMAApzW5woyIbXtrGvnsAI\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgADAQMAApzW5woyIbXtrGvnsAI\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithSQLI(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\" '''\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAQADKgAD5_bHDFnhkQhE_myDAg\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleResults(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"sB0Umf2MBsk\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgADFQAD2EMzEnvdd9kfrCGwAg\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAgADFgAD2EMzEqA6t2tUdswBAg\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleKeywords(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"degWs89raGY vjmvodk8LG8\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAwADWwEAAm9iOwdJbHljxEZDHgI\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAwADgwEAAm9iOweRXewEFMcJ2gI\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywords(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"guhYMplLkEtJ3Q l27rGJHCmOqomg\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgAD8wIAApzW5wrgLgRxhQ_BAgI\"},{\"type\":\"sticker\",\"id\":\"1\",\"sticker_file_id\":\"CAADAgADMAIAAs-71A59r1FSPKQrowI\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywordsWithCompletion(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"CAADAgADAQMAApzW5woyIbX\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgADAQMAApzW5woyIbXtrGvnsAI\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryEscapesWildcards(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"update_id\":457211742,\"inline_query\":{\"id\":\"913797545109391540\",\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"user\",\"username\":\"user\",\"language_code\":\"en-GB\"},\"query\":\"Per\\\\_\\\\%\\\\\\\\x\",\"offset\":\"\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"answerInlineQuery\",\"inline_query_id\":\"913797545109391540\",\"results\":[{\"type\":\"sticker\",\"id\":\"0\",\"sticker_file_id\":\"CAADAgADCwMAApzW5wrRyWd0_dJz5QI\"}]}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_GetUserState_GetsUserState(t *testing.T) {
	resultStickerId, resultMode := GetUserState(1)

	assert.Equal(t, "test", resultStickerId)
	assert.Equal(t, "add", resultMode)
}

func TestHandler_GetUserState_GetsInvalidUserState(t *testing.T) {
	resultStickerId, resultMode := GetUserState(-1)

	assert.Equal(t, "", resultStickerId)
	assert.Equal(t, "", resultMode)
}

func TestHandler_SetUserState_SetsUserState(t *testing.T) {

	resultMode := SetUserStickerAndGetMode(1, "test")

	assert.Equal(t, "add", resultMode)
}

func TestHandler_HandlesKeywordMessageWithNoState(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":900,\"from\":{\"id\":0,\"is_bot\":false,\"first_name\":\"blah\",\"username\":\"blah\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":0,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1527633135,\"text\":\"hi\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":0,\"text\":\"Send a sticker to me then I'll be able to add searchable keywords to it.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordState(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":900,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"blah\",\"username\":\"blah\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1527633135,\"text\":\"hi\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"Added 0 keyword(s).\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordMessageWithRemove(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: "{\"message\":{\"message_id\":900,\"from\":{\"id\":12345,\"is_bot\":false,\"first_name\":\"blah\",\"username\":\"blah\",\"language_code\":\"en-GB\"},\"chat\":{\"id\":12345,\"first_name\":\"user\",\"username\":\"user\",\"type\":\"private\"},\"date\":1527633135,\"text\":\"hi\"}}"}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":0,\"text\":\"Send a sticker to me then I'll be able to add searchable keywords to it.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestGetAllKeywordsForStickerFileId(t *testing.T) {
	result := GetAllKeywordsForStickerFileId("CAADAgAD8wIAApzW5wrgLgRxhQ_BAgI")

	assert.Equal(t, []string{"vader", "thumbs-up", "l27rgjhcmoqomg", "guhympllketj3q"}, result)
}