package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func processMessage(message *Message) (responseMessage string) {
	if message.ReplyToMessage != nil && message.ReplyToMessage.Sticker != nil && len(message.Text) != 0 {
		return addKeywordFromStickerReply(message)
	}

	if len(message.Text) != 0 {
		if message.Text[0] == '/' {
			return processCommand(message)
		} else {
			return processKeywordMessage(message.Chat.ID, message.Text)
		}
	} else if message.Sticker != nil {
		return processStickerMessage(message)
	}

	return "I don't know how to interpret your message."
}

func processCommand(message *Message) string {
	lowerCaseMessage := strings.ToLower(message.Text)
	switch lowerCaseMessage {
	case "/start":
		fallthrough
	case "/help":
		return "Hi, I'm Sticker Manager Bot.\n" +
			"I'll help you manage your stickers by letting you tag them so you can easily find them later.\n" +
			"\n" +
			"Usage:\n" +
			"To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\n" +
			"\n" +
			"You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for."
	case "/add":
		setUserMode(message.Chat.ID, "add")
		return "Okay, send me some tags and I'll add them to the sticker."
	case "/remove":
		setUserMode(message.Chat.ID, "remove")
		return "Okay, I'll remove tags you send me from this sticker."
	default:
		return processOtherCommand(message.Chat.ID, lowerCaseMessage)
	}
}

func processOtherCommand(chatId int64, messageText string) string {
	if strings.HasPrefix(messageText, "/add ") {
		groupId, usersStickerId := setUserMode(chatId, "add")
		if usersStickerId == "" {
			return "Send a sticker to me then I'll be able to add tags to it."
		}
		keywordsText := messageText[5:]
		status, addedTags := addKeywordsToSticker(usersStickerId, keywordsText, groupId)
		switch status {
		case Success:
			return "You are now in add mode.\n\nAdded " + strconv.FormatInt(addedTags, 10) + " " + pluralise("tag", addedTags) + "."
		case NoChange:
			return "You are now in add mode."
		}
	} else if strings.HasPrefix(messageText, "/remove ") {
		usersStickerFileId, _ := GetUserState(chatId)
		groupId := getOrCreateUserGroup(chatId)
		keywordsText := messageText[8:]
		status, removedTags := removeKeywordsFromSticker(usersStickerFileId, keywordsText, groupId)
		switch status {
		case Success:
			return "Removed " + strconv.FormatInt(removedTags, 10) + " " + pluralise("tag", removedTags) + "."
		case NoChange:
			setUserMode(chatId, "remove")
			return "You are now in remove mode."
		}
	} else {
		return "I don't recognise this command."
	}

	return ""
}

func processKeywordMessage(chatId int64, messageText string) string {
	usersStickerId, mode := GetUserState(chatId)
	if usersStickerId == "" {
		return "Send a sticker to me then I'll be able to add tags to it."
	}
	groupId := getOrCreateUserGroup(chatId)
	switch mode {
	case "add":
		status, addedTags := addKeywordsToSticker(usersStickerId, messageText, groupId)
		switch status {
		case Success:
			return "Added " + strconv.FormatInt(addedTags, 10) + " " + pluralise("tag", addedTags) + "."
		case NoChange:
			return "No tags to add"
		}
	case "remove":
		status, removedTags := removeKeywordsFromSticker(usersStickerId, messageText, groupId)
		switch status {
		case Success:
			return "Removed " + strconv.FormatInt(removedTags, 10) + " " + pluralise("tag", removedTags) + "."
		case NoChange:
			return "No tags to Remove"
		}
	}

	return ""
}

func processStickerMessage(message *Message) string {
	groupId, mode := SetUserStickerAndGetMode(message.Chat.ID, message.Sticker.FileID)
	keywordsOnSticker := GetAllKeywordsForStickerFileId(message.Sticker.FileID, groupId)
	if len(keywordsOnSticker) == 0 {
		addStickerSetDefaultTags(message.Sticker, groupId)
		return "That's a nice sticker. Send me some tags and I'll add them to it."
	} else {
		switch mode {
		case "add":
			return "That sticker already has the tags:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me some more tags and I'll add them to it."
		case "remove":
			return "That sticker has the tags:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me tags that you'd like to remove."
		}
	}
	return ""
}

func addStickerSetDefaultTags(sticker *Sticker, groupId int64) {
	safeSetName := url.QueryEscape(sticker.SetName)
	getStickerSetUrl := "https://api.telegram.org/bot" + os.Getenv("TelegramBotApiKey") + "/getStickerSet?name=" + safeSetName
	stickerSetResult := callGetStickerSetApi(getStickerSetUrl)
	if stickerSetResult.Ok {
		setTitleWords := strings.Fields(stickerSetResult.Result.Title)
		keywordsArray := []string{sticker.SetName, strings.Join(setTitleWords, "-")}
		keywordsArray = append(keywordsArray, setTitleWords...)
		for _, sticker := range stickerSetResult.Result.Stickers {
			keywordsArrayWithEmoji := append(keywordsArray, sticker.Emoji)
			addKeywordsArrayToSticker(sticker.FileID, keywordsArrayWithEmoji, groupId)
		}
	} else {

	}
}

func callGetStickerSetApi(url string) GetStickerSetResult {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.WithFields(log.Fields{"url": url, "error": err}).Error("error in http.NewRequest")
		return GetStickerSetResult{false, Stickers{}}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"url": url, "error": err}).Error("error in client.Do")
		return GetStickerSetResult{false, Stickers{}}
	}
	defer func() { checkErr(resp.Body.Close()) }()

	var stickers GetStickerSetResult
	err = json.NewDecoder(resp.Body).Decode(&stickers)
	if err != nil {
		log.WithFields(log.Fields{"url": url, "error": err}).Error("error decoding json")
		return GetStickerSetResult{false, Stickers{}}
	}

	return stickers
}

func addKeywordFromStickerReply(message *Message) (responseMessage string) {
	stickerFileId := message.ReplyToMessage.Sticker.FileID
	groupId := getOrCreateUserGroup(message.Chat.ID)
	status, addedTags := addKeywordsToSticker(stickerFileId, message.Text, groupId)
	switch status {
	case Success:
		return "Added " + strconv.FormatInt(addedTags, 10) + " " + pluralise("tag", addedTags) + "."
	case NoChange:
		return "No tags to add"
	}
	return ""
}

func getKeywordsArray(keywordsString string) []string {
	if len(keywordsString) == 0 {
		return []string{}
	}

	keywordsString = strings.ToLower(keywordsString)

	allKeywords := strings.Fields(keywordsString)

	uniqueKeywords := unique(allKeywords)

	if len(uniqueKeywords) > 10 { // anti DOS
		return uniqueKeywords[:10]
	}

	return uniqueKeywords
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func pluralise(word string, count int64) string {
	if count == 1 {
		return word
	} else {
		return word + "s"
	}
}
