//go:build linux
// +build linux

package ipvs

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/gradusp/go-platform/pkg/jsonview"
	"github.com/gradusp/go-platform/pkg/lazy"
	"github.com/hkwi/nlgo"
	"github.com/mqliang/libipvs"
	"github.com/pkg/errors"
)

//NewAdmin manes inst of Ipvs.Admin
func NewAdmin(ctx context.Context) Admin {
	return &ipvsAdminImpl{
		appCtx: ctx,
		libIpvsAPI: lazy.MakeInitializer(func() interface{} {
			h, e := libipvs.New()
			if e != nil {
				return e
			}
			return h
		}),
	}
}

type (
	libAPI         = libipvs.IPVSHandle
	virtualService = libipvs.Service
	realServer     = libipvs.Destination

	ipvsAdminImpl struct {
		appCtx     context.Context
		libIpvsAPI lazy.Initializer
	}
)

const (
	ipvsImpl = "ipvsAdmin"
	libIpvs  = "lib-ipvs"

	fwdMAT    = "nat"
	fwdDIRECT = "dr"
	fwdTUN    = "tun"
)

func (impl *ipvsAdminImpl) libIpvsHandler() (h libAPI, e error) {
	switch t := impl.libIpvsAPI.Value().(type) {
	case error:
		e = errors.Wrap(t, ipvsImpl+"/"+libIpvs+"/init")
	case libipvs.IPVSHandle:
		h = t
	}
	return
}

//ListVirtualServers impl IpvsAdmin
func (impl *ipvsAdminImpl) ListVirtualServers(_ context.Context, consumer VirtualServerConsumer) error {
	const api = ipvsImpl + "/ListVirtualServers"

	var services []*virtualService
	lib, err := impl.libIpvsHandler()
	if err != nil {
		return errors.Wrap(err, api)
	}
	if services, err = lib.ListServices(); err != nil {
		return errors.Wrapf(err, "%s: %s/ListServices", api, libIpvs)
	}
	for i := range services {
		src := services[i]
		dest := VirtualServer{
			Identity:       impl.address2Identity(src),
			ScheduleMethod: ScheduleMethod(src.SchedName),
		}
		if err = consumer(dest); err != nil {
			return err
		}
	}
	return nil
}

//ListRealServers impl IpvsAdmin
func (impl *ipvsAdminImpl) ListRealServers(_ context.Context, identity VirtualServerIdentity, consumer RealServerConsumer) error {
	const api = ipvsImpl + "/ListRealServers"

	var (
		err   error
		lib   libAPI
		reals []*realServer
	)
	vs := new(virtualService)
	if err = impl.identity2Address(identity, vs); err != nil {
		return errors.Wrap(err, api)
	}
	if lib, err = impl.libIpvsHandler(); err != nil {
		return errors.Wrap(err, api)
	}
	if reals, err = lib.ListDestinations(vs); err != nil {
		return errors.Wrapf(err, "%s: %s/ListDestinations", api, libIpvs)
	}
	for _, r := range reals {
		var res RealServer
		switch r.FwdMethod {
		case libipvs.IP_VS_CONN_F_MASQ:
			res.PacketForwarder = fwdMAT
		case libipvs.IP_VS_CONN_F_TUNNEL:
			res.PacketForwarder = fwdTUN
		case libipvs.IP_VS_CONN_F_DROUTE:
			res.PacketForwarder = fwdDIRECT
		default:
			res.PacketForwarder = PacketForwarder(r.FwdMethod.String())
		}
		res.Address = Address(net.JoinHostPort(r.Address.String(), fmt.Sprintf("%v", r.Port)))
		res.Weight = r.Weight
		res.UpperThreshold = r.UThresh
		res.LowerThreshold = r.LThresh
		if err = consumer(res); err != nil {
			break
		}
	}
	return err
}

