// nolint:lll // long strings
package rabbitmqconsum_test

import (
	"context"
	"errors"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/rs/zerolog/log"
	rabbitmqconsum "github.com/soldatov-s/go-garage/providers/rabbitmq/consumer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen --source=../../../vendor/github.com/streadway/amqp/delivery.go -destination=./delivery_mocks_test.go -package=rabbitmqconsum_test

var errEndTest = errors.New("end test")

func TestConsumer_Subscribe(t *testing.T) {
	type fields struct {
		config *rabbitmqconsum.Config
	}
	tests := []struct {
		name    string
		fields  fields
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ch := NewMockConnector(ctrl)
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

			msg := make(chan amqp.Delivery)
			// nolint:gocritic // no necessary simplify
			msgOut := (<-chan amqp.Delivery)(msg)

			ch.EXPECT().Consume(
				tt.fields.config.RabbitQueue,   // queue
				tt.fields.config.RabbitConsume, // consume
				false,                          // auto-ack
				false,                          // exclusive
				false,                          // no-local
				false,                          // no-wait
				nil,                            // args
			).AnyTimes().Return(msgOut, nil)

			ctx := context.Background()
			ctx = log.Logger.WithContext(ctx)
			errorGroup, ctx := errgroup.WithContext(ctx)

			subs := NewMockSubscriber(ctrl)
			subs.EXPECT().Shutdown(ctx).AnyTimes()
			data := []byte("test message")
			subs.EXPECT().Consume(ctx, data).AnyTimes()

			acknowledger := NewMockAcknowledger(ctrl)
			acknowledger.EXPECT().Ack(uint64(0), true).AnyTimes()

			c, err := rabbitmqconsum.NewConsumer(ctx, tt.fields.config, ch)
			assert.Nil(t, err)

			err = c.Subscribe(ctx, errorGroup, subs)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.err)
			}

			msg <- amqp.Delivery{
				Body:         []byte("test message"),
				Acknowledger: acknowledger,
			}

			errorGroup.Go(func() error {
				time.Sleep(2 * time.Second)
				return errEndTest
			})

			err = errorGroup.Wait()
			assert.ErrorIs(t, err, errEndTest)
		})
	}
}
