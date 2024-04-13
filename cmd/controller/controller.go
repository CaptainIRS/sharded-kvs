package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	"google.golang.org/grpc"
)

type kvServer struct {
	pb.UnimplementedKVServer
}

type Member string

func (m Member) String() string {
	return string(m)
}

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

var (
	port     = flag.Int("port", 8080, "The server port")
	replica  = flag.Int("replica", 0, "Replica ID")
	nodes    = flag.Int("nodes", 1, "Number of nodes")
	replicas = flag.Int("replicas", 1, "Number of replicas")
	ch       = *&consistent.Consistent{}
)

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	return &pb.GetResponse{Value: fmt.Sprintf("Get response from replica %d of controller. Key %s is stored at %s.", *replica, key, member)}, nil
}

func (s *kvServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	return &pb.PutResponse{
		Message: fmt.Sprintf("Put response from replica %d of controller. Key %s is stored at %s", *replica, key, member),
	}, nil
}

func (s *kvServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	return &pb.DeleteResponse{
		Message: fmt.Sprintf("Delete response from replica %d of controller. Key %s is stored at %s", *replica, key, member),
	}, nil
}

func main() {
	flag.Parse()
	log.Printf("Starting replica %d of controller on port %d", *replica, *port)

	members := []consistent.Member{}
	for n := 0; n < *nodes; n++ {
		members = append(members, Member(fmt.Sprintf("node-%d", n)))
	}

	cfg := consistent.Config{
		PartitionCount:    20,
		ReplicationFactor: *replicas,
		Load:              1.25,
		Hasher:            hasher{},
	}
	ch = *consistent.New(members, cfg)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterKVServer(grpcServer, &kvServer{})
	grpcServer.Serve(lis)
}
