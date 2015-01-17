package casper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-martini/martini"
	log "github.com/golang/glog"
)

type martiniEntrances struct {
	martini *martini.ClassicMartini
}

func init() {
	registerEntrance("martini", NewMartiniFace)
}

func NewMartiniFace() entrance {
	return &martiniEntrances{}
}

func (p *martiniEntrances) StartService(app *App, addr string) {
	p.martini = martini.Classic()
	p.martini.Post("/"+app.Name, martiniHandle(app))
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

		// send msg to next
		id, ch, err := p.sendMsg(apiName, reqBody)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if ch == nil {
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

		objResp := HttpResponse{
			Code:    load.Code,
			Message: load.Message,
			Result:  load.Result}

		bResp, _ := json.Marshal(objResp)
		w.Write(bResp)

		log.Infoln("Data arrived. Responding to HTTP response...", string(bResp))
	}
}
