package main

import (
	"strings"
	"os"
	"database/sql"
	"strconv"
	"github.com/lib/pq"
	"github.com/adam-hanna/arrayOperations"
)

func getAllStickerIdsForKeywords(keywordsString string) []string {
	keywordsString = EscapeSql(keywordsString)
	keywords := getKeywordsArray(keywordsString)

	if len(keywordsString) == 0 {
		return []string{}
	}

	keywordsString = EscapeSql(keywordsString)

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

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

	return allStickerFileIds
}

func SetUserMode(chatId int64, userMode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	query := `
INSERT INTO sessions (chat_id, mode) VALUES ($1, $2)
ON CONFLICT (chat_id)
  DO UPDATE set mode = excluded.mode;`
	_, err = db.Exec(query, chatId, userMode)
	checkErr(err)
}

func SetUserStickerAndGetMode(chatId int64, usersStickerId string) (mode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	query := `
INSERT INTO sessions (chat_id, file_id)
VALUES ($1, $2)
ON CONFLICT (chat_id)
  DO UPDATE set file_id = excluded.file_id
RETURNING mode;`
	err = db.QueryRow(query, chatId, usersStickerId).Scan(&mode)
	checkErr(err)

	return
}

func GetUserState(chatId int64) (usersStickerId string, usersMode string) {
	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()

	query := `
SELECT
  file_id,
  mode
FROM sessions
WHERE chat_id = $1`
	rows, err := db.Query(query, chatId)
	defer rows.Close()
	checkErr(err)

	for rows.Next() {
		rows.Scan(&usersStickerId, &usersMode)
		checkErr(err)
	}
	checkErr(err)

	return
}

func addKeywordsToSticker(stickerFileId string, keywordsString string) (responseMessage string) {
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return "No keywords to add"
	}

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	defer db.Close()
	checkErr(err)

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

	stickerResultRows, err := insertStickersStatement.Query(stickerFileId)
	defer stickerResultRows.Close()
	checkErr(err)

	var stickerId int
	for stickerResultRows.Next() {
		err = stickerResultRows.Scan(&stickerId)
		checkErr(err)
	}
	err = insertStickersStatement.Close()
	checkErr(err)

	var keywordsAdded int64
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		keywordsResultRows, err := insertKeywordsStatement.Query(keyword)
		checkErr(err)

		var keywordId int
		for keywordsResultRows.Next() {
			err = keywordsResultRows.Scan(&keywordId)
			checkErr(err)
		}
		checkErr(err)
		err = keywordsResultRows.Close()
		checkErr(err)

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

	connStr := os.Getenv("pgDBConnectionString")
	db, err := sql.Open("postgres", connStr)
	checkErr(err)
	defer db.Close()
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
