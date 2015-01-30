package casper

import (
	"encoding/json"
	"time"
)

const (
	REQ_TIMEOUT = time.Duration(15) * time.Second
	REQ_X_API   = "X-API"

	SESSION_KEY = "sessionID"
	USER_KEY    = "userKey"
)

type Entrance interface {
	Type() string
	Init(messenger Messenger, configs EntranceConfig) error
	Run() error
}

type EntranceConfig map[string]interface{}

type EntranceOptions struct {
	Type    string         `json:"type"`
	Options EntranceConfig `json:"options"`
}

func (p EntranceConfig) GetConfigString(sectionName string) (value string, ok bool) {
	if val, exist := p[sectionName]; !exist {
		return
	} else if strVal, ok := val.(string); ok {
		return strVal, true
	}
	return
}

func (p EntranceConfig) FillToObject(v interface{}) (err error) {
	if data, e := json.Marshal(p); e != nil {
		err = e
		return
	} else {
		err = json.Unmarshal(data, v)
	}
	return
}
