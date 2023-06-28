package prototest

import (
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RunGRPCServer starts a gRPC server that listens on a random port and returns a
// client connection to it.
//
// register is called to register the server's gRPC services.
func RunGRPCServer(
	t *testing.T,
	reg func(grpc.ServiceRegistrar),
) grpc.ClientConnInterface {
	server := grpc.NewServer()
	reg(server)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		lis.Close()
	})

	result := make(chan error, 1)
	go func() {
		result <- server.Serve(lis)
	}()
	t.Cleanup(func() {
		server.Stop()
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	})

	conn, err := grpc.Dial(
		lis.Addr().String(),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}
