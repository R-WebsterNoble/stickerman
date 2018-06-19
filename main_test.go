package main

import (
	"testing"
	"os"
	"database/sql"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"sort"
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	testDbName := "stampstest"

	dbConStr := os.Getenv("pgDBConnectionString")
	if strings.HasPrefix(dbConStr, "host=localhost") {
		adminDb := setupTestDB(testDbName)
		defer tearDownDB(adminDb, testDbName)
	}

	return m.Run()
}

func setupTestDB(dbName string) (adminDb *sql.DB) {
	adminConnStr := os.Getenv("pgAdminDBConnectionString")

	adminDb, err := sql.Open("postgres", adminConnStr)
	checkErr(err)

	_, err = adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)

	_, err = adminDb.Exec("CREATE DATABASE " + dbName)
	checkErr(err)

	schema, err := ioutil.ReadFile("schema.sql")
	checkErr(err)

	_, err = db.Exec(string(schema))
	checkErr(err)

	return adminDb
}

func tearDownDB(adminDb *sql.DB, dbName string) {
	db.Close()
	defer adminDb.Close()

	_, err := adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)
}

func setupStickerKeywords(stickerFileId string, keywords ...string) {
	transaction, err := db.Begin()
	defer func() {
		err = transaction.Rollback()
		if err != nil && err != sql.ErrTxDone {
			panic(err)
		}
	}()
	checkErr(err)

	query := `
INSERT INTO stickers (file_id) VALUES ($1)
ON CONFLICT (file_id)
  DO UPDATE set file_id = excluded.file_id
RETURNING id;`
	insertStickersStatement, err := transaction.Prepare(query)
	defer insertStickersStatement.Close()
	checkErr(err)

	query1 := `
INSERT INTO keywords (keyword) VALUES ($1)
ON CONFLICT (keyword)
  DO UPDATE set keyword = excluded.keyword
RETURNING id;`
	insertKeywordsStatement, err := transaction.Prepare(query1)
	defer insertKeywordsStatement.Close()
	checkErr(err)

	query2 := `
INSERT INTO sticker_keywords (sticker_id, keyword_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING id;`
	insertStickersKeywordsStatement, err := transaction.Prepare(query2)
	defer insertStickersKeywordsStatement.Close()
	checkErr(err)

	var stickerId int64
	err = insertStickersStatement.QueryRow(stickerFileId).Scan(&stickerId)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	err = insertStickersStatement.Close()
	checkErr(err)

	var keywordIds [] int64
	var stickerIds [] int64
	for _, keyword := range keywords {
		var keywordId int64
		err = insertKeywordsStatement.QueryRow(keyword).Scan(&keywordId)
		if err != sql.ErrNoRows {
			checkErr(err)
		}
		keywordIds = append(keywordIds, keywordId)
		var thisStickerId int64
		err := insertStickersKeywordsStatement.QueryRow(stickerId, keywordId).Scan(&thisStickerId)
		//if err != sql.ErrNoRows {
		checkErr(err)
		//}
		stickerIds = append(stickerIds, thisStickerId)
	}

	err = transaction.Commit()
	checkErr(err)
}

func setupUserState(stickerFileId string, userMode string) {
	query := `INSERT INTO sessions (chat_id, file_id, mode) VALUES (12345, $1, $2)`
	_, err := db.Exec(query, stickerFileId, userMode)
	checkErr(err)
}

func cleanUpDb() {
	keywordsCleanupQuery := `DELETE FROM keywords WHERE keyword ILIKE 'keyword%'`
	_, err := db.Exec(keywordsCleanupQuery)
	checkErr(err)

	stickersCleanupQuery := `DELETE FROM stickers WHERE file_id ILIKE 'StickerFileId%'`
	_, err = db.Exec(stickersCleanupQuery)
	checkErr(err)

	userStateCleanupQuery := `DELETE FROM sessions WHERE chat_id = 12345`
	_, err = db.Exec(userStateCleanupQuery)
	checkErr(err)
}

func TestHandler_HandlesUnknownMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211654,"edited_message":{"message_id":64,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524691085,"edit_date":1524693406,"text":"hig"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "unable to process request: neither message or update found", response.Body)
}

func TestHandler_HandlesInvalidJson(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: `!`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "error while Unmarshaling", response.Body)
}

func TestHandler_HandlesMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/start","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"` +
		"This Bot is designed to help you find stickers.\\n" +
		"\\n" +
		"Usage:\\n" +
		"To search for a stickers in any chat type: @DevStampsBot followed by your search keywords.\\n" +
		"\\n" +
		"To add new sticker and keywords to the bot, first send the sticker to this chat." + `"}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesSticker(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"Feroxdoon2","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"That's a nice sticker. Send me some keywords and I'll add them to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReply(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"Feroxdoon2","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 keyword(s)."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithExistingKeyword(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"Feroxdoon2","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"Keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 0 keyword(s)."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"Feroxdoon2","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword1 keyword2"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 2 keyword(s)."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithResult(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithSQLI(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "'''")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"` + "'''" + `","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleResults(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword")
	setupStickerKeywords("StickerFileId2", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId2"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword1", "keyword2")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword1 Keyword2","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword1", "keyword2")
	setupStickerKeywords("StickerFileId2", "keyword1", "keyword0")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword1 keyword2","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywordsWithCompletion(t *testing.T) {
	t.Skip("skipping test: Completion is broken")

	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword-completed")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryEscapesWildcards(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", `keyword-per\_\%\\x`)
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword-Per\\_\\%\\\\x","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}]}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_GetUserState_GetsUserState(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")

	resultStickerId, resultMode := GetUserState(12345)

	assert.Equal(t, "StickerFileId", resultStickerId)
	assert.Equal(t, "add", resultMode)
}

func TestHandler_GetUserState_GetsInvalidUserState(t *testing.T) {
	defer cleanUpDb()
	resultStickerId, resultMode := GetUserState(-1)

	assert.Equal(t, "", resultStickerId)
	assert.Equal(t, "", resultMode)
}

func TestHandler_SetUserState_SetsUserState(t *testing.T) {
	defer cleanUpDb()
	resultMode := SetUserStickerAndGetMode(1, "test")

	assert.Equal(t, "add", resultMode)
}

func TestHandler_HandlesKeywordMessageWithNoState(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":0,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":0,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"hi"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":0,"text":"Send a sticker to me then I'll be able to add searchable keywords to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordState(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"hi"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 keyword(s)."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordMessageWithRemove(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "remove")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"hi"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You have deleted 0 keywords."}`
	assert.Equal(t, expected, response.Body)
}

func TestGetAllKeywordsForStickerFileId(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword1", "keyword2")

	result := GetAllKeywordsForStickerFileId("StickerFileId")

	sort.Strings(result)
	assert.Equal(t, []string{"keyword1", "keyword2"}, result)
}
