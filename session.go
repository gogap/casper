package casper

import (
	"github.com/hoisie/redis"
)

var redisClient redis.Client

func SessionSetByte(sessionid, key string, val []byte, overtime int64) (ok bool, err error) {
	ok, err = redisClient.Hset(sessionid, key, val)
	if overtime > 0 {
		ok, err = redisClient.Expire(sessionid, overtime)
	}

	return
}

func SessionSetOvertime(sessionid string, overtime int64) (ok bool, err error) {
	return redisClient.Expire(sessionid, overtime)
}

func SessionGetByte(sessionid, key string) (val []byte, err error) {
	return redisClient.Hget(sessionid, key)
}

func SessionDel(sessionid string) (ok bool, err error) {
	return redisClient.Del(sessionid)
}
