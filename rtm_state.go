package slopher

import (
	"errors"

	"golang.org/x/net/context"
)

type Entity struct {
	Name string
	*Bot
	*User
	*Self

	PlacesByID map[string]*Place
}

func (self *Entity) IsBot() bool {
	if self.Self != nil || self.Bot != nil {
		return true
	}
	// User must not be nil
	return self.User.IsBot
}

func (self *Entity) IsSelf() bool {
	return self.Self != nil
}

func (self *Entity) GetID() string {
	if self.IsBot() {
		return self.Bot.ID
	} else if self.IsSelf() {
		return self.Self.ID
	}
	return self.User.ID
}

func (self *Entity) addPlace(place *Place) *Place {
	self.PlacesByID[place.ID] = place
	return place
}

func (self *Entity) delPlace(place *Place) *Place {
	delete(self.PlacesByID, place.ID)
	return place
}

type Place struct {
	ID        string
	Name      string // Name of target user or channel name prefixed with "#"
	IsChannel bool
	IsIM      bool
	IsGroup   bool
	*Channel
	*Group
	*IM
}

func (self *Place) SendMessageString(ctx context.Context, s string) error {
	msg := &Message{
		BaseMessage: BaseMessage{
			ChannelID: self.ID,
			Type:      "message",
			Text:      s,
		},
	}
	return self.SendMessage(ctx, msg)
}

func (self *Place) SendMessage(ctx context.Context, msg *Message) error {
	rtm, ok := RTMProcessorFromContext(ctx)
	if !ok {
		return errors.New("No RTMProcessor in context")
	}
	return rtm.SendWSMessage(ctx, msg)
}

type RTMStateManager interface {
	AddHooks(context.Context) error
	RTMStart(context.Context, *RTMStartResponse) error
}

func NewContextForStateManager(ctx context.Context, state_mgr RTMStateManager) context.Context {
	return context.WithValue(ctx, rtmStateManagerKey, state_mgr)
}

func StateManagerFromContext(ctx context.Context) (RTMStateManager, bool) {
	u, ok := ctx.Value(rtmStateManagerKey).(RTMStateManager)
	return u, ok
}

func GetDefaultStateManager() *StateManager {
	return &StateManager{}
}

type StateManager struct {
	Team     *Team
	Self     *Self
	Bots     []*Bot
	Users    []*User
	Channels []*Channel
	IMs      []*IM
	Groups   []*Group

	EntitiesByID   map[string]*Entity
	EntitiesByName map[string]*Entity
	PlacesByID     map[string]*Place
	PlacesByName   map[string]*Place
}

func (self *StateManager) addEntity(entity *Entity) *Entity {
	if entity.User != nil {
		self.EntitiesByID[entity.User.ID] = entity
	}
	if entity.Bot != nil {
		self.EntitiesByID[entity.Bot.ID] = entity
	}
	if entity.Self != nil {
		self.EntitiesByID[entity.Self.ID] = entity
	}
	self.EntitiesByName[entity.Name] = entity
	return entity
}

func (self *StateManager) addEntityFromUser(user *User) *Entity {
	// Bots also have User records, so we want to combine them into the
	// same Entity. Lookup by Name to find.
	entity := self.FindEntityByName(user.Name)
	if entity != nil {
		entity.User = user
		return self.addEntity(entity)
	}
	return self.addEntity(&Entity{
		Name:       user.Name,
		User:       user,
		PlacesByID: make(map[string]*Place),
	})
}

func (self *StateManager) addEntityFromSelf(selfobj *Self) *Entity {
	entity := self.FindEntityByName(selfobj.Name)
	if entity != nil {
		entity.Self = selfobj
		return self.addEntity(entity)
	}
	return self.addEntity(&Entity{
		Name:       selfobj.Name,
		Self:       selfobj,
		PlacesByID: make(map[string]*Place),
	})
}

func (self *StateManager) addEntityFromBot(bot *Bot) *Entity {
	// Bots also have User records, so we want to combine them into the
	// same Entity. Lookup by Name to find.
	entity := self.FindEntityByName(bot.Name)
	if entity != nil {
		entity.Bot = bot
		return self.addEntity(entity)
	}
	return self.addEntity(&Entity{
		Name:       bot.Name,
		Bot:        bot,
		PlacesByID: make(map[string]*Place),
	})
}

