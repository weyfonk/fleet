// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1 (interfaces: Interface)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
)

// CoreInterface is a mock of Interface interface.
type CoreInterface struct {
	ctrl     *gomock.Controller
	recorder *CoreInterfaceMockRecorder
}

// CoreInterfaceMockRecorder is the mock recorder for CoreInterface.
type CoreInterfaceMockRecorder struct {
	mock *CoreInterface
}

// NewCoreInterface creates a new mock instance.
func NewCoreInterface(ctrl *gomock.Controller) *CoreInterface {
	mock := &CoreInterface{ctrl: ctrl}
	mock.recorder = &CoreInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *CoreInterface) EXPECT() *CoreInterfaceMockRecorder {
	return m.recorder
}

// ConfigMap mocks base method.
func (m *CoreInterface) ConfigMap() v1.ConfigMapController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConfigMap")
	ret0, _ := ret[0].(v1.ConfigMapController)
	return ret0
}

// ConfigMap indicates an expected call of ConfigMap.
func (mr *CoreInterfaceMockRecorder) ConfigMap() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigMap", reflect.TypeOf((*CoreInterface)(nil).ConfigMap))
}

// Endpoints mocks base method.
func (m *CoreInterface) Endpoints() v1.EndpointsController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Endpoints")
	ret0, _ := ret[0].(v1.EndpointsController)
	return ret0
}

// Endpoints indicates an expected call of Endpoints.
func (mr *CoreInterfaceMockRecorder) Endpoints() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Endpoints", reflect.TypeOf((*CoreInterface)(nil).Endpoints))
}

// Event mocks base method.
func (m *CoreInterface) Event() v1.EventController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Event")
	ret0, _ := ret[0].(v1.EventController)
	return ret0
}

// Event indicates an expected call of Event.
func (mr *CoreInterfaceMockRecorder) Event() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Event", reflect.TypeOf((*CoreInterface)(nil).Event))
}

// Namespace mocks base method.
func (m *CoreInterface) Namespace() v1.NamespaceController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Namespace")
	ret0, _ := ret[0].(v1.NamespaceController)
	return ret0
}

// Namespace indicates an expected call of Namespace.
func (mr *CoreInterfaceMockRecorder) Namespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Namespace", reflect.TypeOf((*CoreInterface)(nil).Namespace))
}

// Node mocks base method.
func (m *CoreInterface) Node() v1.NodeController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Node")
	ret0, _ := ret[0].(v1.NodeController)
	return ret0
}

// Node indicates an expected call of Node.
func (mr *CoreInterfaceMockRecorder) Node() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Node", reflect.TypeOf((*CoreInterface)(nil).Node))
}

// PersistentVolume mocks base method.
func (m *CoreInterface) PersistentVolume() v1.PersistentVolumeController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PersistentVolume")
	ret0, _ := ret[0].(v1.PersistentVolumeController)
	return ret0
}

// PersistentVolume indicates an expected call of PersistentVolume.
func (mr *CoreInterfaceMockRecorder) PersistentVolume() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PersistentVolume", reflect.TypeOf((*CoreInterface)(nil).PersistentVolume))
}

// PersistentVolumeClaim mocks base method.
func (m *CoreInterface) PersistentVolumeClaim() v1.PersistentVolumeClaimController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PersistentVolumeClaim")
	ret0, _ := ret[0].(v1.PersistentVolumeClaimController)
	return ret0
}

// PersistentVolumeClaim indicates an expected call of PersistentVolumeClaim.
func (mr *CoreInterfaceMockRecorder) PersistentVolumeClaim() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PersistentVolumeClaim", reflect.TypeOf((*CoreInterface)(nil).PersistentVolumeClaim))
}

// Pod mocks base method.
func (m *CoreInterface) Pod() v1.PodController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Pod")
	ret0, _ := ret[0].(v1.PodController)
	return ret0
}

// Pod indicates an expected call of Pod.
func (mr *CoreInterfaceMockRecorder) Pod() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pod", reflect.TypeOf((*CoreInterface)(nil).Pod))
}

// Secret mocks base method.
func (m *CoreInterface) Secret() v1.SecretController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Secret")
	ret0, _ := ret[0].(v1.SecretController)
	return ret0
}

// Secret indicates an expected call of Secret.
func (mr *CoreInterfaceMockRecorder) Secret() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Secret", reflect.TypeOf((*CoreInterface)(nil).Secret))
}

// Service mocks base method.
func (m *CoreInterface) Service() v1.ServiceController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Service")
	ret0, _ := ret[0].(v1.ServiceController)
	return ret0
}

// Service indicates an expected call of Service.
func (mr *CoreInterfaceMockRecorder) Service() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Service", reflect.TypeOf((*CoreInterface)(nil).Service))
}

// ServiceAccount mocks base method.
func (m *CoreInterface) ServiceAccount() v1.ServiceAccountController {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServiceAccount")
	ret0, _ := ret[0].(v1.ServiceAccountController)
	return ret0
}

// ServiceAccount indicates an expected call of ServiceAccount.
func (mr *CoreInterfaceMockRecorder) ServiceAccount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServiceAccount", reflect.TypeOf((*CoreInterface)(nil).ServiceAccount))
}
