package main

import (
	"context"
	"flag"
	"fmt"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Parse()

	var choice string
	var hostChoice string
	fmt.Println("Select cluster to communicate : ")
	fmt.Println("0. node0")
	fmt.Println("1. node1")
	fmt.Println("2. node2")
	fmt.Scanln(&hostChoice)

	address := "node-" + hostChoice + ".kvs.svc.localho.st:8080"

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
			if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: key}); err != nil {
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
			if _, err := client.Put(context.Background(), &pb.PutRequest{Key: key, Value: value}); err != nil {
				fmt.Println(err)
			}
		// Delete
		case "3":
			var key string
			fmt.Print("Enter key to delete: ")
			fmt.Scanln(&key)
			if _, err := client.Delete(context.Background(), &pb.DeleteRequest{Key: key}); err != nil {
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