func (self *StateManager) addPlace(place *Place) *Place {
	self.PlacesByID[place.ID] = place
	self.PlacesByName[place.Name] = place

	members := make([]string, 0)

	if place.Channel != nil {
		members = place.Channel.Members
	} else if place.Group != nil {
		members = place.Group.Members
	}

	for _, user_id := range members {
		if entity := self.FindEntity(user_id); entity != nil {
			entity.addPlace(place)
		}
	}
	return place
}

func (self *StateManager) addPlaceFromChannel(channel *Channel) *Place {
	return self.addPlace(&Place{
		ID:        channel.ID,
		Name:      "#" + channel.Name,
		Channel:   channel,
		IsChannel: true,
	})
}

func (self *StateManager) addPlaceFromGroup(group *Group) *Place {
	return self.addPlace(&Place{
		ID:      group.ID,
		Name:    "#" + group.Name,
		Group:   group,
		IsGroup: true,
	})
}

func (self *StateManager) addPlaceFromIM(im *IM) *Place {
	var name string
	// We want to use the User's name for the name of Place, if it exists.
	if entity := self.FindEntity(im.UserID); entity != nil {
		name = entity.Name
	} else {
		name = im.UserID
	}
	return self.addPlace(&Place{
		ID:   im.ID,
		Name: name,
		IM:   im,
		IsIM: true,
	})
}

func (self *StateManager) FindEntity(id string) *Entity {
	return self.EntitiesByID[id]
}

func (self *StateManager) FindEntityByName(name string) *Entity {
	return self.EntitiesByName[name]
}

func (self *StateManager) FindPlace(id string) *Place {
	return self.PlacesByID[id]
}

func (self *StateManager) FindPlaceByName(name string) *Place {
	return self.PlacesByName[name]
}

func (self *StateManager) AddHooks(ctx context.Context) error {
	rtm, ok := RTMProcessorFromContext(ctx)
	if !ok {
		return errors.New("No RTMProcessor in context")
	}

	rtm.addHook("team_join", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMTeamJoinMessage)
		self.addEntityFromUser(msg.User)
	})

	rtm.addHook("bot_added", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMBotAddedMessage)
		self.addEntityFromBot(msg.Bot)
	})

	rtm.addHook("user_change", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMUserChangedMessage)

		// Name might have changed, so find by ID first.
		entity := self.FindEntity(msg.User.ID)

		if entity != nil && entity.Name != msg.User.Name {
			// Handle name change.
			delete(self.EntitiesByName, entity.Name)
			self.EntitiesByName[msg.User.Name] = entity
		}
		self.addEntityFromUser(msg.User)
	})

	rtm.addHook("channel_created", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMChannelCreatedMessage)
		// Make sure these are set
		msg.Channel.IsChannel = true
		self.addPlaceFromChannel(msg.Channel)
	})

	rtm.addHook("im_created", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMIMCreatedMessage)
		// Make sure these are set
		msg.IM.IsIM = true
		msg.IM.UserID = msg.UserID
		self.addPlaceFromIM(msg.IM)
	})

	rtm.addHook("group_joined", func(ctx context.Context, _msg RTMMessage) {
		msg := _msg.(*RTMGroupJoinedMessage)
		// Make sure this is set
		msg.Group.IsGroup = true
		self.addPlaceFromGroup(msg.Group)
	})

	return nil
}

func (self *StateManager) RTMStart(ctx context.Context, resp *RTMStartResponse) error {
	self.Team = resp.Team
	self.Self = resp.Self
	self.Bots = resp.Bots
	self.Users = resp.Users
	self.Channels = resp.Channels
	self.IMs = resp.IMs
	self.Groups = resp.Groups

	self.EntitiesByID = make(map[string]*Entity)
	self.EntitiesByName = make(map[string]*Entity)
	self.PlacesByID = make(map[string]*Place)
	self.PlacesByName = make(map[string]*Place)

	self.addEntityFromSelf(self.Self)

	// Must do users before channels
	for _, user := range self.Users {
		self.addEntityFromUser(user)
	}
	for _, bot := range self.Bots {
		self.addEntityFromBot(bot)
	}

	for _, channel := range self.Channels {
		self.addPlaceFromChannel(channel)
	}
	for _, im := range self.IMs {
		self.addPlaceFromIM(im)
	}
	for _, group := range self.Groups {
		self.addPlaceFromGroup(group)
	}

	return nil
}
