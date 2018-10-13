package main

import (
	"encoding/json"
	"strings"
	"time"
)

// APIResponse is a response from the Telegram API with the result
// stored raw.
type APIResponse struct {
	Ok          bool                `json:"ok,omitempty"`
	Result      json.RawMessage     `json:"result,omitempty"`
	ErrorCode   int                 `json:"error_code,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  *ResponseParameters `json:"parameters,omitempty"`
}

// ResponseParameters are various errors that can be returned in APIResponse.
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"` // optional
	RetryAfter      int   `json:"retry_after,omitempty"`        // optional
}

type ChatId struct {
	Identifier int         `json:"identifier,omitempty"`
	Username   interface{} `json:"username,omitempty"`
}

type Response struct {
	Method string `json:"method,omitempty"`
	ChatId int64  `json:"chat_id,omitempty"`
	Text   string `json:"text,omitempty"`
}

// Update is an update response, from GetUpdates.
type Update struct {
	UpdateID           int                 `json:"update_id,omitempty,omitempty"`
	Message            *Message            `json:"message,omitempty,omitempty"`
	EditedMessage      *Message            `json:"edited_message,omitempty,omitempty"`
	ChannelPost        *Message            `json:"channel_post,omitempty,omitempty"`
	EditedChannelPost  *Message            `json:"edited_channel_post,omitempty,omitempty"`
	InlineQuery        *InlineQuery        `json:"inline_query,omitempty,omitempty"`
	ChosenInlineResult *ChosenInlineResult `json:"chosen_inline_result,omitempty,omitempty"`
	CallbackQuery      *CallbackQuery      `json:"callback_query,omitempty,omitempty"`
	ShippingQuery      *ShippingQuery      `json:"shipping_query,omitempty,omitempty"`
	PreCheckoutQuery   *PreCheckoutQuery   `json:"pre_checkout_query,omitempty,omitempty"`
}

// UpdatesChannel is the channel for getting updates.
type UpdatesChannel <-chan Update

// Clear discards all unprocessed incoming updates.
func (ch UpdatesChannel) Clear() {
	for len(ch) != 0 {
		<-ch
	}
}

// User is a user on Telegram.
type User struct {
	ID           int    `json:"id,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`     // optional
	UserName     string `json:"username,omitempty"`      // optional
	LanguageCode string `json:"language_code,omitempty"` // optional
	IsBot        bool   `json:"is_bot,omitempty"`        // optional
}

// String displays a simple text version of a user.
//
// It is normally a user's username, but falls back to a first/last
// name as available.
func (u *User) String() string {
	if u.UserName != "" {
		return u.UserName
	}

	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}

	return name
}

