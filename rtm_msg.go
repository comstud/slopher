package slopher

import (
	"fmt"
	"golang.org/x/net/context"
)

type RTMMessage interface {
	rawJSONSupporter
	Process(context.Context)
}

var rtmMessageTypeToObj = map[string]RTMMessage{
	"message":         &RTMChannelMessage{},
	"user_typing":     &RTMTypingMessage{},
	"bot_added":       &RTMBotAddedMessage{},
	"team_join":       &RTMTeamJoinMessage{},
	"channel_created": &RTMChannelCreatedMessage{},
	"channel_joined":  &RTMChannelJoinedMessage{},
	"channel_left":    &RTMChannelLeftMessage{},
	"im_created":      &RTMIMCreatedMessage{},
	"user_change":     &RTMUserChangedMessage{},
}

var rtmMessageSubTypeHooks = []string{
	"message_changed",
}

/*
** Channel message
 */

type RTMChannelMessage struct {
	rawJSON

	Message
}

func (self *RTMChannelMessage) Process(ctx context.Context) {
	// Ignore all replies
	if self.ReplyTo != nil {
		return
	}

	if self.SubType == "" || self.SubType == "bot_message" ||
		self.SubType == "me_message" {
		// Treat these all as channel messages.
		if self.BotMessage != nil {
			// Copy this... should already be set for other messages
			self.UserID = self.BotMessage.BotID
		}

		runRTMHooks(ctx, "message", self)
		return
	}

	// Handle subtypes :-/
	if self.SubType == "message_changed" {
		runRTMHooks(ctx, "message_changed", self)
		return
	}

	fmt.Printf("Dropping subtyped message: %s %+v\n", self.raw, *self)

	return
}

/*
** Typing message
 */
type RTMTypingMessage struct {
	rawJSON

	Type      string `json:"type"`
	UserID    string `json:"user"`
	ChannelID string `json:"channel"`
}

func (self *RTMTypingMessage) SetRaw(data []byte) {
	self.raw = data
}

func (self *RTMTypingMessage) GetRaw() []byte {
	return self.raw
}

func (self *RTMTypingMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** Team join message
 */
type RTMTeamJoinMessage struct {
	rawJSON

	Type string `json:"type"`
	User *User  `json:"user"`
}

func (self *RTMTeamJoinMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** Bot added message
 */
type RTMBotAddedMessage struct {
	rawJSON

	Type string `json:"type"`
	Bot  *Bot   `json:"bot"`
}

func (self *RTMBotAddedMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** Channel created
 */
type RTMChannelCreatedMessage struct {
	rawJSON

	Type    string   `json:"type"`
	Channel *Channel `json:"channel"`
}

func (self *RTMChannelCreatedMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** Channel joined
 */
type RTMChannelJoinedMessage struct {
	rawJSON

	Type    string   `json:"type"`
	Channel *Channel `json:"channel"`
}

func (self *RTMChannelJoinedMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** Channel left
 */
type RTMChannelLeftMessage struct {
	rawJSON

	Type      string `json:"type"`
	ChannelID string `json:"channel"`
}

func (self *RTMChannelLeftMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** IM created
 */
type RTMIMCreatedMessage struct {
	rawJSON

	Type    string   `json:"type"`
	UserID  string   `json:"user"`
	Channel *Channel `json:"channel"`
}

func (self *RTMIMCreatedMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}

/*
** User changed
 */
type RTMUserChangedMessage struct {
	rawJSON

	Type string `json:"type"`
	User *User  `json:"user"`
}

func (self *RTMUserChangedMessage) Process(ctx context.Context) {
	runRTMHooks(ctx, self.Type, self)
}
