package rabbitmqconsum_test

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	rabbitmqconsum "github.com/soldatov-s/go-garage/providers/rabbitmq/consumer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestConsumer_Subscribe(t *testing.T) {
	type fields struct {
		config *rabbitmqconsum.Config
	}
	type args struct {
		ctx        context.Context
		subscriber rabbitmqconsum.Subscriber
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "success",
			fields: fields{
				config: &rabbitmqconsum.Config{
					ExchangeName:  "test_exch",
					RoutingKey:    "test",
					RabbitQueue:   "test_queue",
					RabbitConsume: "test_consum",
				},
			},
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ch := NewMockChanneler(ctrl)
			ch.EXPECT().ExchangeDeclare(
				tt.fields.config.ExchangeName,
				"direct",
				true,
				false,
				false,
				false,
				nil,
			).AnyTimes()

			ch.EXPECT().QueueDeclare(
				tt.fields.config.RabbitQueue, // name
				true,                         // durable
				false,                        // delete when unused
				false,                        // exclusive
				false,                        // no-wait
				nil,                          // arguments
			).AnyTimes()

			ch.EXPECT().QueueBind(
				tt.fields.config.RabbitQueue,  // queue name
				tt.fields.config.RoutingKey,   // routing key
				tt.fields.config.ExchangeName, // exchange
				false,
				nil,
			).AnyTimes()

			ch.EXPECT().Consume(
				tt.fields.config.RabbitQueue,   // queue
				tt.fields.config.RabbitConsume, // consume
				false,                          // auto-ack
				false,                          // exclusive
				false,                          // no-local
				false,                          // no-wait
				nil,                            // args
			).AnyTimes().Return(make(<-chan amqp.Delivery), nil)

			ctx := log.Logger.WithContext(tt.args.ctx)

			c, err := rabbitmqconsum.NewConsumer(ctx, tt.fields.config, ch)
			assert.Nil(t, err)

			errorGroup, ctx := errgroup.WithContext(ctx)
			err = c.Subscribe(ctx, errorGroup, tt.args.subscriber)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.err)
			}

			qq := zerolog.Ctx(ctx)
			qq.Info().Msg("ttt")

			tt.args.ctx.Done()
		})
	}
}