//UpdateVirtualSerer impl IpvsAdmin
func (impl *ipvsAdminImpl) UpdateVirtualSerer(_ context.Context, vServer VirtualServer, opts ...AdminOption) error {
	const api = ipvsImpl + "/UpdateVirtualSerer"

	vs := new(virtualService)
	err := impl.identity2Address(vServer.Identity, vs)
	if err != nil {
		return errors.Wrap(err, api)
	}
	vs.SchedName = string(vServer.ScheduleMethod)
	var lib libAPI
	if lib, err = impl.libIpvsHandler(); err != nil {
		return errors.Wrap(err, api)
	}

	var forceAddIfNotExist bool
	for i := range opts {
		switch opts[i].(type) {
		case ForceAddIfNotExist:
			forceAddIfNotExist = true
		}
	}
	if err = lib.UpdateService(vs); err != nil {
		var e nlgo.NlMsgerr
		if errors.As(err, &e) {
			if syscall.Errno(-e.Payload().Error) == syscall.ESRCH {
				err = ErrVirtualServerNotExist
				if forceAddIfNotExist {
					err = lib.NewService(vs)
				}
			} else {
				err = errors.WithMessage(ErrExternal, e.Error())
			}
		}
	}
	return errors.Wrap(err, api)
}

//RemoveVirtualSerer impl IpvsAdmin
func (impl *ipvsAdminImpl) RemoveVirtualSerer(_ context.Context, identity VirtualServerIdentity, opts ...AdminOption) error {
	const api = ipvsImpl + "/RemoveVirtualSerer"

	vs := new(virtualService)
	err := impl.identity2Address(identity, vs)
	if err != nil {
		return errors.Wrapf(err, api)
	}
	var lib libAPI
	if lib, err = impl.libIpvsHandler(); err != nil {
		return errors.Wrap(err, api)
	}

	var keepCalmIfNotExist bool
	for i := range opts {
		switch opts[i].(type) {
		case KeepCalmIfNotExist:
			keepCalmIfNotExist = true
		}
	}
	if err = lib.DelService(vs); err != nil {
		var e nlgo.NlMsgerr
		if errors.As(err, &e) {
			if syscall.Errno(-e.Payload().Error) == syscall.ESRCH {
				err = ErrVirtualServerNotExist
				if keepCalmIfNotExist {
					err = nil
				}
			} else {
				err = errors.WithMessage(ErrExternal, e.Error())
			}
		}
	}
	return errors.Wrapf(err, api)
}

//UpdateRealServer impl IpvsAdmin
func (impl *ipvsAdminImpl) UpdateRealServer(_ context.Context, identity VirtualServerIdentity, realServer RealServer, opts ...AdminOption) error {
	const api = ipvsImpl + "/UpdateRealServer"

	var (
		host string
		port uint32
		err  error
		lib  libAPI
	)
	if host, port, err = realServer.Address.ToHostPort(); err != nil {
		return errors.Wrap(err, api)
	}
	rs := new(libipvs.Destination)
	if rs.Address = net.ParseIP(host); rs.Address == nil {
		return errors.Wrap(errors.Errorf("parse-IP('%s')", host), api)
	}
	rs.Port = uint16(port)
	rs.AddressFamily = syscall.AF_INET
	rs.Weight = realServer.Weight
	rs.LThresh = realServer.LowerThreshold
	rs.UThresh = realServer.UpperThreshold
	switch realServer.PacketForwarder {
	case fwdMAT:
		rs.FwdMethod = libipvs.IP_VS_CONN_F_MASQ
	case fwdDIRECT:
		rs.FwdMethod = libipvs.IP_VS_CONN_F_DROUTE
	case fwdTUN:
		rs.FwdMethod = libipvs.IP_VS_CONN_F_TUNNEL
	default:
		return errors.Wrapf(ErrUnsupported, "%s: packet-forward '%s'", api, realServer.PacketForwarder)
	}

	vs, err := impl.findVirtualService(identity)
	if err != nil {
		return errors.Wrapf(err, api)
	}
	if vs == nil {
		return errors.Wrapf(ErrVirtualServerNotExist, api)
	}

	var forceAddIfNotExist bool
	for i := range opts {
		switch opts[i].(type) {
		case ForceAddIfNotExist:
			forceAddIfNotExist = true
		}
	}
	if lib, err = impl.libIpvsHandler(); err != nil {
		return errors.Wrap(err, api)
	}
	if err = lib.UpdateDestination(vs, rs); err != nil {
		var e nlgo.NlMsgerr
		if errors.As(err, &e) {
			if syscall.Errno(-e.Payload().Error) == syscall.ENOENT {
				err = ErrRealServerNotExist
				if forceAddIfNotExist {
					err = lib.NewDestination(vs, rs)
				}
			} else {
				err = errors.WithMessage(ErrExternal, e.Error())
			}
		}
	}
	return errors.Wrap(err, api)
}

