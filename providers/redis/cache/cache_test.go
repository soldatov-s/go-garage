package rediscache_test

import (
	"context"
	reflect "reflect"
	"testing"
	"unsafe"

	"github.com/go-redis/redis/v8"
	gomock "github.com/golang/mock/gomock"
	rediscache "github.com/soldatov-s/go-garage/providers/redis/cache"
	"github.com/stretchr/testify/assert"
)

func TestCache_Get(t *testing.T) {
	type fields struct {
		config *rediscache.Config
		name   string
	}
	type args struct {
		key      string
		ctx      context.Context
		value    interface{}
		redisCmd func() interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "success test",
			args: args{
				key:   "test",
				ctx:   context.Background(),
				value: reflect.New(reflect.ValueOf("empty").Type()).Interface(),
				redisCmd: func() interface{} {
					cmd := &redis.StringCmd{}
					changeStringCmdVal(cmd, "test data")
					return cmd
				},
			},
			fields: fields{
				name: "test",
				config: &rediscache.Config{
					KeyPrefix: "test",
					ClearTime: 0,
				},
			},
		},
		{
			name: "not found",
			args: args{
				key:   "test",
				ctx:   context.Background(),
				value: reflect.New(reflect.ValueOf("empty").Type()).Interface(),
				redisCmd: func() interface{} {
					cmd := &redis.StringCmd{}
					changeStringCmdErr(cmd, redis.Nil)
					return cmd
				},
			},
			fields: fields{
				name: "test",
				config: &rediscache.Config{
					KeyPrefix: "test",
					ClearTime: 0,
				},
			},
			wantErr: true,
			err:     rediscache.ErrNotFoundInCache,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := NewMockConnector(ctrl)

			c, err := rediscache.NewCache(
				tt.args.ctx,
				tt.fields.name,
				tt.fields.config,
				conn,
			)
			assert.Nil(t, err)

			conn.EXPECT().Get(
				tt.args.ctx,
				c.BuildKey(tt.args.key),
			).MaxTimes(1).Return(tt.args.redisCmd())

			err = c.Get(tt.args.ctx, tt.args.key, tt.args.value)
			if !tt.wantErr {
				assert.Nil(t, err)
				t.Logf("data %q", reflect.Indirect(reflect.ValueOf(tt.args.value)))
			} else {
				assert.ErrorIs(t, err, tt.err)
				t.Logf("err: %s", err)
			}
		})
	}
}

func TestCache_Set(t *testing.T) {
	type fields struct {
		config *rediscache.Config
		name   string
	}
	type args struct {
		key      string
		ctx      context.Context
		value    interface{}
		redisCmd func() interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "success test",
			args: args{
				key:   "test",
				ctx:   context.Background(),
				value: "test value",
				redisCmd: func() interface{} {
					cmd := &redis.StatusCmd{}
					return cmd
				},
			},
			fields: fields{
				name: "test",
				config: &rediscache.Config{
					KeyPrefix: "test",
					ClearTime: 5,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := NewMockConnector(ctrl)

			c, err := rediscache.NewCache(
				tt.args.ctx,
				tt.fields.name,
				tt.fields.config,
				conn,
			)
			assert.Nil(t, err)

			conn.EXPECT().Set(
				tt.args.ctx,
				c.BuildKey(tt.args.key),
				tt.args.value,
				tt.fields.config.ClearTime,
			).MaxTimes(1).Return(tt.args.redisCmd())

			err = c.Set(tt.args.ctx, tt.args.key, tt.args.value)
			if !tt.wantErr {
				assert.Nil(t, err)
			} else {
				assert.ErrorIs(t, err, tt.err)
				t.Logf("err: %s", err)
			}
		})
	}
}

func TestCache_Size(t *testing.T) {
	type fields struct {
		config *rediscache.Config
		name   string
	}
	type args struct {
		key      string
		ctx      context.Context
		value    interface{}
		redisCmd func() interface{}
		keys     []string
		size     int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "success test",
			args: args{
				key:   "test",
				ctx:   context.Background(),
				value: "test value",
				redisCmd: func() interface{} {
					cmd := &redis.ScanCmd{}
					changeScanCmdPage(cmd, []string{"test:key1", "test:key2", "test:key3"})
					return cmd
				},
				keys: []string{"test:key1", "test:key2", "test:key3"},
				size: 3,
			},
			fields: fields{
				name: "test",
				config: &rediscache.Config{
					KeyPrefix: "test",
					ClearTime: 5,
					ScanSize:  10,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := NewMockConnector(ctrl)

			c, err := rediscache.NewCache(
				tt.args.ctx,
				tt.fields.name,
				tt.fields.config,
				conn,
			)
			assert.Nil(t, err)

			var cursor uint64
			conn.EXPECT().Scan(
				tt.args.ctx,
				cursor,
				c.KeyPrefix()+"*",
				tt.fields.config.ScanSize,
			).MaxTimes(1).Return(tt.args.redisCmd())

			size, err := c.Size(tt.args.ctx)
			if !tt.wantErr {
				assert.Nil(t, err)
				assert.Equal(t, tt.args.size, size)
			} else {
				assert.ErrorIs(t, err, tt.err)
				t.Logf("err: %s", err)
			}
		})
	}
}

func changeStringCmdVal(c *redis.StringCmd, v string) {
	pointerVal := reflect.ValueOf(c)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("val")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToY := (*string)(ptrToY)
	*realPtrToY = v
}

func changeStringCmdErr(c *redis.StringCmd, v error) {
	pointerVal := reflect.ValueOf(c)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("err")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToY := (*error)(ptrToY)
	*realPtrToY = v
}

func changeScanCmdPage(c *redis.ScanCmd, page []string) {
	pointerVal := reflect.ValueOf(c)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("page")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToY := (*[]string)(ptrToY)
	*realPtrToY = page
}
