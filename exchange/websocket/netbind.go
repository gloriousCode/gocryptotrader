package websocket

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
)

const (
	envBindInterfacePrimary  = "GCT_BIND_INTERFACE"
	envBindInterfaceFallback = "OPTIONSMM_VPN_INTERFACE"
)

// configureInterfaceDialer applies optional interface binding to outbound websocket dials.
func configureInterfaceDialer(dialer *gws.Dialer) error {
	if dialer == nil {
		return nil
	}
	interfaceName := resolveBindInterface()
	if interfaceName == "" {
		return nil
	}
	dialContext, err := dialContextForInterface(interfaceName)
	if err != nil {
		return err
	}
	dialer.NetDialContext = dialContext
	return nil
}

// resolveBindInterface resolves interface preference from environment.
func resolveBindInterface() string {
	if explicit := strings.TrimSpace(os.Getenv(envBindInterfacePrimary)); explicit != "" {
		return explicit
	}
	return strings.TrimSpace(os.Getenv(envBindInterfaceFallback))
}

// dialContextForInterface creates a dial context bound to IP addresses owned by the interface.
func dialContextForInterface(interfaceName string) (func(context.Context, string, string) (net.Conn, error), error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("vpn interface lookup failed: %w", err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("vpn interface addresses unavailable: %w", err)
	}
	v4, v6 := interfaceIPsFromAddrs(addrs)
	if v4 == nil && v6 == nil {
		return nil, fmt.Errorf("vpn interface %s has no usable IP address", interfaceName)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		remoteIP := parseRemoteIP(address)
		if remoteIP != nil {
			if remoteIP.To4() != nil {
				if v4 == nil {
					return nil, fmt.Errorf("vpn interface %s has no IPv4 address", interfaceName)
				}
				return dialWithLocalAddress(ctx, "tcp4", address, v4)
			}
			if v6 == nil {
				return nil, fmt.Errorf("vpn interface %s has no global IPv6 address", interfaceName)
			}
			return dialWithLocalAddress(ctx, "tcp6", address, v6)
		}
		if strings.HasSuffix(network, "4") {
			if v4 == nil {
				return nil, fmt.Errorf("vpn interface %s has no IPv4 address", interfaceName)
			}
			return dialWithLocalAddress(ctx, "tcp4", address, v4)
		}
		if strings.HasSuffix(network, "6") {
			if v6 == nil {
				return nil, fmt.Errorf("vpn interface %s has no global IPv6 address", interfaceName)
			}
			return dialWithLocalAddress(ctx, "tcp6", address, v6)
		}
		if v4 != nil {
			if conn, dialErr := dialWithLocalAddress(ctx, "tcp4", address, v4); dialErr == nil {
				return conn, nil
			}
		}
		if v6 == nil {
			return nil, fmt.Errorf("vpn interface %s has no global IPv6 address", interfaceName)
		}
		return dialWithLocalAddress(ctx, "tcp6", address, v6)
	}, nil
}

// dialWithLocalAddress dials a remote endpoint with a source address from the bound interface.
func dialWithLocalAddress(ctx context.Context, network, address string, ip net.IP) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
		LocalAddr: &net.TCPAddr{IP: ip},
	}
	return dialer.DialContext(ctx, network, address)
}

// interfaceIPsFromAddrs selects the first usable IPv4 and global IPv6 addresses.
func interfaceIPsFromAddrs(addrs []net.Addr) (net.IP, net.IP) {
	var ipv4 net.IP
	var ipv6 net.IP
	for i := range addrs {
		ip := parseInterfaceAddressIP(addrs[i])
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			if ipv4 == nil {
				ipv4 = ip4
			}
			continue
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		if ipv6 == nil {
			ipv6 = ip
		}
	}
	return ipv4, ipv6
}

// parseInterfaceAddressIP extracts IPs from interface addresses.
func parseInterfaceAddressIP(addr net.Addr) net.IP {
	ipNet, ok := addr.(*net.IPNet)
	if ok {
		return ipNet.IP
	}
	ipAddr, ok := addr.(*net.IPAddr)
	if ok {
		return ipAddr.IP
	}
	return nil
}

// parseRemoteIP extracts an IP literal from host:port endpoints.
func parseRemoteIP(address string) net.IP {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	return net.ParseIP(host)
}