// GroupChat is a group chat.
type GroupChat struct {
	ID    int    `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}

// ChatPhoto represents a chat photo.
type ChatPhoto struct {
	SmallFileID string `json:"small_file_id,omitempty"`
	BigFileID   string `json:"big_file_id,omitempty"`
}

// Chat contains information about the place a message was sent.
type Chat struct {
	ID                  int64      `json:"id,omitempty"`
	Type                string     `json:"type,omitempty"`
	Title               string     `json:"title,omitempty"`                          // optional
	UserName            string     `json:"username,omitempty"`                       // optional
	FirstName           string     `json:"first_name,omitempty"`                     // optional
	LastName            string     `json:"last_name,omitempty"`                      // optional
	AllMembersAreAdmins bool       `json:"all_members_are_administrators,omitempty"` // optional
	Photo               *ChatPhoto `json:"photo,omitempty"`
	Description         string     `json:"description,omitempty,omitempty"` // optional
	InviteLink          string     `json:"invite_link,omitempty,omitempty"` // optional
}

// IsPrivate returns if the Chat is a private conversation.
func (c Chat) IsPrivate() bool {
	return c.Type == "private"
}

// IsGroup returns if the Chat is a group.
func (c Chat) IsGroup() bool {
	return c.Type == "group"
}

// IsSuperGroup returns if the Chat is a supergroup.
func (c Chat) IsSuperGroup() bool {
	return c.Type == "supergroup"
}

// IsChannel returns if the Chat is a channel.
func (c Chat) IsChannel() bool {
	return c.Type == "channel"
}

// ChatConfig returns a ChatConfig struct for chat related methods.
//func (c Chat) ChatConfig() ChatConfig {
//	return ChatConfig{ChatID: c.ID}
//}

// Message is returned by almost every request, and contains data about
// almost anything.
type Message struct {
	MessageID             int                `json:"message_id,omitempty"`
	From                  *User              `json:"from,omitempty"` // optional
	Date                  int                `json:"date,omitempty"`
	Chat                  *Chat              `json:"chat,omitempty"`
	ForwardFrom           *User              `json:"forward_from,omitempty"`            // optional
	ForwardFromChat       *Chat              `json:"forward_from_chat,omitempty"`       // optional
	ForwardFromMessageID  int                `json:"forward_from_message_id,omitempty"` // optional
	ForwardDate           int                `json:"forward_date,omitempty"`            // optional
	ReplyToMessage        *Message           `json:"reply_to_message,omitempty"`        // optional
	EditDate              int                `json:"edit_date,omitempty"`               // optional
	Text                  string             `json:"text,omitempty"`                    // optional
	Entities              *[]MessageEntity   `json:"entities,omitempty"`                // optional
	Audio                 *Audio             `json:"audio,omitempty"`                   // optional
	Document              *Document          `json:"document,omitempty"`                // optional
	Game                  *Game              `json:"game,omitempty"`                    // optional
	Photo                 *[]PhotoSize       `json:"photo,omitempty"`                   // optional
	Sticker               *Sticker           `json:"sticker,omitempty"`                 // optional
	Video                 *Video             `json:"video,omitempty"`                   // optional
	VideoNote             *VideoNote         `json:"video_note,omitempty"`              // optional
	Voice                 *Voice             `json:"voice,omitempty"`                   // optional
	Caption               string             `json:"caption,omitempty"`                 // optional
	Contact               *Contact           `json:"contact,omitempty"`                 // optional
	Location              *Location          `json:"location,omitempty"`                // optional
	Venue                 *Venue             `json:"venue,omitempty"`                   // optional
	NewChatMembers        *[]User            `json:"new_chat_members,omitempty"`        // optional
	LeftChatMember        *User              `json:"left_chat_member,omitempty"`        // optional
	NewChatTitle          string             `json:"new_chat_title,omitempty"`          // optional
	NewChatPhoto          *[]PhotoSize       `json:"new_chat_photo,omitempty"`          // optional
	DeleteChatPhoto       bool               `json:"delete_chat_photo,omitempty"`       // optional
	GroupChatCreated      bool               `json:"group_chat_created,omitempty"`      // optional
	SuperGroupChatCreated bool               `json:"supergroup_chat_created,omitempty"` // optional
	ChannelChatCreated    bool               `json:"channel_chat_created,omitempty"`    // optional
	MigrateToChatID       int64              `json:"migrate_to_chat_id,omitempty"`      // optional
	MigrateFromChatID     int64              `json:"migrate_from_chat_id,omitempty"`    // optional
	PinnedMessage         *Message           `json:"pinned_message,omitempty"`          // optional
	Invoice               *Invoice           `json:"invoice,omitempty"`                 // optional
	SuccessfulPayment     *SuccessfulPayment `json:"successful_payment,omitempty"`      // optional
}

// Time converts the message timestamp into a Time.
func (m *Message) Time() time.Time {
	return time.Unix(int64(m.Date), 0)
}

// IsCommand returns true if message starts with a "bot_command" entity.
func (m *Message) IsCommand() bool {
	if m.Entities == nil || len(*m.Entities) == 0 {
		return false
	}

	entity := (*m.Entities)[0]
	return entity.Offset == 0 && entity.Type == "bot_command"
}

// Command checks if the message was a command and if it was, returns the
// command. If the Message was not a command, it returns an empty string.
//
// If the command contains the at name syntax, it is removed. Use
// CommandWithAt() if you do not want that.
func (m *Message) Command() string {
	command := m.CommandWithAt()

	if i := strings.Index(command, "@"); i != -1 {
		command = command[:i]
	}

	return command
}

// CommandWithAt checks if the message was a command and if it was, returns the
// command. If the Message was not a command, it returns an empty string.
//
// If the command contains the at name syntax, it is not removed. Use Command()
// if you want that.
func (m *Message) CommandWithAt() string {
	if !m.IsCommand() {
		return ""
	}

	// IsCommand() checks that the message begins with a bot_command entity
	entity := (*m.Entities)[0]
	return m.Text[1:entity.Length]
}

// CommandArguments checks if the message was a command and if it was,
// returns all text after the command name. If the Message was not a
// command, it returns an empty string.
//
// Note: The first character after the command name is omitted:
// - "/foo bar baz" yields "bar baz", not " bar baz"
// - "/foo-bar baz" yields "bar baz", too
// Even though the latter is not a command conforming to the spec, the API
// marks "/foo" as command entity.
func (m *Message) CommandArguments() string {
	if !m.IsCommand() {
		return ""
	}

	// IsCommand() checks that the message begins with a bot_command entity
	entity := (*m.Entities)[0]
	if len(m.Text) == entity.Length {
		return "" // The command makes up the whole message
	}

	return m.Text[entity.Length+1:]
}

// MessageEntity contains information about data in a Message.
type MessageEntity struct {
	Type   string `json:"type,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Length int    `json:"length,omitempty"`
	URL    string `json:"url,omitempty"`  // optional
	User   *User  `json:"user,omitempty"` // optional
}

