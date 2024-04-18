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
// host = flag.String("host", "localhost", "The server host")
// port = flag.Int("port", 8080, "The server port")
)

func main() {
	flag.Parse()
	// conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *host, *port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	panic(err)
	// }
	// client := pb.NewKVClient(conn)
	// for i := 0; i < 100; i++ {
	// 	if _, err := client.Put(context.Background(), &pb.PutRequest{Key: fmt.Sprintf("key%d", i), Value: fmt.Sprintf("value%d", i)}); err != nil {
	// 		fmt.Println(err)
	// 	}
	// }
	// for i := 0; i < 100; i++ {
	// 	if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
	// 		fmt.Println(err)
	// 	} else {
	// 		fmt.Println(resp.Value)
	// 	}
	// }
	// for i := 0; i < 10; i++ {
	// 	if _, err := client.Delete(context.Background(), &pb.DeleteRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
	// 		fmt.Println(err)
	// 	}
	// }
	// for i := 0; i < 20; i++ {
	// 	if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: fmt.Sprintf("key%d", i)}); err != nil {
	// 		fmt.Println(err)
	// 	} else {
	// 		fmt.Println(resp.Value)
	// 	}
	// }

	var choice string
	var hostChoice string
	var port int
	fmt.Println("Select cluster to communicate : ")
	fmt.Println("0. node0")
	fmt.Println("1. node1")
	fmt.Println("2. node2")
	fmt.Scanln(&hostChoice)

	host := "node-" + hostChoice + ".kvs.svc.localho.st"
	fmt.Println("Enter port no : ")
	fmt.Scanln(&port)

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client := pb.NewKVClient(conn)

	for {
		fmt.Println("Menu:")
		fmt.Println("1. Get")
		fmt.Println("2. Put")
		fmt.Println("3. Delete")
		fmt.Println("4. Quit")
		fmt.Print("Enter your choice: ")
		fmt.Scanln(&choice)

		switch choice {
		// Get
		case "1":
			var key string
			fmt.Print("Enter key to get: ")
			fmt.Scanln(&key)
			if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: fmt.Sprintf("key%d", key)}); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(resp.Value)
			}
		// Put
		case "2":
			var key, value string
			fmt.Print("Enter key to put: ")
			fmt.Scanln(&key)
			fmt.Print("Enter value to put: ")
			fmt.Scanln(&value)
			if _, err := client.Put(context.Background(), &pb.PutRequest{Key: fmt.Sprintf("key%d", key), Value: fmt.Sprintf("value%d", value)}); err != nil {
				fmt.Println(err)
			}
		// Delete
		case "3":
			var key string
			fmt.Print("Enter key to delete: ")
			fmt.Scanln(&key)
			if _, err := client.Delete(context.Background(), &pb.DeleteRequest{Key: fmt.Sprintf("key%d", key)}); err != nil {
				fmt.Println(err)
			}
		case "4":
			fmt.Println("Quitting...")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}
