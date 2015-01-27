package casper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-martini/martini"
	log "github.com/golang/glog"
	uuid "github.com/nu7hatch/gouuid"
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
	p.martini.Post(p.config.Path, p.martiniHandle())
	p.martini.Options(p.config.Path, p.martiniOptionsHandle)

	listenAddr := p.config.GetListenAddress()
	log.Infof("[Entrance-%s] start at: %s\n", p.Type(), listenAddr)
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

func (p *EntranceMartini) martiniOptionsHandle(w http.ResponseWriter, r *http.Request) {
	p.setBasicHeaders(w, r)
}

func (p *EntranceMartini) martiniHandle() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		apiName := r.Header.Get(REQ_X_API)
		if apiName == "" {
			log.Errorln("request api name nil")
			http.NotFound(w, r)
			return
		}
		log.Infoln("handle:", apiName)

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Errorln("request body err:", p.config.Path, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read request body error"))
			return
		}
		log.Infoln("httpRequest:", p.config.Path, string(reqBody))

		// cookie
		sessionids := ""
		userids := ""
		sessionid, err := r.Cookie(SESSION_KEY)
		if err != nil || sessionid == nil {
			uuidTmp, _ := uuid.NewV4()
			sessionids = uuidTmp.String()
		} else {
			sessionids = sessionid.Value
		}

		userid, err := r.Cookie(USER_KEY)
		if userid != nil {
			userids = userid.Value
		}

		// Componet message
		var comMsg *ComponentMessage
		if msg, e := p.messenger.NewMessage(reqBody); e != nil {
			log.Errorln("NewMessage err", e)
			return
		} else {
			comMsg = msg
		}
		comMsg.Payload.SetContext(REQ_X_API, apiName)
		comMsg.Payload.SetContext(SESSION_KEY, sessionids) // 会话ID
		comMsg.Payload.SetContext(USER_KEY, userids)

		// send msg to next
		id, ch, err := p.messenger.SendMessage(apiName, comMsg)
		if err != nil {
			log.Errorln("sendMsg err:", comMsg.Id, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if ch == nil {
			log.Errorln("sendMsg return nil:", comMsg.Id)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Service Internal Error"))
			return
		}
		defer close(ch)
		defer p.messenger.OnMessageEvent(id, MSG_EVENT_PROCESSED)

		// Wait for response from IN port
		log.Infoln("Waiting for response: ", apiName, string(reqBody))
		var load *Payload
		select {
		case load = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Couldn't process request in a given time"))
			return
		}

		// SESSION
		cmd := make(map[string]string)
		load.GetCommandObject(CMD_SET_SESSION, &cmd)
		log.Infoln("get session:", sessionids, cmd)
		for k, v := range cmd {
			if k == USER_KEY {
				log.Infoln("add cookie:", k, v)
				http.SetCookie(w, &http.Cookie{Name: k, Value: v, Domain: p.config.Domain, Path: "/"})
			}
			log.Infoln("set session:", sessionids, k, v)
			SessionSetByte(sessionids, k, []byte(v))
		}

		http.SetCookie(w, &http.Cookie{Name: SESSION_KEY, Value: sessionids, Domain: p.config.Domain, Path: "/"})

		p.setBasicHeaders(w, r)

		rstStr := string(load.result)
		if rstStr == "" {
			rstStr = "NULL"
		}
		if load.Code != 0 {
			rstStr = "NULL"
		}
		resp := fmt.Sprintf("{\n\"code\":%v,\n\"message\":\"%v\",\n\"result\":%v\n}", load.Code, load.Message, rstStr)
		log.Infoln("Data arrived. Responding to HTTP response...", resp)
		w.Write([]byte(resp))
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
