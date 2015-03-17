package slopher

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
	"log"
	"net/http"
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

type RTMHook func(context.Context, RTMMessage)

func (self *RTMStartResponse) SetRaw(data []byte) {
	self.raw = data
}

func (self *RTMStartResponse) GetRaw() []byte {
	return self.raw
}

type RTMProcessor struct {
	done          chan struct{}
	ws            *websocket.Conn
	log           *log.Logger
	hooks         map[string][]RTMHook
	seq_id        int64
	WSUrl         string
	Running       bool
	Stopping      bool
	AutoReconnect bool
}

func (self *RTMProcessor) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, rtmProcessorKey, self)
}

func RTMProcessorFromContext(ctx context.Context) (*RTMProcessor, bool) {
	rtm, ok := ctx.Value(rtmProcessorKey).(*RTMProcessor)
	return rtm, ok
}

func NewRTMProcessor(ctx context.Context) (*RTMProcessor, error) {
	cli, ok := ClientFromContext(ctx)
	if !ok {
		return nil, errors.New("No Client found in context")
	}
	state_mgr, ok := StateManagerFromContext(ctx)
	if !ok {
		return nil, errors.New("No RTMStateManager found in context")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rtm_resp, err := cli.RTMStart(ctx)
	if err != nil {
		return nil, err
	}

	if !rtm_resp.Ok {
		return nil, errors.New("Received !Ok response")
	}

	if rtm_resp.WSUrl == "" {
		return nil, errors.New("Websocket URL is empty")
	}

	rtm := &RTMProcessor{
		done:          make(chan struct{}),
		log:           cli.log,
		seq_id:        1,
		WSUrl:         rtm_resp.WSUrl,
		AutoReconnect: true,
	}

	ctx = rtm.NewContext(ctx)

	if err := rtm.initHooks(ctx); err != nil {
		return nil, err
	}

	if err := state_mgr.RTMStart(ctx, rtm_resp); err != nil {
		return nil, err
	}

	return rtm, nil
}

func (self *RTMProcessor) initHooks(ctx context.Context) error {
	state_mgr, ok := StateManagerFromContext(ctx)
	if !ok {
		return errors.New("No RTMStateManager found in context")
	}
	self.hooks = make(map[string][]RTMHook)
	for mtype, _ := range rtmMessageTypeToObj {
		self.hooks[mtype] = make([]RTMHook, 0)
	}
	return state_mgr.AddHooks(ctx)
}

func runRTMHooks(ctx context.Context, name string, msg RTMMessage) {
	rtm, _ := RTMProcessorFromContext(ctx)
	for _, hook := range rtm.hooks[name] {
		hook(ctx, msg)
	}
}

func (self *RTMProcessor) wsConnect(ctx context.Context) (*websocket.Conn, error) {
	var hdr http.Header
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(self.WSUrl, hdr)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to %s: %s",
			self.WSUrl, err)
	}
	return conn, nil
}

func (self *RTMProcessor) wsReconnect(ctx context.Context) (*websocket.Conn, error) {
	cli, ok := ClientFromContext(ctx)
	if !ok {
		return nil, errors.New("No Client found in context")
	}

	state_mgr, ok := StateManagerFromContext(ctx)
	if !ok {
		return nil, errors.New("No RTMStateManager found in context")
	}

	var delay time.Duration
	var err error

	for {
		if self.Stopping {
			return nil, nil
		}
		self.log.Print("Attempting reconnect (start.rtm)...")
		if rtm_resp, err := cli.RTMStart(ctx); err == nil {
			if !rtm_resp.Ok {
				err = errors.New("Received !Ok response")
			} else if rtm_resp.WSUrl == "" {
				err = errors.New("Websocket URL is empty")
			} else {
				self.WSUrl = rtm_resp.WSUrl
				if conn, err := self.wsConnect(ctx); err == nil {
					state_mgr.RTMStart(ctx, rtm_resp)
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

func (self *RTMProcessor) processMessage(ctx context.Context, msgtype int, data []byte) error {
	smsg, ok := WSMessageTypeToName[msgtype]
	if !ok {
		smsg = "Unknown"
	}

	self.log.Printf("Got message (%s) from WebSocket: %s\n", smsg,
		data)

	if msgtype != websocket.TextMessage {
		return nil
	}

	// First decode JSON just to get the message type
	mtype := &struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(data, mtype); err != nil {
		self.log.Printf("Error decoding Websocket message: %s\n",
			err)
		return err
	}

	obj_ptr, ok := rtmMessageTypeToObj[mtype.Type]
	if !ok {
		self.log.Printf("Warning: ignoring unknown message type: %s\n",
			mtype.Type)
		return nil
	}
	ref_val := reflect.ValueOf(obj_ptr).Elem()
	nobj_val := reflect.New(ref_val.Type())

	nobj := nobj_val.Interface().(RTMMessage)
	nobj.SetRaw(data)
	if err := json.Unmarshal(data, nobj); err != nil {
		self.log.Printf("Error decoding Websocket message: %s\n",
			err)
		return err
	}

	nobj.Process(ctx)

	return nil
}

func (self *RTMProcessor) Start(ctx context.Context) error {
	if self.Running {
		return errors.New("RTMProcessor already running")
	}

	var conn *websocket.Conn

	conn, err := self.wsConnect(ctx)
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

			msgtype, data, err := conn.ReadMessage()
			if err != nil {
				self.log.Printf("Closing RTM connection to %s due to "+
					"read error: %s", self.WSUrl, err)
				self.ws = nil
				conn.Close()
				if !self.AutoReconnect {
					break
				}

				conn, err = self.wsReconnect(ctx)
				if err != nil {
					break
				}
				continue
			}

			self.processMessage(ctx, msgtype, data)
		}
		self.Running = false
		if self.ws != nil {
			self.ws.Close()
		}
		close(self.done)
	}()
	return nil
}

func (self *RTMProcessor) Done(ctx context.Context) chan struct{} {
	return self.done
}

func (self *RTMProcessor) Stop(ctx context.Context, wait bool) {
	if !self.Running {
		return
	}

	self.Stopping = true
	self.log.Printf("Writing Close message\n")
	if self.ws != nil {
		err := self.ws.WriteControl(websocket.CloseMessage, make([]byte, 0), time.Time{})
		if err != nil {
			self.log.Printf("Failed to write close message: %s\n", err)
			return
		}
	}

	if !wait {
		return
	}

	<-self.done
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

func (self *RTMProcessor) sendMessage(ctx context.Context, msg *Message) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		self.log.Printf("Error converting message to json %+v: %s\n",
			*msg, err)
		return err
	}

	self.log.Printf("Sending to WS Socket: %s\n", bytes)
	err = self.ws.WriteMessage(websocket.TextMessage, bytes)
	if err != nil {
		self.log.Printf("Got error sending %+v: %s\n", *msg, err)
	}
	return err
}

func (self *RTMProcessor) SendWSMessage(ctx context.Context, msg *Message) error {
	// Race
	seq_id := self.seq_id
	self.seq_id++

	msg.Id = &seq_id
	return self.sendMessage(ctx, msg)
}
