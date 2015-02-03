package casper

import (
	"encoding/json"
	"fmt"

	uuid "github.com/nu7hatch/gouuid"
)

type componentCommands map[string][]interface{}
type componentContext map[string]interface{}

type callChain struct {
	ComponentName string
	Endpoint      string
}

type ComponentMetadata struct {
	Name   string `json:"name"`
	MQType string `json:"mq_type"`
	In     string `json:"in"`
}

type ComponentMessage struct {
	Id       string               `json:"id"`
	entrance *ComponentMetadata   `json:"entrance"`
	graph    []*ComponentMetadata `json:"graph"`
	chain    []string             `json:"chain"`
	Payload  *Payload             `json:"payload"`
}

type Payload struct {
	Code    uint64            `json:"code"`
	Message string            `json:"message"`
	context componentContext  `json:"context"`
	command componentCommands `json:"command"`
	result  interface{}       `json:"result"`
}

func NewComponentMessage(entrance *ComponentMetadata, result interface{}) (msg *ComponentMessage, err error) {
	msgId := ""
	if u, e := uuid.NewV4(); e != nil {
		err = e
		return
	} else {
		msgId = u.String()
	}

	msg = &ComponentMessage{
		Id:       msgId,
		entrance: entrance,
		graph:    nil,
		chain:    nil,
		Payload: &Payload{
			Code:    0,
			Message: "OK",
			context: nil,
			command: nil,
			result:  result}}

	return msg, nil

}

func (p *ComponentMessage) SetEntrance(entrance ComponentMetadata) {
	p.entrance = &entrance
}

func (p *ComponentMessage) TopGraph() *ComponentMetadata {
	if len(p.graph) >= 1 {
		return p.graph[0]
	}

	return nil
}

func (p *ComponentMessage) PopGraph() *ComponentMetadata {
	if len(p.graph) >= 1 {
		p.graph = p.graph[1:]
	}
	if len(p.graph) >= 1 {
		return p.graph[0]
	}

	return nil
}

func (p *ComponentMessage) Serialize() ([]byte, error) {
	type Msg struct {
		Id       string               `json:"id"`
		Entrance *ComponentMetadata   `json:"entrance"`
		Graph    []*ComponentMetadata `json:"graph"`
		Chain    []string             `json:"chain"`
		Payload  struct {
			Code    uint64            `json:"code"`
			Message string            `json:"message"`
			Context componentContext  `json:"context"`
			Command componentCommands `json:"command"`
			Result  interface{}       `json:"result"`
		} `json:"payload"`
	}

	tmp := &Msg{}
	tmp.Id = p.Id
	tmp.Entrance = p.entrance
	tmp.Graph = p.graph
	tmp.Chain = p.chain
	if p.Payload != nil {
		tmp.Payload.Code = p.Payload.Code
		tmp.Payload.Message = p.Payload.Message
		tmp.Payload.Context = p.Payload.context
		tmp.Payload.Command = p.Payload.command
		tmp.Payload.Result = p.Payload.result
	}

	return json.Marshal(tmp)
}

func (p *ComponentMessage) FromJson(jsonStr []byte) (err error) {
	var tmp struct {
		Id       string               `json:"id"`
		Entrance *ComponentMetadata   `json:"entrance"`
		Graph    []*ComponentMetadata `json:"graph"`
		Chain    []string             `json:"chain"`
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

	p.Id = tmp.Id
	p.entrance = tmp.Entrance
	p.graph = tmp.Graph
	p.chain = tmp.Chain
	p.Payload = &Payload{
		Code:    tmp.Payload.Code,
		Message: tmp.Payload.Message,
		context: tmp.Payload.Context,
		command: tmp.Payload.Command,
		result:  tmp.Payload.Result}

	return nil
}

func (p *Payload) UnmarshalResult(v interface{}) (err error) {
	if p.result == nil {
		return nil
	}

	if bJson, e := json.Marshal(p.result); e != nil {
		err = fmt.Errorf("marshal result to json failed, error is:%v", e)
		return
	} else if e := json.Unmarshal(bJson, v); e != nil {
		err = fmt.Errorf("unmarshal result to object failed, error is:%v", e)
		return
	}
	return
}

func (p *Payload) GetResult() interface{} {
	return p.result
}

func (p *Payload) SetResult(result interface{}) {
	p.result = result
}

func (p *Payload) SetContext(key string, val interface{}) {
	if p.context == nil {
		p.context = make(map[string]interface{})
	}
	p.context[key] = val
}

func (p *Payload) GetContext(key string) (val interface{}, exist bool) {
	if p.context == nil {
		return nil, false
	}
	val, exist = p.context[key]
	return
}

func (p *Payload) GetContextString(key string) (val string, err error) {
	if p.context == nil {
		return "", fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if strVal, ok := v.(string); ok {
		val = strVal
		return
	}

	return
}

func (p *Payload) GetContextStringArray(key string) (vals []string, err error) {
	if p.context == nil {
		return nil, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if vInterfaces, ok := v.([]interface{}); ok {
		tmpArray := []string{}
		for i, vStr := range vInterfaces {
			if str, ok := vStr.(string); ok {
				tmpArray = append(tmpArray, str)
			} else {
				err = fmt.Errorf("the context key of %s's value type at index of %d is not string", key, i)
				return
			}
		}
		vals = tmpArray
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not array", key)
		return
	}
	return
}

func (p *Payload) GetContextInt(key string) (val int, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int", key)
	}
	return
}

func (p *Payload) GetContextInt32(key string) (val int32, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int32); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int32", key)
	}
	return
}

