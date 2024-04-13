package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
)

type kvServer struct {
	pb.UnimplementedKVServer
}

var (
	port    = flag.Int("port", 8080, "The server port")
	node    = flag.Int("node", 0, "Node ID")
	replica = flag.Int("replica", 0, "Replica ID")
)

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	return &pb.GetResponse{Value: fmt.Sprintf("Hello, World! from replica %d of node %d", *replica, *node)}, nil
}

func main() {
	flag.Parse()
	log.Printf("Starting replica %d of node %d on port %d", *replica, *node, *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterKVServer(grpcServer, &kvServer{})
	grpcServer.Serve(lis)
}