//RemoveRealServer impl IpvsAdmin
func (impl *ipvsAdminImpl) RemoveRealServer(_ context.Context, identity VirtualServerIdentity, addr Address, opts ...AdminOption) error {
	const api = ipvsImpl + "/RemoveRealServer"

	vs, err := impl.findVirtualService(identity)
	if err != nil {
		return errors.Wrap(err, api)
	}
	if vs == nil {
		return errors.Wrapf(ErrVirtualServerNotExist, api)
	}
	var (
		h   string
		p   uint32
		lib libAPI
	)
	if h, p, err = addr.ToHostPort(); err != nil {
		return errors.Wrapf(err, api)
	}
	rs := new(libipvs.Destination)
	if rs.Address = net.ParseIP(h); rs.Address == nil {
		return errors.Wrap(errors.Errorf("parse-IP('%s')", h), api)
	}
	rs.Port = uint16(p)
	rs.AddressFamily = syscall.AF_INET

	if lib, err = impl.libIpvsHandler(); err != nil {
		return errors.Wrap(err, api)
	}
	var keepCalmIfNotExist bool
	for i := range opts {
		switch opts[i].(type) {
		case KeepCalmIfNotExist:
			keepCalmIfNotExist = true
		}
	}
	if err = lib.DelDestination(vs, rs); err != nil {
		var e nlgo.NlMsgerr
		if errors.As(err, &e) {
			if syscall.Errno(-e.Payload().Error) == syscall.ENOENT {
				err = ErrRealServerNotExist
				if keepCalmIfNotExist {
					err = nil
				}
			} else {
				err = errors.WithMessage(ErrExternal, e.Error())
			}
		}
	}
	return errors.Wrap(err, api)
}

func (impl *ipvsAdminImpl) findVirtualService(identity VirtualServerIdentity) (*virtualService, error) {
	lib, err := impl.libIpvsHandler()
	if err != nil {
		return nil, err
	}
	var services []*virtualService
	if services, err = lib.ListServices(); err != nil {
		return nil, err
	}
	for _, s := range services {
		id := impl.address2Identity(s)
		if IsIdentitiesEq(id, identity) {
			return s, nil
		}
	}
	return nil, nil
}

func (impl *ipvsAdminImpl) address2Identity(vs *virtualService) VirtualServerIdentity {
	var ret VirtualServerIdentity
	if vs.Address == nil {
		ret = VirtualServerFMark{FirewallMark: vs.FWMark}
	} else {
		ret = VirtualServerAddress{
			Address:         Address(net.JoinHostPort(vs.Address.String(), fmt.Sprintf("%v", vs.Port))),
			NetworkProtocol: NetworkProtocol(vs.Protocol.String()),
		}
	}
	return ret
}

func (impl *ipvsAdminImpl) identity2Address(identity VirtualServerIdentity, vs *virtualService) error {
	const api = ipvsImpl + "/identity2Address"

	switch t := identity.(type) {
	case VirtualServerAddress:
		h, p, err := t.Address.ToHostPort()
		if err != nil {
			return errors.Wrap(err, api)
		}
		if vs.Address = net.ParseIP(h); vs.Address == nil {
			return errors.Wrap(errors.Errorf("parse-IP('%s')", h), api)
		}
		vs.Port = uint16(p)
		switch t.NetworkProtocol {
		case "tcp":
			vs.Protocol = syscall.IPPROTO_TCP
			vs.AddressFamily = syscall.AF_INET
		case "udp":
			vs.Protocol = syscall.IPPROTO_UDP
			vs.AddressFamily = syscall.AF_INET
		default:
			return errors.Errorf("%s: unsupported protocol(%v)", api, t.NetworkProtocol)
		}
	case VirtualServerFMark:
		vs.FWMark = t.FirewallMark
	default:
		return errors.Wrapf(ErrUnsupported,
			"%s: identity %s", api, jsonview.String(identity))
	}
	return nil
}
