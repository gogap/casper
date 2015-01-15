package casper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	uuid "github.com/nu7hatch/gouuid"
)

type componentCommands map[string]interface{}
type componentContext map[string]interface{}

type callChain struct {
	ComponentName string
	Endpoint      string
}

type ComponentMessage struct {
	ID       string   `json:"id"`
	entrance string   `json:"entrance"`
	graph    []string `json:"graph"`
	chain    []string `json:"chain"`
	Payload  *Payload `json:"payload"`
}

type Payload struct {
	Code    uint64            `json:"code"`
	Message string            `json:"message"`
	context componentContext  `json:"context,omitempty"`
	command componentCommands `json:"command,omitempty"`
	Result  interface{}       `json:"result"`
}

func NewComponentMessage(entrance string) (msg *ComponentMessage, err error) {
	msgID := ""
	if u, e := uuid.NewV4(); e != nil {
		err = e
		return
	} else {
		msgID = u.String()
	}

	msg = &ComponentMessage{
		ID:       msgID,
		entrance: strings.TrimSpace(entrance),
		graph:    nil,
		chain:    nil,
		Payload: &Payload{
			Code:    0,
			Message: "OK",
			context: make(map[string]interface{}),
			command: make(map[string]interface{}),
			Result:  nil}}

	return
}

func (p *ComponentMessage) SetEntrance(entrance string) {
	p.entrance = entrance
}

func (p *ComponentMessage) TopGraph() string {
	if len(p.graph) >= 1 {
		return p.graph[0]
	}
	
	return ""
}

func (p *ComponentMessage) PopGraph() string {
	fmt.Println(p.graph)
	
	if len(p.graph) >= 1 {
		p.graph = p.graph[1:]
	}
	if len(p.graph) >= 1 {
		return p.graph[0]
	}

	return ""
}

func (p *ComponentMessage) Serialize() ([]byte, error) {
	var tmp struct {
		ID       string   `json:"id"`
		Entrance string   `json:"entrance"`
		App      string   `json:"app"`
		Graph    []string `json:"graph"`
		Chain    []string `json:"chain"`
		Payload  struct {
			Code    uint64            `json:"code"`
			Message string            `json:"message"`
			Context componentContext  `json:"context,omitempty"`
			Command componentCommands `json:"command,omitempty"`
			Result  interface{}       `json:"result"`
		} `json:"payload"`
	}

	tmp.ID = p.ID
	tmp.Entrance = p.entrance
	tmp.Graph = p.graph
	tmp.Chain = p.chain
	if p.Payload != nil {
		tmp.Payload.Code = p.Payload.Code
		tmp.Payload.Message = p.Payload.Message
		tmp.Payload.Context = p.Payload.context
		tmp.Payload.Command = p.Payload.command
		tmp.Payload.Result = p.Payload.Result
	}

	return json.Marshal(tmp)
}

func (p *ComponentMessage) FromJson(jsonStr []byte) (err error) {
	var tmp struct {
		ID       string   `json:"id"`
		Entrance string   `json:"entrance"`
		App      string   `json:"app"`
		Graph    []string `json:"graph"`
		Chain    []string `json:"chain"`
		Payload  struct {
			Code    uint64            `json:"code"`
			Message string            `json:"message"`
			Context componentContext  `json:"context,omitempty"`
			Command componentCommands `json:"command,omitempty"`
			Result  interface{}       `json:"result"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(jsonStr, &tmp); err != nil {
		return err
	}

	p.ID = tmp.ID
	p.entrance = tmp.Entrance
	p.graph = tmp.Graph
	p.chain = tmp.Chain
	p.Payload = &Payload{
		Code:    tmp.Payload.Code,
		Message: tmp.Payload.Message,
		context: tmp.Payload.Context,
		command: tmp.Payload.Command,
		Result:  tmp.Payload.Result}

	return nil
}

func (p *Payload) UnmarshalResult(v interface{}) error {
	if p.Result == nil {
		return nil
	}

	if str, ok := p.Result.(string); ok {
		var byteData []byte
		var err error
		if byteData, err = base64.StdEncoding.DecodeString(str); err != nil {
			return err
		}
		return json.Unmarshal(byteData, v)
	} else if reflect.TypeOf(p.Result) == reflect.TypeOf(v) {
		v = p.Result
		return nil
	}

	return fmt.Errorf("Type of %s & %s error", reflect.TypeOf(p.Result).Name(), reflect.TypeOf(v).Name())
}

func (p *Payload) SetContext(key string, val interface{}) {
	if p.context == nil {
		p.context = make(map[string]interface{})
	}
	p.context[key] = val
}

func (p *Payload) GetContext(key string, del bool) (val interface{}, ok bool) {
	if p.context == nil {
		return nil, false
	}

	val, ok = p.context[key]

	if del == true {
		delete(p.context, key)
	}

	return
}

func (p *Payload) SetCommand(key string, command interface{}) {
	if p.command == nil {
		p.command = make(map[string]interface{})
	}
	p.command[key] = command
}

func (p *Payload) GetCommand(key string, del bool) (val interface{}, ok bool) {
	if p.command == nil {
		return nil, false
	}

	val, ok = p.command[key]

	if del == true {
		delete(p.command, key)
	}

	return
}
