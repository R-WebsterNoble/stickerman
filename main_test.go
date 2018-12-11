package main

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	currentlyTesting = true
	run := m.Run()
	return run
}

func setupHttpHandler(t *testing.T, body string) (*http.Request, error, *httptest.ResponseRecorder, http.HandlerFunc) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health-check", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler)
	return req, err, rr, handler
}

func setupTestDB(dbName string) (adminDb *sql.DB) {
	adminConnStr := os.Getenv("pgAdminDBConnectionString")

	adminDb, err := sql.Open("postgres", adminConnStr+"postgres")
	checkErr(err)

	_, err = adminDb.Exec("DROP DATABASE IF EXISTS " + dbName)
	checkErr(err)

	_, err = adminDb.Exec("CREATE DATABASE " + dbName)
	checkErr(err)

	newDb, err := sql.Open("postgres", adminConnStr+dbName)
	checkErr(err)
	defer func() { checkErr(newDb.Close()) }()

	schema, err := ioutil.ReadFile("schema.sql")
	checkErr(err)

	_, err = newDb.Exec("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO stickerman;")
	checkErr(err)

	_, err = newDb.Exec("GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO stickerman;")
	checkErr(err)

	_, err = newDb.Exec("ALTER DATABASE " + dbName + " OWNER TO stickerman;")
	checkErr(err)

	_, err = newDb.Exec(string(schema))
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

	groupId, _ = SetUserStickerAndGetMode(12345, stickerFileId)

	addKeywordsArrayToSticker(stickerFileId, keywords, groupId)

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
	testWaitGroup.Wait()
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
	requestBody := `{"update_id":457211654,"edited_message":{"message_id":64,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524691085,"edit_date":1524693406,"text":"hig"}}`

	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(responseRecorder, req)

	//// Check the status code is what we expect.
	//if status := responseRecorder.Code; status != http.StatusOK {
	//	t.Errorf("handler returned wrong status code: got %v want %v",
	//		status, http.StatusOK)
	//}

	//// Check the response body is what we expect.
	//expected := `{"alive": true}`
	//if responseRecorder.Body.String() != expected {
	//	t.Errorf("handler returned unexpected body: got %v want %v",
	//		responseRecorder.Body.String(), expected)
	//}

	//request := events.APIGatewayProxyRequest{Body: `{"update_id":457211654,"edited_message":{"message_id":64,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524691085,"edit_date":1524693406,"text":"hig"}}`}

	assert.IsType(t, err, nil)
	assert.Equal(t, "unable to process request: neither message nor update found\n", responseRecorder.Body.String())
}

