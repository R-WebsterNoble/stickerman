package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func processMessage(message *Message) BotResponce {
	if message.ReplyToMessage != nil && message.ReplyToMessage.Sticker != nil && len(message.Text) != 0 {
		responseText := addKeywordFromStickerReply(message)
		return &TextMessageResponse{Text: responseText}
	}

	if len(message.Text) != 0 {
		if message.Text[0] == '/' {
			return processCommand(message)
		} else {
			return processKeywordMessage(message.Chat.ID, message.Text)
		}
	} else if message.Sticker != nil {
		responceText := processStickerMessage(message)
		return &TextMessageResponse{Text: responceText}
	}

	return &TextMessageResponse{Method: "sendMessage", Text: "I don't know how to interpret your message."}
}

func processCommand(message *Message) (responseMessage BotResponce) {
	switch strings.ToLower(message.Text) {
	case "/start":
		fallthrough
	case "/help":
		return &TextMessageResponse{Text: "Hi, I'm Sticker Manager Bot.\n" +
			"I'll help you manage your stickers by letting you tag them so you can easily find them later.\n" +
			"\n" +
			"Usage:\n" +
			"To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\n" +
			"\n" +
			"You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for."}
	case "/add":
		setUserMode(message.Chat.ID, "add")
		return &TextMessageResponse{Text: "Okay, send me some tags and I'll add them to the sticker."}
	case "/remove":
		setUserMode(message.Chat.ID, "remove")
		return &TextMessageResponse{Text: "Okay, I'll remove tags you send me from this sticker."}
	case "/testresponsekeyboard":
		s := "s"
		return &InlineKeyboardMarkupResponseMessage{
			Text: "blah",
			ReplyMarkup: InlineKeyboardMarkup{
				[][]InlineKeyboardButton{
					{InlineKeyboardButton{Text: "a1", CallbackData: &s}, InlineKeyboardButton{Text: "a2", CallbackData: &s}},
					{InlineKeyboardButton{Text: "b1", CallbackData: &s}, InlineKeyboardButton{Text: "b2", CallbackData: &s}},
				},
			},
		}
	case "/testthing":
		return &TextMessageResponse{Text: "/removething"}
	default:
		return processOtherCommand(message)
	}
}

func processOtherCommand(message *Message) BotResponce {
	if strings.HasPrefix(message.Text, "/add ") {
		groupId, usersStickerId := setUserMode(message.Chat.ID, "add")
		if usersStickerId == "" {
			return &TextMessageResponse{Text: "Send a sticker to me then I'll be able to add tags to it."}
		}
		keywordsText := message.Text[5:]
		responseText := "You are now in add mode.\n" + addKeywordsToSticker(usersStickerId, keywordsText, groupId)
		return &TextMessageResponse{Text: responseText}
	} else if strings.HasPrefix(message.Text, "/remove ") {
		usersStickerFileId, _ := GetUserState(message.Chat.ID)
		groupId := getOrCreateUserGroup(message.Chat.ID)
		keywordsText := message.Text[8:]
		responseText := removeKeywordsFromSticker(usersStickerFileId, keywordsText, groupId)
		return &TextMessageResponse{Text: responseText}
	} else {
		return &TextMessageResponse{Text: "I don't recognise this command."}
	}
}

func processKeywordMessage(chatId int64, messageText string) BotResponce {
	usersStickerId, mode := GetUserState(chatId)
	if usersStickerId == "" {
		return &TextMessageResponse{Text: "Send a sticker to me then I'll be able to add tags to it."}
	}
	groupId := getOrCreateUserGroup(chatId)
	switch mode {
	case "add":
		responseText := addKeywordsToSticker(usersStickerId, messageText, groupId)
		return &TextMessageResponse{Text: responseText}
	case "remove":
		responseText := removeKeywordsFromSticker(usersStickerId, messageText, groupId)
		return &TextMessageResponse{Text: responseText}
	}

	return &TextMessageResponse{Text: ""}
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
	return addKeywordsToSticker(stickerFileId, message.Text, groupId)
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
