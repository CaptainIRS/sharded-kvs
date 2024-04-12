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
	port         = flag.Int("port", 8080, "The server port")
	replicaGroup = flag.Int("replica-group", 0, "The replica group this node belongs to")
	replicaIndex = flag.Int("replica-index", 0, "The index of this replica in the group")
)

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	return &pb.GetResponse{Value: fmt.Sprintf("Hello, World! from replica %d in group %d", *replicaIndex, *replicaGroup)}, nil
}

func main() {
	flag.Parse()
	log.Printf("!! Starting node replica-%d-node-%d on port %d", *replicaGroup, *replicaIndex, *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterKVServer(grpcServer, &kvServer{})
	grpcServer.Serve(lis)
}