func TestHandler_HandlesInvalidJson(t *testing.T) {
	requestBody := `!`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := "error while Unmarshaling\n"
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesMessage(t *testing.T) {
	requestBody := `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/start","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"` +
		"Hi, I'm Sticker Manager Bot.\\n" +
		"I'll help you manage your stickers by letting you tag them so you can easily find them later.\\n" +
		"\\n" +
		"Usage:\\n" +
		"To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\\n" +
		"\\n" +
		"You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for.\"}"
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesSticker(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"😀","set_name":"SetName","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"That's a nice sticker. Send me some tags and I'll add them to it.\n\nI'll also setup some default tags for every sticker in the pack for you."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesStickerReply(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesStickerReplyWithExistingKeyword(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	requestBody := `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"Keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"No tags to add"}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesStickerReplyWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"message":{"message_id":359,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1525458701,"reply_to_message":{"message_id":321,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524777329,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}},"text":"keyword1 keyword2"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 2 tags."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesEmptyInlineQuery(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryWithResult(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryWithSQLI(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword'''")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"` + "keyword'''" + `","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryWithMultipleResults(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword")
	setupStickerKeywords("StickerFileId2", "keyword")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryWithMultipleKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword1", "keyword2")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword1 Keyword2","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryMatchesAllKeywords(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword1", "keyword2", "keyword3")
	setupStickerKeywords("StickerFileId2", "keyword1", "keyword0")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword1 keyword2 keyword3","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryMatchesAllKeywordsWithCompletion(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword-completed")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInlineQueryEscapesWildcards(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", `keyword-per\_\%\\x`)
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword-Per\\_\\%\\\\x","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
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
	requestBody := `{"message":{"message_id":900,"from":{"id":0,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":1,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":1,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesKeywordState(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	requestBody := `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesKeywordMessageWithRemove(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "remove")
	requestBody := `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"No tags to Remove"}`
	assert.Equal(t, expected, responseRecorder.Body.String())
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
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":0,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesAddingKeywordToStickerFromSession(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"👉","set_name":"","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)
	handler.ServeHTTP(responseRecorder, req)
	requestBody = `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"keyword"}}`

	req, err, responseRecorder, handler = setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Added 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesRemovingKeywordFromStickerUsingSession(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId", "keyword")
	setRemoveRequestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/remove"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, setRemoveRequestBody)
	handler.ServeHTTP(responseRecorder, req)
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"keyword"}}`
	req, err, responseRecorder, handler = setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Removed 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesAddCommand(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Okay, send me some tags and I'll add them to the sticker."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesAddCommandWithKeywordButNoSession(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
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

	requestBody := `{"message":{"message_id":900,"from":{"id":12345,"is_bot":false,"first_name":"blah","username":"blah","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1527633135,"text":"keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	checkErr(err)

	expected := `{"method":"sendMessage","chat_id":12345,"text":"Send a sticker to me then I'll be able to add tags to it."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesAddCommandWithKeyword(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/add keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You are now in add mode.\n\nAdded 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesAddCommandWithRemove(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	setupStickerKeywords("StickerFileId", "keyword")
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/remove keyword"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"Removed 1 tag."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesInvalidCommand(t *testing.T) {
	requestBody := `{"update_id":457214899,"message":{"message_id":3682,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1530123765,"text":"/blah"}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"I don't recognise this command."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_InlineQueryResultsAreMostRecentFirst(t *testing.T) {
	defer cleanUpDb()
	setupStickerKeywords("StickerFileId1", "keyword")
	setupStickerKeywords("StickerFileId2", "keyword")
	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())

	cleanUpDb()

	setupStickerKeywords("StickerFileId2", "keyword")
	setupStickerKeywords("StickerFileId1", "keyword")
	requestBody = `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler = setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected = `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId1"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId2"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())

}

func TestHandler_HandlesInlineQueryWithPagination(t *testing.T) {
	defer cleanUpDb()
	for i := 0; i < 51; i++ {
		setupStickerKeywords("StickerFileId"+strconv.Itoa(i), "keyword")
	}

	requestBody := `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":""}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId50"},{"type":"sticker","id":"1","sticker_file_id":"StickerFileId49"},{"type":"sticker","id":"2","sticker_file_id":"StickerFileId48"},{"type":"sticker","id":"3","sticker_file_id":"StickerFileId47"},{"type":"sticker","id":"4","sticker_file_id":"StickerFileId46"},{"type":"sticker","id":"5","sticker_file_id":"StickerFileId45"},{"type":"sticker","id":"6","sticker_file_id":"StickerFileId44"},{"type":"sticker","id":"7","sticker_file_id":"StickerFileId43"},{"type":"sticker","id":"8","sticker_file_id":"StickerFileId42"},{"type":"sticker","id":"9","sticker_file_id":"StickerFileId41"},{"type":"sticker","id":"10","sticker_file_id":"StickerFileId40"},{"type":"sticker","id":"11","sticker_file_id":"StickerFileId39"},{"type":"sticker","id":"12","sticker_file_id":"StickerFileId38"},{"type":"sticker","id":"13","sticker_file_id":"StickerFileId37"},{"type":"sticker","id":"14","sticker_file_id":"StickerFileId36"},{"type":"sticker","id":"15","sticker_file_id":"StickerFileId35"},{"type":"sticker","id":"16","sticker_file_id":"StickerFileId34"},{"type":"sticker","id":"17","sticker_file_id":"StickerFileId33"},{"type":"sticker","id":"18","sticker_file_id":"StickerFileId32"},{"type":"sticker","id":"19","sticker_file_id":"StickerFileId31"},{"type":"sticker","id":"20","sticker_file_id":"StickerFileId30"},{"type":"sticker","id":"21","sticker_file_id":"StickerFileId29"},{"type":"sticker","id":"22","sticker_file_id":"StickerFileId28"},{"type":"sticker","id":"23","sticker_file_id":"StickerFileId27"},{"type":"sticker","id":"24","sticker_file_id":"StickerFileId26"},{"type":"sticker","id":"25","sticker_file_id":"StickerFileId25"},{"type":"sticker","id":"26","sticker_file_id":"StickerFileId24"},{"type":"sticker","id":"27","sticker_file_id":"StickerFileId23"},{"type":"sticker","id":"28","sticker_file_id":"StickerFileId22"},{"type":"sticker","id":"29","sticker_file_id":"StickerFileId21"},{"type":"sticker","id":"30","sticker_file_id":"StickerFileId20"},{"type":"sticker","id":"31","sticker_file_id":"StickerFileId19"},{"type":"sticker","id":"32","sticker_file_id":"StickerFileId18"},{"type":"sticker","id":"33","sticker_file_id":"StickerFileId17"},{"type":"sticker","id":"34","sticker_file_id":"StickerFileId16"},{"type":"sticker","id":"35","sticker_file_id":"StickerFileId15"},{"type":"sticker","id":"36","sticker_file_id":"StickerFileId14"},{"type":"sticker","id":"37","sticker_file_id":"StickerFileId13"},{"type":"sticker","id":"38","sticker_file_id":"StickerFileId12"},{"type":"sticker","id":"39","sticker_file_id":"StickerFileId11"},{"type":"sticker","id":"40","sticker_file_id":"StickerFileId10"},{"type":"sticker","id":"41","sticker_file_id":"StickerFileId9"},{"type":"sticker","id":"42","sticker_file_id":"StickerFileId8"},{"type":"sticker","id":"43","sticker_file_id":"StickerFileId7"},{"type":"sticker","id":"44","sticker_file_id":"StickerFileId6"},{"type":"sticker","id":"45","sticker_file_id":"StickerFileId5"},{"type":"sticker","id":"46","sticker_file_id":"StickerFileId4"},{"type":"sticker","id":"47","sticker_file_id":"StickerFileId3"},{"type":"sticker","id":"48","sticker_file_id":"StickerFileId2"},{"type":"sticker","id":"49","sticker_file_id":"StickerFileId1"}],"cache_time":0,"is_personal":true,"next_offset":"50"}`
	assert.Equal(t, expected, responseRecorder.Body.String())

	requestBody = `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":"Keyword","offset":"50"}}`

	req, err, responseRecorder, handler = setupHttpHandler(t, requestBody)
	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected = `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"StickerFileId0"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_HandlesStickerAndSetsDefaultTags(t *testing.T) {
	defer cleanUpDb()
	requestBody := `{"update_id":457211708,"message":{"message_id":315,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524775382,"sticker":{"width":512,"height":512,"emoji":"😀","set_name":"VaultBoySet","thumb":{"file_id":"ThumbFileId","file_size":4670,"width":128,"height":128},"file_id":"StickerFileId","file_size":24458}}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)

	requestBody = `{"update_id":457211742,"inline_query":{"id":"913797545109391540","from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"query":" Fallout-Vault-Boy Fallout Vault Boy 😂","offset":""}}`
	req, err, responseRecorder, handler = setupHttpHandler(t, requestBody)
	handler.ServeHTTP(responseRecorder, req)

	assert.IsType(t, err, nil)
	expected := `{"method":"answerInlineQuery","inline_query_id":"913797545109391540","results":[{"type":"sticker","id":"0","sticker_file_id":"CAADAQADrwgAAr-MkARNRpJexr9oegI"}],"cache_time":0,"is_personal":true,"next_offset":""}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestGetUserGroup_GetsUserGroupHandlesNoUser(t *testing.T) {
	defer cleanUpDb()

	result := GetUserGroup(-1)

	assert.Empty(t, result)
}

func TestGetUserGroup_GetsUserGroup(t *testing.T) {
	defer cleanUpDb()
	getOrCreateUserGroup(12345)

	result := GetUserGroup(12345)

	assert.NotEmpty(t, result)
}

func TestAssignUserGroup_AssignesUserGroup(t *testing.T) {
	defer cleanUpDb()
	getOrCreateUserGroup(12345)
	getOrCreateUserGroup(0)
	newGroupUuid := GetUserGroup(0)

	assignUserToGroup(12345, newGroupUuid)

	swappedUsersGroup := GetUserGroup(12345)
	assert.Equal(t, swappedUsersGroup, newGroupUuid)
}

func TestHandler_AbleToGetGroup(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")
	requestBody := `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/Group","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	usersGroup := GetUserGroup(12345)
	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"Your group ID is \\\"" + usersGroup + "\\\".\\nOther users can join your group using\\n/JoinGroup " + usersGroup + "\"}"
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_AbleToJoinGroup(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")

	getOrCreateUserGroup(0)
	newGroupUuid := GetUserGroup(0)

	requestBody := `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/JoinGroup ` + newGroupUuid + `","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You have moved into the group."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_UnAbleToJoinGroupAlreadyIn(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")

	currentGroupUuid := GetUserGroup(12345)

	requestBody := `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/JoinGroup ` + currentGroupUuid + `","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := `{"method":"sendMessage","chat_id":12345,"text":"You are already in that group."}`
	assert.Equal(t, expected, responseRecorder.Body.String())
}

func TestHandler_UnAbleToJoinGroupinvalid(t *testing.T) {
	defer cleanUpDb()
	setupUserState("StickerFileId", "add")

	requestBody := `{"update_id":457211650,"message":{"message_id":65,"from":{"id":12345,"is_bot":false,"first_name":"user","username":"user","language_code":"en-GB"},"chat":{"id":12345,"first_name":"user","username":"user","type":"private"},"date":1524692383,"text":"/JoinGroup blah","entities":[{"offset":0,"length":6,"type":"bot_command"}]}}`
	req, err, responseRecorder, handler := setupHttpHandler(t, requestBody)

	handler.ServeHTTP(responseRecorder, req)
	assert.IsType(t, err, nil)
	expected := "{\"method\":\"sendMessage\",\"chat_id\":12345,\"text\":\"That Group Id is not in the correct format, I'm expecting something that looks like this:\\n/JoinGroup 123e4567-e89b-12d3-a456-426655440000.\"}"
	assert.Equal(t, expected, responseRecorder.Body.String())
}
