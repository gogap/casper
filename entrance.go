package casper

import (
	"fmt"
)

var entrances map[string]entranceType = make(map[string]entranceType)

// 入口服务的接口
type entrance interface {
	StartService(*App, string) // 开始服务
}

type entranceType func() entrance

func registerEntrance(name string, one entranceType) {
	if one == nil {
		panic("Register nil entrance")
	}
	if _, dup := entrances[name]; dup {
		panic("Register entrance duplicate for " + name)
	}
	entrances[name] = one
}

func NewEntrance(typeName string) (entrance, error) {
	if newFun, ok := entrances[typeName]; ok {
		return newFun(), nil
	}

	
	return nil, fmt.Errorf("No entrance types " + typeName)	
}
