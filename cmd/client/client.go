package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

var (
	kvctlConfigPath = flag.String("kvctlConfigPath", "./kvctl/config/samples/kvctl_v1_kvstore.yaml", "Path to kvctl config")
	dummyData       = flag.Bool("dummyData", false, "Should insert dummy data?")
)

type KvctlSpec struct {
	Shards        int            `yaml:"shards"`
	Replicas      int            `yaml:"replicas"`
	IngressDomain string         `yaml:"ingressDomain"`
	PodSpec       map[string]any `yaml:"podSpec"`
}
type KvctlMetadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}
type KvctlConfig struct {
	ApiVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   KvctlMetadata `yaml:"metadata"`
	Spec       KvctlSpec     `yaml:"spec"`
}

func main() {
	flag.Parse()
	kvctlConfig := KvctlConfig{}
	if buf, err := os.ReadFile(*kvctlConfigPath); err != nil {
		log.Fatalf("Unable to read config file at %s: %s", *kvctlConfigPath, err)
		panic(err)
	} else {
		if err := yaml.Unmarshal(buf, &kvctlConfig); err != nil {
			log.Fatalf("Unable to parse config file at %s: %s", *kvctlConfigPath, err)
			panic(err)
		}
	}

	if *dummyData {
		address := fmt.Sprintf("%s-shard-%d.%s:80", kvctlConfig.Metadata.Name, 0, kvctlConfig.Spec.IngressDomain)
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			panic(err)
		}
		client := pb.NewKVClient(conn)
		for i := range 100 {
			fmt.Printf("Setting key %d\n", i)
			key := strconv.Itoa(i)
			value := strconv.Itoa(100 - i)
			if _, err := client.Put(context.Background(), &pb.PutRequest{Key: key, Value: value}); err != nil {
				fmt.Println(err)
			}
		}
		return
	}

	var choice int
	fmt.Println(kvctlConfig.ApiVersion)
	fmt.Println("Select shard to communicate: ")
	for i := 0; i < kvctlConfig.Spec.Shards; i++ {
		fmt.Printf("%d. %s-shard-%d\n", i, kvctlConfig.Metadata.Name, i)
	}
	fmt.Print("Choice: ")
	fmt.Scanln(&choice)

	address := fmt.Sprintf("%s-shard-%d.%s:80", kvctlConfig.Metadata.Name, choice, kvctlConfig.Spec.IngressDomain)

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
		case 1:
			var key string
			fmt.Print("Enter key to get: ")
			fmt.Scanln(&key)
			if resp, err := client.Get(context.Background(), &pb.GetRequest{Key: key}); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(resp.Value)
			}
		// Put
		case 2:
			var key, value string
			fmt.Print("Enter key to put: ")
			fmt.Scanln(&key)
			fmt.Print("Enter value to put: ")
			fmt.Scanln(&value)
			if _, err := client.Put(context.Background(), &pb.PutRequest{Key: key, Value: value}); err != nil {
				fmt.Println(err)
			}
		// Delete
		case 3:
			var key string
			fmt.Print("Enter key to delete: ")
			fmt.Scanln(&key)
			if _, err := client.Delete(context.Background(), &pb.DeleteRequest{Key: key}); err != nil {
				fmt.Println(err)
			}
		case 4:
			fmt.Println("Quitting...")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}
