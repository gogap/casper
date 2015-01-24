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

	CMD_HTTP_HEADER_WRITE = "CMD_HTTP_HEADER_WRITE"
	CMD_SET_SESSION       = "CMD_SET_SESSION"
)

type EntranceConfig map[string]interface{}

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

type Entrance interface {
	Type() string
	Init(app *App, configs EntranceConfig) error
	Run()
}
