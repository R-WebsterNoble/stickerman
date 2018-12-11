package main

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"strconv"
	"strings"
)

type DbOperationStatus int

const (
	Success       DbOperationStatus = 0
	InvalidFormat DbOperationStatus = 1
	NoChange      DbOperationStatus = 2
)

func GetAllStickerIdsForKeywords(keywordsString string, groupId int64, offset int) (allStickerFileIds []string, nextOffset int) {
	keywordsString = EscapeSql(keywordsString)
	keywordsString = keywordsString + "%"
	keywords := getKeywordsArray(keywordsString)

	keywordCount := len(keywords)

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT ARRAY(
    SELECT r.fid
    FROM (
           SELECT
             k1.fid,
             MAX(k1.skid),
             row_number()
             OVER ( ORDER BY MAX(k1.skid) DESC ) AS rn
           FROM
             (
               SELECT
                 s.id      AS sid,
                 s.file_id AS fid,
                 sk.id     AS skid
               FROM keywords k
                 JOIN sticker_keywords sk ON sk.keyword_id = k.id
                 JOIN stickers s ON sk.sticker_id = s.id
               WHERE sk.group_id = $1
                     AND k.keyword ILIKE $2
             ) k1`)

	for i := 1; i < keywordCount; i++ {
		iStr := strconv.Itoa(i + 1)
		queryBuilder.WriteString(`
             JOIN
             (
               SELECT
                 s.id as sid` + iStr + `,
                 s.file_id,
                 sk.id
               FROM keywords k
                 JOIN sticker_keywords sk ON sk.keyword_id = k.id
                 JOIN stickers s ON sk.sticker_id = s.id
               WHERE sk.group_id = $1
                     AND k.keyword ILIKE $` + strconv.Itoa(i+2) + `
             ) k` + iStr + ` ON k1.sid = k` + iStr + `.sid` + iStr)
	}

	queryBuilder.WriteString(`
           GROUP BY k1.fid
           ORDER by MAX(k1.skid) DESC
           LIMIT 51
           OFFSET ` + strconv.Itoa(offset) + `
         ) AS r
    ORDER BY r.rn
);`)

	query := queryBuilder.String()

	parameters := make([]interface{}, keywordCount+1)
	parameters[0] = groupId
	for i, keyword := range keywords {
		parameters[i+1] = keyword
	}

	err := db.QueryRow(query, parameters...).Scan(pq.Array(&allStickerFileIds))
	checkErr(err)

	if len(allStickerFileIds) > 50 {
		allStickerFileIds = allStickerFileIds[:50]
		nextOffset = offset + 50
	}

	return
}

func GetAllKeywordsForStickerFileId(stickerFileId string, groupId int64) (keywords []string) {

	query := `
SELECT array_agg(k.keyword)
FROM
 keywords k
 JOIN sticker_keywords sk ON sk.keyword_id = k.id
 JOIN stickers s ON sk.sticker_id = s.id
WHERE sk.group_id = $1 
AND s.file_id = $2`

	err := db.QueryRow(query, groupId, stickerFileId).Scan(pq.Array(&keywords))
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	return
}

func setUserMode(chatId int64, mode string) (groupId int64) {
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
          ON CONFLICT (chat_id) DO NOTHING 
          returning group_id;`

		err = db.QueryRow(insertQuery, chatId, mode).Scan(&groupId)
		if err == sql.ErrNoRows {
			return setUserMode(chatId, mode)
		} else {
			checkErr(err)
		}
	} else {
		checkErr(err)
	}
	SetSession(chatId, groupId, mode)
	return
}

func SetSession(chatId int64, groupId int64, mode string) {
	selectQuery := `
UPDATE sessions 
    SET group_id = $1,
        mode = $2
WHERE chat_id = $3;`
	_, err := db.Exec(selectQuery, groupId, mode, chatId)
	checkErr(err)
}

func GetStickerFileId(chatId int64) (stickerFileId string) {
	selectQuery := `
SELECT file_id FROM sessions WHERE chat_id = $1`

	var dbUsersStickerFileId sql.NullString
	err := db.QueryRow(selectQuery, chatId).Scan(&dbUsersStickerFileId)
	checkErr(err)

	if dbUsersStickerFileId.Valid {
		stickerFileId = dbUsersStickerFileId.String
	}
	return
}

func SetUserStickerAndGetMode(chatId int64, usersStickerId string) (groupId int64, mode string) {
	selectQuery := `SELECT group_id, mode FROM sessions WHERE chat_id = $1;`

	err := db.QueryRow(selectQuery, chatId).Scan(&groupId, &mode)
	if err == sql.ErrNoRows {
		insertQuery := `
          WITH inserted AS (
            INSERT INTO groups DEFAULT VALUES RETURNING id
          )
          INSERT INTO sessions (chat_id, file_id, group_id) SELECT
                                                              $1,
                                                              $2,
                                                              inserted.id
                                                            from inserted
          ON CONFLICT (chat_id) DO NOTHING
          RETURNING group_id;`
		var nullableGroupId sql.NullInt64
		err = db.QueryRow(insertQuery, chatId, usersStickerId).Scan(&nullableGroupId)
		if err == sql.ErrNoRows || !nullableGroupId.Valid {
			SetUserStickerAndGetMode(chatId, usersStickerId)
		} else {
			checkErr(err)
			groupId = nullableGroupId.Int64
		}
		mode = "add" // default value
		return
	} else {
		checkErr(err)
	}

	query := `
	UPDATE sessions
	SET file_id = $1
	WHERE chat_id = $2
	`
	_, err = db.Exec(query, usersStickerId, chatId)
	checkErr(err)
	return
}

