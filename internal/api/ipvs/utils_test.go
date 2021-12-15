package ipvs

import (
	"testing"

	"github.com/gradusp/crispy-ipvs/pkg/ipvs"
	ipvsAdm "github.com/gradusp/crispy-ipvs/pkg/net/ipvs"
	"github.com/stretchr/testify/assert"
)

func TestNetworkProtocolConv(t *testing.T) {
	type T = struct {
		proto ipvsAdm.NetworkProtocol
		exp   ipvs.NetworkTransport
	}
	protos := []T{
		{"tcp", ipvs.NetworkTransport_TCP},
		{"udp", ipvs.NetworkTransport_UDP},
	}
	for _, p := range protos {
		pb, err := NetworkProtocolConv{p.proto}.ToPb()
		if !assert.NoError(t, err) {
			return
		}
		if !assert.Equal(t, p.exp, pb) {
			return
		}
	}
}
