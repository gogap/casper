package casper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-martini/martini"
	log "github.com/golang/glog"
	uuid "github.com/nu7hatch/gouuid"
)

type EntranceMartiniConf struct {
	Host          string
	Port          int32
	Domain        string
	Headers       map[string]string
	IsEnableHttps bool
}

func (p *EntranceMartiniConf) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

type EntranceMartini struct {
	app     *App
	config  EntranceMartiniConf
	martini *martini.ClassicMartini
}

func init() {
	entrancefactory.RegisterEntrance(new(EntranceMartini))
}

func (p *EntranceMartini) Type() string {
	return "martini"
}

func (p *EntranceMartini) Init(app *App, configs EntranceConfig) (err error) {
	if e := configs.FillToObject(&p.config); e != nil {
		err = fmt.Errorf("[Entrance-%s] fill config failed", p.Type())
		return
	}

	if app == nil {
		err = fmt.Errorf("[Entrance-%s] app is nil", p.Type())
		return
	} else {
		p.app = app
	}
	return
}

func (p *EntranceMartini) Run() {
	p.martini = martini.Classic()
	p.martini.Post("/"+p.app.Name(), p.martiniHandle())
	p.martini.Options("/"+p.app.Name(), martiniOptionsHandle)

	listenAddr := p.config.GetListenAddress()
	log.Infof("[Entrance-%s] start at: %s\n", p.Type(), listenAddr)
	p.martini.RunOnAddr(listenAddr)

	return
}

func martiniOptionsHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Headers", "X-API")
	w.Header().Set("Access-Control-Allow-Origin", "http://investor.rijin.com")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Methods", "POST")
	w.Header().Add("P3P", `CP="CURa ADMa DEVa PSAo PSDo OUR BUS UNI PUR INT DEM STA PRE COM NAV OTC NOI DSP COR"`)
	w.Header().Add("Access-Control-Allow-Headers", "X-API, X-REQUEST-ID, X-API-TRANSACTION, X-API-TRANSACTION-TIMEOUT, X-RANGE, Origin, X-Requested-With, Content-Type, Accept")
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
			log.Errorln("request body err:", p.app.Name, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read request body error"))
			return
		}
		log.Infoln("httpRequest:", p.app.Name, string(reqBody))

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
		coMsg, _ := NewComponentMessage("", reqBody)
		coMsg.Payload.SetContext(REQ_X_API, apiName)
		coMsg.Payload.SetContext(SESSION_KEY, sessionids) // 会话ID
		coMsg.Payload.SetContext(USER_KEY, userids)

		// send msg to next
		id, ch, err := p.app.sendMsg(apiName, coMsg)
		if err != nil {
			log.Errorln("sendMsg err:", coMsg.ID, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if ch == nil {
			log.Errorln("sendMsg return nil:", coMsg.ID)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Service Internal Error"))
			return
		}
		defer close(ch)
		defer p.app.delRequest(id)

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

		for key, value := range p.config.Headers {
			w.Header().Set(key, value)
		}

		resp := fmt.Sprintf("{\n\"code\":%v,\n\"message\":\"%v\",\n\"result\":%v\n}", load.Code, load.Message, string(load.result))
		log.Infoln("Data arrived. Responding to HTTP response...", resp)
		w.Write([]byte(resp))
	}
}
