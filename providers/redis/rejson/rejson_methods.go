// Based on KromDaniel/rejonson
package rejson

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

func concatWithCmd(cmdName string, args []interface{}) []interface{} {
	res := make([]interface{}, 0)
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

func jsonDelExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.DEL", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonGetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringCmd, error) {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.GET", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonSetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StatusCmd, error) {
	cmd := redis.NewStatusCmd(ctx, concatWithCmd("JSON.SET", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonMGetExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringSliceCmd, error) {
	cmd := redis.NewStringSliceCmd(ctx, concatWithCmd("JSON.MGET", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonTypeExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringCmd, error) {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.TYPE", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonNumIncrByExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringCmd, error) {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.NUMINCRBY", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonNumMultByExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringCmd, error) {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.NUMMULTBY", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonStrAppendExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.STRAPPEND", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonStrLenExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.STRLEN", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonArrAppendExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRAPPEND", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsoArrIndexExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRINDEX", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonArrInsertExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRINSERT", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonArrLenExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRLEN", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonArrPopExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringCmd, error) {
	cmd := redis.NewStringCmd(ctx, concatWithCmd("JSON.ARRPOP", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonArrTrimExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.ARRTRIM", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonObjKeysExecute(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.StringSliceCmd, error) {
	cmd := redis.NewStringSliceCmd(ctx, concatWithCmd("JSON.OBJKEYS", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}

func jsonObjLen(ctx context.Context, c *redisProcessor, args ...interface{}) (*redis.IntCmd, error) {
	cmd := redis.NewIntCmd(ctx, concatWithCmd("JSON.OBJLEN", args)...)
	if err := c.Process(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "process command")
	}
	return cmd, nil
}
