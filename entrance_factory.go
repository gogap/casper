package casper

import (
	"fmt"
	"reflect"
)

type EntranceFactory interface {
	RegisterEntrance(entrance Entrance)
	NewEntrance(messengerr Messenger, typ string, configs EntranceConfig) Entrance
}

type DefaultEntranceFactory struct {
	entrances map[string]reflect.Type
}

func NewDefaultEntranceFactory() *DefaultEntranceFactory {
	fact := new(DefaultEntranceFactory)
	fact.entrances = make(map[string]reflect.Type)
	return fact
}

func (p *DefaultEntranceFactory) RegisterEntrance(entrance Entrance) {
	if entrance == nil {
		panic("entrance is nil")
	}
	if _, exist := p.entrances[entrance.Type()]; exist {
		panic(fmt.Errorf("entrance of %s already exist", entrance.Type()))
	}

	vof := reflect.ValueOf(entrance)
	vType := vof.Type()
	if vof.Kind() == reflect.Ptr {
		vType = vof.Elem().Type()
	}

	p.entrances[entrance.Type()] = vType
	return
}

func (p *DefaultEntranceFactory) NewEntrance(messengerr Messenger, typ string, configs EntranceConfig) Entrance {
	if entranceType, exist := p.entrances[typ]; !exist {
		panic(fmt.Errorf("entrance of %s not exist", typ))
	} else {
		if vOfEntrance := reflect.New(entranceType); vOfEntrance.CanInterface() {
			iEntrance := vOfEntrance.Interface()
			if entrance, ok := iEntrance.(Entrance); ok {
				entrance.Init(messengerr, configs)
				return entrance
			} else {
				panic(fmt.Errorf("convert value to interface{} of Entrance failed, entrance type is: %s", typ))
			}
		}
		panic(fmt.Errorf("create entrance of %s failed", typ))
	}
}
