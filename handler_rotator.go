package casper

import (
	"encoding/json"
	"fmt"
	"os"
)

type handler func(*Payload) (interface{}, error)

type HandlerRotator struct {
	handler  handler
	handlers map[string]handler
}
type RotatorParams map[string]handler

var (
	rotatorConfig map[string]string = make(map[string]string)
)

func NewHandlerRotator(componentName string, handlers RotatorParams) (rotator *HandlerRotator) {
	if handlers == nil {
		panic(fmt.Errorf("handlers can not be empty."))
	}
	rotator = new(HandlerRotator)
	if rotator.handlers == nil {
		rotator.handlers = make(map[string]handler, len(handlers))
		rotator.handlers = handlers
		var name string
		if name = rotatorConfig[componentName]; name == "" {
			panic(fmt.Errorf("can not found default component handler's name in config: %s", componentName))
		}
		if handlers[name] == nil {
			panic(fmt.Errorf("can not found default component handler in rotator params: %s", name))
		} else {
			rotator.handler = handlers[name]
		}
	}
	return
}

func (p *HandlerRotator) Handler(payload *Payload) (interface{}, error) {
	return p.handler(payload)
}

func (p *HandlerRotator) Rotate(name string) error {
	if h := p.handlers[name]; h == nil {
		return fmt.Errorf("handler not found with handler' name: %s. ", name)
	} else {
		p.handler = h
	}
	return nil
}

func BuildHandlerRotatorConfig(configPath string) {
	var conf struct {
		ComHandlers []struct {
			Name    string `json:"component_name"`
			Handler string `json:"handler"`
		} `json:"handlers"`
	}
	r, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		panic(err)
	}
	for _, v := range conf.ComHandlers {
		rotatorConfig[v.Name] = v.Handler
	}
}
