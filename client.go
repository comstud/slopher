package slopher

import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
)

type Client struct {
    Uri       string
    AuthToken string
    log       *log.Logger
}

func NewClient(uri string, auth_token string, logger *log.Logger) *Client {
    return &Client{Uri: uri, AuthToken: auth_token, log: logger}
}

type RTMStartResponse struct {
    raw           []byte     `json:"-"`

    Ok              bool     `json:"ok"`
    WSUrl           string   `json:"url"`
    CacheVersion    string   `json:"cache_version"`
    LatestTimeStamp string   `json:"latest_event_ts"`

    Self           *Self     `json:"self,omitempty"`
    Bots          []*Bot     `json:"bots"`
    Users         []*User    `json:"users"`
    Channels      []*Channel `json:"channels"`
    IMs           []*Channel `json:"ims"`
    Groups        []*Group   `json:"groups"`
    Team           *Team     `json:"team"`
}

func (self *Client) RTMStart() (*RTMStartResponse, error) {
    rtm_resp := &RTMStartResponse{}

    if err := self.apiCall("rtm.start", apiArgs{}, rtm_resp); err != nil {
        return nil, err
    }

    return rtm_resp, nil
}

func (self *Client) NewRTMProcessor(config *RTMConfig) (*RTMProcessor, error) {
    rtm_resp, err := self.RTMStart()
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
        Client:    self,
        RTMConfig: config,
        WSUrl:     rtm_resp.WSUrl,
        doneChan:  make(chan int, 1),
        seq_id:    1,
    }

    rtm.initHooks()
    rtm.StateManager.RTMStart(rtm, rtm_resp)

    return rtm, nil
}

type JoinChannelResponse struct {
    raw     []byte   `json:"-"`

    Ok       bool    `json:"ok"`
    Channel *Channel `json:"channel"`
}

func (self *JoinChannelResponse) SetRaw(data []byte) {
    self.raw = data
}

func (self *JoinChannelResponse) GetRaw() []byte {
    return self.raw
}

func (self *Client) JoinChannel(name string) (*JoinChannelResponse, error) {
    resp := &JoinChannelResponse{}
    err := self.apiCall("channels.join", apiArgs{"name": name}, resp)
    if err != nil {
        return nil, err
    }

    return resp, nil
}

type LeaveChannelResponse struct {
    raw     []byte   `json:"-"`

    Ok       bool    `json:"ok"`
    /* Returns other keys on error */
}

func (self *LeaveChannelResponse) SetRaw(data []byte) {
    self.raw = data
}

func (self *LeaveChannelResponse) GetRaw() []byte {
    return self.raw
}

func (self *Client) LeaveChannel(id string) (*LeaveChannelResponse, error) {
    resp := &LeaveChannelResponse{}
    err := self.apiCall("channels.leave", apiArgs{"channel": id}, resp)
    if err != nil {
        return nil, err
    }

    return resp, nil
}

// Private methods
type apiArgs map[string]string
type apiResponse interface {
    SetRaw([]byte)
    GetRaw() []byte
}


func (self *Client) apiCall(method string, args apiArgs, apiresp apiResponse) error {
    if self.Uri == "" {
        self.Uri = "https://slack.com/api"
    }

    // FIXME: Don't manually generate this!
    full_uri := self.Uri + fmt.Sprintf("/%s?token=%s", method, self.AuthToken)
    for k, v := range(args) {
        full_uri += fmt.Sprintf("&%s=%s", k, v)
    }

    resp, err := http.Get(full_uri)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    self.log.Printf("apiCall(%s) response: %s\n", method, body)

    if err := json.Unmarshal(body, apiresp); err != nil {
        return err
    }

    apiresp.SetRaw(body)
    return nil
}

