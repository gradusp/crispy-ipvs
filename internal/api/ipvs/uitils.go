package ipvs

import (
	"net"
	"strconv"

	"github.com/gradusp/crispy-ipvs/pkg/ipvs"
	ipvsAdm "github.com/gradusp/crispy-ipvs/pkg/net/ipvs"
	"github.com/pkg/errors"
)

type (
	//VirtualServerIdentityConv ...
	VirtualServerIdentityConv struct {
		Identity ipvsAdm.VirtualServerIdentity
	}

	//NetworkProtocolConv ...
	NetworkProtocolConv struct {
		NetworkProtocol ipvsAdm.NetworkProtocol
	}

	//VirtualServerConv ...
	VirtualServerConv struct {
		VirtualServer ipvsAdm.VirtualServer
	}

	//RealServerConv ...
	RealServerConv struct {
		RealServer ipvsAdm.RealServer
	}

	//AddressConv ...
	AddressConv struct {
		Address ipvsAdm.Address
	}
)

//ToPb converts to *ipvs.VirtualServerIdentity
func (identity VirtualServerIdentityConv) ToPb() (*ipvs.VirtualServerIdentity, error) {
	const api = "VirtualServerIdentityConv/ToPb"

	var ret ipvs.VirtualServerIdentity
	switch t := identity.Identity.(type) {
	case ipvsAdm.VirtualServerFMark:
		ret.By = &ipvs.VirtualServerIdentity_FirewallMark{FirewallMark: t.FirewallMark}
	case ipvsAdm.VirtualServerAddress:
		var addr ipvs.VirtualServerAddress
		var e error
		if addr.Host, addr.Port, e = t.Address.ToHostPort(); e != nil {
			return nil, errors.Wrap(e, api)
		}
		addr.Network, e = NetworkProtocolConv{NetworkProtocol: t.NetworkProtocol}.ToPb()
		if e != nil {
			return nil, errors.Wrap(e, api)
		}
		ret.By = &ipvs.VirtualServerIdentity_Address{Address: &addr}
	default:
		return nil, errors.Errorf("%s: unconvertible", api)
	}

	return &ret, nil
}

//FromPb converts from *ipvs.VirtualServerIdentity
func (identity *VirtualServerIdentityConv) FromPb(src *ipvs.VirtualServerIdentity) error {
	const api = "VirtualServerIdentityConv/FromPb"

	switch t := src.GetBy().(type) {
	case *ipvs.VirtualServerIdentity_Address:
		h := t.Address.GetHost()
		p := strconv.Itoa(int(t.Address.GetPort()))
		a := ipvsAdm.VirtualServerAddress{
			NetworkProtocol: ipvsAdm.NetworkProtocol(ipvs.NetworkTransport2String[t.Address.Network]),
			Address:         ipvsAdm.Address(net.JoinHostPort(h, p)),
		}
		if err := a.NetworkProtocol.Valid(); err != nil {
			return errors.Wrap(err, api)
		}
		identity.Identity = a
		return nil
	case *ipvs.VirtualServerIdentity_FirewallMark:
		identity.Identity = ipvsAdm.VirtualServerFMark{
			FirewallMark: t.FirewallMark,
		}
		return nil
	}
	return errors.Wrapf(errors.New("inconvertible virtual server identity"), api)
}

//ToPb to *ipvs.NetworkTransport
func (conv NetworkProtocolConv) ToPb() (ipvs.NetworkTransport, error) {
	const api = "NetworkProtocolConv/ToPb"
	return ipvs.String2NetworkTransport[string(conv.NetworkProtocol)],
		errors.Wrap(conv.NetworkProtocol.Valid(), api)
}

//FromPb conv from *ipvs.VirtualServer
func (conv *VirtualServerConv) FromPb(src *ipvs.VirtualServer) error {
	const api = "VirtualServerConv/FromPb"

	var ret ipvsAdm.VirtualServer
	var identity VirtualServerIdentityConv
	err := identity.FromPb(src.GetIdentity())
	if err != nil {
		return errors.Wrap(err, api)
	}
	ret.ScheduleMethod = ipvsAdm.ScheduleMethod(ipvs.ScheduleMethod2String[src.GetScheduleMethod()])
	if err = ret.ScheduleMethod.Valid(); err != nil {
		return errors.Wrap(err, api)
	}
	ret.Identity = identity.Identity
	conv.VirtualServer = ret
	return nil
}

//ToPb conv to *ipvs.VirtualServer
func (conv VirtualServerConv) ToPb() (*ipvs.VirtualServer, error) {
	const api = "VirtualServerConv/ToPb"

	var ret ipvs.VirtualServer
	var err error

	ret.Identity, err =
		VirtualServerIdentityConv{Identity: conv.VirtualServer.Identity}.ToPb()

	if err != nil {
		return nil, errors.Wrap(err, api)
	}
	if err = conv.VirtualServer.ScheduleMethod.Valid(); err != nil {
		return nil, errors.Wrap(err, api)
	}
	ret.ScheduleMethod = ipvs.String2ScheduleMethod[string(conv.VirtualServer.ScheduleMethod)]
	return &ret, nil
}

//ToPb conv to *ipvs.RealServer
func (conv RealServerConv) ToPb() (*ipvs.RealServer, error) {
	const api = "RealServerConv/ToPb"

	ret := ipvs.RealServer{
		Address:        new(ipvs.RealServerAddress),
		Weight:         conv.RealServer.Weight,
		UpperThreshold: conv.RealServer.UpperThreshold,
		LowerThreshold: conv.RealServer.LowerThreshold,
	}
	err := conv.RealServer.PacketForwarder.Valid()
	if err != nil {
		return nil, errors.Wrap(err, api)
	}
	ret.PacketForwarder = ipvs.String2PacketFwdMethod[string(conv.RealServer.PacketForwarder)]
	if ret.Address.Host, ret.Address.Port, err = conv.RealServer.ToHostPort(); err != nil {
		return nil, errors.Wrap(err, api)
	}

	return &ret, nil
}

//FromPb conv from *ipvs.RealServer
func (conv *RealServerConv) FromPb(src *ipvs.RealServer) error {
	const api = "RealServerConv/FromPb"

	var ac AddressConv
	ac.FromPb(src.GetAddress())
	ret := ipvsAdm.RealServer{
		Address:         ac.Address,
		Weight:          src.GetWeight(),
		UpperThreshold:  src.GetUpperThreshold(),
		LowerThreshold:  src.GetLowerThreshold(),
		PacketForwarder: ipvsAdm.PacketForwarder(ipvs.PacketFwdMethod2String[src.GetPacketForwarder()]),
	}
	if e := ret.PacketForwarder.Valid(); e != nil {
		return errors.Wrap(e, api)
	}
	if ret.LowerThreshold > ret.UpperThreshold {
		return errors.Wrap(
			errors.Errorf("lowerThreshold(%v) > upperThreshold(%v)", ret.LowerThreshold, ret.UpperThreshold),
			api,
		)
	}
	conv.RealServer = ret
	return nil
}

//FromPb ...
func (conv *AddressConv) FromPb(src *ipvs.RealServerAddress) {
	conv.Address = ipvsAdm.Address(net.JoinHostPort(src.GetHost(), strconv.Itoa(int(src.GetPort()))))
}
