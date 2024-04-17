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
	for i := 0; i < 100; i++ {
		if _, err := client.Put(context.Background(), &pb.PutRequest{Key: fmt.Sprintf("key%d", i), Value: fmt.Sprintf("value%d", i)}); err != nil {
			fmt.Println(err)
		}
	}
	for i := 0; i < 100; i++ {
		if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(resp.Value)
		}
	}
	for i := 0; i < 10; i++ {
		if _, err := client.Delete(context.Background(), &pb.DeleteRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
			fmt.Println(err)
		}
	}
	for i := 0; i < 20; i++ {
		if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(resp.Value)
		}
	}
}