//// ParseURL attempts to parse a URL contained within a MessageEntity.
//func (entity MessageEntity) ParseURL() (*url.URL, error) {
//	if entity.URL == "" {
//		return nil, errors.New(ErrBadURL)
//	}
//
//	return url.Parse(entity.URL)
//}

// PhotoSize contains information about photos.
type PhotoSize struct {
	FileID   string `json:"file_id,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	FileSize int    `json:"file_size,omitempty"` // optional
}

// Audio contains information about audio.
type Audio struct {
	FileID    string `json:"file_id,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Performer string `json:"performer,omitempty"` // optional
	Title     string `json:"title,omitempty"`     // optional
	MimeType  string `json:"mime_type,omitempty"` // optional
	FileSize  int    `json:"file_size,omitempty"` // optional
}

// Document contains information about a document.
type Document struct {
	FileID    string     `json:"file_id,omitempty"`
	Thumbnail *PhotoSize `json:"thumb,omitempty"`     // optional
	FileName  string     `json:"file_name,omitempty"` // optional
	MimeType  string     `json:"mime_type,omitempty"` // optional
	FileSize  int        `json:"file_size,omitempty"` // optional
}

// Sticker contains information about a sticker.
type Sticker struct {
	FileID    string     `json:"file_id,omitempty"`
	Width     int        `json:"width,omitempty"`
	Height    int        `json:"height,omitempty"`
	Thumbnail *PhotoSize `json:"thumb,omitempty"` // optional
	Emoji     string     `json:"emoji,omitempty"` // optional
	FileSize  int        `json:"file_size,omitempty"`
	SetName   string     `json:"set_name,omitempty"`
	// optional
}

// Stickers contains information about a sticker pack.
type Stickers struct {
	Name          string    `json:"name,omitempty"`
	Title         string    `json:"title,omitempty"`
	ContainsMasks bool      `json:"contains_masks,omitempty"`
	Stickers      []Sticker `json:"stickers,omitempty"`
}

type GetStickerSetResult struct {
	Ok     bool     `json:"ok,omitempty"`
	Result Stickers `json:"result,omitempty"`
}

// Video contains information about a video.
type Video struct {
	FileID    string     `json:"file_id,omitempty"`
	Width     int        `json:"width,omitempty"`
	Height    int        `json:"height,omitempty"`
	Duration  int        `json:"duration,omitempty"`
	Thumbnail *PhotoSize `json:"thumb,omitempty"`     // optional
	MimeType  string     `json:"mime_type,omitempty"` // optional
	FileSize  int        `json:"file_size,omitempty"` // optional
}

// VideoNote contains information about a video.
type VideoNote struct {
	FileID    string     `json:"file_id,omitempty"`
	Length    int        `json:"length,omitempty"`
	Duration  int        `json:"duration,omitempty"`
	Thumbnail *PhotoSize `json:"thumb,omitempty"`     // optional
	FileSize  int        `json:"file_size,omitempty"` // optional
}

// Voice contains information about a voice.
type Voice struct {
	FileID   string `json:"file_id,omitempty"`
	Duration int    `json:"duration,omitempty"`
	MimeType string `json:"mime_type,omitempty"` // optional
	FileSize int    `json:"file_size,omitempty"` // optional
}

