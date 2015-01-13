package casper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/golang/glog"
	"github.com/cascades-fbp/cascades/runtime"
	. "github.com/gogap/base_component"
)

const (
	REQ_TIMEOUT = time.Duration(15) * time.Second
)

type Response struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

func handle(p *App) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("http Handler:", r.Method, r.RequestURI)

		apiName := r.Header.Get(REQ_X_API)
		if apiName == "" {
			log.Errorln("request api name nil")
			http.NotFound(w, r)
			return
		}
		port := p.GetApi(apiName)
		if port == nil {
			log.Errorln("request api 404")
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
		log.Infoln("req:", apiName, string(reqBody))

		// Componet message
		componentMsg, _ := NewComponentMessage()
		componentMsg.SetEntrance(port.outPort[0].Url)
		componentMsg.Payload.SetContext(REQ_X_API, apiName)
		componentMsg.Payload.Result = reqBody

		// new request
		ch := p.AddRequest(componentMsg.ID)
		defer p.DelRequest(componentMsg.ID)
		defer close(ch)

		// Send Component message
		msgBytes, err := componentMsg.Serialize()
		if err != nil {
			log.Errorln("Service Internal Error")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Service Internal Error"))
			return
		}
		log.Infoln("ToNextComponent:", port.outPort[0].Url, string(msgBytes))
		port.outPort[0].Socket.SendMessage(runtime.NewPacket(msgBytes))

		// Wait for response from IN port
		log.Infoln("Waiting for response from a channel port (from INPUT port)")
		var load *Payload
		select {
		case load = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Couldn't process request in a given time"))
			return
		}

		objResp := Response{
			Code:    load.Code,
			Message: load.Message,
			Result:  load.Result}

		bResp, _ := json.Marshal(objResp)
		w.Write(bResp)
		log.Infoln("Data arrived. Responding to HTTP response...", string(bResp))
	}
}
