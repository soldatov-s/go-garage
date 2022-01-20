package rabbitmqpub_test

import (
	"context"
	"errors"
	"testing"

	gomock "github.com/golang/mock/gomock"
	rabbitmqpub "github.com/soldatov-s/go-garage/providers/rabbitmq/publisher"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

var errTestSend = errors.New("failed to send")

func TestPublisher_SendMessage(t *testing.T) {
	type args struct {
		ctx     context.Context
		message interface{}
	}
	tests := []struct {
		name    string
		config  *rabbitmqpub.Config
		args    args
		wantErr bool
	}{
		{
			name: "normal send",
			config: &rabbitmqpub.Config{
				ExchangeName: "test_echange",
				RoutingKey:   "test_key",
			},
			args: args{
				ctx:     context.Background(),
				message: "test data",
			},
		},
		{
			name: "failed send",
			config: &rabbitmqpub.Config{
				ExchangeName: "test_echange",
				RoutingKey:   "test_key",
			},
			args: args{
				ctx:     context.Background(),
				message: "test data",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ch := NewMockConnector(ctrl)
			ch.EXPECT().ExchangeDeclare(
				tt.args.ctx,
				tt.config.ExchangeName,
				"direct",
				true,
				false,
				false,
				false,
				nil,
			).AnyTimes()

			ampqMsg := amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte("\"test data\""),
			}

			if !tt.wantErr {
				ch.EXPECT().Publish(
					tt.args.ctx,
					tt.config.ExchangeName,
					tt.config.RoutingKey,
					false,
					false,
					ampqMsg,
				).AnyTimes()
			} else {
				ch.EXPECT().Publish(
					tt.args.ctx,
					tt.config.ExchangeName,
					tt.config.RoutingKey,
					false,
					false,
					ampqMsg,
				).AnyTimes().Return(errTestSend)
			}

			p, err := rabbitmqpub.NewPublisher(
				context.Background(),
				tt.config,
				ch,
			)
			assert.Nil(t, err)
			if err := p.SendMessage(tt.args.ctx, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Publisher.SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
