package slopher

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "github.com/gorilla/websocket"
    "reflect"
    "time"
)

var WSMessageTypeToName = map[int]string{
    websocket.TextMessage:   "Text",
    websocket.BinaryMessage: "Binary",
    websocket.CloseMessage:  "Close",
    websocket.PingMessage:   "Ping",
    websocket.PongMessage:   "Pong",
}

type RTMHook func(*RTMProcessor, RTMMessage)

func (self *RTMStartResponse) SetRaw(data []byte) {
    self.raw = data
}

func (self *RTMStartResponse) GetRaw() []byte {
    return self.raw
}

type RTMConfig struct {
    StateManager    RTMStateManager
    AutoReconnect   bool
}

type RTMProcessor struct {
    ws          *websocket.Conn
    doneChan    chan int

    *Client
    *RTMConfig
    WSUrl       string
    Running     bool
    Stopping    bool

    hooks       map[string][]RTMHook
    seq_id      int
}

func GetDefaultRTMConfig() *RTMConfig {
    return &RTMConfig{
        StateManager:  GetDefaultStateManager(),
        AutoReconnect: true,
    }
}

func (self *RTMProcessor) initHooks() {
    self.hooks = make(map[string][]RTMHook)
    for mtype, _ := range(rtmMessageTypeToObj) {
        self.hooks[mtype] = make([]RTMHook, 0)
    }
    self.StateManager.AddHooks(self)
}

func (self *RTMProcessor) runHooks(name string, msg RTMMessage) {
    for _, hook := range(self.hooks[name]) {
        hook(self, msg)
    }
}

func (self *RTMProcessor) wsConnect() (*websocket.Conn, error) {
    var hdr http.Header;
    dialer := websocket.DefaultDialer;
    conn, _, err := dialer.Dial(self.WSUrl, hdr)
    if err != nil {
        return nil, fmt.Errorf("Couldn't connect to %s: %s",
            self.WSUrl, err)
    }
    return conn, nil
}

func (self *RTMProcessor) wsReconnect() (*websocket.Conn, error) {
    var delay time.Duration
    var err error

    for {
        if self.Stopping {
            return nil, nil
        }
        self.log.Print("Attempting reconnect (start.rtm)...")
        if rtm_resp, err := self.RTMStart(); err == nil {
            if !rtm_resp.Ok {
                err = errors.New("Received !Ok response")
            } else if rtm_resp.WSUrl == "" {
                err = errors.New("Websocket URL is empty")
            } else {
                self.WSUrl = rtm_resp.WSUrl
                if conn, err := self.wsConnect(); err == nil {
                    self.StateManager.RTMStart(self, rtm_resp)
                    return conn, nil
                }
            }
        }
        delay += time.Second + (delay / 2)
        if delay > 30 {
            delay = 30
        }
        self.log.Print("Reconnect failed (will try again after %s): %s",
            delay, err)
        time.Sleep(delay)
    }
}

func (self *RTMProcessor) GetDoneChannel() chan int {
    return self.doneChan
}

func (self *RTMProcessor) Start() error {
    if self.Running {
        return errors.New("RTMProcessor already running")
    }

    var conn *websocket.Conn

    conn, err := self.wsConnect()
    if err != nil {
        return err
    }

    self.ws = conn
    self.Running = true

    go func() {
        for {
            if self.Stopping {
                break
            }

            messageType, data, err := conn.ReadMessage()
            if err != nil {
                self.log.Printf("Closing RTM connection to %s due to " +
                    "read error: %s", self.WSUrl, err)
                self.ws = nil
                conn.Close()
                if !self.AutoReconnect {
                    break
                }

                conn, err = self.wsReconnect()
                if err != nil {
                    break
                }
                continue
            }

            smsg, ok := WSMessageTypeToName[messageType]
            if !ok {
                smsg = "Unknown"
            }

            self.log.Printf("Got message (%s) from WebSocket: %s\n", smsg,
                data)

            if messageType != websocket.TextMessage {
                continue
            }

            // First decode JSON just to get the message type
            mtype := &struct {
                Type string `json:"type"`
            }{}

            if err = json.Unmarshal(data, mtype); err != nil {
                self.log.Printf("Error decoding Websocket message: %s\n",
                    err)
                continue
            }

            obj_ptr, ok := rtmMessageTypeToObj[mtype.Type]
            if !ok {
                self.log.Printf("Warning: ignoring unknown message type: %s\n",
                    mtype.Type)
                continue
            }
            ref_val := reflect.ValueOf(obj_ptr).Elem()
            nobj_val := reflect.New(ref_val.Type())

            nobj := nobj_val.Interface().(RTMMessage)
            nobj.SetRaw(data)
            if err = json.Unmarshal(data, nobj); err != nil {
                self.log.Printf("Error decoding Websocket message: %s\n",
                    err)
                continue
            }

            nobj.Process(self)
        }
        self.Running = false
        if self.ws != nil {
            self.ws.Close()
        }
        self.doneChan <- 1
    }()

    return nil
}

func (self *RTMProcessor) Stop(wait bool) {
    if !self.Running {
        return
    }

    self.Stopping = true
    self.log.Printf("Writing Close message\n")
    err := self.ws.WriteControl(websocket.CloseMessage, make([]byte, 0), time.Time{})
    if err != nil {
        self.log.Printf("Failed to write close message: %s\n", err)
        return
    }

    if !wait {
        return
    }

    <-self.doneChan
}

func (self *RTMProcessor) addHook(name string, fn RTMHook) {
    self.hooks[name] = append(self.hooks[name], fn)
}

func (self *RTMProcessor) OnChannelMessage(fn RTMHook) {
    self.addHook("message", fn)
}

func (self *RTMProcessor) OnTyping(fn RTMHook) {
    self.addHook("user_typing", fn)
}

func (self *RTMProcessor) OnTeamJoin(fn RTMHook) {
    self.addHook("team_join", fn)
}

func (self *RTMProcessor) OnChannelCreated(fn RTMHook) {
    self.addHook("channel_created", fn)
}

func (self *RTMProcessor) OnChannelJoined(fn RTMHook) {
    self.addHook("channel_joined", fn)
}

func (self *RTMProcessor) OnChannelLeft(fn RTMHook) {
    self.addHook("channel_left", fn)
}

func (self *RTMProcessor) OnIMCreated(fn RTMHook) {
    self.addHook("im_created", fn)
}

/* TODO: Need to make sure these are serialized */
func (self *RTMProcessor) sendMessage(v interface {}) {
    err := self.ws.WriteJSON(v)
    if err != nil {
        self.log.Printf("Got error sending %+v: %s\n", v, err)
    }
}

// Caller responsible for escaping &, <, and >.
func (self *RTMProcessor) SendChannelMessage(target string, text string) {
    // Race
    seq_id := self.seq_id
    self.seq_id++

    v := map[string]interface {}{
        "id": seq_id,
        "type": "message",
        "channel": target,
        "text": text,
    }
    self.sendMessage(v)
}
