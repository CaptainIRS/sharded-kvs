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
	"sync"
	"syscall"
	"time"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	kvstore "github.com/CaptainIRS/sharded-kvs/internal/raft/kvstore"
	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

type kvServer struct {
	pb.UnimplementedKVServer
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

type ErrWritesDisabled struct {}

func (e ErrWritesDisabled) Error() string {
	return "Writes disabled. Please try again later."
}

var (
	address                 = flag.String("address", "localhost", "This shard's IP address")
	kvPort                  = flag.Int("port", 8080, "The key-value server port")
	shardPort               = flag.Int("shardPort", 8081, "The shard RPC server port")
	raftPort                = flag.Int("raftPort", 8082, "The Raft RPC server port")
	replicaPort             = flag.Int("replicaPort", 8083, "The leader RPC server port")
	shard                   = flag.String("shard", "shard-0", "Shard ID")
	replica                 = flag.String("replica", "shard-0-replica-0", "Replica ID")
	folder                  = flag.String("folder", "/data", "Folder to store data")
	configFile              = flag.String("configFile", "/etc/config/config.yaml", "Path of the KV Store configuration file")
	shouldBootstrap         = flag.Bool("shouldBootstrap", false, "Should this replica bootstrap Raft?")
	ch                      = consistent.Consistent{}
	shardClients            = make(map[string]pb.ShardRPCClient)
	replicaClients          = make(map[string]pb.ReplicaRPCClient)
	kvStore                 *kvstore.KVStore
	isRestart               bool
	redistributingKeys      = false
	redistributingKeysMutex = sync.Mutex{}
	consistentHashCfg       = consistent.Config{
		PartitionCount:    consistent.DefaultPartitionCount,
		ReplicationFactor: consistent.DefaultReplicationFactor,
		Load:              consistent.DefaultLoad,
		Hasher:            hasher{},
	}
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
	return shardclient.Put(ctx, &pb.InternalPutRequest{Key: key, Value: in.Value, IsRedistribution: false})
}

func (s *kvServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	key := in.Key
	member := ch.LocateKey([]byte(key))
	shardclient := shardClients[member.String()]
	log.Printf("Forwarding delete request for key %s to %s", key, member)
	return shardclient.Delete(ctx, &pb.DeleteRequest{Key: key})
}

func ReadConfig() (kvConfig KVStoreConfig) {
	if buf, err := os.ReadFile(*configFile); err != nil {
		log.Fatalf("Unable to read config file at %s", *configFile)
		panic(err)
	} else {
		if err := yaml.Unmarshal(buf, &kvConfig); err != nil {
			log.Fatalf("Unable to parse config file at %s", *configFile)
			panic(err)
		}
	}
	return kvConfig
}

func RedistributeKeys() {
	redistributingKeysMutex.Lock()
	if redistributingKeys {
		redistributingKeysMutex.Unlock()
		return
	}
	redistributingKeys = true
	redistributingKeysMutex.Unlock()
	defer func() {
		redistributingKeysMutex.Lock()
		redistributingKeys = false
		redistributingKeysMutex.Unlock()
	}()

	kvConfig := ReadConfig()
	if len(kvConfig.NewShards) == 0 {
		log.Println("Waiting for new configuration...")
		return
	}
	PopulateShardClients(kvConfig.NewShards)
	newMembers := []consistent.Member{}
	for _, shard := range kvConfig.NewShards {
		newMembers = append(newMembers, Member(shard.Name))
	}
	newCh := *consistent.New(newMembers, consistentHashCfg)
	for key := range kvStore.Keys() {
		oldLocation := ch.LocateKey([]byte(key)).String()
		newLocation := newCh.LocateKey([]byte(key)).String()

		if oldLocation != newLocation {
			if oldLocation == *shard {
				log.Printf("Key: %s, Old location: %s, New location: %s", key, oldLocation, newLocation)
				if value, err := kvStore.Get(key); err != nil {
					log.Printf("Error when getting key %s for relocation: %s", key, err)
				} else {
					targetClient := shardClients[newLocation]
					if _, err := targetClient.Put(context.Background(), &pb.InternalPutRequest{Key: key, Value: value, IsRedistribution: true}); err != nil {
						log.Printf("Error when sending key %s to %s: %s", key, newLocation, err)
					} else {
						if err := kvStore.Delete(key); err != nil {
							log.Printf("Error when purging key %s: %s", key, err)
						}
					}
				}
			}
		}
	}
}

func (s *shardRpcServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	log.Printf("Serving get request for key %s", in.Key)
	if value, err := kvStore.Get(in.Key); err != nil {
		return nil, err
	} else {
		return &pb.GetResponse{Value: value}, nil
	}
}

func (s *shardRpcServer) Put(ctx context.Context, in *pb.InternalPutRequest) (*pb.PutResponse, error) {
	if err := kvStore.Set(in.Key, in.Value); err == raft.ErrNotLeader {
		leader := kvStore.Leader()
		if leader == "" {
			return nil, fmt.Errorf("no leader found")
		} else {
			log.Printf("Forwarding put request for key %s to leader %s", in.Key, leader)
			return replicaClients[leader].Put(ctx, in)
		}
	} else if err != nil {
		return nil, err
	}
	log.Printf("Serving put request for key %s", in.Key)
	return &pb.PutResponse{}, nil
}

func (s *shardRpcServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if err := kvStore.Delete(in.Key); err == raft.ErrNotLeader {
		leader := kvStore.Leader()
		if leader == "" {
			return nil, fmt.Errorf("no leader found")
		} else {
			log.Printf("Forwarding delete request to leader %s", leader)
			return replicaClients[leader].Delete(ctx, in)
		}
	} else if err != nil {
		return nil, err
	}
	log.Printf("Serving delete request for key %s", in.Key)
	return &pb.DeleteResponse{}, nil
}

func (s *shardRpcServer) PauseWrites(ctx context.Context, in *pb.PauseWritesRequest) (*pb.PauseWritesResponse, error) {
	leader := kvStore.Leader()
	if leader == "" {
		return nil, fmt.Errorf("no leader found")
	} else {
		return replicaClients[leader].PauseWrites(ctx, in)
	}
}

func (s *shardRpcServer) RedistributeKeys(ctx context.Context, in *pb.RedistributeKeysRequest) (*pb.RedistributeKeysResponse, error) {
	leader := kvStore.Leader()
	if leader == "" {
		return nil, fmt.Errorf("no leader found")
	} else {
		return replicaClients[leader].RedistributeKeys(ctx, in)
	}
}

func (s *shardRpcServer) ResumeWrites(ctx context.Context, in *pb.ResumeWritesRequest) (*pb.ResumeWritesResponse, error) {
	leader := kvStore.Leader()
	if leader == "" {
		return nil, fmt.Errorf("no leader found")
	} else {
		for replica, replicaClient := range replicaClients {
			if _, err := replicaClient.ReloadConfig(ctx, &pb.ReloadConfigRequest{}); err != nil {
				log.Printf("Config reload request sent to %s", replica)
			}
		}
		return replicaClients[leader].ResumeWrites(ctx, in)
	}
}

func (s *replicaRpcServer) Put(ctx context.Context, in *pb.InternalPutRequest) (*pb.PutResponse, error) {
	log.Printf("Received put request for key %s", in.Key)
	if !in.IsRedistribution && !kvStore.GetWritesEnabled() {
		return nil, ErrWritesDisabled{}
	}
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

func (s *replicaRpcServer) PauseWrites(ctx context.Context, in *pb.PauseWritesRequest) (*pb.PauseWritesResponse, error) {
	kvConfig := ReadConfig()
	if len(kvConfig.NewShards) == 0 {
		log.Println("Waiting for new configuration...")
		return &pb.PauseWritesResponse{
			IsPaused: false,
		}, nil
	}
	if err := kvStore.SetWritesEnabled(false); err != nil {
		log.Printf("Could not disable writes: %s", err)
		return nil, err
	} else {
		log.Println("Writes disabled successfully")
		return &pb.PauseWritesResponse{
			IsPaused: true,
		}, nil
	}
}

func (s *replicaRpcServer) RedistributeKeys(ctx context.Context, in *pb.RedistributeKeysRequest) (*pb.RedistributeKeysResponse, error) {
	go RedistributeKeys()

	kvConfig := ReadConfig()
	newMembers := []consistent.Member{}
	for _, shard := range kvConfig.NewShards {
		newMembers = append(newMembers, Member(shard.Name))
	}
	newCh := *consistent.New(newMembers, consistentHashCfg)
	pendingKeys := 0
	for key := range kvStore.Keys() {
		oldLocation := ch.LocateKey([]byte(key)).String()
		newLocation := newCh.LocateKey([]byte(key)).String()
		if oldLocation != newLocation && oldLocation == *shard {
			pendingKeys++
		}
	}
	log.Printf("Pending keys to be redistributed: %d", pendingKeys)
	return &pb.RedistributeKeysResponse{
		PendingKeys: int32(pendingKeys),
	}, nil
}

func (s *replicaRpcServer) ResumeWrites(ctx context.Context, in *pb.ResumeWritesRequest) (*pb.ResumeWritesResponse, error) {
	kvConfig := ReadConfig()
	if len(kvConfig.NewShards) != 0 {
		return &pb.ResumeWritesResponse{
			IsResumed: false,
		}, nil
	}
	reloadSuccessful := true
	for replica, replicaClient := range replicaClients {
		if res, err := replicaClient.ReloadConfig(ctx, &pb.ReloadConfigRequest{}); err != nil {
			log.Printf("Could not reload config for %s: %s", replica, err)
			return &pb.ResumeWritesResponse{
				IsResumed: false,
			}, nil
		} else {
			if !res.ReloadSuccessful {
				log.Printf("Waiting for %s to reload config", replica)
				reloadSuccessful = false
			}
		}
	}
	if !reloadSuccessful {
		return &pb.ResumeWritesResponse{
			IsResumed: reloadSuccessful,
		}, nil
	}
	if err := kvStore.SetWritesEnabled(true); err != nil {
		log.Printf("Could not enable writes: %s", err)
		return nil, err
	} else {
		log.Println("Writes resumed successfully")
		return &pb.ResumeWritesResponse{
			IsResumed: true,
		}, nil
	}
}

func (s *replicaRpcServer) Join(ctx context.Context, in *pb.JoinRequest) (*pb.JoinResponse, error) {
	log.Printf("Processing join request from %s at %s", in.Replica, in.Address)
	if err := kvStore.Join(in.Address, *raftPort, *shard, in.Replica); err != nil {
		return nil, err
	} else {
		return &pb.JoinResponse{}, nil
	}
}

func (s *replicaRpcServer) Leader(ctx context.Context, in *pb.LeaderRequest) (*pb.LeaderResponse, error) {
	leader := kvStore.Leader()
	if leader == "" {
		return nil, fmt.Errorf("no leader found")
	}
	return &pb.LeaderResponse{Leader: leader}, nil
}

func (s *replicaRpcServer) DemoteVoter(ctx context.Context, in *pb.DemoteVoterRequest) (*pb.DemoteVoterResponse, error) {
	log.Printf("Processing demote voter request from %s at %s", in.Replica, in.Address)
	if err := kvStore.DemoteVoter(in.Address, *raftPort, *shard, in.Replica); err != nil {
		return nil, err
	} else {
		return &pb.DemoteVoterResponse{}, nil
	}
}

func ReloadConfig() bool {
	kvConfig := KVStoreConfig{}
	if buf, err := os.ReadFile(*configFile); err != nil {
		log.Fatalf("Unable to read config file at %s", *configFile)
		panic(err)
	} else {
		kvConfig = KVStoreConfig{}
		if err := yaml.Unmarshal(buf, &kvConfig); err != nil {
			log.Fatalf("Unable to parse config file at %s", *configFile)
			panic(err)
		}
		if len(kvConfig.NewShards) != 0 {
			return false
		}
	}
	members := []consistent.Member{}
	for _, shard := range kvConfig.Shards {
		members = append(members, Member(shard.Name))
	}
	ch = *consistent.New(members, consistentHashCfg)

	PopulateShardClients(kvConfig.Shards)
	return true

}
func (s *replicaRpcServer) ReloadConfig(ctx context.Context, in *pb.ReloadConfigRequest) (*pb.ReloadConfigResponse, error) {
	reloadSuccessful := ReloadConfig()
	return &pb.ReloadConfigResponse{
		ReloadSuccessful: reloadSuccessful,
	}, nil
}

type KVStoreShard struct {
	Name     string   `yaml:"name"`
	Replicas []string `yaml:"replicas"`
}

type KVStoreConfig struct {
	Shards    []KVStoreShard `yaml:"shards"`
	NewShards []KVStoreShard `yaml:"newShards,omitempty"`
}

func PopulateShardClients(shards []KVStoreShard) {
	for _, destinationShard := range shards {
		for {
			_, exists := shardClients[destinationShard.Name]
			if exists {
				break
			}
			conn, err := grpc.Dial(fmt.Sprintf("%s:%d", destinationShard.Name, *shardPort), grpc.WithInsecure())
			if err != nil {
				log.Printf("Failed to connect to %s. Retrying...", destinationShard.Name)
				time.Sleep(1 * time.Second)
				continue
			}
			shardClients[destinationShard.Name] = pb.NewShardRPCClient(conn)
			break
		}
	}
}

func JoinCluster(ctx context.Context, replicas []string) {
	leaderFound := false
	leader := kvStore.Leader()
	if leader != "" {
		leaderFound = true
	}
	for !leaderFound {
		for _, destinationReplica := range replicas {
			resp, err := replicaClients[fmt.Sprintf("%s.%s:%d", destinationReplica, *shard, *raftPort)].Leader(ctx, &pb.LeaderRequest{})
			if err != nil {
				log.Printf("Failed to get leader ID from %s.%s. Retrying...", destinationReplica, *shard)
				time.Sleep(1 * time.Second)
				continue
			}
			if resp.Leader != "" {
				leaderFound = true
				leader = resp.Leader
				break
			}
		}
	}

	for !kvStore.HasLatestLogs() {
		// Wait for the logs to be synced
		time.Sleep(1 * time.Second)
	}
	leaderClient := replicaClients[leader]
	leaderClient.Join(ctx, &pb.JoinRequest{Replica: *replica, Address: *address})
}

func PopulateReplicaClients(ctx context.Context, kvConfig KVStoreConfig) {
	replicas := []string{}
	for _, currentShard := range append(kvConfig.Shards, kvConfig.NewShards...) {
		if currentShard.Name == *shard {
			replicas = append(replicas, currentShard.Replicas...)
		}
	}
	for _, destinationReplica := range replicas {
		for {
			if _, exists := replicaClients[fmt.Sprintf("%s.%s:%d", destinationReplica, *shard, *raftPort)]; exists {
				break
			}
			conn, err := grpc.Dial(fmt.Sprintf("%s.%s:%d", destinationReplica, *shard, *replicaPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to %s.%s. Retrying...", destinationReplica, *shard)
				time.Sleep(1 * time.Second)
				continue
			}
			replicaRpcClient := pb.NewReplicaRPCClient(conn)
			replicaClients[fmt.Sprintf("%s.%s:%d", destinationReplica, *shard, *raftPort)] = replicaRpcClient
			break
		}
	}
	JoinCluster(ctx, replicas)
}
func StartShardRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *shardPort))
	if err != nil {
		panic(err)
	}
	log.Printf("Starting shard server on port %d", *shardPort)
	grpcServer := grpc.NewServer()
	pb.RegisterShardRPCServer(grpcServer, &shardRpcServer{})
	grpcServer.Serve(lis)
}
func StartReplicaRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *replicaPort))
	if err != nil {
		panic(err)
	}
	log.Printf("Starting replica server on port %d", *replicaPort)
	grpcServer := grpc.NewServer()
	pb.RegisterReplicaRPCServer(grpcServer, &replicaRpcServer{})
	grpcServer.Serve(lis)
}
func StartKVRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *kvPort))
	if err != nil {
		panic(err)
	}
	log.Printf("Starting key-value server on port %d", *kvPort)
	grpcServer := grpc.NewServer()
	pb.RegisterKVServer(grpcServer, &kvServer{})
	grpcServer.Serve(lis)
}
func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	log.SetFlags(0)
	flag.Parse()
	log.Printf("Starting replica %s of shard %s on port %d", *replica, *shard, *kvPort)

	kvConfig := KVStoreConfig{}
	if buf, err := os.ReadFile(*configFile); err != nil {
		log.Fatalf("Unable to read config file at %s", *configFile)
		panic(err)
	} else {
		log.Print(string(buf))
		if err := yaml.Unmarshal(buf, &kvConfig); err != nil {
			log.Fatalf("Unable to parse config file at %s", *configFile)
			panic(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	if _, err := os.Stat(path.Join(*folder, "bolt")); os.IsNotExist(err) {
		isRestart = false
	} else {
		isRestart = true
	}

	members := []consistent.Member{}
	for _, shard := range kvConfig.Shards {
		members = append(members, Member(shard.Name))
	}

	ch = *consistent.New(members, consistentHashCfg)

	kvStore = kvstore.NewKVStore()
	log.Printf("Starting Raft server on port %d", *raftPort)

	err := kvStore.Open(*folder, *address, *raftPort, *shard, *replica, !isRestart && *shouldBootstrap)
	if err != nil {
		panic(err)
	}

	go PopulateShardClients(append(kvConfig.Shards, kvConfig.NewShards...))
	go PopulateReplicaClients(ctx, kvConfig)

	go StartShardRPCServer()
	go StartReplicaRPCServer()
	go StartKVRPCServer()

	<-signalCh

	log.Printf("Shutting down")
	if err := kvStore.Close(); err != nil {
		log.Printf("Failed to close KV store: %v", err)
	}

	if leader := kvStore.Leader(); leader == "" {
		log.Printf("No leader found")
	} else {
		leaderClient := replicaClients[string(leader)]
		leaderClient.DemoteVoter(ctx, &pb.DemoteVoterRequest{Replica: *replica, Address: *address})
	}
	log.Printf("Waiting for 5 seconds before shutting down...")
	time.Sleep(10 * time.Second)
	cancel()
}
