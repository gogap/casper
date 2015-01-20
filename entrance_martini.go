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
	log.Infoln("martiniEntrance start at:", addr)
	p.martini.RunOnAddr(addr)
}

func martiniHandle(p *App) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		apiName := r.Header.Get(REQ_X_API)
		if apiName == "" {
			log.Errorln("request api name nil")
			http.NotFound(w, r)
			return
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Errorln("request body err:", p.Name, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read request body error"))
			return
		}
		log.Infoln("httpRequest:", p.Name, string(reqBody))

		// cookie
		sessionids := ""
		userids := ""
		sessionid, err := r.Cookie(SESSION_HEADER_KEY)
		if err != nil || sessionid == nil {
			uuidTmp, _ := uuid.NewV4()
			sessionids = uuidTmp.String()
		} else {
			sessionids = sessionid.Value
		}

		userid, err := r.Cookie(USER_HEADER_KEY)
		if userid != nil {
			userids = userid.Value
		}

		// Componet message
		coMsg, _ := NewComponentMessage("")
		coMsg.Payload.SetContext(REQ_X_API, apiName)
		coMsg.Payload.SetContext(SESSION_HEADER_KEY, sessionids)
		coMsg.Payload.SetContext(USER_HEADER_KEY, userids)
		coMsg.Payload.Result = reqBody

		// send msg to next
		id, ch, err := p.sendMsg(apiName, coMsg)
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
		defer p.delRequest(id)

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

		// deal cmd
		cmd := make(map[string]string)
		err = load.GetCommandObject(CMD_HTTP_HEADER_WRITE, cmd)
		if err != nil {
			for k, v := range cmd {
				r.AddCookie(&http.Cookie{Name: k, Value: v, Path: "/"})
			}
		}

		objResp := HttpResponse{
			Code:    load.Code,
			Message: load.Message,
			Result:  load.Result}

		bResp, _ := json.Marshal(objResp)
		w.Write(bResp)
		log.Infoln("Data arrived. Responding to HTTP response...", string(bResp))
	}
}
