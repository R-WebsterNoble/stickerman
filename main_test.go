package main

import (
	"database/sql"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
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
	err := db.Close()
	checkErr(err)
	defer func() { checkErr(adminDb.Close()) }()

	_, err = adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)
}

func setupStickerKeywords(stickerFileId string, keywords ...string) (groupId int64) {
	transaction, err := db.Begin()
	defer func() {
		err = transaction.Rollback()
		if err != nil && err != sql.ErrTxDone {
			panic(err)
		}
	}()
	checkErr(err)

	insertStickersQuery := `
INSERT INTO stickers (file_id) VALUES ($1)
ON CONFLICT (file_id)
  DO UPDATE set file_id = excluded.file_id
RETURNING id;`
	insertStickersStatement, err := transaction.Prepare(insertStickersQuery)
	checkErr(err)
	defer func() { checkErr(insertStickersStatement.Close()) }()

	insertKeywordsQuery := `
INSERT INTO keywords (keyword) VALUES ($1)
ON CONFLICT (keyword)
  DO UPDATE set keyword = excluded.keyword
RETURNING id;`
	insertKeywordsStatement, err := transaction.Prepare(insertKeywordsQuery)
	checkErr(err)
	defer func() { checkErr(insertKeywordsStatement.Close()) }()

	insertSessionQuery := `
          WITH inserted AS (
            INSERT INTO groups DEFAULT VALUES RETURNING id
          )
          INSERT INTO sessions (chat_id, file_id, group_id) SELECT
                                                              $1,
                                                              $2,
                                                              inserted.id
                                                            from inserted
          ON CONFLICT (chat_id)
            DO UPDATE SET chat_id = excluded.chat_id
          returning group_id;`
	insertSessionStatement, err := transaction.Prepare(insertSessionQuery)
	checkErr(err)
	defer func() { checkErr(insertSessionStatement.Close()) }()

	insertStickersKeywordsQuery := `
INSERT INTO sticker_keywords (sticker_id, keyword_id, group_id) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING id;`
	insertStickersKeywordsStatement, err := transaction.Prepare(insertStickersKeywordsQuery)
	checkErr(err)
	defer func() { checkErr(insertStickersKeywordsStatement.Close()) }()

	var stickerId int64
	err = insertStickersStatement.QueryRow(stickerFileId).Scan(&stickerId)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	err = insertStickersStatement.Close()
	checkErr(err)

	err = insertSessionStatement.QueryRow(12345, stickerFileId).Scan(&groupId)

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
		err := insertStickersKeywordsStatement.QueryRow(stickerId, keywordId, groupId).Scan(&thisStickerId)
		//if err != sql.ErrNoRows {
		checkErr(err)
		//}
		stickerIds = append(stickerIds, thisStickerId)
	}

	err = transaction.Commit()
	checkErr(err)

	return
}

func setupUserState(stickerFileId string, userMode string) {
	query := `
          WITH inserted AS (
            INSERT INTO groups DEFAULT VALUES RETURNING id
          )
          INSERT INTO sessions (chat_id, file_id, mode, group_id) SELECT
                                                              $1,
                                                              $2,
															  $3,
                                                              inserted.id
                                                            from inserted
          ON CONFLICT (chat_id)
            DO UPDATE SET chat_id = excluded.chat_id
          returning group_id;`
	_, err := db.Exec(query, 12345, stickerFileId, userMode)
	checkErr(err)
}

func cleanUpDb() {
	keywordsCleanupQuery := `DELETE FROM keywords WHERE keyword ILIKE 'keyword%'`
	_, err := db.Exec(keywordsCleanupQuery)
	checkErr(err)

	stickersCleanupQuery := `DELETE FROM stickers WHERE file_id ILIKE 'StickerFileId%'`
	_, err = db.Exec(stickersCleanupQuery)
	checkErr(err)

	groupsCleanupQuery := `DELETE FROM groups g USING sessions s WHERE s.group_id = g.id and s.chat_id in (0, 12345)`
	_, err = db.Exec(groupsCleanupQuery)
	checkErr(err)

	orphanedGroupsCleanupQuery := `  
      DELETE from groups g
      where not exists
      (select 1
       from sessions s
       where s.group_id = g.id
      );`
	_, err = db.Exec(orphanedGroupsCleanupQuery)
	checkErr(err)

	userStateCleanupQuery := `DELETE FROM sessions WHERE chat_id IN ( 0, 12345)`
	_, err = db.Exec(userStateCleanupQuery)
	checkErr(err)
}

