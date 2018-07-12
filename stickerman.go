package main

import "strings"

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

func processCommand(message *Message) (responseMessage string) {
	switch strings.ToLower(message.Text) {
	case "/start":
		fallthrough
	case "/help":
		return "This Bot is designed to help you find stickers.\n" +
			"\n" +
			"Usage:\n" +
			"To search for a stickers in any chat type: @DevStampsBot followed by your search keywords.\n" +
			"\n" +
			"To add a new sticker and keywords to the bot, first send the sticker to this chat."
	case "/add":
		setUserMode(message.Chat.ID, "add")
		return "Okay, send me some keywords and I'll add them to the sticker."
	case "/remove":
		setUserMode(message.Chat.ID, "remove")
		return "Okay, I'll remove keywords you send me from this sticker."
	default:
		return processOtherCommand(message)
	}
}

func processOtherCommand(message *Message) string {
	if strings.HasPrefix(message.Text, "/add ") {
		groupId, usersStickerId := setUserMode(message.Chat.ID, "add")
		if usersStickerId == "" {
			return "Send a sticker to me then I'll be able to add searchable keywords to it."
		}
		keywordsText := message.Text[5:]
		return "You and now in add mode.\n" + addKeywordsToSticker(usersStickerId, keywordsText, groupId)
	} else if strings.HasPrefix(message.Text, "/remove ") {
		usersStickerId, _ := GetUserState(message.Chat.ID)
		groupId := getOrCreateUserGroup(message.Chat.ID)
		keywordsText := message.Text[8:]
		return removeKeywordsFromSticker(usersStickerId, keywordsText, groupId)
	} else {
		return "I don't recognise this command."
	}
}

func processKeywordMessage(chatId int64, messageText string) string {
	usersStickerId, mode := GetUserState(chatId)
	if usersStickerId == "" {
		return "Send a sticker to me then I'll be able to add searchable keywords to it."
	}
	groupId := getOrCreateUserGroup(chatId)
	switch mode {
	case "add":
		return addKeywordsToSticker(usersStickerId, messageText, groupId)
	case "remove":
		return removeKeywordsFromSticker(usersStickerId, messageText, groupId)
	}

	return ""
}

func processStickerMessage(message *Message) string {
	groupId, mode := SetUserStickerAndGetMode(message.Chat.ID, message.Sticker.FileID)
	keywordsOnSticker := GetAllKeywordsForStickerFileId(message.Sticker.FileID, groupId)
	if len(keywordsOnSticker) == 0 {
		switch mode {
		case "add":
			return "That's a nice sticker. Send me some keywords and I'll add them to it."
		case "remove":
			return "Okay, send me some keywords to remove them from this sticker."
		}
	} else {
		switch mode {
		case "add":
			return "That sticker already has the keywords:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me some more keywords and I'll add them to it."
		case "remove":
			return "That sticker has the keywords:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me keywords that you'd like to remove."
		}
	}

	return ""
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

	return strings.Fields(keywordsString)
}
