package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	capture "github.com/CaptainIRS/sharded-kvs/internal"
	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	kvstore "github.com/CaptainIRS/sharded-kvs/internal/raft/kvstore"
	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
)

type kvServer struct {
	pb.UnimplementedKVServer

	kvStore *kvstore.KVStore
}

type nodeRpcServer struct {
	pb.UnimplementedNodeRPCServer
}

type replicaRpcServer struct {
	pb.UnimplementedReplicaRPCServer
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
	address        = flag.String("address", "localhost", "This node's IP address")
	kvport         = flag.Int("port", 8080, "The key-value server port")
	nodeport       = flag.Int("nodeport", 8081, "The node RPC server port")
	raftport       = flag.Int("raftport", 8082, "The Raft RPC server port")
	replicaport    = flag.Int("replicaport", 8083, "The leader RPC server port")
	node           = flag.Int("node", 0, "Node ID")
	replica        = flag.Int("replica", 0, "Replica ID")
	nodes          = flag.Int("nodes", 1, "Number of nodes")
	replicas       = flag.Int("replicas", 1, "Number of replicas")
	folder         = flag.String("folder", "/data", "Folder to store data")
	ch             = *&consistent.Consistent{}
	nodeclients    = make(map[string]pb.NodeRPCClient)
	replicaclients = make(map[string]pb.ReplicaRPCClient)
	kvStore        *kvstore.KVStore
	isRestart      bool
)

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	nodeclient := nodeclients[member.String()]
	log.Printf("Forwarding get request for key %s to %s", key, member)
	return nodeclient.Get(ctx, &pb.GetRequest{Key: key})
}

func (s *kvServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	nodeclient := nodeclients[member.String()]
	log.Printf("Forwarding put request for key %s to %s", key, member)
	return nodeclient.Put(ctx, &pb.PutRequest{Key: key, Value: in.Value})
}

func (s *kvServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	nodeclient := nodeclients[member.String()]
	log.Printf("Forwarding delete request for key %s to %s", key, member)
	return nodeclient.Delete(ctx, &pb.DeleteRequest{Key: key})
}

func (s *kvServer) RangeQuery(ctx context.Context, in *pb.RangeQueryRequest) (*pb.RangeQueryResponse, error) {
	key1 := in.Key1
	key2 := in.Key2

	key1Int, _ := strconv.ParseInt(key1, 10, 64)
	key2Int, _ := strconv.ParseInt(key2, 10, 64)

	response := ""

	for i := key1Int; i <= key2Int; i++ {
		currentKey := strconv.FormatInt(i, 10)
		member := ch.LocateKey([]byte(currentKey))
		nodeclient := nodeclients[member.String()]
		log.Printf("Forwarding get request for key %s to %s", currentKey, member)
		resp, err := nodeclient.Get(ctx, &pb.GetRequest{Key: currentKey})
		if err != nil {
			response = response + "For key : " + currentKey + " " + err.Error() + "\n"
		} else {
			response = response + "For key : " + currentKey + " " + resp.Value + "\n"
		}
	}

	return &pb.RangeQueryResponse{Value: response}, nil
}

func (s *nodeRpcServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	log.Printf("Serving get request for key %s", in.Key)
	if value, err := kvStore.Get(in.Key); err != nil {
		return nil, err
	} else {
		return &pb.GetResponse{Value: value}, nil
	}
}

func (s *nodeRpcServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	if err := kvStore.Set(in.Key, in.Value); err == raft.ErrNotLeader {
		_, leaderId := kvStore.Leader()
		if leaderId == "" {
			return nil, fmt.Errorf("No leader found")
		} else {
			log.Printf("Forwarding put request for key %s to leader %s", in.Key, leaderId)
			return replicaclients[string(leaderId)].Put(ctx, in)
		}
	} else if err != nil {
		return nil, err
	}
	log.Printf("Serving put request for key %s", in.Key)
	return &pb.PutResponse{}, nil
}

func (s *nodeRpcServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if err := kvStore.Delete(in.Key); err == raft.ErrNotLeader {
		_, leaderId := kvStore.Leader()
		if leaderId == "" {
			return nil, fmt.Errorf("No leader found")
		} else {
			log.Printf("Forwarding delete request to leader %s", leaderId)
			return replicaclients[string(leaderId)].Delete(ctx, in)
		}
	} else if err != nil {
		return nil, err
	}
	log.Printf("Serving delete request for key %s", in.Key)
	return &pb.DeleteResponse{}, nil
}

func (s *replicaRpcServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	log.Printf("Received put request for key %s", in.Key)
	if err := kvStore.Set(in.Key, in.Value); err != nil {
		return nil, err
	} else {
		return &pb.PutResponse{}, nil
	}
}

func (s *replicaRpcServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	log.Printf("Received delete request for key %s", in.Key)
	if err := kvStore.Delete(in.Key); err != nil {
		return nil, err
	} else {
		return &pb.DeleteResponse{}, nil
	}
}

func (s *replicaRpcServer) Join(ctx context.Context, in *pb.JoinRequest) (*pb.JoinResponse, error) {
	log.Printf("Processing join request from replica-%d at %s", in.ReplicaId, in.Address)
	if err := kvStore.Join(in.Address, *raftport, *node, int(in.ReplicaId)); err != nil {
		return nil, err
	} else {
		return &pb.JoinResponse{}, nil
	}
}

