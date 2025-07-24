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

type shardRpcServer struct {
	pb.UnimplementedShardRPCServer
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
	address        = flag.String("address", "localhost", "This shard's IP address")
	kvPort         = flag.Int("port", 8080, "The key-value server port")
	shardPort       = flag.Int("shardPort", 8081, "The shard RPC server port")
	raftPort       = flag.Int("raftPort", 8082, "The Raft RPC server port")
	replicaPort    = flag.Int("replicaPort", 8083, "The leader RPC server port")
	shard           = flag.Int("shard", 0, "Shard ID")
	replica        = flag.Int("replica", 0, "Replica ID")
	shards          = flag.Int("shards", 1, "Number of shards")
	replicas       = flag.Int("replicas", 1, "Number of replicas")
	folder         = flag.String("folder", "/data", "Folder to store data")
	ch             = consistent.Consistent{}
	shardClients    = make(map[string]pb.ShardRPCClient)
	replicaClients = make(map[string]pb.ReplicaRPCClient)
	kvStore        *kvstore.KVStore
	isRestart      bool
)

func (s *kvServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	shardclient := shardClients[member.String()]
	log.Printf("Forwarding get request for key %s to %s", key, member)
	return shardclient.Get(ctx, &pb.GetRequest{Key: key})
}

func (s *kvServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	shardclient := shardClients[member.String()]
	log.Printf("Forwarding put request for key %s to %s", key, member)
	return shardclient.Put(ctx, &pb.PutRequest{Key: key, Value: in.Value})
}

func (s *kvServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	shardclient := shardClients[member.String()]
	log.Printf("Forwarding delete request for key %s to %s", key, member)
	return shardclient.Delete(ctx, &pb.DeleteRequest{Key: key})
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
		shardclient := shardClients[member.String()]
		log.Printf("Forwarding get request for key %s to %s", currentKey, member)
		resp, err := shardclient.Get(ctx, &pb.GetRequest{Key: currentKey})
		if err != nil {
			response = response + "For key : " + currentKey + " " + err.Error() + "\n"
		} else {
			response = response + "For key : " + currentKey + " " + resp.Value + "\n"
		}
	}

	return &pb.RangeQueryResponse{Value: response}, nil
}

func (s *shardRpcServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	log.Printf("Serving get request for key %s", in.Key)
	if value, err := kvStore.Get(in.Key); err != nil {
		return nil, err
	} else {
		return &pb.GetResponse{Value: value}, nil
	}
}

func (s *shardRpcServer) Put(ctx context.Context, in *pb.PutRequest) (*pb.PutResponse, error) {
	if err := kvStore.Set(in.Key, in.Value); err == raft.ErrNotLeader {
		_, leaderId := kvStore.Leader()
		if leaderId == "" {
			return nil, fmt.Errorf("no leader found")
		} else {
			log.Printf("Forwarding put request for key %s to leader %s", in.Key, leaderId)
			return replicaClients[string(leaderId)].Put(ctx, in)
		}
	} else if err != nil {
		return nil, err
	}
	log.Printf("Serving put request for key %s", in.Key)
	return &pb.PutResponse{}, nil
}