func GetUserState(chatId int64) (usersStickerId string, usersMode string) {

	query := `
SELECT
  file_id,
  mode
FROM sessions
WHERE chat_id = $1`
	var dbUsersStickerFileId sql.NullString
	err := db.QueryRow(query, chatId).Scan(&dbUsersStickerFileId, &usersMode)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	if dbUsersStickerFileId.Valid {
		usersStickerId = dbUsersStickerFileId.String
	}

	return
}

func addKeywordsToSticker(stickerFileId string, keywordsString string, groupId int64) (status DbOperationStatus, addedTags int64) {
	keywordsArray := getKeywordsArray(keywordsString)
	return addKeywordsArrayToSticker(stickerFileId, keywordsArray, groupId)
}

func addKeywordsArrayToSticker(stickerFileId string, keywords []string, groupId int64) (status DbOperationStatus, addedTags int64) {
	if len(keywords) == 0 {
		return NoChange, 0
	}

	stickerId := getStickerId(stickerFileId)

	//transaction, err := db.Begin()
	//defer func() {
	//	err = transaction.Rollback()
	//	if err != nil && err != sql.ErrTxDone {
	//		panic(err)
	//	}
	//}()
	//checkErr(err)

	stickerKeywordQuery := `
INSERT INTO sticker_keywords (sticker_id, keyword_id, group_id) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;`
	//insertStickersKeywordsStatement, err := transaction.Prepare(stickerKeywordQuery)
	//defer insertStickersKeywordsStatement.Close()
	//checkErr(err)

	var keywordsAdded int64
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)

		keywordId := getKeywordId(keyword)

		stickersKeywordsResult, err := db.Exec(stickerKeywordQuery, stickerId, keywordId, groupId)
		checkErr(err)

		numRowsAffected, err := stickersKeywordsResult.RowsAffected()
		checkErr(err)

		keywordsAdded += numRowsAffected
	}

	if keywordsAdded == 0 {
		return NoChange, 0
	}
	return Success, keywordsAdded

	//err = transaction.Commit()
	//checkErr(err)
}

func getStickerId(stickerFileId string, ) (stickerId int64) {
	selectQuery := `SELECT id FROM stickers WHERE file_id = $1;`
	err := db.QueryRow(selectQuery, stickerFileId).Scan(&stickerId)
	if err == sql.ErrNoRows {
		insertQuery := `INSERT INTO stickers (file_id) VALUES ($1) ON CONFLICT DO NOTHING RETURNING id;`
		err := db.QueryRow(insertQuery, stickerFileId).Scan(&stickerId)
		if err == sql.ErrNoRows {
			return getStickerId(stickerFileId)
		} else {
			checkErr(err)
		}
	} else {
		checkErr(err)
	}
	return
}

func getKeywordId(keywordFileId string, ) (keywordId int64) {
	selectQuery := `SELECT id FROM keywords WHERE keyword = $1;`
	err := db.QueryRow(selectQuery, keywordFileId).Scan(&keywordId)
	if err == sql.ErrNoRows {
		insertQuery := `INSERT INTO keywords (keyword) VALUES ($1) ON CONFLICT DO NOTHING RETURNING id;`
		err := db.QueryRow(insertQuery, keywordFileId).Scan(&keywordId)
		if err == sql.ErrNoRows {
			return getKeywordId(keywordFileId)
		} else {
			checkErr(err)
		}
	} else {
		checkErr(err)
	}
	return
}

func removeKeywordsFromSticker(stickerFileId string, keywordsString string, groupId int64) (status DbOperationStatus, removedTags int64) {
	keywordsString = EscapeSql(keywordsString)
	keywords := getKeywordsArray(keywordsString)

	if len(keywords) == 0 {
		return NoChange, 0
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

	keywordsRemoved, err := result.RowsAffected()

	if keywordsRemoved == 0 {
		return NoChange, 0
	}

	return Success, keywordsRemoved
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
ON CONFLICT (chat_id) DO NOTHING RETURNING group_id;
`
		err = db.QueryRow(insertQuery, chatId).Scan(&groupId)
		if err == sql.ErrNoRows {
			return getOrCreateUserGroup(chatId)
		} else {
			checkErr(err)
		}
	} else {
		checkErr(err)
	}
	return
}

func GetUserGroup(chatId int64) (groupUuid string) {
	query := `SELECT uuid
from groups
       JOIN sessions s on groups.id = s.group_id
where chat_id = $1`
	err := db.QueryRow(query, chatId).Scan(&groupUuid)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	return
}

func assignUserToGroup(chatId int64, groupGuid string) DbOperationStatus {

	guid, err := uuid.Parse(groupGuid)
	if err != nil {
		return InvalidFormat
	}

	query := `
WITH groupId AS (
  SELECT id FROM groups WHERE uuid = $2 LIMIT 1
)
UPDATE sessions
SET group_id = (SELECT id FROM groupId)
WHERE EXISTS(SELECT id FROM groupId) AND chat_id = $1 AND group_id <> (SELECT id FROM groupId);`
	result, err := db.Exec(query, chatId, guid)
	checkErr(err)

	rowsAffected, err := result.RowsAffected()
	checkErr(err)
	if rowsAffected == 0 {
		return NoChange
	}

	return Success
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
