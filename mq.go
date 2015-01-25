package casper

import (
	"fmt"
)

var mqs map[string]mqType = make(map[string]mqType)

// 消息接口
type MessageQueue interface {
	Ready() error                   // 初始化
	RecvMessage() ([]byte, error)   // 读一条消息
	SendToNext([]byte) (int, error) // 发送一条消息到下一节点
}

type mqType func(string) MessageQueue

func registerMq(name string, one mqType) {
	if one == nil {
		panic("Register MQ nil")
	}
	if _, dup := mqs[name]; dup {
		panic("Register MQ duplicate for " + name)
	}
	mqs[name] = one
}

func NewMQ(compMeta *ComponentMetadata) (MessageQueue, error) {
	if compMeta == nil {
		return nil, fmt.Errorf("MessageQueue's metadata is nil")
	}

	if compMeta.MQType == "" {
		return nil, fmt.Errorf("MessageQueue's type nil")
	}
	if compMeta.In == "" {
		return nil, fmt.Errorf("MessageQueue's in nil")
	}

	if newFun, ok := mqs[compMeta.MQType]; ok {
		return newFun(compMeta.In), nil
	}

	return nil, fmt.Errorf("no MessageQueue types " + compMeta.MQType)
}
