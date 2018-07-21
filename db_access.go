package main

import (
	"strings"
	"database/sql"
	"strconv"
	"github.com/lib/pq"
	"github.com/adam-hanna/arrayOperations"
)

func GetAllStickerIdsForKeywords(keywordsString string, groupId int64) []string {
	keywordsString = EscapeSql(keywordsString)
	keywords := getKeywordsArray(keywordsString)

	if len(keywordsString) == 0 {
		return []string{}
	}

	query := `
SELECT array_agg(s.file_id)
FROM
  keywords k
  JOIN sticker_keywords sk ON sk.keyword_id = k.id
  JOIN stickers s ON sk.sticker_id = s.id
WHERE sk.group_id = $1
AND k.keyword ILIKE ANY ($2)
GROUP BY k.keyword;
`
	rows, err := db.Query(query, groupId, pq.Array(keywords))
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

			intersectionResult, ok := arrayOperations.Intersect(allStickerFileIds, fileIdsForKeyword)
			if !ok {
				return allStickerFileIds
			}

			allStickerFileIds, ok = intersectionResult.Interface().([]string)
			if !ok {
				return allStickerFileIds
			}
		}
	}
	checkErr(err)

	if len(allStickerFileIds) > 50 {
		allStickerFileIds = allStickerFileIds[:50]
	}

	return allStickerFileIds
}

func GetAllKeywordsForStickerFileId(stickerFileId string, groupId int64) (keywords []string) {

	query := `
SELECT array_agg(k.keyword)
FROM
 keywords k
 JOIN sticker_keywords sk ON sk.keyword_id = k.id
 JOIN stickers s ON sk.sticker_id = s.id
WHERE sk.group_id = $1 
AND s.file_id = $2
`

	err := db.QueryRow(query, groupId, stickerFileId).Scan(pq.Array(&keywords))
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	return
}

func setUserMode(chatId int64, mode string) (groupId int64, usersStickerId string) {
	groupIdQuery := `SELECT group_id FROM sessions WHERE chat_id = $1;`
	err := db.QueryRow(groupIdQuery, chatId).Scan(&groupId)

	if err == sql.ErrNoRows {
		insertQuery := `
          WITH inserted AS (
            INSERT INTO groups DEFAULT VALUES RETURNING id
          )
          INSERT INTO sessions (chat_id, group_id, mode) SELECT
                                                              $1,
                                                              inserted.id,
															  $2
                                                            from inserted
          ON CONFLICT (chat_id)
            DO UPDATE SET chat_id = excluded.chat_id
          returning group_id;`

		err = db.QueryRow(insertQuery, chatId, mode).Scan(&groupId)
		checkErr(err)
		return
	}
	checkErr(err)

	query := `
INSERT INTO sessions (chat_id, group_id, mode) VALUES ($1, $2, $3)
ON CONFLICT (chat_id)
  DO UPDATE set mode = excluded.mode
  RETURNING file_id;`
	err = db.QueryRow(query, chatId, groupId, mode).Scan(&usersStickerId)
	checkErr(err)

	return
}