// Contact contains information about a contact.
//
// Note that LastName and UserID may be empty.
type Contact struct {
	PhoneNumber string `json:"phone_number,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"` // optional
	UserID      int    `json:"user_id,omitempty"`   // optional
}

// Location contains information about a place.
type Location struct {
	Longitude float64 `json:"longitude,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
}

// Venue contains information about a venue, including its Location.
type Venue struct {
	Location     Location `json:"location,omitempty"`
	Title        string   `json:"title,omitempty"`
	Address      string   `json:"address,omitempty"`
	FoursquareID string   `json:"foursquare_id,omitempty"` // optional
}

// UserProfilePhotos contains a set of user profile photos.
type UserProfilePhotos struct {
	TotalCount int           `json:"total_count,omitempty"`
	Photos     [][]PhotoSize `json:"photos,omitempty"`
}

// File contains information about a file to download from Telegram.
type File struct {
	FileID   string `json:"file_id,omitempty"`
	FileSize int    `json:"file_size,omitempty"` // optional
	FilePath string `json:"file_path,omitempty"` // optional
}

// Link returns a full path to the download URL for a File.
//
// It requires the Bot Token to create the link.
//func (f *File) Link(token string) string {
//	return fmt.Sprintf(FileEndpoint, token, f.FilePath)
//}

// ReplyKeyboardMarkup allows the Bot to set a custom keyboard.
type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard,omitempty"`
	ResizeKeyboard  bool               `json:"resize_keyboard,omitempty"`   // optional
	OneTimeKeyboard bool               `json:"one_time_keyboard,omitempty"` // optional
	Selective       bool               `json:"selective,omitempty"`         // optional
}

// KeyboardButton is a button within a custom keyboard.
type KeyboardButton struct {
	Text            string `json:"text,omitempty"`
	RequestContact  bool   `json:"request_contact,omitempty"`
	RequestLocation bool   `json:"request_location,omitempty"`
}

// ReplyKeyboardHide allows the Bot to hide a custom keyboard.
type ReplyKeyboardHide struct {
	HideKeyboard bool `json:"hide_keyboard,omitempty"`
	Selective    bool `json:"selective,omitempty"` // optional
}

// ReplyKeyboardRemove allows the Bot to hide a custom keyboard.
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard,omitempty"`
	Selective      bool `json:"selective,omitempty"`
}

// InlineKeyboardMarkup is a custom keyboard presented for an inline bot.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard,omitempty"`
}

// InlineKeyboardButton is a button within a custom keyboard for
// inline query responses.
//
// Note that some values are references as even an empty string
// will change behavior.
//
// CallbackGame, if set, MUST be first button in first row.
type InlineKeyboardButton struct {
	Text                         string        `json:"text,omitempty"`
	URL                          *string       `json:"url,omitempty,omitempty"`                              // optional
	CallbackData                 *string       `json:"callback_data,omitempty,omitempty"`                    // optional
	SwitchInlineQuery            *string       `json:"switch_inline_query,omitempty,omitempty"`              // optional
	SwitchInlineQueryCurrentChat *string       `json:"switch_inline_query_current_chat,omitempty,omitempty"` // optional
	CallbackGame                 *CallbackGame `json:"callback_game,omitempty,omitempty"`                    // optional
	Pay                          bool          `json:"pay,omitempty,omitempty"`                              // optional
}

// CallbackQuery is data sent when a keyboard button with callback data
// is clicked.
type CallbackQuery struct {
	ID              string   `json:"id,omitempty"`
	From            *User    `json:"from,omitempty"`
	Message         *Message `json:"message,omitempty"`           // optional
	InlineMessageID string   `json:"inline_message_id,omitempty"` // optional
	ChatInstance    string   `json:"chat_instance,omitempty"`
	Data            string   `json:"data,omitempty"`            // optional
	GameShortName   string   `json:"game_short_name,omitempty"` // optional
}

// ForceReply allows the Bot to have users directly reply to it without
// additional interaction.
type ForceReply struct {
	ForceReply bool `json:"force_reply,omitempty"`
	Selective  bool `json:"selective,omitempty"` // optional
}

