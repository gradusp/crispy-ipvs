//go:build !linux
// +build !linux

package ipvs

import (
	"context"
	"runtime"

	"github.com/pkg/errors"
)

//NewAdmin manes inst of Ipvs.Admin
func NewAdmin(_ context.Context) Admin {
	return new(fakeIpvsAdmin)
}

type fakeIpvsAdmin struct{}

var errNotSupport = errors.Errorf("not supported in OS('%s)'", runtime.GOOS)

//ListVirtualServers impl IpvsAdmin
func (fakeIpvsAdmin) ListVirtualServers(_ context.Context, _ VirtualServerConsumer) error {
	return errNotSupport
}

//ListRealServers impl IpvsAdmin
func (fakeIpvsAdmin) ListRealServers(_ context.Context, _ VirtualServerIdentity, _ RealServerConsumer) error {
	return errNotSupport
}

//UpdateVirtualSerer impl IpvsAdmin
func (fakeIpvsAdmin) UpdateVirtualSerer(_ context.Context, _ VirtualServer, _ ...AdminOption) error {
	return errNotSupport
}

//RemoveVirtualSerer impl IpvsAdmin
func (fakeIpvsAdmin) RemoveVirtualSerer(_ context.Context, _ VirtualServerIdentity, _ ...AdminOption) error {
	return errNotSupport
}

//UpdateRealServer impl IpvsAdmin
func (fakeIpvsAdmin) UpdateRealServer(_ context.Context, _ VirtualServerIdentity, _ RealServer, _ ...AdminOption) error {
	return errNotSupport
}

//RemoveRealServer impl IpvsAdmin
func (fakeIpvsAdmin) RemoveRealServer(_ context.Context, _ VirtualServerIdentity, _ Address, _ ...AdminOption) error {
	return errNotSupport
}
