package main

import (
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
)

var (
	REDIS_COMMANDS = [...]string{
		"info", "exists", "type", "keys", "ttl", "set", "get", "getset", "setnx",
		"incr", "incrby", "decr", "decrby", "rpush", "lpush", "llen",
		"lrange", "ltrim", "lindex", "lset", "lrem", "lpop", "rpop",
		"sadd", "srem", "spop", "scard", "sismember", "smembers", "srandmember",
		"zadd", "zrem", "zincrby", "zrange", "zrevrange", "zrangebyscore",
		"zcard", "zscore", "zremrangebyscore", "expire", "expireat", "hlen",
		"hkeys", "hvals", "hgetall", "hset", "hget", "hincrby", "hexists", "hdel", "hmset"}
	CONNECTED       = false
	ERROR_MSG       = "错误的指令方式"
	UNSUPPORTED_MSG = "暂不支持该指令"
	OK_MSG          = "OK"
	client          redis.Conn
)

func contains(command string) bool {
	for _, c := range REDIS_COMMANDS {
		if c == command {
			return true
		}
	}
	return false
}

func evalRedisCommand(param martini.Params) (int, string) {
	var err error
	command := param["command"]
	inputs := strings.Split(command, " ")

	if !CONNECTED {
		if strings.ToUpper(inputs[0]) != "CONNECT" {
			return jsonRedisRetFail("请首先连接服务器")
		}
		if len(inputs) != 2 {
			return jsonRedisRetFail(ERROR_MSG)
		}
		client, err = redis.Dial("tcp", inputs[1])
		if err != nil {
			return jsonRedisRetFail(err.Error())
		} else {
			CONNECTED = true
			return jsonRedisRet(OK_MSG)
		}
	}

	if !contains(inputs[0]) {
		return jsonRedisRetFail(ERROR_MSG)
	}
	switch strings.ToUpper(inputs[0]) {
	case "GET":
		ret, err := client.Do(inputs[0], inputs[1])
		if err != nil {
			return jsonRedisRetFail(err.Error())
		}
		if ret == nil {
			return jsonRedisRet("nil")
		}
		resp, _ := redis.String(ret, err)
		return jsonRedisRet(resp)
	case "SET":
		if len(inputs) != 3 {
			return jsonRedisRetFail("SET需要两个参数, SET KEY VALUE")
		}
		_, err = client.Do(inputs[0], inputs[1], inputs[2])
		if err != nil {
			return jsonRedisRetFail(err.Error())
		}
		return jsonRedisRet(OK_MSG)
	case "DEL":
		if len(inputs) != 2 {
			return jsonRedisRetFail("DEL需要一个参数, DEL KEY")
		}
		_, err = client.Do(inputs[0], inputs[1])
		if err != nil {
			return jsonRedisRetFail(err.Error())
		}
		return jsonRedisRet(OK_MSG)
	case "KEYS":
		if len(inputs) != 2 {
			return jsonRedisRetFail("KEYS需要两个参数, KEYS PATTERN")
		}
		ret, err := redis.Strings(client.Do(inputs[0], inputs[1]))
		if err != nil {
			return jsonRedisRetFail(err.Error())
		}
		return jsonRedisRet(strings.Join(ret, ","))
	default:
		return jsonRedisRetFail(UNSUPPORTED_MSG)
	}
}
