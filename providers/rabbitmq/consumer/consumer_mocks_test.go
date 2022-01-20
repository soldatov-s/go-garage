// Code generated by MockGen. DO NOT EDIT.
// Source: ./consumer.go

// Package rabbitmqconsum_test is a generated GoMock package.
package rabbitmqconsum_test

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	amqp "github.com/streadway/amqp"
)

// MockConnector is a mock of Connector interface.
type MockConnector struct {
	ctrl     *gomock.Controller
	recorder *MockConnectorMockRecorder
}

// MockConnectorMockRecorder is the mock recorder for MockConnector.
type MockConnectorMockRecorder struct {
	mock *MockConnector
}

// NewMockConnector creates a new mock instance.
func NewMockConnector(ctrl *gomock.Controller) *MockConnector {
	mock := &MockConnector{ctrl: ctrl}
	mock.recorder = &MockConnectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnector) EXPECT() *MockConnectorMockRecorder {
	return m.recorder
}

// Consume mocks base method.
func (m *MockConnector) Consume(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Consume", ctx, queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	ret0, _ := ret[0].(<-chan amqp.Delivery)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Consume indicates an expected call of Consume.
func (mr *MockConnectorMockRecorder) Consume(ctx, queue, consumer, autoAck, exclusive, noLocal, noWait, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Consume", reflect.TypeOf((*MockConnector)(nil).Consume), ctx, queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

// ExchangeDeclare mocks base method.
func (m *MockConnector) ExchangeDeclare(ctx context.Context, name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExchangeDeclare", ctx, name, kind, durable, autoDelete, internal, noWait, args)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExchangeDeclare indicates an expected call of ExchangeDeclare.
func (mr *MockConnectorMockRecorder) ExchangeDeclare(ctx, name, kind, durable, autoDelete, internal, noWait, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExchangeDeclare", reflect.TypeOf((*MockConnector)(nil).ExchangeDeclare), ctx, name, kind, durable, autoDelete, internal, noWait, args)
}

// IsClosed mocks base method.
func (m *MockConnector) IsClosed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsClosed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsClosed indicates an expected call of IsClosed.
func (mr *MockConnectorMockRecorder) IsClosed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsClosed", reflect.TypeOf((*MockConnector)(nil).IsClosed))
}

// QueueBind mocks base method.
func (m *MockConnector) QueueBind(ctx context.Context, name, key, exchange string, noWait bool, args amqp.Table) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueBind", ctx, name, key, exchange, noWait, args)
	ret0, _ := ret[0].(error)
	return ret0
}

// QueueBind indicates an expected call of QueueBind.
func (mr *MockConnectorMockRecorder) QueueBind(ctx, name, key, exchange, noWait, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueBind", reflect.TypeOf((*MockConnector)(nil).QueueBind), ctx, name, key, exchange, noWait, args)
}

// QueueDeclare mocks base method.
func (m *MockConnector) QueueDeclare(ctx context.Context, name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueDeclare", ctx, name, durable, autoDelete, exclusive, noWait, args)
	ret0, _ := ret[0].(amqp.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueueDeclare indicates an expected call of QueueDeclare.
func (mr *MockConnectorMockRecorder) QueueDeclare(ctx, name, durable, autoDelete, exclusive, noWait, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueDeclare", reflect.TypeOf((*MockConnector)(nil).QueueDeclare), ctx, name, durable, autoDelete, exclusive, noWait, args)
}

// MockSubscriber is a mock of Subscriber interface.
type MockSubscriber struct {
	ctrl     *gomock.Controller
	recorder *MockSubscriberMockRecorder
}

// MockSubscriberMockRecorder is the mock recorder for MockSubscriber.
type MockSubscriberMockRecorder struct {
	mock *MockSubscriber
}

// NewMockSubscriber creates a new mock instance.
func NewMockSubscriber(ctrl *gomock.Controller) *MockSubscriber {
	mock := &MockSubscriber{ctrl: ctrl}
	mock.recorder = &MockSubscriberMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscriber) EXPECT() *MockSubscriberMockRecorder {
	return m.recorder
}

// Consume mocks base method.
func (m *MockSubscriber) Consume(ctx context.Context, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Consume", ctx, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Consume indicates an expected call of Consume.
func (mr *MockSubscriberMockRecorder) Consume(ctx, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Consume", reflect.TypeOf((*MockSubscriber)(nil).Consume), ctx, data)
}

// Shutdown mocks base method.
func (m *MockSubscriber) Shutdown(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Shutdown", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Shutdown indicates an expected call of Shutdown.
func (mr *MockSubscriberMockRecorder) Shutdown(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Shutdown", reflect.TypeOf((*MockSubscriber)(nil).Shutdown), ctx)
}