func TestHandler_HandlesUnknownMessage(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211654,"edited_message":{"message_id":64,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524691085,"edit_date":1524693406,"text":"hig"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	assert.Equal(t, "unable to process request: neither message nor update found", response.Body)
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
		"Hi, I'm Sticker Manager Bot.\\n" +
		"I'll help you manage your stickers by letting you tag them so you can easily find them later.\\n" +
		"\\n" +
		"Usage:\\n" +
		"To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\\n" +
		"\\n" +
		"You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for.\"}"
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesSticker(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"A","set_name":"SetName","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"That's a nice sticker. Send me some tags and I'll add them to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReply(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithExistingKeyword(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"Keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 0 tags."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerReplyWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword1 keyword2"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 2 tags."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithResult(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithSQLI(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword'''")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"` + "keyword'''" + `","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleResults(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword")
	setupStickerKeywords("StickerFileId2", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword1", "keyword2")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword1 Keyword2","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword1", "keyword2", "keyword3")
	setupStickerKeywords("StickerFileId2", "keyword1", "keyword0")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword1 keyword2 keyword3","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryMatchesAllKeywordsWithCompletion(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword-completed")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInlineQueryEscapesWildcards(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", `keyword-per\_\%\\x`)
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword-Per\\_\\%\\\\x","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
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
	groupId, resultMode := SetUserStickerAndGetMode(12345, "StickerFileId")

	assert.Equal(t, "add", resultMode)
	assert.NotNil(t, groupId)
}

func TestHandler_SetUserState_SetsUserStateWithExistingState(t *testing.T) {
	defer cleanUpDb()
	SetUserStickerAndGetMode(12345, "StickerFileId1")
	SetUserStickerAndGetMode(12345, "StickerFileId2")

	var resultStickerId string
	var resultMode string
	query := `
        SELECT
          file_id,
          mode
        FROM sessions
        WHERE chat_id = 12345`

	err := db.QueryRow(query).Scan(&resultStickerId, &resultMode)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	assert.Equal(t, "StickerFileId2", resultStickerId)
	assert.Equal(t, "add", resultMode)
}

func TestHandler_HandlesKeywordMessageWithNoState(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":0,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":0,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":0,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordState(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesKeywordMessageWithRemove(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "remove")
	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You have deleted 0 tags."}`
	assert.Equal(t, expected, response.Body)
}

func TestGetAllKeywordsForStickerFileId(t *testing.T) {
	defer cleanUpDb()
	groupId := setupStickerKeywords("StickerFileId", "keyword1", "keyword2")

	result := GetAllKeywordsForStickerFileId("StickerFileId", groupId)

	sort.Strings(result)
	assert.Equal(t, []string{"keyword1", "keyword2"}, result)
}

func TestGetUserGroup_GetsANewGroup(t *testing.T) {
	defer cleanUpDb()

	result := getOrCreateUserGroup(12345)

	assert.NotEqual(t, 0, result)
}

func TestGetUserGroup_GetsAnExistingGroup(t *testing.T) {
	defer cleanUpDb()
	groupId := getOrCreateUserGroup(12345)

	result := getOrCreateUserGroup(12345)

	assert.Equal(t, groupId, result)
}

func TestHandler_HandlesInlineQueryDoesNotGetStickersFromOtherGroup(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":0,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesAddingKeywordToStickerFromSession(t *testing.T) {
	defer cleanUpDb()
	stickerRequest := events.APIGatewayProxyRequest{Body: `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`}
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"keyword"}}`}
	_, err := Handler(stickerRequest)

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesRemovingKeywordToStickerFromSession(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	setRemoveRequest := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/remove"}}`}
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"keyword"}}`}
	_, err := Handler(setRemoveRequest)

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You have deleted 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesAddCommand(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add"}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Okay, send me some tags and I'll add them to the sticker."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesAddCommandWithKeywordButNoSession(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add keyword"}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesUserWithNoStickerInSession(t *testing.T) {
	defer cleanUpDb()

	query := `
          WITH inserted AS (
            INSERT INTO groups DEFAULT VALUES RETURNING id
          )
          INSERT INTO sessions (chat_id, group_id) SELECT
                                                              12345,                                                            
                                                              inserted.id
                                                            from inserted
          ON CONFLICT (chat_id)
            DO UPDATE SET chat_id = excluded.chat_id
          returning group_id;`
	_, err := db.Exec(query)
	checkErr(err)

	usersStickerId, usersMode := GetUserState(12345)

	assert.Equal(t, "", usersStickerId)
	assert.Equal(t, "add", usersMode)
}

func TestHandler_HandlesDoesNotAddKeywordsWhenUserHasNoStickerIdInSession(t *testing.T) {
	defer cleanUpDb()

	query := `
         WITH inserted AS (
           INSERT INTO groups DEFAULT VALUES RETURNING id
         )
         INSERT INTO sessions (chat_id, group_id) SELECT
                                                             12345,
                                                             inserted.id
                                                           from inserted
         ON CONFLICT (chat_id)
           DO UPDATE SET chat_id = excluded.chat_id
         returning group_id;`
	_, err := db.Exec(query)
	checkErr(err)

	request := events.APIGatewayProxyRequest{Body: `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`}

	response, err := Handler(request)
	checkErr(err)

	expected := `{"method":"sendMessage","chat_id":12345,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesAddCommandWithKeyword(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add keyword"}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You are now in add mode.\nAdded 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesAddCommandWithRemove(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	setupStickerKeywords("StickerFileId", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/remove keyword"}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You have deleted 1 tag."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesInvalidCommand(t *testing.T) {
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/blah"}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"I don't recognise this command."}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_InlineQueryResultsAreMostRecentFirst(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword")
	setupStickerKeywords("StickerFileId2", "keyword")
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)

	cleanUpDb()

	setupStickerKeywords("StickerFileId2", "keyword")
	setupStickerKeywords("StickerFileId1", "keyword")
	request = events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err = Handler(request)

	assert.IsType(t, err, nil)
	expected = `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId2"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)

}

func TestHandler_HandlesInlineQueryWithPagination(t *testing.T) {
	defer cleanUpDb()
	for i := 0; i < 51; i++ {
		setupStickerKeywords("StickerFileId"+strconv.Itoa(i), "keyword")
	}

	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`}

	response, err := Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId50"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId49"},{"type":"sticker","id":"2","sticker_file_id":"StickerFileId48"},{"type":"sticker","id":"3","sticker_file_id":"StickerFileId47"},{"type":"sticker","id":"4","sticker_file_id":"StickerFileId46"},{"type":"sticker","id":"5","sticker_file_id":"StickerFileId45"},{"type":"sticker","id":"6","sticker_file_id":"StickerFileId44"},{"type":"sticker","id":"7","sticker_file_id":"StickerFileId43"},{"type":"sticker","id":"8","sticker_file_id":"StickerFileId42"},{"type":"sticker","id":"9","sticker_file_id":"StickerFileId41"},{"type":"sticker","id":"10","sticker_file_id":"StickerFileId40"},{"type":"sticker","id":"11","sticker_file_id":"StickerFileId39"},{"type":"sticker","id":"12","sticker_file_id":"StickerFileId38"},{"type":"sticker","id":"13","sticker_file_id":"StickerFileId37"},{"type":"sticker","id":"14","sticker_file_id":"StickerFileId36"},{"type":"sticker","id":"15","sticker_file_id":"StickerFileId35"},{"type":"sticker","id":"16","sticker_file_id":"StickerFileId34"},{"type":"sticker","id":"17","sticker_file_id":"StickerFileId33"},{"type":"sticker","id":"18","sticker_file_id":"StickerFileId32"},{"type":"sticker","id":"19","sticker_file_id":"StickerFileId31"},{"type":"sticker","id":"20","sticker_file_id":"StickerFileId30"},{"type":"sticker","id":"21","sticker_file_id":"StickerFileId29"},{"type":"sticker","id":"22","sticker_file_id":"StickerFileId28"},{"type":"sticker","id":"23","sticker_file_id":"StickerFileId27"},{"type":"sticker","id":"24","sticker_file_id":"StickerFileId26"},{"type":"sticker","id":"25","sticker_file_id":"StickerFileId25"},{"type":"sticker","id":"26","sticker_file_id":"StickerFileId24"},{"type":"sticker","id":"27","sticker_file_id":"StickerFileId23"},{"type":"sticker","id":"28","sticker_file_id":"StickerFileId22"},{"type":"sticker","id":"29","sticker_file_id":"StickerFileId21"},{"type":"sticker","id":"30","sticker_file_id":"StickerFileId20"},{"type":"sticker","id":"31","sticker_file_id":"StickerFileId19"},{"type":"sticker","id":"32","sticker_file_id":"StickerFileId18"},{"type":"sticker","id":"33","sticker_file_id":"StickerFileId17"},{"type":"sticker","id":"34","sticker_file_id":"StickerFileId16"},{"type":"sticker","id":"35","sticker_file_id":"StickerFileId15"},{"type":"sticker","id":"36","sticker_file_id":"StickerFileId14"},{"type":"sticker","id":"37","sticker_file_id":"StickerFileId13"},{"type":"sticker","id":"38","sticker_file_id":"StickerFileId12"},{"type":"sticker","id":"39","sticker_file_id":"StickerFileId11"},{"type":"sticker","id":"40","sticker_file_id":"StickerFileId10"},{"type":"sticker","id":"41","sticker_file_id":"StickerFileId9"},{"type":"sticker","id":"42","sticker_file_id":"StickerFileId8"},{"type":"sticker","id":"43","sticker_file_id":"StickerFileId7"},{"type":"sticker","id":"44","sticker_file_id":"StickerFileId6"},{"type":"sticker","id":"45","sticker_file_id":"StickerFileId5"},{"type":"sticker","id":"46","sticker_file_id":"StickerFileId4"},{"type":"sticker","id":"47","sticker_file_id":"StickerFileId3"},{"type":"sticker","id":"48","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"49","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":"50"}`
	assert.Equal(t, expected, response.Body)

	request = events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":"50"}}`}

	response, err = Handler(request)

	assert.IsType(t, err, nil)
	expected = `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId0"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}

func TestHandler_HandlesStickerAndSetsDefaultTags(t *testing.T) {
	defer cleanUpDb()
	request := events.APIGatewayProxyRequest{Body: `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"A","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`}

	response, err := Handler(request)
	assert.IsType(t, err, nil)

	request = events.APIGatewayProxyRequest{Body: `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"VaultBoySet Fallout-Vault-Boy Fallout Vault Boy 😂","offset":""}}`}
	
	response, err = Handler(request)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"CAADAQADrwgAAr-MkARNRpJexr9oegI"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, response.Body)
}