func SetUserStickerAndGetMode(chatId int64, usersStickerId string) (groupId int64, mode string) {
	selectQuery := `SELECT group_id, mode FROM sessions WHERE chat_id = $1;`

	dbErr := db.QueryRow(selectQuery, chatId).Scan(&groupId, &mode)
	if dbErr == sql.ErrNoRows {
		insertQuery := `
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

		dbErr = db.QueryRow(insertQuery, chatId, usersStickerId).Scan(&groupId)
		mode = "add" // default value
		return
	}

	query := `
	UPDATE sessions
	SET file_id = $1
	WHERE chat_id = $2
	`
	_, dbErr = db.Exec(query, usersStickerId, chatId)
	checkErr(dbErr)
	return
}

func GetUserState(chatId int64) (usersStickerId string, usersMode string) {

	query := `
SELECT
  file_id,
  mode
FROM sessions
WHERE chat_id = $1`
	var dbUsersStickerId sql.NullString
	err := db.QueryRow(query, chatId).Scan(&dbUsersStickerId, &usersMode)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	if dbUsersStickerId.Valid {
		usersStickerId = dbUsersStickerId.String
	}

	return
}

func addKeywordsToSticker(stickerFileId string, keywordsString string, groupId int64) (responseMessage string) {
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return "No tags to add"
	}

	transaction, err := db.Begin()
	defer func() {
		err = transaction.Rollback()
		if err != nil && err != sql.ErrTxDone {
			panic(err)
		}
	}()
	checkErr(err)

	stickerQuery := `
INSERT INTO stickers (file_id) VALUES ($1)
ON CONFLICT (file_id)
  DO UPDATE set file_id = excluded.file_id
RETURNING id;`
	insertStickersStatement, err := transaction.Prepare(stickerQuery)
	defer insertStickersStatement.Close()
	checkErr(err)

	keywordQuery := `
INSERT INTO keywords (keyword) VALUES ($1)
ON CONFLICT (keyword)
  DO UPDATE set keyword = excluded.keyword
RETURNING id;`
	insertKeywordsStatement, err := transaction.Prepare(keywordQuery)
	defer insertKeywordsStatement.Close()
	checkErr(err)

	stickerKeywordQuery := `
INSERT INTO sticker_keywords (sticker_id, keyword_id, group_id) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;`
	insertStickersKeywordsStatement, err := transaction.Prepare(stickerKeywordQuery)
	defer insertStickersKeywordsStatement.Close()
	checkErr(err)

	var stickerId int
	err = insertStickersStatement.QueryRow(stickerFileId).Scan(&stickerId)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	err = insertStickersStatement.Close()
	checkErr(err)

	var keywordsAdded int64
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)

		var keywordId int
		err = insertKeywordsStatement.QueryRow(keyword).Scan(&keywordId)
		if err != sql.ErrNoRows {
			checkErr(err)
		}

		stickersKeywordsResult, err := insertStickersKeywordsStatement.Exec(stickerId, keywordId, groupId)
		checkErr(err)

		numRowsAffected, err := stickersKeywordsResult.RowsAffected()
		checkErr(err)

		keywordsAdded += numRowsAffected
	}

	responseMessage = "Added " + strconv.FormatInt(keywordsAdded, 10) + " tag(s)."

	err = transaction.Commit()
	checkErr(err)

	return
}

func removeKeywordsFromSticker(stickerFileId string, keywordsString string, groupId int64) string {
	keywordsString = EscapeSql(keywordsString)
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return "No tags to remove"
	}

	query := `
DELETE FROM sticker_keywords sk
USING keywords k, stickers s
WHERE sk.keyword_id = k.id
      AND sk.sticker_id = s.id
      AND s.file_id = $1
      AND sk.group_id = $3
      AND k.keyword ILIKE ANY ($2);`
	result, err := db.Exec(query, stickerFileId, pq.Array(keywords), groupId)
	checkErr(err)

	numRows, err := result.RowsAffected()

	return "You have deleted " + strconv.FormatInt(numRows, 10) + " tag(s)."
}

func getOrCreateUserGroup(chatId int64) (groupId int64) {

	selectQuery := `SELECT group_id FROM sessions WHERE chat_id = $1;`

	err := db.QueryRow(selectQuery, chatId).Scan(&groupId)
	if err == sql.ErrNoRows {

		insertQuery := `
WITH inserted AS (
  INSERT INTO groups DEFAULT VALUES RETURNING id
)
INSERT INTO sessions (chat_id, group_id) SELECT
                                           $1,
                                           inserted.id
                                         from inserted
ON CONFLICT (chat_id)
  DO UPDATE SET chat_id = excluded.chat_id
returning group_id;
`
		err = db.QueryRow(insertQuery, chatId).Scan(&groupId)
	}
	checkErr(err)
	return
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
