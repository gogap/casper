package casper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-martini/martini"
	log "github.com/golang/glog"
	uuid "github.com/nu7hatch/gouuid"
)

type martiniEntrance struct {
	martini *martini.ClassicMartini
}

func init() {
	registerEntrance("martini", NewMartiniEntrances)
}

func NewMartiniEntrances() entrance {
	return &martiniEntrance{}
}

func (p *martiniEntrance) StartService(app *App, addr string) {
	p.martini = martini.Classic()
	p.martini.Post("/"+app.Name, martiniHandle(app))
	p.martini.Options("/"+app.Name, martiniOptionsHandle)
	log.Infoln("martiniEntrance start at:", addr)
	p.martini.RunOnAddr(addr)
}

func martiniOptionsHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	//w.Header().Add("Access-Control-Allow-Headers", "X-API")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Methods", "POST")
	w.Header().Add("P3P", `CP="CURa ADMa DEVa PSAo PSDo OUR BUS UNI PUR INT DEM STA PRE COM NAV OTC NOI DSP COR"`)
	w.Header().Add("Access-Control-Allow-Headers", "X-API, X-REQUEST-ID, X-API-TRANSACTION, X-API-TRANSACTION-TIMEOUT, X-RANGE, Origin, X-Requested-With, Content-Type, Accept")
}

func martiniHandle(app *App) func(http.ResponseWriter, *http.Request) {
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
			log.Errorln("request body err:", app.Name, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read request body error"))
			return
		}
		log.Infoln("httpRequest:", app.Name, string(reqBody))

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
		coMsg, _ := NewComponentMessage("")
		coMsg.Payload.SetContext(REQ_X_API, apiName)
		coMsg.Payload.SetContext(SESSION_KEY, sessionids) // 会话ID
		coMsg.Payload.SetContext(USER_KEY, userids)
		coMsg.Payload.Result = reqBody

		// send msg to next
		id, ch, err := app.sendMsg(apiName, coMsg)
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
		defer app.delRequest(id)

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
		var cmd map[string]string
		err = load.GetCommandObject(CMD_SET_SESSION, &cmd)
		if err != nil {
			log.Errorln("session CMD:", err.Error())
		} else {
			log.Infoln("get session:", sessionids, cmd)
			for k, v := range cmd {
				if k == USER_KEY {
					log.Infoln("add cookie:", k, v)
					http.SetCookie(w, &http.Cookie{Name: k, Value: v, Domain: app.domain, Path: "/"})
				}
				log.Infoln("set session:", sessionids, k, v)
				SessionSetByte(sessionids, k, []byte(v))
			}
		}

		http.SetCookie(w, &http.Cookie{Name: SESSION_KEY, Value: sessionids, Domain: app.domain, Path: "/"})
		w.Header().Set("content-type", "application/json") //返回数据格式是json
		w.Header().Set("Access-Control-Allow-Origin", "http://investor.rijin.com")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "POST")
		w.Header().Add("Access-Control-Allow-Headers", "X-API, X-REQUEST-ID, X-API-TRANSACTION, X-API-TRANSACTION-TIMEOUT, X-RANGE, Origin, X-Requested-With, Content-Type, Accept")
		w.Header().Add("P3P", `CP="CURa ADMa DEVa PSAo PSDo OUR BUS UNI PUR INT DEM STA PRE COM NAV OTC NOI DSP COR"`)

		objResp := HttpResponse{
			Code:    load.Code,
			Message: load.Message,
			Result:  load.Result}

		bResp, _ := json.Marshal(objResp)
		w.Write(bResp)
		log.Infoln("Data arrived. Responding to HTTP response...", string(bResp))
	}
}
