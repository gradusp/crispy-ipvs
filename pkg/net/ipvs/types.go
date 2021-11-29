package ipvs

import (
	"net"
	"strconv"

	"github.com/gradusp/crispy-ipvs/pkg/ipvs"
	"github.com/pkg/errors"
)

type (
	//NetworkProtocol IP net proto
	NetworkProtocol string

	//ScheduleMethod VIP schedule packets method
	ScheduleMethod string

	//PacketForwarder ...
	PacketForwarder string

	//Address IP net address
	Address string

	//VirtualServerIdentity ...
	VirtualServerIdentity interface {
		isVirtualServerIdentity()
	}

	//VirtualServerAddress ...
	VirtualServerAddress struct {
		NetworkProtocol
		Address
	}

	//VirtualServerFMark ...
	VirtualServerFMark struct {
		FirewallMark uint32
	}

	//VirtualServer a virtual IP server
	VirtualServer struct {
		Identity       VirtualServerIdentity
		ScheduleMethod ScheduleMethod
	}

	//RealServer the ral IP server
	RealServer struct {
		Address
		PacketForwarder PacketForwarder
		Weight          uint32
		UpperThreshold  uint32
		LowerThreshold  uint32
	}
)

var (
	//ErrUnsupported ...
	ErrUnsupported = errors.New("unsupported")

	//ErrVirtualServerNotExist ...
	ErrVirtualServerNotExist = errors.New("virtual server not exit")

	//ErrRealServerNotExist ...
	ErrRealServerNotExist = errors.New("real server not exit")
)

//Valid ...
func (np NetworkProtocol) Valid() error {
	s := string(np)
	if _, ok := ipvs.String2NetworkTransport[s]; ok {
		return errors.Wrapf(ErrUnsupported, "NetworkProtocol(%s)", s)
	}
	return nil
}

//Valid ...
func (sm ScheduleMethod) Valid() error {
	s := string(sm)
	if _, ok := ipvs.String2ScheduleMethod[s]; ok {
		return errors.Wrapf(ErrUnsupported, "ScheduleMethod(%s)", s)
	}
	return nil
}

//Valid ...
func (pf PacketForwarder) Valid() error {
	s := string(pf)
	if _, ok := ipvs.String2PacketFwdMethod[s]; ok {
		return errors.Wrapf(ErrUnsupported, "PacketForwarder(%s)", s)
	}
	return nil
}

//ToHostPort ...
func (n Address) ToHostPort() (string, uint32, error) {
	const api = "Address/ToHostPort"

	h, p, e := net.SplitHostPort(string(n))
	if e != nil {
		return "", 0, errors.Wrap(e, api)
	}
	var i int
	if i, e = strconv.Atoi(p); e != nil {
		return "", 0, errors.Wrap(e, api)
	}
	if i < 0 {
		return "", 0, errors.Errorf("wrong port(%v)", i)
	}
	return h, uint32(i), nil
}

func (VirtualServerAddress) isVirtualServerIdentity() {}

func (VirtualServerFMark) isVirtualServerIdentity() {}
