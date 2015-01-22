package casper

import (
	"github.com/hoisie/redis"
)

var redisClient redis.Client

func SessionSetByte(sessionid, key string, val []byte) (ok bool, err error) {
	return redisClient.Hset(sessionid, key, val)
}

func SessionGetByte(sessionid, key string) (val []byte, err error) {
	return redisClient.Hget(sessionid, key)
}

func SessionDel(sessionid string) (ok bool, err error) {
	return redisClient.Del(sessionid)
}
