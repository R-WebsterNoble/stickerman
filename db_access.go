package main

import (
	"strings"
	"database/sql"
	"strconv"
	"github.com/lib/pq"
	"github.com/adam-hanna/arrayOperations"
	"io/ioutil"
)

func SetupDB(sqlFile string) {
	schema, err := ioutil.ReadFile(sqlFile)
	checkErr(err)

	_, err = db.Exec(string(schema))
	checkErr(err)
}

func GetAllStickerIdsForKeywords(keywordsString string) []string {
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
WHERE k.keyword ILIKE ANY ($1)
GROUP BY k.keyword;
`
	rows, err := db.Query(query, pq.Array(keywords))
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

func GetAllKeywordsForStickerFileId(stickerFileId string) (keywords []string) {

	query := `
SELECT array_agg(k.keyword)
FROM
 keywords k
 JOIN sticker_keywords sk ON sk.keyword_id = k.id
 JOIN stickers s ON sk.sticker_id = s.id
WHERE s.file_id = $1`

	err := db.QueryRow(query, stickerFileId).Scan(pq.Array(&keywords))
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	return
}

func SetUserMode(chatId int64, userMode string) {

	query := `
INSERT INTO sessions (chat_id, mode) VALUES ($1, $2)
ON CONFLICT (chat_id)
  DO UPDATE set mode = excluded.mode;`
	_, err := db.Exec(query, chatId, userMode)
	checkErr(err)
}

func SetUserStickerAndGetMode(chatId int64, usersStickerId string) (mode string) {

	query := `
INSERT INTO sessions (chat_id, file_id)
VALUES ($1, $2)
ON CONFLICT (chat_id)
  DO UPDATE set file_id = excluded.file_id
RETURNING mode;`
	err := db.QueryRow(query, chatId, usersStickerId).Scan(&mode)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	return
}

func GetUserState(chatId int64) (usersStickerId string, usersMode string) {

	query := `
SELECT
  file_id,
  mode
FROM sessions
WHERE chat_id = $1`
	err := db.QueryRow(query, chatId).Scan(&usersStickerId, &usersMode)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	return
}

func addKeywordsToSticker(stickerFileId string, keywordsString string) (responseMessage string) {
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return "No keywords to add"
	}

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
ON CONFLICT DO NOTHING;`
	insertStickersKeywordsStatement, err := transaction.Prepare(query2)
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

		stickersKeywordsResult, err := insertStickersKeywordsStatement.Exec(stickerId, keywordId)
		checkErr(err)

		numRowsAffected, err := stickersKeywordsResult.RowsAffected()
		checkErr(err)

		keywordsAdded += numRowsAffected
	}

	responseMessage = "Added " + strconv.FormatInt(keywordsAdded, 10) + " keyword(s)."

	err = transaction.Commit()
	checkErr(err)

	return
}

func removeKeywordsFromSticker(stickerFileId string, keywordsString string) string {
	keywordsString = EscapeSql(keywordsString)
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return "No keywords to remove"
	}

	query := `
DELETE FROM sticker_keywords sk
USING keywords k, stickers s
WHERE sk.keyword_id = k.id
      AND sk.sticker_id = s.id
      and s.file_id = $1
      and k.keyword ILIKE ANY ($2);`
	result, err := db.Exec(query, stickerFileId, pq.Array(keywords))
	checkErr(err)

	numRows, err := result.RowsAffected()

	return "You have deleted " + strconv.FormatInt(numRows, 10) + " keywords."
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
