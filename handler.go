package casper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gogap/base_component"
	"github.com/nu7hatch/gouuid"

	"github.com/gogap/casper/utils"
)

const (
	timeout = time.Duration(15) * time.Second
)

type HandlerRequest struct {
	ResponseChan chan base_component.ComponentMessage
	Request      *base_component.ComponentMessage
}

type Response struct {
	Code    int64       `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

func handler(out chan HandlerRequest) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {

		log.Debug("Handler:", req.Method, req.RequestURI)

		id, _ := uuid.NewV4()
		msg := utils.RequestToComponentMessage(req)
		msg.Id = id.String()

		hr := &HandlerRequest{
			ResponseChan: make(chan base_component.ComponentMessage),
			Request:      msg,
		}

		// Send request to OUT port
		log.Debug("Sending request to out channel (for OUTPUT port)")
		select {
		case out <- *hr:
		case <-time.Tick(timeout):
			timeout_respond(rw)
			return
		}

		// Wait for response from IN port
		log.Debug("Waiting for response from a channel port (from INPUT port)")

		var resp base_component.ComponentMessage
		select {
		case resp = <-hr.ResponseChan:
		case <-time.Tick(timeout):
			timeout_respond(rw)
			return
		}

		log.Debug("Data arrived. Responding to HTTP response...")
		for cmdName, _ := range resp.Payload.Commands {
			//TODO command process
			switch cmdName {
			case "write_header":
				{

				}
			}
		}
		objResp := Response{
			Code:    resp.Payload.Code,
			Message: resp.Payload.Message,
			Result:  resp.Payload.Result}

		bResp, _ := json.Marshal(objResp)

		fmt.Fprint(rw, bResp)
	}
}

func timeout_respond(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(rw, "Couldn't process request in a given time")
}