// ChatMember is information about a member in a chat.
type ChatMember struct {
	User                  *User  `json:"user,omitempty"`
	Status                string `json:"status,omitempty"`
	UntilDate             int64  `json:"until_date,omitempty,omitempty"`                // optional
	CanBeEdited           bool   `json:"can_be_edited,omitempty,omitempty"`             // optional
	CanChangeInfo         bool   `json:"can_change_info,omitempty,omitempty"`           // optional
	CanPostMessages       bool   `json:"can_post_messages,omitempty,omitempty"`         // optional
	CanEditMessages       bool   `json:"can_edit_messages,omitempty,omitempty"`         // optional
	CanDeleteMessages     bool   `json:"can_delete_messages,omitempty,omitempty"`       // optional
	CanInviteUsers        bool   `json:"can_invite_users,omitempty,omitempty"`          // optional
	CanRestrictMembers    bool   `json:"can_restrict_members,omitempty,omitempty"`      // optional
	CanPinMessages        bool   `json:"can_pin_messages,omitempty,omitempty"`          // optional
	CanPromoteMembers     bool   `json:"can_promote_members,omitempty,omitempty"`       // optional
	CanSendMessages       bool   `json:"can_send_messages,omitempty,omitempty"`         // optional
	CanSendMediaMessages  bool   `json:"can_send_media_messages,omitempty,omitempty"`   // optional
	CanSendOtherMessages  bool   `json:"can_send_other_messages,omitempty,omitempty"`   // optional
	CanAddWebPagePreviews bool   `json:"can_add_web_page_previews,omitempty,omitempty"` // optional
}

// IsCreator returns if the ChatMember was the creator of the chat.
func (chat ChatMember) IsCreator() bool { return chat.Status == "creator" }

// IsAdministrator returns if the ChatMember is a chat administrator.
func (chat ChatMember) IsAdministrator() bool { return chat.Status == "administrator" }

// IsMember returns if the ChatMember is a current member of the chat.
func (chat ChatMember) IsMember() bool { return chat.Status == "member" }

// HasLeft returns if the ChatMember left the chat.
func (chat ChatMember) HasLeft() bool { return chat.Status == "left" }

// WasKicked returns if the ChatMember was kicked from the chat.
func (chat ChatMember) WasKicked() bool { return chat.Status == "kicked" }

// Game is a game within Telegram.
type Game struct {
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	Photo        []PhotoSize     `json:"photo,omitempty"`
	Text         string          `json:"text,omitempty"`
	TextEntities []MessageEntity `json:"text_entities,omitempty"`
	Animation    Animation       `json:"animation,omitempty"`
}

