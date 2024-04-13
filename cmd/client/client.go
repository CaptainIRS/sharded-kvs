package main

import (
	"context"
	"flag"
	"fmt"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	host = flag.String("host", "localhost", "The server host")
	port = flag.Int("port", 8080, "The server port")
)

func main() {
	flag.Parse()
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *host, *port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client := pb.NewKVClient(conn)
	resp, err := client.Get(context.Background(), &pb.GetRequest{Key: "hello"})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Value)
}