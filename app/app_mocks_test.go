// Code generated by MockGen. DO NOT EDIT.
// Source: ./app.go

// Package app_test is a generated GoMock package.
package app_test

import (
	context "context"
	http "net/http"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	base "github.com/soldatov-s/go-garage/base"
	httpx "github.com/soldatov-s/go-garage/x/httpx"
	errgroup "golang.org/x/sync/errgroup"
)

// MockHTTPServer is a mock of HTTPServer interface.
type MockHTTPServer struct {
	ctrl     *gomock.Controller
	recorder *MockHTTPServerMockRecorder
}

// MockHTTPServerMockRecorder is the mock recorder for MockHTTPServer.
type MockHTTPServerMockRecorder struct {
	mock *MockHTTPServer
}

// NewMockHTTPServer creates a new mock instance.
func NewMockHTTPServer(ctrl *gomock.Controller) *MockHTTPServer {
	mock := &MockHTTPServer{ctrl: ctrl}
	mock.recorder = &MockHTTPServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHTTPServer) EXPECT() *MockHTTPServerMockRecorder {
	return m.recorder
}

// RegisterEndpoint mocks base method.
func (m_2 *MockHTTPServer) RegisterEndpoint(method, endpoint string, handler http.Handler, m ...httpx.MiddleWareFunc) error {
	m_2.ctrl.T.Helper()
	varargs := []interface{}{method, endpoint, handler}
	for _, a := range m {
		varargs = append(varargs, a)
	}
	ret := m_2.ctrl.Call(m_2, "RegisterEndpoint", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterEndpoint indicates an expected call of RegisterEndpoint.
func (mr *MockHTTPServerMockRecorder) RegisterEndpoint(method, endpoint, handler interface{}, m ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{method, endpoint, handler}, m...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterEndpoint", reflect.TypeOf((*MockHTTPServer)(nil).RegisterEndpoint), varargs...)
}

// MockEnityMetricsGateway is a mock of EnityMetricsGateway interface.
type MockEnityMetricsGateway struct {
	ctrl     *gomock.Controller
	recorder *MockEnityMetricsGatewayMockRecorder
}

// MockEnityMetricsGatewayMockRecorder is the mock recorder for MockEnityMetricsGateway.
type MockEnityMetricsGatewayMockRecorder struct {
	mock *MockEnityMetricsGateway
}

// NewMockEnityMetricsGateway creates a new mock instance.
func NewMockEnityMetricsGateway(ctrl *gomock.Controller) *MockEnityMetricsGateway {
	mock := &MockEnityMetricsGateway{ctrl: ctrl}
	mock.recorder = &MockEnityMetricsGatewayMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnityMetricsGateway) EXPECT() *MockEnityMetricsGatewayMockRecorder {
	return m.recorder
}

// GetMetrics mocks base method.
func (m *MockEnityMetricsGateway) GetMetrics() *base.MapMetricsOptions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMetrics")
	ret0, _ := ret[0].(*base.MapMetricsOptions)
	return ret0
}

// GetMetrics indicates an expected call of GetMetrics.
func (mr *MockEnityMetricsGatewayMockRecorder) GetMetrics() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMetrics", reflect.TypeOf((*MockEnityMetricsGateway)(nil).GetMetrics))
}

// MockEnityAliveGateway is a mock of EnityAliveGateway interface.
type MockEnityAliveGateway struct {
	ctrl     *gomock.Controller
	recorder *MockEnityAliveGatewayMockRecorder
}

// MockEnityAliveGatewayMockRecorder is the mock recorder for MockEnityAliveGateway.
type MockEnityAliveGatewayMockRecorder struct {
	mock *MockEnityAliveGateway
}

// NewMockEnityAliveGateway creates a new mock instance.
func NewMockEnityAliveGateway(ctrl *gomock.Controller) *MockEnityAliveGateway {
	mock := &MockEnityAliveGateway{ctrl: ctrl}
	mock.recorder = &MockEnityAliveGatewayMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnityAliveGateway) EXPECT() *MockEnityAliveGatewayMockRecorder {
	return m.recorder
}

// GetAliveHandlers mocks base method.
func (m *MockEnityAliveGateway) GetAliveHandlers() *base.MapCheckOptions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAliveHandlers")
	ret0, _ := ret[0].(*base.MapCheckOptions)
	return ret0
}

// GetAliveHandlers indicates an expected call of GetAliveHandlers.
func (mr *MockEnityAliveGatewayMockRecorder) GetAliveHandlers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAliveHandlers", reflect.TypeOf((*MockEnityAliveGateway)(nil).GetAliveHandlers))
}

// MockEnityReadyGateway is a mock of EnityReadyGateway interface.
type MockEnityReadyGateway struct {
	ctrl     *gomock.Controller
	recorder *MockEnityReadyGatewayMockRecorder
}

// MockEnityReadyGatewayMockRecorder is the mock recorder for MockEnityReadyGateway.
type MockEnityReadyGatewayMockRecorder struct {
	mock *MockEnityReadyGateway
}

// NewMockEnityReadyGateway creates a new mock instance.
func NewMockEnityReadyGateway(ctrl *gomock.Controller) *MockEnityReadyGateway {
	mock := &MockEnityReadyGateway{ctrl: ctrl}
	mock.recorder = &MockEnityReadyGatewayMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnityReadyGateway) EXPECT() *MockEnityReadyGatewayMockRecorder {
	return m.recorder
}

// GetReadyHandlers mocks base method.
func (m *MockEnityReadyGateway) GetReadyHandlers() *base.MapCheckOptions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetReadyHandlers")
	ret0, _ := ret[0].(*base.MapCheckOptions)
	return ret0
}

// GetReadyHandlers indicates an expected call of GetReadyHandlers.
func (mr *MockEnityReadyGatewayMockRecorder) GetReadyHandlers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetReadyHandlers", reflect.TypeOf((*MockEnityReadyGateway)(nil).GetReadyHandlers))
}

// MockEnityGateway is a mock of EnityGateway interface.
type MockEnityGateway struct {
	ctrl     *gomock.Controller
	recorder *MockEnityGatewayMockRecorder
}

// MockEnityGatewayMockRecorder is the mock recorder for MockEnityGateway.
type MockEnityGatewayMockRecorder struct {
	mock *MockEnityGateway
}

// NewMockEnityGateway creates a new mock instance.
func NewMockEnityGateway(ctrl *gomock.Controller) *MockEnityGateway {
	mock := &MockEnityGateway{ctrl: ctrl}
	mock.recorder = &MockEnityGatewayMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnityGateway) EXPECT() *MockEnityGatewayMockRecorder {
	return m.recorder
}

// GetFullName mocks base method.
func (m *MockEnityGateway) GetFullName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFullName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetFullName indicates an expected call of GetFullName.
func (mr *MockEnityGatewayMockRecorder) GetFullName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFullName", reflect.TypeOf((*MockEnityGateway)(nil).GetFullName))
}

// Shutdown mocks base method.
func (m *MockEnityGateway) Shutdown(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Shutdown", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Shutdown indicates an expected call of Shutdown.
func (mr *MockEnityGatewayMockRecorder) Shutdown(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Shutdown", reflect.TypeOf((*MockEnityGateway)(nil).Shutdown), ctx)
}

// Start mocks base method.
func (m *MockEnityGateway) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", ctx, errorGroup)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockEnityGatewayMockRecorder) Start(ctx, errorGroup interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockEnityGateway)(nil).Start), ctx, errorGroup)
}
