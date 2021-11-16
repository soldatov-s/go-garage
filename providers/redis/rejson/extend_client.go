// Based on KromDaniel/rejonson
package rejson

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type redisProcessor struct {
	Process func(ctx context.Context, cmd redis.Cmder) error
}

/*
Client is an extended redis.Client, stores a pointer to the original redis.Client
*/
type Client struct {
	*redis.Client
	*redisProcessor
}

/*
Pipeline is an extended redis.Pipeline, stores a pointer to the original redis.Pipeliner
*/
type Pipeline struct {
	redis.Pipeliner
	*redisProcessor
}

func (cl *Client) Pipeline() *Pipeline {
	pip := cl.Client.Pipeline()
	return ExtendPipeline(pip)
}

func (cl *Client) TXPipeline() *Pipeline {
	pip := cl.Client.TxPipeline()
	return ExtendPipeline(pip)
}

func (pl *Pipeline) Pipeline() *Pipeline {
	pip := pl.Pipeliner.Pipeline()
	return ExtendPipeline(pip)
}

/*
JSONDel

returns intCmd -> deleted 1 or 0
read more: https://oss.redislabs.com/rejson/commands/#jsondel
*/
func (cl *redisProcessor) JSONDel(ctx context.Context, key, path string) (*redis.IntCmd, error) {
	cmd, err := jsonDelExecute(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonDelExecute")
	}
	return cmd, nil
}

/*
JsonGet

Possible args:

(Optional) INDENT + indent-string
(Optional) NEWLINE + line-break-string
(Optional) SPACE + space-string
(Optional) NOESCAPE
(Optional) path ...string

returns stringCmd -> the JSON string
read more: https://oss.redislabs.com/rejson/commands/#jsonget
*/
func (cl *redisProcessor) JSONGet(ctx context.Context, key string, args ...interface{}) (*redis.StringCmd, error) {
	cmd, err := jsonGetExecute(ctx, cl, append([]interface{}{key}, args...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsonGetExecute")
	}
	return cmd, nil
}

/*
jsonSet

Possible args:
(Optional)
*/
func (cl *redisProcessor) JSONSet(ctx context.Context, key, path, json string, args ...interface{}) (*redis.StatusCmd, error) {
	cmd, err := jsonSetExecute(ctx, cl, append([]interface{}{key, path, json}, args...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsonSetExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONMGet(ctx context.Context, key string, args ...interface{}) (*redis.StringSliceCmd, error) {
	cmd, err := jsonMGetExecute(ctx, cl, append([]interface{}{key}, args...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsonSetExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONType(ctx context.Context, key, path string) (*redis.StringCmd, error) {
	cmd, err := jsonTypeExecute(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonTypeExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONNumIncrBy(ctx context.Context, key, path string, num int) (*redis.StringCmd, error) {
	cmd, err := jsonNumIncrByExecute(ctx, cl, key, path, num)
	if err != nil {
		return nil, errors.Wrap(err, "jsonNumIncrByExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONNumMultBy(ctx context.Context, key, path string, num int) (*redis.StringCmd, error) {
	cmd, err := jsonNumMultByExecute(ctx, cl, key, path, num)
	if err != nil {
		return nil, errors.Wrap(err, "jsonNumMultByExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONStrAppend(ctx context.Context, key, path, appendString string) (*redis.IntCmd, error) {
	cmd, err := jsonStrAppendExecute(ctx, cl, key, path, appendString)
	if err != nil {
		return nil, errors.Wrap(err, "jsonStrAppendExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONStrLen(ctx context.Context, key, path string) (*redis.IntCmd, error) {
	cmd, err := jsonStrLenExecute(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonStrLenExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrAppend(ctx context.Context, key, path string, jsons ...interface{}) (*redis.IntCmd, error) {
	cmd, err := jsonArrAppendExecute(ctx, cl, append([]interface{}{key, path}, jsons...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsonArrAppendExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrIndex(ctx context.Context,
	key, path string, jsonScalar interface{}, startAndStop ...interface{}) (*redis.IntCmd, error) {
	cmd, err := jsoArrIndexExecute(ctx, cl, append([]interface{}{key, path, jsonScalar}, startAndStop...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsoArrIndexExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrInsert(
	ctx context.Context,
	key, path string,
	index int,
	jsons ...interface{}) (*redis.IntCmd, error) {
	cmd, err := jsonArrInsertExecute(ctx, cl, append([]interface{}{key, path, index}, jsons...)...)
	if err != nil {
		return nil, errors.Wrap(err, "jsonArrInsertExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrLen(ctx context.Context, key, path string) (*redis.IntCmd, error) {
	cmd, err := jsonArrLenExecute(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonArrLenExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrPop(ctx context.Context, key, path string, index int) (*redis.StringCmd, error) {
	cmd, err := jsonArrPopExecute(ctx, cl, key, path, index)
	if err != nil {
		return nil, errors.Wrap(err, "jsonArrPopExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONArrTrim(ctx context.Context, key, path string, start, stop int) (*redis.IntCmd, error) {
	cmd, err := jsonArrTrimExecute(ctx, cl, key, path, start, stop)
	if err != nil {
		return nil, errors.Wrap(err, "jsonArrTrimExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONObjKeys(ctx context.Context, key, path string) (*redis.StringSliceCmd, error) {
	cmd, err := jsonObjKeysExecute(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonObjKeysExecute")
	}
	return cmd, nil
}

func (cl *redisProcessor) JSONObjLen(ctx context.Context, key, path string) (*redis.IntCmd, error) {
	cmd, err := jsonObjLen(ctx, cl, key, path)
	if err != nil {
		return nil, errors.Wrap(err, "jsonObjLen")
	}
	return cmd, nil
}
