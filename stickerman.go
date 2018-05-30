package main

import "strings"

func cleanKeywords(queryString string) []string {
	if len(queryString) == 0 {
		return []string{}
	}

	queryString = strings.ToLower(queryString)
	queryString = strings.Replace(queryString, ",", " ", -1)
	queryString = strings.Replace(queryString, ":", " ", -1)
	queryString = strings.Replace(queryString, ".", " ", -1)

	return strings.Fields(queryString)
}

func processMessage(message *Message) (responseMessage string) {
	if message.ReplyToMessage != nil && message.ReplyToMessage.Sticker != nil && len(message.Text) != 0 {
		return addKeywordFromStickerReply(message)
	}

	if len(message.Text) != 0 {
		if message.Text[0] == '/' {
			switch message.Text {
			case "/start":
				fallthrough
			case "/help":
				return "This Bot is designed to help you find Stickers.\n" +
					"\n" +
					"Usage:\n" +
					"To search for Stickers in any chat type: @DevStampsBot followed by your search keywords.\n" +
					"\n" +
					"To add new Stickers and keywords to the bot, send the sticker to this chat then reply to the sticker with a message containing the keywords you want to add."
			case "/add":
				SetUserMode(message.Chat.ID, "add")
				return "You are now adding keywords"
			case "/remove":
				SetUserMode(message.Chat.ID, "remove")
				return "You are now removing keywords from the sticker"
			}
		} else {
			return ProcessKeywordMessage(message)
		}
	} else if message.Sticker != nil {
		return ProcessStickerMessage(message)
	}

	return "I don't know how to interpret your message"
}

func ProcessKeywordMessage(message *Message) (responseMessage string) {
	usersStickerId, mode := GetUserState(message.Chat.ID)
	if usersStickerId == "" {
		responseMessage = "Send a sticker to me then I'll be able to add searchable keywords to it"
	}

	switch mode {
	case "add":
		responseMessage = addKeywordsToSticker(usersStickerId, message.Text)
	case "remove":
		return removeKeywordsFromSticker(usersStickerId, message.Text)
	}

	return responseMessage
}

func ProcessStickerMessage(message *Message) (responseMessage string) {
	mode := SetUserStickerAndGetMode(message.Chat.ID, message.Sticker.FileID)
	return "Now you are " + mode + "ing keywords to " + message.Sticker.Emoji + " sticker"
}

func addKeywordFromStickerReply(message *Message) (responseMessage string) {
	stickerFileId := message.ReplyToMessage.Sticker.FileID
	return addKeywordsToSticker(stickerFileId, message.Text)
}
