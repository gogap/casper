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
	uuid "github.com/nu7hatch/gouuid"

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

type EntranceMartiniConf struct {
	Host         string            `json:"host"`
	Port         int32             `json:"port"`
	Domain       string            `json:"domain"`
	Path         string            `json:"path"`
	AllowOrigin  []string          `json:"allow_origin"`
	AllowHeaders []string          `json:"allow_headers"`
	allowHeaders string            `json:"-"`
	allowOrigin  map[string]bool   `json:"-"`
	Headers      map[string]string `json:"headers"`
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
		err = fmt.Errorf("[Entrance-%s] fill config failed", p.Type())
		return
	}

	p.config.allowHeaders = strings.Join(p.config.AllowHeaders, ",")
	p.config.allowOrigin = make(map[string]bool)
	for _, origin := range p.config.AllowOrigin {
		p.config.allowOrigin[origin] = true
	}

	if messenger == nil {
		err = fmt.Errorf("[Entrance-%s] Messenger is nil", p.Type())
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

	logs.Info("[entrance-%s] start at: %s", p.Type(), listenAddr)

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

	for key, value := range p.config.Headers {
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

		apiName := r.Header.Get(REQ_X_API)
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

		// cookie
		strSessionId := ""
		strUserId := ""

		sessionid, err := r.Cookie(SESSION_KEY)
		if err != nil || sessionid == nil {
			uuidTmp, _ := uuid.NewV4()
			strSessionId = uuidTmp.String()
		} else {
			strSessionId = sessionid.Value
		}

		if userid, e := r.Cookie(USER_KEY); e != nil {
			logs.Debug("get cookie error:", USER_KEY, e)
		} else if userid != nil {
			strUserId = userid.Value
		}

		// Componet message
		var comMsg *ComponentMessage
		if comMsg, err = p.messenger.NewMessage(mapResult); err != nil {
			logs.Error(errorcode.ERR_COULD_NOT_NEW_COMPONENT_MSG.New(errors.Params{"err": err}))
			writeJson(respInternalError, w)
			return
		}

		comMsg.Payload.SetContext(REQ_X_API, apiName)
		comMsg.Payload.SetContext(SESSION_KEY, strSessionId) // 会话ID
		comMsg.Payload.SetContext(USER_KEY, strUserId)

		// send msg to next

		msgId := ""
		var ch chan *Payload

		if msgId, ch, err = p.messenger.SendMessage(apiName, comMsg); err != nil {
			logs.Error(errorcode.ERR_SEND_COMPONENT_MSG_ERROR.New(errors.Params{"id": msgId, "err": err}))
			writeJson(respInternalError, w)
			return
		}

		fmt.Println(msgId, ch)

		defer close(ch)
		defer p.messenger.OnMessageEvent(msgId, MSG_EVENT_PROCESSED)

		// Wait for response from IN port
		logs.Debug("Waiting for response: ", apiName)
		var load *Payload
		select {
		case load = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			writeJson(respRequestTimeout, w)
			return
		}

		// SESSION
		cmd := make(map[string]string)
		load.GetCommandObject(CMD_SET_SESSION, &cmd)
		logs.Debug("get session:", strSessionId, cmd)

		for k, v := range cmd {
			if k == USER_KEY {
				http.SetCookie(w, &http.Cookie{Name: k, Value: v, Domain: p.config.Domain, Path: "/"})
			}
			SessionSetByte(strSessionId, k, []byte(v), (3 * 24 * 60 * 60))
		}

		http.SetCookie(w, &http.Cookie{Name: SESSION_KEY, Value: strSessionId, Domain: p.config.Domain, Path: "/"})

		respObj := httpRespStruct{Code: load.Code,
			Message: load.Message,
			Result:  load.result}

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
