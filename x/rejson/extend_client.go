// Based on KromDaniel/rejonson
package rejson

import (
	"context"

	"github.com/go-redis/redis/v8"
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
func (cl *redisProcessor) JSONDel(ctx context.Context, key, path string) *redis.IntCmd {
	return jsonDelExecute(ctx, cl, key, path)
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
func (cl *redisProcessor) JSONGet(ctx context.Context, key string, args ...interface{}) *redis.StringCmd {
	return jsonGetExecute(ctx, cl, append([]interface{}{key}, args...)...)
}

/*
jsonSet

Possible args:
(Optional)
*/
func (cl *redisProcessor) JSONSet(ctx context.Context, key, path, json string, args ...interface{}) *redis.StatusCmd {
	return jsonSetExecute(ctx, cl, append([]interface{}{key, path, json}, args...)...)
}

func (cl *redisProcessor) JSONMGet(ctx context.Context, key string, args ...interface{}) *redis.StringSliceCmd {
	return jsonMGetExecute(ctx, cl, append([]interface{}{key}, args...)...)
}

func (cl *redisProcessor) JSONType(ctx context.Context, key, path string) *redis.StringCmd {
	return jsonTypeExecute(ctx, cl, key, path)
}

func (cl *redisProcessor) JSONNumIncrBy(ctx context.Context, key, path string, num int) *redis.StringCmd {
	return jsonNumIncrByExecute(ctx, cl, key, path, num)
}

func (cl *redisProcessor) JSONNumMultBy(ctx context.Context, key, path string, num int) *redis.StringCmd {
	return jsonNumMultByExecute(ctx, cl, key, path, num)
}

func (cl *redisProcessor) JSONStrAppend(ctx context.Context, key, path, appendString string) *redis.IntCmd {
	return jsonStrAppendExecute(ctx, cl, key, path, appendString)
}

func (cl *redisProcessor) JSONStrLen(ctx context.Context, key, path string) *redis.IntCmd {
	return jsonStrLenExecute(ctx, cl, key, path)
}

func (cl *redisProcessor) JSONArrAppend(ctx context.Context, key, path string, jsons ...interface{}) *redis.IntCmd {
	return jsonArrAppendExecute(ctx, cl, append([]interface{}{key, path}, jsons...)...)
}

func (cl *redisProcessor) JSONArrIndex(ctx context.Context,
	key, path string, jsonScalar interface{}, startAndStop ...interface{}) *redis.IntCmd {
	return jsoArrIndexExecute(ctx, cl, append([]interface{}{key, path, jsonScalar}, startAndStop...)...)
}

func (cl *redisProcessor) JSONArrInsert(ctx context.Context, key, path string, index int, jsons ...interface{}) *redis.IntCmd {
	return jsonArrInsertExecute(ctx, cl, append([]interface{}{key, path, index}, jsons...)...)
}

func (cl *redisProcessor) JSONArrLen(ctx context.Context, key, path string) *redis.IntCmd {
	return jsonArrLenExecute(ctx, cl, key, path)
}

func (cl *redisProcessor) JSONArrPop(ctx context.Context, key, path string, index int) *redis.StringCmd {
	return jsonArrPopExecute(ctx, cl, key, path, index)
}

func (cl *redisProcessor) JSONArrTrim(ctx context.Context, key, path string, start, stop int) *redis.IntCmd {
	return jsonArrTrimExecute(ctx, cl, key, path, start, stop)
}

func (cl *redisProcessor) JSONObjKeys(ctx context.Context, key, path string) *redis.StringSliceCmd {
	return jsonObjKeysExecute(ctx, cl, key, path)
}

func (cl *redisProcessor) JSONObjLen(ctx context.Context, key, path string) *redis.IntCmd {
	return jsonObjLen(ctx, cl, key, path)
}