// Animation is a GIF animation demonstrating the game.
type Animation struct {
	FileID   string    `json:"file_id,omitempty"`
	Thumb    PhotoSize `json:"thumb,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	MimeType string    `json:"mime_type,omitempty"`
	FileSize int       `json:"file_size,omitempty"`
}

// GameHighScore is a user's score and position on the leaderboard.
type GameHighScore struct {
	Position int  `json:"position,omitempty"`
	User     User `json:"user,omitempty"`
	Score    int  `json:"score,omitempty"`
}

// CallbackGame is for starting a game in an inline keyboard button.
type CallbackGame struct{}

// WebhookInfo is information about a currently set webhook.
type WebhookInfo struct {
	URL                  string `json:"url,omitempty"`
	HasCustomCertificate bool   `json:"has_custom_certificate,omitempty"`
	PendingUpdateCount   int    `json:"pending_update_count,omitempty"`
	LastErrorDate        int    `json:"last_error_date,omitempty"`    // optional
	LastErrorMessage     string `json:"last_error_message,omitempty"` // optional
}

// IsSet returns true if a webhook is currently set.
func (info WebhookInfo) IsSet() bool {
	return info.URL != ""
}

// InlineQuery is a Query from Telegram for an inline request.
type InlineQuery struct {
	ID       string    `json:"id,omitempty"`
	From     *User     `json:"from,omitempty"`
	Location *Location `json:"location,omitempty"` // optional
	Query    string    `json:"query,omitempty"`
	Offset   string    `json:"offset,omitempty"`
}

// InlineQueryResultArticle is an inline query response article.
type InlineQueryResultArticle struct {
	Type                string                `json:"type,omitempty"`                            // required
	ID                  string                `json:"id,omitempty"`                              // required
	Title               string                `json:"title,omitempty"`                           // required
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"` // required
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	URL                 string                `json:"url,omitempty"`
	HideURL             bool                  `json:"hide_url,omitempty"`
	Description         string                `json:"description,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	ThumbWidth          int                   `json:"thumb_width,omitempty"`
	ThumbHeight         int                   `json:"thumb_height,omitempty"`
}

// InlineQueryResultPhoto is an inline query response photo.
type InlineQueryResultPhoto struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	URL                 string                `json:"photo_url,omitempty"` // required
	MimeType            string                `json:"mime_type,omitempty"`
	Width               int                   `json:"photo_width,omitempty"`
	Height              int                   `json:"photo_height,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	Title               string                `json:"title,omitempty"`
	Description         string                `json:"description,omitempty"`
	Caption             string                `json:"caption,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultGIF is an inline query response GIF.
type InlineQueryResultGIF struct {
	Type                string                `json:"type,omitempty"`    // required
	ID                  string                `json:"id,omitempty"`      // required
	URL                 string                `json:"gif_url,omitempty"` // required
	Width               int                   `json:"gif_width,omitempty"`
	Height              int                   `json:"gif_height,omitempty"`
	Duration            int                   `json:"gif_duration,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	Title               string                `json:"title,omitempty"`
	Caption             string                `json:"caption,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultMPEG4GIF is an inline query response MPEG4 GIF.
type InlineQueryResultMPEG4GIF struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	URL                 string                `json:"mpeg4_url,omitempty"` // required
	Width               int                   `json:"mpeg4_width,omitempty"`
	Height              int                   `json:"mpeg4_height,omitempty"`
	Duration            int                   `json:"mpeg4_duration,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	Title               string                `json:"title,omitempty"`
	Caption             string                `json:"caption,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultVideo is an inline query response video.
type InlineQueryResultVideo struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	URL                 string                `json:"video_url,omitempty"` // required
	MimeType            string                `json:"mime_type,omitempty"` // required
	ThumbURL            string                `json:"thumb_url,omitempty"`
	Title               string                `json:"title,omitempty"`
	Caption             string                `json:"caption,omitempty"`
	Width               int                   `json:"video_width,omitempty"`
	Height              int                   `json:"video_height,omitempty"`
	Duration            int                   `json:"video_duration,omitempty"`
	Description         string                `json:"description,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultAudio is an inline query response audio.
type InlineQueryResultAudio struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	URL                 string                `json:"audio_url,omitempty"` // required
	Title               string                `json:"title,omitempty"`     // required
	Caption             string                `json:"caption,omitempty"`
	Performer           string                `json:"performer,omitempty"`
	Duration            int                   `json:"audio_duration,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultVoice is an inline query response voice.
type InlineQueryResultVoice struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	URL                 string                `json:"voice_url,omitempty"` // required
	Title               string                `json:"title,omitempty"`     // required
	Caption             string                `json:"caption,omitempty"`
	Duration            int                   `json:"voice_duration,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
}

