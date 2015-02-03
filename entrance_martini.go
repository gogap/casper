package casper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/gogap/logs"

	"github.com/gogap/casper/errorcode"
	"github.com/gogap/errors"
)

var (
	respInternalError  = httpRespStruct{Code: http.StatusInternalServerError, Message: "internal server error"}
	respRequestTimeout = httpRespStruct{Code: http.StatusRequestTimeout, Message: "request timeout"}

	respNotFound   = httpRespStruct{Code: http.StatusNotFound, Message: "api not found"}
	respBadRequest = httpRespStruct{Code: http.StatusBadRequest, Message: "bad request"}
	respNotAJson   = httpRespStruct{Code: http.StatusBadRequest, Message: "request data should be json struct"}
)

const (
	CTX_HTTP_COOKIES = "CTX_HTTP_COOKIES"
	CTX_HTTP_HEADERS = "CTX_HTTP_HEADERS"

	CMD_HTTP_HEADERS_SET = "CMD_HTTP_HEADERS_SET"
	CMD_HTTP_COOKIES_SET = "CMD_HTTP_COOKIES_SET"
)

const (
	DefaultAPIHeader = "X-API"
)

type EntranceToContextConf struct {
	Cookies []string `json:"cookies"`
	Headers []string `json:"headers"`
}

type EntranceMartiniConf struct {
	Host   string `json:"host"`
	Port   int32  `json:"port"`
	Domain string `json:"domain"`
	Path   string `json:"path"`

	AllowOrigin  []string              `json:"allow_origin"`
	AllowHeaders []string              `json:"allow_headers"`
	P3P          string                `json:"p3p"`
	Server       string                `json:"server"`
	ToContext    EntranceToContextConf `json:"to_context"`
	apiHeader    string                `json:"api_header"`

	allowHeaders    string            `json:"-"`
	allowOrigin     map[string]bool   `json:"-"`
	responseHeaders map[string]string `json:"-"`
}

func (p *EntranceMartiniConf) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

type EntranceMartini struct {
	config    EntranceMartiniConf
	martini   *martini.ClassicMartini
	messenger Messenger
}

type httpRespStruct struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func init() {
	entrancefactory.RegisterEntrance(new(EntranceMartini))
}

func (p *EntranceMartini) Type() string {
	return "martini"
}

func (p *EntranceMartini) Init(messenger Messenger, configs EntranceConfig) (err error) {
	if e := configs.FillToObject(&p.config); e != nil {
		err = errorcode.ERR_CONFIG_TO_OBJECT_FAILED.New(errors.Params{"err": e})
		return
	}

	p.config.allowHeaders = strings.Join(p.config.AllowHeaders, ",")
	p.config.allowOrigin = make(map[string]bool)
	for _, origin := range p.config.AllowOrigin {
		p.config.allowOrigin[origin] = true
	}

	if p.config.responseHeaders == nil {
		p.config.responseHeaders = make(map[string]string)
	}

	if p.config.P3P == "" {
		p.config.responseHeaders["P3P"] = p.config.P3P
	}

	if p.config.Server == "" {
		p.config.responseHeaders["Server"] = p.config.Server
	} else {
		p.config.responseHeaders["Server"] = "casper"
	}

	if p.config.apiHeader == "" {
		p.config.apiHeader = DefaultAPIHeader
	}

	if messenger == nil {
		err = errorcode.ERR_MESSENGER_IS_NIL.New(errors.Params{"type": p.Type()})
		return
	} else {
		p.messenger = messenger
	}
	return
}

func (p *EntranceMartini) Run() error {
	p.martini = martini.Classic()
	p.martini.Post(p.config.Path, p.postHandler())
	p.martini.Options(p.config.Path, p.optionsHandle())

	listenAddr := p.config.GetListenAddress()

	logs.Info("entrance", p.Type(), "start:", listenAddr)

	p.martini.RunOnAddr(listenAddr)

	return nil
}

func (p *EntranceMartini) setBasicHeaders(w http.ResponseWriter, r *http.Request) {
	refer := r.Referer()
	if refer == "" {
		refer = r.Header.Get("Origin")
	}

	if _, err := url.Parse(refer); err == nil {
		refProtocol, refDomain := parse_refer(refer)
		if p.config.allowOrigin["*"] ||
			p.config.allowOrigin[refDomain] {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			origin := refProtocol + "://" + refDomain
			if origin == "://" { //issue of post man, chrome limit.
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}

	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", p.config.allowHeaders)
	w.Header().Set("Content-Type", "application/json")

	for key, value := range p.config.responseHeaders {
		w.Header().Set(key, value)
	}
}

func (p *EntranceMartini) optionsHandle() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.setBasicHeaders(w, r)
	}
}

func writeJson(respObj interface{}, w http.ResponseWriter) {
	if bJson, e := json.Marshal(respObj); e != nil {
		logs.Error(e)
		return
	} else {
		strResp := string(bJson)
		w.Write(bJson)
		logs.Pretty(strResp, "response:")
	}
}

