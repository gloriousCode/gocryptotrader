package websocket

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveBindInterface(t *testing.T) {
	t.Run("PrimaryWins", func(t *testing.T) {
		t.Setenv(envBindInterfaceFallback, "ppp0")
		t.Setenv(envBindInterfacePrimary, "utun9")
		require.Equal(t, "utun9", resolveBindInterface())
	})

	t.Run("FallsBack", func(t *testing.T) {
		t.Setenv(envBindInterfacePrimary, "")
		t.Setenv(envBindInterfaceFallback, "ppp0")
		require.Equal(t, "ppp0", resolveBindInterface())
	})
}

func TestInterfaceIPsFromAddrs(t *testing.T) {
	t.Parallel()

	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("192.168.3.2"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)},
		&net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(64, 128)},
	}
	v4, v6 := interfaceIPsFromAddrs(addrs)
	require.Equal(t, net.ParseIP("192.168.3.2").To4(), v4)
	require.Equal(t, net.ParseIP("2001:db8::1"), v6)
}

func TestParseRemoteIP(t *testing.T) {
	t.Parallel()

	require.Equal(t, net.ParseIP("1.2.3.4"), parseRemoteIP("1.2.3.4:443"))
	require.Equal(t, net.ParseIP("2606:4700:4700::1111"), parseRemoteIP("[2606:4700:4700::1111]:443"))
	require.Nil(t, parseRemoteIP("www.okx.com:443"))
}