func (s *replicaRpcServer) LeaderID(ctx context.Context, in *pb.LeaderIDRequest) (*pb.LeaderIDResponse, error) {
	_, leaderId := kvStore.Leader()
	if leaderId == "" {
		return nil, fmt.Errorf("No leader found")
	}
	return &pb.LeaderIDResponse{LeaderId: string(leaderId)}, nil
}

func (s *replicaRpcServer) DemoteVoter(ctx context.Context, in *pb.DemoteVoterRequest) (*pb.DemoteVoterResponse, error) {
	log.Printf("Processing demote voter request from replica-%d at %s", in.ReplicaId, in.Address)
	if err := kvStore.DemoteVoter(in.Address, *raftport, *node, int(in.ReplicaId)); err != nil {
		return nil, err
	} else {
		return &pb.DemoteVoterResponse{}, nil
	}
}

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	log.SetFlags(0)
	flag.Parse()
	log.Printf("Starting replica %d of node %d on port %d", *replica, *node, *kvport)

	ctx, cancel := context.WithCancel(context.Background())

	if _, err := os.Stat(path.Join(*folder, "bolt")); os.IsNotExist(err) {
		isRestart = false
	} else {
		isRestart = true
	}

	go capture.RunPacketCapture(ctx, *address, fmt.Sprintf("%d", *raftport))

	members := []consistent.Member{}
	for n := 0; n < *nodes; n++ {
		members = append(members, Member(fmt.Sprintf("node-%d", n)))
	}

	cfg := consistent.Config{
		PartitionCount:    7,
		ReplicationFactor: 20,
		Load:              1.25,
		Hasher:            hasher{},
	}
	ch = *consistent.New(members, cfg)

	kvStore = kvstore.NewKVStore()
	log.Printf("Starting Raft server on port %d", *raftport)
	err := kvStore.Open(*folder, *address, *raftport, *node, *replica, !isRestart && *replica == 0)
	if err != nil {
		panic(err)
	}

	go func() {
		for n := 0; n < *nodes; n++ {
			for {
				conn, err := grpc.Dial(fmt.Sprintf("node-%d:%d", n, *nodeport), grpc.WithInsecure())
				if err != nil {
					log.Printf("Failed to connect to node-%d. Retrying...", n)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("Connected to node-%d", n)
				nodeclients[fmt.Sprintf("node-%d", n)] = pb.NewNodeRPCClient(conn)
				break
			}
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *nodeport))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting node server on port %d", *nodeport)
		grpcServer := grpc.NewServer()
		pb.RegisterNodeRPCServer(grpcServer, &nodeRpcServer{})
		grpcServer.Serve(lis)
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *replicaport))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting replica server on port %d", *replicaport)
		grpcServer := grpc.NewServer()
		pb.RegisterReplicaRPCServer(grpcServer, &replicaRpcServer{})
		grpcServer.Serve(lis)
	}()

	go func() {
		for r := 0; r < *replicas; r++ {
			for {
				conn, err := grpc.Dial(fmt.Sprintf("node-%d-replica-%d.node-%d:%d", *node, r, *node, *replicaport), grpc.WithInsecure())
				if err != nil {
					log.Printf("Failed to connect to replica-%d. Retrying...", r)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("Connected to replica-%d", r)
				replicaRpcClient := pb.NewReplicaRPCClient(conn)
				replicaclients[fmt.Sprintf("node-%d-replica-%d.node-%d.kvs.svc.localho.st:%d", *node, r, *node, *raftport)] = replicaRpcClient
				break
			}
		}

		leaderFound := false
		leaderId, _ := kvStore.Leader()
		if leaderId != "" {
			leaderFound = true
		}
		for !leaderFound {
			for r := 0; r < *replicas; r++ {
				resp, err := replicaclients[fmt.Sprintf("node-%d-replica-%d.node-%d.kvs.svc.localho.st:%d", *node, r, *node, *raftport)].LeaderID(ctx, &pb.LeaderIDRequest{})
				if err != nil {
					log.Printf("Failed to get leader ID from replica-%d. Retrying...", r)
					time.Sleep(1 * time.Second)
					continue
				}
				if resp.LeaderId != "" {
					leaderFound = true
					leaderId = raft.ServerAddress(resp.LeaderId)
					break
				}
			}
		}

		for !kvStore.HasLatestLogs() {
			time.Sleep(1 * time.Second)
		}
		leaderClient := replicaclients[string(leaderId)]
		leaderClient.Join(ctx, &pb.JoinRequest{ReplicaId: int32(*replica), Address: *address})
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *kvport))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting key-value server on port %d", *kvport)
		grpcServer := grpc.NewServer()
		pb.RegisterKVServer(grpcServer, &kvServer{})
		grpcServer.Serve(lis)
	}()

	<-signalCh

	log.Printf("Shutting down")
	leaderId, _ := kvStore.Leader()
	leaderClient := replicaclients[string(leaderId)]
	if err := kvStore.Close(); err != nil {
		log.Printf("Failed to close KV store: %v", err)
	}
	leaderClient.DemoteVoter(ctx, &pb.DemoteVoterRequest{ReplicaId: int32(*replica), Address: *address})
	time.Sleep(5 * time.Second)
	cancel()
}
