package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Parse()

	var choice string
	var hostChoice string
	fmt.Println("Select shard to communicate : ")
	fmt.Println("0. shard0")
	fmt.Println("1. shard1")
	fmt.Println("2. shard2")
	fmt.Scanln(&hostChoice)

	address := "shard-" + hostChoice + ".kvs.svc.localho.st:80"

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
		fmt.Println("4. Range Query")
		fmt.Println("5. Quit")
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
		// Range Query
		case "4":
			var input1, input2 string
			fmt.Print("Enter the first number: ")
			fmt.Scanln(&input1)
			fmt.Print("Enter the second number: ")
			fmt.Scanln(&input2)

			num1, err1 := strconv.ParseInt(input1, 10, 64)
			num2, err2 := strconv.ParseInt(input2, 10, 64)

			if err1 != nil || err2 != nil || num1 > num2 {
				fmt.Println("Error: Both inputs should be numbers and num1 should be lesser than num2")
				continue
			}

			if resp, err := client.RangeQuery(context.Background(), &pb.RangeQueryRequest{Key1: input1, Key2: input2}); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(resp.Value)
			}
		case "5":
			fmt.Println("Quitting...")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}
