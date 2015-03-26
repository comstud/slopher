package slopher

import (
	"bytes"
	"encoding/json"
	//	"errors"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	//	"net/textproto"
	//	"net/url"
	"strings"
)

const DEFAULT_URI = "https://slack.com/api"

type contextKeyType int

var clientKey contextKeyType = 0
var rtmProcessorKey contextKeyType = 1
var rtmStateManagerKey contextKeyType = 2

type Client struct {
	transport *http.Transport
	client    *http.Client
	Uri       string
	AuthToken string
	log       *log.Logger
}

type APIArgs map[string]string

type rawJSONSupporter interface {
	SetRaw(data []byte)
	GetRaw() []byte
}

type rawJSON struct {
	raw []byte `json:"-"`
}

func (self *rawJSON) SetRaw(data []byte) {
	self.raw = data
}

func (self *rawJSON) GetRaw() []byte {
	return self.raw
}

type baseAPIResponse struct {
	rawJSON
	Ok bool `json:"ok"`
}

func NewClient(uri string, auth_token string, logger *log.Logger) *Client {
	if uri == "" {
		uri = DEFAULT_URI
	}
	tr := &http.Transport{}
	return &Client{
		transport: tr,
		client:    &http.Client{Transport: tr},
		Uri:       uri,
		AuthToken: auth_token,
		log:       logger,
	}
}

func (self *Client) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, clientKey, self)
}

func ClientFromContext(ctx context.Context) (*Client, bool) {
	u, ok := ctx.Value(clientKey).(*Client)
	return u, ok
}

type RTMStartResponse struct {
	baseAPIResponse

	WSUrl           string `json:"url"`
	CacheVersion    string `json:"cache_version"`
	LatestTimeStamp string `json:"latest_event_ts"`

	Self     *Self      `json:"self,omitempty"`
	Bots     []*Bot     `json:"bots"`
	Users    []*User    `json:"users"`
	Channels []*Channel `json:"channels"`
	IMs      []*Channel `json:"ims"`
	Groups   []*Group   `json:"groups"`
	Team     *Team      `json:"team"`
}

func (self *Client) RTMStart(ctx context.Context) (*RTMStartResponse, error) {
	rtm_resp := &RTMStartResponse{}

	if err := self.apiCall(ctx, "rtm.start", nil, rtm_resp); err != nil {
		return nil, err
	}

	return rtm_resp, nil
}

type JoinChannelResponse struct {
	baseAPIResponse

	Channel *Channel `json:"channel"`
}

func (self *Client) JoinChannel(ctx context.Context, name string) (*JoinChannelResponse, error) {
	resp := &JoinChannelResponse{}
	apiargs := APIArgs{"name": name}

	err := self.apiCall(ctx, "channels.join", apiargs, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type LeaveChannelResponse struct {
	baseAPIResponse
}

func (self *Client) LeaveChannel(ctx context.Context, id string) (*LeaveChannelResponse, error) {
	resp := &LeaveChannelResponse{}
	apiargs := APIArgs{"channel": id}

	err := self.apiCall(ctx, "channels.leave", apiargs, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type PostChatMessageResponse struct {
	baseAPIResponse

	TS        string   `json:"ts"`
	ChannelID string   `json:"channel"`
	Message   *Message `json:"message"`
}

func (self *Client) PostChatMessage(ctx context.Context, args APIArgs, attachments []Attachment) (*PostChatMessageResponse, error) {
	resp := &PostChatMessageResponse{}

	if args == nil {
		args = APIArgs{}
	}

	if attachments != nil && len(attachments) > 0 {
		a, err := json.Marshal(attachments)
		if err != nil {
			return nil, err
		}
		args["attachments"] = string(a)
	}

	err := self.apiCall(ctx, "chat.postMessage", args, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type UploadFileResponse struct {
	baseAPIResponse

	File *SharedFile `json:"file"`
	/* Returns other keys on error */
}

func (self *Client) UploadFile(ctx context.Context, filename, content, filetype, title string, channels []string) (*UploadFileResponse, error) {
	resp := &UploadFileResponse{}

	args := APIArgs{
		"_filename":      filename,
		"_file_contents": content,
		"title":          title,
		"filetype":       filetype,
	}

	if channels != nil {
		args["channels"] = strings.Join(channels, ",")
	}

	err := self.apiCall(ctx, "files.upload", args, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type IMListResponse struct {
	baseAPIResponse

	IMs []Channel `json:"ims"`
}

func (self *Client) IMList(ctx context.Context) (*IMListResponse, error) {
	resp := &IMListResponse{}

	err := self.apiCall(ctx, "im.list", nil, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type ChatDeleteResponse struct {
	baseAPIResponse

	ChannelID string `json:"channel"`
	TS        string `json:"ts"`
}

func (self *Client) ChatDelete(ctx context.Context, channel_id, ts string) (*ChatDeleteResponse, error) {
	resp := &ChatDeleteResponse{}

	args := APIArgs{
		"channel": channel_id,
		"ts":      ts,
	}

	err := self.apiCall(ctx, "chat.delete", args, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Private methods
func (self *Client) apiCall(ctx context.Context, method string, args APIArgs, apiresp rawJSONSupporter) error {
	full_uri := self.Uri + fmt.Sprintf("/%s?token=%s", method, self.AuthToken)

	self.log.Printf("apiCall(%s) sending: %+v\n", method, args)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if filename, ok := args["_filename"]; ok {
		contents := args["_file_contents"]

		/*
			var qE = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition",
					fmt.Sprintf(`form-data; name="file"; filename="%s"`,
						qE.Replace(filename[0])))
				h.Set("Content-Type", "text/html")

				fw, err := w.CreatePart(h)
		*/
		fw, err := w.CreateFormFile("file", filename)
		if err != nil {
			return err
		}

		if _, err := fw.Write([]byte(contents)); err != nil {
			return err
		}
		delete(args, "_filename")
		delete(args, "_file_contents")
	}

	for k, v := range args {
		ff, err := w.CreateFormField(k)
		if err != nil {
			return err
		}
		if _, err := ff.Write([]byte(v)); err != nil {
			return err
		}
	}

	w.Close()

	req, err := http.NewRequest("POST", full_uri, &b)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	var body []byte

	errch := make(chan error, 1)

	go func() {
		resp, err := self.client.Do(req)
		if err != nil {
			errch <- err
			return
		}

		defer resp.Body.Close()

		body, err = ioutil.ReadAll(resp.Body)
		errch <- err
	}()

	select {
	case <-ctx.Done():
		self.transport.CancelRequest(req)
		<-errch
		return ctx.Err()
	case err := <-errch:
		if err != nil {
			return err
		}
	}

	self.log.Printf("apiCall(%s) response: %s\n", method, body)

	if err := json.Unmarshal(body, apiresp); err != nil {
		return err
	}

	apiresp.SetRaw(body)
	return nil
}