// InlineQueryResultDocument is an inline query response document.
type InlineQueryResultDocument struct {
	Type                string                `json:"type,omitempty"`  // required
	ID                  string                `json:"id,omitempty"`    // required
	Title               string                `json:"title,omitempty"` // required
	Caption             string                `json:"caption,omitempty"`
	URL                 string                `json:"document_url,omitempty"` // required
	MimeType            string                `json:"mime_type,omitempty"`    // required
	Description         string                `json:"description,omitempty"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	ThumbWidth          int                   `json:"thumb_width,omitempty"`
	ThumbHeight         int                   `json:"thumb_height,omitempty"`
}

// InlineQueryResultLocation is an inline query response location.
type InlineQueryResultLocation struct {
	Type                string                `json:"type,omitempty"`      // required
	ID                  string                `json:"id,omitempty"`        // required
	Latitude            float64               `json:"latitude,omitempty"`  // required
	Longitude           float64               `json:"longitude,omitempty"` // required
	Title               string                `json:"title,omitempty"`     // required
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
	InputMessageContent interface{}           `json:"input_message_content,omitempty,omitempty"`
	ThumbURL            string                `json:"thumb_url,omitempty"`
	ThumbWidth          int                   `json:"thumb_width,omitempty"`
	ThumbHeight         int                   `json:"thumb_height,omitempty"`
}

// InlineQueryResultGame is an inline query response game.
type InlineQueryResultGame struct {
	Type          string                `json:"type,omitempty"`
	ID            string                `json:"id,omitempty"`
	GameShortName string                `json:"game_short_name,omitempty"`
	ReplyMarkup   *InlineKeyboardMarkup `json:"reply_markup,omitempty,omitempty"`
}

// ChosenInlineResult is an inline query result chosen by a User
type ChosenInlineResult struct {
	ResultID        string    `json:"result_id,omitempty"`
	From            *User     `json:"from,omitempty"`
	Location        *Location `json:"location,omitempty"`
	InlineMessageID string    `json:"inline_message_id,omitempty"`
	Query           string    `json:"query,omitempty"`
}

// InputTextMessageContent contains text for displaying
// as an inline query result.
type InputTextMessageContent struct {
	Text                  string `json:"message_text,omitempty"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

// InputLocationMessageContent contains a location for displaying
// as an inline query result.
type InputLocationMessageContent struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// InputVenueMessageContent contains a venue for displaying
// as an inline query result.
type InputVenueMessageContent struct {
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	Title        string  `json:"title,omitempty"`
	Address      string  `json:"address,omitempty"`
	FoursquareID string  `json:"foursquare_id,omitempty"`
}

// InputContactMessageContent contains a contact for displaying
// as an inline query result.
type InputContactMessageContent struct {
	PhoneNumber string `json:"phone_number,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
}

// Invoice contains basic information about an invoice.
type Invoice struct {
	Title          string `json:"title,omitempty"`
	Description    string `json:"description,omitempty"`
	StartParameter string `json:"start_parameter,omitempty"`
	Currency       string `json:"currency,omitempty"`
	TotalAmount    int    `json:"total_amount,omitempty"`
}

// LabeledPrice represents a portion of the price for goods or services.
type LabeledPrice struct {
	Label  string `json:"label,omitempty"`
	Amount int    `json:"amount,omitempty"`
}

// ShippingAddress represents a shipping address.
type ShippingAddress struct {
	CountryCode string `json:"country_code,omitempty"`
	State       string `json:"state,omitempty"`
	City        string `json:"city,omitempty"`
	StreetLine1 string `json:"street_line1,omitempty"`
	StreetLine2 string `json:"street_line2,omitempty"`
	PostCode    string `json:"post_code,omitempty"`
}

// OrderInfo represents information about an order.
type OrderInfo struct {
	Name            string           `json:"name,omitempty,omitempty"`
	PhoneNumber     string           `json:"phone_number,omitempty,omitempty"`
	Email           string           `json:"email,omitempty,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty,omitempty"`
}

// ShippingOption represents one shipping option.
type ShippingOption struct {
	ID     string          `json:"id,omitempty"`
	Title  string          `json:"title,omitempty"`
	Prices *[]LabeledPrice `json:"prices,omitempty"`
}

// SuccessfulPayment contains basic information about a successful payment.
type SuccessfulPayment struct {
	Currency                string     `json:"currency,omitempty"`
	TotalAmount             int        `json:"total_amount,omitempty"`
	InvoicePayload          string     `json:"invoice_payload,omitempty"`
	ShippingOptionID        string     `json:"shipping_option_id,omitempty,omitempty"`
	OrderInfo               *OrderInfo `json:"order_info,omitempty,omitempty"`
	TelegramPaymentChargeID string     `json:"telegram_payment_charge_id,omitempty"`
	ProviderPaymentChargeID string     `json:"provider_payment_charge_id,omitempty"`
}

// ShippingQuery contains information about an incoming shipping query.
type ShippingQuery struct {
	ID              string           `json:"id,omitempty"`
	From            *User            `json:"from,omitempty"`
	InvoicePayload  string           `json:"invoice_payload,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// PreCheckoutQuery contains information about an incoming pre-checkout query.
type PreCheckoutQuery struct {
	ID               string     `json:"id,omitempty"`
	From             *User      `json:"from,omitempty"`
	Currency         string     `json:"currency,omitempty"`
	TotalAmount      int        `json:"total_amount,omitempty"`
	InvoicePayload   string     `json:"invoice_payload,omitempty"`
	ShippingOptionID string     `json:"shipping_option_id,omitempty,omitempty"`
	OrderInfo        *OrderInfo `json:"order_info,omitempty,omitempty"`
}

// Error is an error containing extra information returned by the Telegram API.
type Error struct {
	Message string
	ResponseParameters
}

func (e Error) Error() string {
	return e.Message
}
