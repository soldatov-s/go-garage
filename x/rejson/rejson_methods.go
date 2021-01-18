// Based on KromDaniel/rejonson
package rejson

import (
	"context"

	"github.com/go-redis/redis/v8"
)

func concatWithCmd(cmdName string, args []interface{}) []interface{} {
	res := make([]interface{}, 1)
	res[0] = cmdName
	for _, v := range args {
		if str, ok := v.(string); ok {
			if str == "" {
				continue
			}
		}
		res = append(res, v)
	}
	return res
}

func jsonDelExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.DEL", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonGetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.GET", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonSetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, concatWithCmd("JSON.SET", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonMGetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx, concatWithCmd("JSON.MGET", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonTypeExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.TYPE", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonNumIncrByExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.NUMINCRBY", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonNumMultByExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.NUMMULTBY", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonStrAppendExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.STRAPPEND", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonStrLenExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.STRLEN", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonArrAppendExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRAPPEND", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsoArrIndexExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRINDEX", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonArrInsertExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRINSERT", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonArrLenExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRLEN", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonArrPopExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.ARRPOP", args)...)
	_ = c.Process(ctx, cmd)
	return cmd
}

func jsonArrTrimExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRTRIM", args)...)
	c.Process(ctx, cmd)
	return cmd
}

func jsonObjKeysExecute(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx, concatWithCmd("JSON.OBJKEYS", args)...)
	c.Process(ctx, cmd)
	return cmd
}

func jsonObjLen(ctx context.Context, c *redisProcessor, args ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.OBJLEN", args)...)
	c.Process(ctx, cmd)
	return cmd
}
