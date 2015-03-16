package slopher

type RTMMessage interface {
    apiResponse
    Process(*RTMProcessor)
}

var rtmMessageTypeToObj = map[string]RTMMessage{
    "message":         &RTMChannelMessage{},
    "user_typing":     &RTMTypingMessage{},
    "team_join":       &RTMTeamJoinMessage{},
    "channel_created": &RTMChannelCreatedMessage{},
    "channel_joined":  &RTMChannelJoinedMessage{},
    "channel_left":    &RTMChannelLeftMessage{},
    "im_created":      &RTMIMCreatedMessage{},
}

/*
** Channel message
*/

type RTMChannelMessage struct {
    raw       []byte  `json:"-"`

    Type      string  `json:"type"`
    ReplyTo   *int    `json:"reply_to"`
    *Message
}

func (self *RTMChannelMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMChannelMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMChannelMessage) Process(rtm *RTMProcessor) {
    // Ignore all replies
    if self.ReplyTo != nil {
        return
    }
    rtm.runHooks(self.Type, self)
}

/*
** Typing message
*/
type RTMTypingMessage struct {
    raw       []byte    `json:"-"`

    Type        string  `json:"type"`
    UserID      string  `json:"user"`
    ChannelID   string  `json:"channel"`
}

func (self *RTMTypingMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMTypingMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMTypingMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}

/*
** Team join message
*/
type RTMTeamJoinMessage struct {
    raw       []byte    `json:"-"`

    Type        string  `json:"type"`
    User        *User   `json:"user"`
}

func (self *RTMTeamJoinMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMTeamJoinMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMTeamJoinMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}

/*
** Channel created
*/
type RTMChannelCreatedMessage struct {
    raw       []byte     `json:"-"`

    Type        string   `json:"type"`
    Channel     *Channel `json:"channel"`
}

func (self *RTMChannelCreatedMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMChannelCreatedMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMChannelCreatedMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}

/*
** Channel joined
*/
type RTMChannelJoinedMessage struct {
    raw       []byte     `json:"-"`

    Type        string   `json:"type"`
    Channel     *Channel `json:"channel"`
}

func (self *RTMChannelJoinedMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMChannelJoinedMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMChannelJoinedMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}

/*
** Channel left
*/
type RTMChannelLeftMessage struct {
    raw       []byte     `json:"-"`

    Type        string   `json:"type"`
    ChannelID   string   `json:"channel"`
}

func (self *RTMChannelLeftMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMChannelLeftMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMChannelLeftMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}

/*
** IM created
*/
type RTMIMCreatedMessage struct {
    raw       []byte     `json:"-"`

    Type        string   `json:"type"`
    UserID      string   `json:"user"`
    Channel    *Channel  `json:"channel"`
}

func (self *RTMIMCreatedMessage) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMIMCreatedMessage) GetRaw() []byte {
    return self.raw
}

func (self *RTMIMCreatedMessage) Process(rtm *RTMProcessor) {
    rtm.runHooks(self.Type, self)
}