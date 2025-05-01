// Package mock is a generated GoMock package. DO NOT EDIT
package mock

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockHashBucket is a mock of HashBucket interface.
type MockHashBucket struct {
	ctrl     *gomock.Controller
	recorder *MockHashBucketMockRecorder
}

// MockHashBucketMockRecorder is the mock recorder for MockHashBucket.
type MockHashBucketMockRecorder struct {
	mock *MockHashBucket
}

// NewMockHashBucket creates a new mock instance.
func NewMockHashBucket(ctrl *gomock.Controller) *MockHashBucket {
	mock := &MockHashBucket{ctrl: ctrl}
	mock.recorder = &MockHashBucketMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHashBucket) EXPECT() *MockHashBucketMockRecorder {
	return m.recorder
}

// InBucket mocks base method.
func (m *MockHashBucket) InBucket(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InBucket", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// InBucket indicates an expected call of InBucket.
func (mr *MockHashBucketMockRecorder) InBucket(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InBucket", reflect.TypeOf((*MockHashBucket)(nil).InBucket), arg0)
}
