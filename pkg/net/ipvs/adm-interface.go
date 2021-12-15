package ipvs

import (
	"context"
)

type (
	//VirtualServerConsumer ...
	VirtualServerConsumer = func(vs VirtualServer) error

	//RealServerConsumer ...
	RealServerConsumer = func(rs RealServer) error

	//AdminOption option type for IpvsAdmin op-s
	AdminOption interface {
		isIpvsAdminOption()
	}

	//Admin Linux IPVS admin
	Admin interface {
		ListVirtualServers(ctx context.Context, cons VirtualServerConsumer) error
		ListRealServers(ctx context.Context, vsKey VirtualServerIdentity, cons RealServerConsumer) error

		UpdateVirtualServer(ctx context.Context, serv VirtualServer, opts ...AdminOption) error
		RemoveVirtualServer(ctx context.Context, vsKey VirtualServerIdentity, opts ...AdminOption) error

		UpdateRealServer(ctx context.Context, vsKey VirtualServerIdentity, serv RealServer, opts ...AdminOption) error
		RemoveRealServer(ctx context.Context, vsKey VirtualServerIdentity, servAddress Address, opts ...AdminOption) error
	}

	//KeepCalmIfNotExist ...
	KeepCalmIfNotExist struct{}

	//ForceAddIfNotExist ...
	ForceAddIfNotExist struct{}
)

func (KeepCalmIfNotExist) isIpvsAdminOption() {}

func (ForceAddIfNotExist) isIpvsAdminOption() {}
