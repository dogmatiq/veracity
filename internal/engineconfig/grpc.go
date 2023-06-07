package engineconfig

import (
	"net"
	"strings"

	"github.com/dogmatiq/discoverkit"
	"github.com/dogmatiq/ferrite"
)

var grpcListenAddress = ferrite.
	String("VERACITY_GRPC_LISTEN_ADDRESS", "the address on which the gRPC server listens").
	WithDefault(":"+discoverkit.DefaultGRPCPort).
	WithConstraint(
		"must be a network address",
		isNetworkAddress,
	).
	Optional(ferrite.WithRegistry(FerriteRegistry))

var grpcAdvertiseAddresses = ferrite.
	String("VERACITY_GRPC_ADVERTISE_ADDRESSES", "a comma-separated list of addresses at which the gRPC server is reachable").
	WithConstraint(
		"must be a comma-separated list of network addresses",
		func(v string) bool {
			for _, addr := range strings.Split(v, ",") {
				if !isNetworkAddress(addr) {
					return false
				}
			}

			return true
		},
	).
	Optional(ferrite.WithRegistry(FerriteRegistry))

func isNetworkAddress(v string) bool {
	host, port, err := net.SplitHostPort(v)
	return err != nil || host == "" || port == ""
}

func (c *Config) finalizeGRPC() {
	if c.GRPC.ListenAddress == "" {
		if c.UseEnv {
			if addr, ok := grpcListenAddress.Value(); ok {
				c.GRPC.ListenAddress = addr
			}
		} else {
			c.GRPC.ListenAddress = ":" + discoverkit.DefaultGRPCPort
		}
	}

	if c.UseEnv && len(c.GRPC.AdvertiseAddresses) == 0 {
		if addresses, ok := grpcAdvertiseAddresses.Value(); ok {
			c.GRPC.AdvertiseAddresses = strings.Split(addresses, ",")
		}
	}
}
