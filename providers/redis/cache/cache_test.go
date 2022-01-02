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
		key   string
		ctx   context.Context
		value interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success test",
			args: args{
				key:   "test",
				ctx:   context.Background(),
				value: reflect.New(reflect.ValueOf("empty").Type()).Interface(),
			},
			fields: fields{
				name: "test",
				config: &rediscache.Config{
					KeyPrefix: "test",
					ClearTime: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := NewMockConnector(ctrl)
			cmd := &redis.StringCmd{}
			changeStringCmdVal(cmd, "test data")
			conn.EXPECT().Get(
				tt.args.ctx,
				tt.fields.config.KeyPrefix+":"+tt.args.key,
			).AnyTimes().Return(cmd)

			c, err := rediscache.NewCache(
				tt.args.ctx,
				tt.fields.name,
				tt.fields.config,
				conn,
			)
			assert.Nil(t, err)

			if err := c.Get(tt.args.ctx, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Cache.Get() error = %v, wantErr %v", err, tt.wantErr)
			}

			t.Logf("data %q", reflect.Indirect(reflect.ValueOf(tt.args.value)))
		})
	}
}

func changeStringCmdVal(f *redis.StringCmd, v string) {
	pointerVal := reflect.ValueOf(f)
	val := reflect.Indirect(pointerVal)
	member := val.FieldByName("val")
	ptrToY := unsafe.Pointer(member.UnsafeAddr())
	realPtrToY := (*string)(ptrToY)
	*realPtrToY = v
}
