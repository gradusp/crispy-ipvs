// +build !linux

package ipvs

import (
	"context"
	"runtime"

	"github.com/pkg/errors"
)

//NewAdmin manes inst of Ipvs.Admin
func NewAdmin(_ context.Context) Admin {
	return new(fakeOdIpvsAdmin)
}

type fakeOdIpvsAdmin struct{}

var errNotSupport = errors.Errorf("not supported in OS('%s)'", runtime.GOOS)

//ListVirtualServers impl IpvsAdmin
func (fakeOdIpvsAdmin) ListVirtualServers(_ context.Context, _ VirtualServerConsumer) error {
	return errNotSupport
}

//ListRealServers impl IpvsAdmin
func (fakeOdIpvsAdmin) ListRealServers(_ context.Context, _ VirtualServerIdentity, _ RealServerConsumer) error {
	return errNotSupport
}

//UpdateVirtualSerer impl IpvsAdmin
func (fakeOdIpvsAdmin) UpdateVirtualSerer(_ context.Context, _ VirtualServer, _ ...AdminOption) error {
	return errNotSupport
}

//RemoveVirtualSerer impl IpvsAdmin
func (fakeOdIpvsAdmin) RemoveVirtualSerer(_ context.Context, _ VirtualServerIdentity, _ ...AdminOption) error {
	return errNotSupport
}

//UpdateRealServer impl IpvsAdmin
func (fakeOdIpvsAdmin) UpdateRealServer(_ context.Context, _ VirtualServerIdentity, _ RealServer, _ ...AdminOption) error {
	return errNotSupport
}

//RemoveRealServer impl IpvsAdmin
func (fakeOdIpvsAdmin) RemoveRealServer(_ context.Context, _ VirtualServerIdentity, _ Address, _ ...AdminOption) error {
	return errNotSupport
}
