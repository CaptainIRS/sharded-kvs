package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
)

type kvServer struct {
	pb.UnimplementedKVServer
}

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	replicaGroup, replicaIndex := os.Getenv("REPLICA_GROUP"), os.Getenv("REPLICA_INDEX")
	return &pb.GetResponse{Value: fmt.Sprintf("Hello, World from replica %s in group %s", replicaIndex, replicaGroup)}, nil
}

var (
	port = flag.Int("port", 8080, "The server port")
)

func main() {
	replicaGroup, ok := os.LookupEnv("REPLICA_GROUP")
	if !ok {
		log.Fatal("REPLICA_GROUP environment variable not set")
	}
	log.Printf("REPLICA_GROUP: %s", replicaGroup)

	replicaIndex, ok := os.LookupEnv("REPLICA_INDEX")
	if !ok {
		log.Fatal("REPLICA_INDEX environment variable not set")
	}
	log.Printf("REPLICA_INDEX: %s", replicaIndex)

	flag.Parse()
	log.Printf("Starting server on port %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterKVServer(grpcServer, &kvServer{})
	grpcServer.Serve(lis)
}
