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

type shardedKVServer struct {
	pb.UnimplementedShardedKVServer
}

func (s *shardedKVServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	return &pb.GetResponse{Value: "Hello, World!"}, nil
}

var (
	port = flag.Int("port", 8080, "The server port")
)

func main() {
	flag.Parse()
	log.Printf("Starting server on port %d", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterShardedKVServer(grpcServer, &shardedKVServer{})
	grpcServer.Serve(lis)
}
