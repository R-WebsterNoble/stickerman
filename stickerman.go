package main

import "strings"

func processMessage(message *Message) (responseMessage string) {
	if message.ReplyToMessage != nil && message.ReplyToMessage.Sticker != nil && len(message.Text) != 0 {
		return addKeywordFromStickerReply(message)
	}

	if len(message.Text) != 0 {
		if message.Text[0] == '/' {
			switch strings.ToLower(message.Text) {
			case "/start":
				fallthrough
			case "/help":
				return "This Bot is designed to help you find stickers.\n" +
					"\n" +
					"Usage:\n" +
					"To search for a stickers in any chat type: @DevStampsBot followed by your search keywords.\n" +
					"\n" +
					"To add new sticker and keywords to the bot, first send the sticker to this chat."
			case "/add":
				SetUserMode(message.Chat.ID, "add")
				return "Okay, send me some keywords and I'll add them to the sticker."
			case "/remove":
				SetUserMode(message.Chat.ID, "remove")
				return "Okay, I'll remove keywords you send me from this sticker."
			default:
				return "I don't recognise this command."
			}
		} else {
			return ProcessKeywordMessage(message)
		}
	} else if message.Sticker != nil {
		return ProcessStickerMessage(message)
	}

	return "I don't know how to interpret your message."
}

func ProcessKeywordMessage(message *Message) string {
	usersStickerId, mode := GetUserState(message.Chat.ID)
	if usersStickerId == "" {
		return "Send a sticker to me then I'll be able to add searchable keywords to it."
	}
	groupId := upsertUserGroup(message.Chat.ID)
	switch mode {
	case "add":
		return addKeywordsToSticker(usersStickerId, message.Text, groupId)
	case "remove":
		return removeKeywordsFromSticker(usersStickerId, message.Text, groupId)
	}

	return ""
}

func ProcessStickerMessage(message *Message) (responseMessage string) {
	groupId, mode := SetUserStickerAndGetMode(message.Chat.ID, message.Sticker.FileID)
	keywordsOnSticker := GetAllKeywordsForStickerFileId(message.Sticker.FileID, groupId)
	if len(keywordsOnSticker) == 0 {
		switch mode {
		case "add":
			responseMessage = "That's a nice sticker. Send me some keywords and I'll add them to it."
		case "remove":
			responseMessage = "Okay, send me some keywords to remove them from this sticker."
		}
	} else {
		switch mode {
		case "add":
			responseMessage = "That sticker already has the keywords:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me some more keywords and I'll add them to it."
		case "remove":
			responseMessage = "That sticker has the keywords:\n" +
				"\n" +
				strings.Join(keywordsOnSticker, "\n") +
				"\n" +
				"\n" +
				"Send me keywords that you'd like to remove."
		}
	}

	return responseMessage
}

func addKeywordFromStickerReply(message *Message) (responseMessage string) {
	stickerFileId := message.ReplyToMessage.Sticker.FileID
	groupId := upsertUserGroup(message.Chat.ID)
	return addKeywordsToSticker(stickerFileId, message.Text, groupId)
}

func getKeywordsArray(keywordsString string) []string {
	if len(keywordsString) == 0 {
		return []string{}
	}

	keywordsString = strings.ToLower(keywordsString)
	keywordsString = strings.Replace(keywordsString, ",", " ", -1)
	keywordsString = strings.Replace(keywordsString, ":", " ", -1)
	keywordsString = strings.Replace(keywordsString, ".", " ", -1)

	return strings.Fields(keywordsString)
}
