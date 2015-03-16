package slopher

type Entity struct {
    Name     string
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

func (self *Entity) addPlace(place *Place) *Place {
    self.PlacesByID[place.ID] = place
    return place
}

func (self *Entity) delPlace(place *Place) *Place {
    delete(self.PlacesByID, place.ID)
    return place
}

type Place struct {
    ID       string
    Name     string // Name of target user or channel name prefixed with "#"
    *Channel
}

func (self *Place)SendMessage(rtm *RTMProcessor, msg string) {
    rtm.SendChannelMessage(self.ID, msg)
}

type RTMStateManager interface {
    AddHooks(*RTMProcessor)
    RTMStart(*RTMProcessor, *RTMStartResponse)
}

func GetDefaultStateManager() *DefaultStateManager {
    return &DefaultStateManager{}
}

type DefaultStateManager struct {

    Team            *Team
    Self            *Self
    Bots            []*Bot
    Users           []*User
    Channels        []*Channel
    IMs             []*Channel
    Groups          []*Group

    EntitiesByID    map[string]*Entity
    EntitiesByName  map[string]*Entity
    PlacesByID      map[string]*Place
    PlacesByName    map[string]*Place
}

func (self *DefaultStateManager) addEntity(entity *Entity) *Entity {
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

func (self *DefaultStateManager) addEntityFromUser(user *User) *Entity {
    // Bots also have User records, so we want to combine them into the
    // same Entity. Lookup by Name to find.
    entity := self.FindEntityByName(user.Name)
    if entity != nil {
        entity.User = user
        return self.addEntity(entity)
    }
    return self.addEntity(&Entity{
        Name: user.Name,
        User: user,
        PlacesByID: make(map[string]*Place),
    })
}

func (self *DefaultStateManager) addEntityFromSelf(selfobj *Self) *Entity {
    entity := self.FindEntityByName(selfobj.Name)
    if entity != nil {
        entity.Self = selfobj
        return self.addEntity(entity)
    }
    return self.addEntity(&Entity{
        Name: selfobj.Name,
        Self: selfobj,
        PlacesByID: make(map[string]*Place),
    })
}

func (self *DefaultStateManager) addEntityFromBot(bot *Bot) *Entity {
    // Bots also have User records, so we want to combine them into the
    // same Entity. Lookup by Name to find.
    entity := self.FindEntityByName(bot.Name)
    if entity != nil {
        entity.Bot = bot
        return self.addEntity(entity)
    }
    return self.addEntity(&Entity{
        Name: bot.Name,
        Bot: bot,
        PlacesByID: make(map[string]*Place),
    })
}

func (self *DefaultStateManager) addPlace(place *Place) *Place {
    self.PlacesByID[place.ID] = place
    self.PlacesByName[place.Name] = place
    if place.Channel != nil {
        for _, user_id := range(place.Members) {
            entity := self.FindEntity(user_id)
            if entity != nil {
                entity.addPlace(place)
            }
        }
    }
    return place
}

func (self *DefaultStateManager) addPlaceFromChannel(channel *Channel) *Place {
    name := channel.Name
    if channel.IsChannel {
        name = "#" + name
    } else if channel.IsIM {
        entity := self.FindEntity(channel.UserID)
        if entity == nil {
            name = channel.UserID
        } else {
            name = entity.Name
        }
    } else {
        name = channel.ID
    }

    return self.addPlace(&Place{
        ID: channel.ID,
        Name: name,
        Channel: channel,
    })
}

func (self *DefaultStateManager) FindEntity(id string) *Entity {
    return self.EntitiesByID[id]
}

func (self *DefaultStateManager) FindEntityByName(name string) *Entity {
    return self.EntitiesByName[name]
}

func (self *DefaultStateManager) FindPlace(id string) *Place {
    return self.PlacesByID[id]
}

func (self *DefaultStateManager) FindPlaceByName(name string) *Place {
    return self.PlacesByName[name]
}

func (self *DefaultStateManager) AddHooks(rtm *RTMProcessor) {
    rtm.addHook("team_join", func(rtm *RTMProcessor, _msg RTMMessage) {
        msg := _msg.(*RTMTeamJoinMessage)
        self.addEntityFromUser(msg.User)
    })

    rtm.addHook("channel_created", func(rtm *RTMProcessor, _msg RTMMessage) {
        msg := _msg.(*RTMChannelCreatedMessage)
        // Make sure these are set
        msg.Channel.IsChannel = true
        self.addPlaceFromChannel(msg.Channel)
    })

    rtm.addHook("im_created", func(rtm *RTMProcessor, _msg RTMMessage) {
        msg := _msg.(*RTMIMCreatedMessage)
        // Make sure these are set
        msg.Channel.IsIM = true
        msg.Channel.UserID = msg.UserID
        self.addPlaceFromChannel(msg.Channel)
    })
}

func (self *DefaultStateManager) RTMStart(rtm *RTMProcessor, resp *RTMStartResponse) {
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
    for _, user := range(self.Users) {
        self.addEntityFromUser(user)
    }
    for _, bot := range(self.Bots) {
        self.addEntityFromBot(bot)
    }

    for _, channel := range(self.Channels) {
        self.addPlaceFromChannel(channel)
    }
    for _, channel := range(self.IMs) {
        self.addPlaceFromChannel(channel)
    }
}
