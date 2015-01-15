package casper

import (
	"fmt"
)

// 消息接口
type mqI interface {
	SetPara(key string, val interface{}) // 设置参数
	Ready() error                        // 初始化
	RecvMessage() ([]byte, error)        // 读一条消息
	SendToNext([]byte) (int, error)      // 发送一条消息到下一节点
}

type mqType func() mqI

var mqs map[string]mqType = make(map[string]mqType)

func registerMq(name string, one mqType) {
	if one == nil {
		panic("register MQ nil")
	}
	if _, dup := mqs[name]; dup {
		panic("register mq duplicate for " + name)
	}
	mqs[name] = one
}

func NewMq(typeName string) (mqI, error) {
	if newFun, ok := mqs[typeName]; ok {
		return newFun(), nil
	}

	return nil, fmt.Errorf("no mq types " + typeName)
}