func (p *Payload) GetContextInt64(key string) (val int64, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int64); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int64", key)
	}
	return
}

func (p *Payload) GetContextObject(key string, v interface{}) (err error) {
	if v == nil {
		err = fmt.Errorf("the v should not be nil, it should be a Pointer")
		return
	}

	if p.context == nil {
		return fmt.Errorf("the context container is nil")
	}

	if val, exist := p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	} else if val == nil {
		err = fmt.Errorf("the context key of %s is exist, but the value is nil", key)
		return
	} else {
		if bJson, e := json.Marshal(val); e != nil {
			err = fmt.Errorf("marshal object of %s to json failed, error is:%v", key, e)
			return
		} else if e := json.Unmarshal(bJson, v); e != nil {
			err = fmt.Errorf("unmarshal json to object %s failed, error is:%v", key, e)
			return
		}
	}
	return
}

func (p *Payload) SetCommand(command string, values []interface{}) {
	if p.command == nil {
		p.command = make(map[string][]interface{})
	}
	p.command[command] = values
}

func (p *Payload) AppendCommand(command string, value interface{}) {
	if p.command == nil {
		p.command = make(map[string][]interface{})
	}

	if values, ok := p.command[command]; !ok {
		p.command[command] = []interface{}{value}
	} else {
		values = append(values, value)
		p.command[command] = values
	}
	return
}

func (p *Payload) GetCommand(key string) (val []interface{}, exist bool) {
	if p.command == nil {
		return nil, false
	}
	val, exist = p.command[key]
	return
}

func (p *Payload) GetCommandValueSize(key string) int {
	if p.command == nil {
		return 0
	} else {
		if vals, exist := p.command[key]; exist {
			if vals != nil {
				return len(vals)
			}
			return 0
		}
	}
	return 0
}

func (p *Payload) GetCommandStringArray(command string) (vals []string, err error) {
	if size := p.GetCommandValueSize(command); size > 0 {
		values, _ := p.GetCommand(command)

		tmpVals := []string{}
		for _, iStr := range values {
			if strV, ok := iStr.(string); ok {
				tmpVals = append(tmpVals, strV)
			} else {
				err = fmt.Errorf("the value of %v are not string type", iStr)
				return
			}
		}
		vals = tmpVals
		return
	}
	err = fmt.Errorf("command values is nil or command not exist")
	return
}

func (p *Payload) GetCommandObjectArray(command string, values []interface{}) (err error) {

	if values == nil {
		err = fmt.Errorf("the values should not be nil, it should be a interface{}")
		return
	}

	if len(values) == 0 {
		return
	}

	if p.GetCommandValueSize(command) < len(values) {
		err = fmt.Errorf("the command of %s is exist, but the recv values length is greater than command values", command)
		return
	}

	vals, _ := p.GetCommand(command)

	for i, objVal := range vals {
		var bJson []byte
		var e error
		if bJson, e = json.Marshal(objVal); e != nil {
			err = fmt.Errorf("marshal object of %s to json failed, error is:%v", command, e)
			return
		}

		if e = json.Unmarshal(bJson, &values[i]); e != nil {
			err = fmt.Errorf("unmarshal json to object %s failed, error is:%v", command, e)
			return
		}
	}

	return
}