func (p *EntranceMartini) postHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.setBasicHeaders(w, r)

		var err error

		apiName := r.Header.Get(p.config.apiHeader)
		if apiName == "" {
			logs.Error(errorcode.ERR_API_NOT_FOUND.New(errors.Params{"apiName": apiName}))
			writeJson(respNotFound, w)
			return
		}

		logs.Info("handle", apiName)

		var reqBody []byte
		if reqBody, err = ioutil.ReadAll(r.Body); err != nil {
			logs.Error(errorcode.ERR_BAD_REQUEST.New(errors.Params{"path": p.config.Path, "err": err}))
			writeJson(respBadRequest, w)
			return
		} else if strings.TrimSpace(string(reqBody)) == "" {
			reqBody = []byte("{}")
		}

		logs.Debug("http request:", p.config.Path, string(reqBody))

		var mapResult map[string]interface{}

		if e := json.Unmarshal(reqBody, &mapResult); e != nil {
			logs.Error(errorcode.ERR_REQUEST_SHOULD_BE_JSON.New())
			writeJson(respNotAJson, w)
			return
		}

		// Componet message
		var comMsg *ComponentMessage
		if comMsg, err = p.messenger.NewMessage(mapResult); err != nil {
			logs.Error(errorcode.ERR_COULD_NOT_NEW_COMPONENT_MSG.New(errors.Params{"err": err}))
			writeJson(respInternalError, w)
			return
		}

		cookies := map[string]string{}
		if p.config.ToContext.Cookies != nil {
			for _, cookieName := range p.config.ToContext.Cookies {
				if cookie, e := r.Cookie(cookieName); e == nil {
					cookies[cookieName] = cookie.Value
				}
			}
		}

		headers := map[string]string{}
		if p.config.ToContext.Headers != nil {
			for _, headerName := range p.config.ToContext.Headers {
				headers[headerName] = r.Header.Get(headerName)
			}
		}

		comMsg.Payload.SetContext(CTX_HTTP_COOKIES, cookies)
		comMsg.Payload.SetContext(CTX_HTTP_HEADERS, headers)
		comMsg.Payload.SetContext(p.config.apiHeader, apiName)

		logs.Pretty("request_cookies:", cookies)
		logs.Pretty("request_headers:", headers)

		// send msg to next
		msgId := ""
		var ch chan *Payload

		if msgId, ch, err = p.messenger.SendMessage(apiName, comMsg); err != nil {
			logs.Error(errorcode.ERR_SEND_COMPONENT_MSG_ERROR.New(errors.Params{"id": msgId, "err": err}))
			writeJson(respInternalError, w)
			return
		}

		if msgId != "" {
			w.Header().Set("X-Response-Id", msgId)
		}

		defer close(ch)
		defer p.messenger.OnMessageEvent(msgId, MSG_EVENT_PROCESSED)

		// Wait for response from IN port
		logs.Debug("Waiting for response: ", apiName)
		var payload *Payload
		select {
		case payload = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			writeJson(respRequestTimeout, w)
			return
		}

		// Cookies
		cmdCookiesSize := payload.GetCommandValueSize(CMD_HTTP_COOKIES_SET)
		cmdCookies := make([]interface{}, cmdCookiesSize)
		for i := 0; i < cmdCookiesSize; i++ {
			cookie := new(http.Cookie)
			cmdCookies[i] = cookie
		}

		if err = payload.GetCommandObjectArray(CMD_HTTP_COOKIES_SET, cmdCookies); err != nil {
			err = errorcode.ERR_PARSE_COMMAND_TO_OBJECT_FAILED.New(errors.Params{"cmd": CMD_HTTP_COOKIES_SET, "err": err})
			logs.Error(err)
			writeJson(respInternalError, w)
			return
		}

		for _, cookie := range cmdCookies {
			if c, ok := cookie.(*http.Cookie); ok {
				c.Domain = p.config.Domain
				c.Path = "/"
				logs.Pretty("write cookie:", c)
				http.SetCookie(w, c)
			} else {
				err = errorcode.ERR_PARSE_COMMAND_TO_OBJECT_FAILED.New(errors.Params{"cmd": CMD_HTTP_COOKIES_SET, "err": "object could not parser to cookies"})
				logs.Error(err)
				writeJson(respInternalError, w)
				return
			}
		}

		cmdHeadersSize := payload.GetCommandValueSize(CMD_HTTP_HEADERS_SET)
		cmdHeaders := make([]interface{}, cmdHeadersSize)
		for i := 0; i < cmdHeadersSize; i++ {
			header := new(NameValue)
			cmdHeaders[i] = header
		}

		if err = payload.GetCommandObjectArray(CMD_HTTP_HEADERS_SET, cmdHeaders); err != nil {
			err = errorcode.ERR_PARSE_COMMAND_TO_OBJECT_FAILED.New(errors.Params{"cmd": CMD_HTTP_HEADERS_SET, "err": err})
			logs.Error(err)
			writeJson(respInternalError, w)
			return
		}

		for _, header := range cmdHeaders {
			if nv, ok := header.(*NameValue); ok {
				w.Header().Add(nv.Name, nv.Value)
				logs.Pretty("write header:", nv)
			} else {
				err = errorcode.ERR_PARSE_COMMAND_TO_OBJECT_FAILED.New(errors.Params{"cmd": CMD_HTTP_HEADERS_SET, "err": "object could not parser to headers"})
				logs.Error(err)
				writeJson(respInternalError, w)
				return
			}
		}

		respObj := httpRespStruct{Code: payload.Code,
			Message: payload.Message,
			Result:  payload.result}

		writeJson(respObj, w)
	}
}

func parse_refer(url string) (protocol string, domain string) {
	url = strings.TrimSpace(url)

	if len(url) > 0 {
		start0 := strings.Index(url, "://")
		url0 := url[start0+3 : len(url)]
		surls := strings.Split(url0, "/")

		if len(surls) > 0 {
			domain = surls[0]
		}

		protocol = url[0:start0]
	}

	return
}