func (s *shardRpcServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if err := kvStore.Delete(in.Key); err == raft.ErrNotLeader {
		_, leaderId := kvStore.Leader()
		if leaderId == "" {
			return nil, fmt.Errorf("no leader found")
		} else {
			log.Printf("Forwarding delete request to leader %s", leaderId)
			return replicaClients[string(leaderId)].Delete(ctx, in)
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
	if err := kvStore.Join(in.Address, *raftPort, *shard, int(in.ReplicaId)); err != nil {
		return nil, err
	} else {
		return &pb.JoinResponse{}, nil
	}
}

func (s *replicaRpcServer) LeaderID(ctx context.Context, in *pb.LeaderIDRequest) (*pb.LeaderIDResponse, error) {
	_, leaderId := kvStore.Leader()
	if leaderId == "" {
		return nil, fmt.Errorf("no leader found")
	}
	return &pb.LeaderIDResponse{LeaderId: string(leaderId)}, nil
}

func (s *replicaRpcServer) DemoteVoter(ctx context.Context, in *pb.DemoteVoterRequest) (*pb.DemoteVoterResponse, error) {
	log.Printf("Processing demote voter request from replica-%d at %s", in.ReplicaId, in.Address)
	if err := kvStore.DemoteVoter(in.Address, *raftPort, *shard, int(in.ReplicaId)); err != nil {
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
	log.Printf("Starting replica %d of shard %d on port %d", *replica, *shard, *kvPort)

	ctx, cancel := context.WithCancel(context.Background())

	if _, err := os.Stat(path.Join(*folder, "bolt")); os.IsNotExist(err) {
		isRestart = false
	} else {
		isRestart = true
	}

	go capture.RunPacketCapture(ctx, *address, fmt.Sprintf("%d", *raftPort))

	members := []consistent.Member{}
	for n := 0; n < *shards; n++ {
		members = append(members, Member(fmt.Sprintf("shard-%d", n)))
	}

	cfg := consistent.Config{
		PartitionCount:    7,
		ReplicationFactor: 20,
		Load:              1.25,
		Hasher:            hasher{},
	}
	ch = *consistent.New(members, cfg)

	kvStore = kvstore.NewKVStore()
	log.Printf("Starting Raft server on port %d", *raftPort)
	err := kvStore.Open(*folder, *address, *raftPort, *shard, *replica, !isRestart && *replica == 0)
	if err != nil {
		panic(err)
	}

	go func() {
		for n := 0; n < *shards; n++ {
			for {
				conn, err := grpc.Dial(fmt.Sprintf("shard-%d:%d", n, *shardPort), grpc.WithInsecure())
				if err != nil {
					log.Printf("Failed to connect to shard-%d. Retrying...", n)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("Connected to shard-%d", n)
				shardClients[fmt.Sprintf("shard-%d", n)] = pb.NewShardRPCClient(conn)
				break
			}
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *shardPort))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting shard server on port %d", *shardPort)
		grpcServer := grpc.NewServer()
		pb.RegisterShardRPCServer(grpcServer, &shardRpcServer{})
		grpcServer.Serve(lis)
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *replicaPort))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting replica server on port %d", *replicaPort)
		grpcServer := grpc.NewServer()
		pb.RegisterReplicaRPCServer(grpcServer, &replicaRpcServer{})
		grpcServer.Serve(lis)
	}()

	go func() {
		for r := 0; r < *replicas; r++ {
			for {
				conn, err := grpc.Dial(fmt.Sprintf("shard-%d-replica-%d.shard-%d:%d", *shard, r, *shard, *replicaPort), grpc.WithInsecure())
				if err != nil {
					log.Printf("Failed to connect to replica-%d. Retrying...", r)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("Connected to replica-%d", r)
				replicaRpcClient := pb.NewReplicaRPCClient(conn)
				replicaClients[fmt.Sprintf("shard-%d-replica-%d.shard-%d.kvs.svc.localho.st:%d", *shard, r, *shard, *raftPort)] = replicaRpcClient
				break
			}
		}

		leaderFound := false
		_, leaderId := kvStore.Leader()
		if leaderId != "" {
			leaderFound = true
		}
		for !leaderFound {
			for r := 0; r < *replicas; r++ {
				resp, err := replicaClients[fmt.Sprintf("shard-%d-replica-%d.shard-%d.kvs.svc.localho.st:%d", *shard, r, *shard, *raftPort)].LeaderID(ctx, &pb.LeaderIDRequest{})
				if err != nil {
					log.Printf("Failed to get leader ID from replica-%d. Retrying...", r)
					time.Sleep(1 * time.Second)
					continue
				}
				if resp.LeaderId != "" {
					leaderFound = true
					leaderId = raft.ServerID(resp.LeaderId)
					break
				}
			}
		}

		for !kvStore.HasLatestLogs() {
			time.Sleep(1 * time.Second)
		}
		leaderClient := replicaClients[string(leaderId)]
		leaderClient.Join(ctx, &pb.JoinRequest{ReplicaId: int32(*replica), Address: *address})
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *kvPort))
		if err != nil {
			panic(err)
		}
		log.Printf("Starting key-value server on port %d", *kvPort)
		grpcServer := grpc.NewServer()
		pb.RegisterKVServer(grpcServer, &kvServer{})
		grpcServer.Serve(lis)
	}()

	<-signalCh

	log.Printf("Shutting down")
	if err := kvStore.Close(); err != nil {
		log.Printf("Failed to close KV store: %v", err)
	}

	if _, leaderId := kvStore.Leader(); leaderId == "" {
		log.Printf("No leader found")
	} else {
		leaderClient := replicaClients[string(leaderId)]
		leaderClient.DemoteVoter(ctx, &pb.DemoteVoterRequest{ReplicaId: int32(*replica), Address: *address})
	}
	log.Printf("Waiting for 5 seconds before shutting down...")
	time.Sleep(10 * time.Second)
	cancel()
}
