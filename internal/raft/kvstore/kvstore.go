package kvstore

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	common "github.com/CaptainIRS/sharded-kvs/internal/raft"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
)

const (
	SET = iota
	DELETE
)

type KVStore struct {
	raft *raft.Raft

	mutex sync.Mutex
	store *map[string]string
}

func NewKVStore() *KVStore {
	return &KVStore{store: &map[string]string{}}
}

type KeyNotFound struct{}

func (k KeyNotFound) Error() string {
	return "Key not found"
}

func (k *KVStore) Get(key string) (string, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	if value, ok := (*k.store)[key]; ok {
		return value, nil
	}
	return "", KeyNotFound{}
}

func (k *KVStore) Set(key, value string) error {
	if k.raft.State() != raft.Leader {
		return raft.ErrNotLeader
	}
	log.Printf("Setting key %s to value %s", key, value)
	newEntry, err := proto.Marshal(&pb.KVFSMLogEntry{
		Operation: SET,
		Key:       key,
		Value:     &value,
	})
	if err != nil {
		return err
	}
	future := k.raft.Apply(newEntry, 10*time.Second)
	return future.Error()
}

func (k *KVStore) Delete(key string) error {
	if k.raft.State() != raft.Leader {
		return raft.ErrNotLeader
	}
	log.Printf("Deleting key %s", key)
	newEntry, err := proto.Marshal(&pb.KVFSMLogEntry{
		Operation: DELETE,
		Key:       key,
	})
	if err != nil {
		return err
	}
	future := k.raft.Apply(newEntry, 10*time.Second)
	return future.Error()
}

func (k *KVStore) Open(dir, ip string, raftPort, nodeId, replicaId int) error {
	var fsm = NewKVFsm(k.store)
	id := fmt.Sprintf("node-%d-replica-%d.node-%d.kvs.svc.localho.st:%d", nodeId, replicaId, nodeId, raftPort)
	address := fmt.Sprintf("%s:%d", ip, raftPort)
	r, err := common.SetupRaft(dir, id, address, replicaId == 0, fsm)
	if err != nil {
		return err
	}
	k.raft = r
	return nil
}

func (k *KVStore) Join(ip string, raftPort, nodeId, replicaId int) error {
	id := fmt.Sprintf("node-%d-replica-%d.node-%d.kvs.svc.localho.st:%d", nodeId, replicaId, nodeId, raftPort)
	address := fmt.Sprintf("%s:%d", ip, raftPort)

	configFuture := k.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return fmt.Errorf("Failed to get raft configuration: %s", err)
	}
	for _, member := range configFuture.Configuration().Servers {
		if member.ID == raft.ServerID(id) || member.Address == raft.ServerAddress(address) {
			log.Printf("Removing existing node (%s, %s) which is different from (%s, %s)", member.ID, member.Address, id, address)
			if err := k.raft.RemoveServer(member.ID, 0, 0).Error(); err != nil {
				return fmt.Errorf("Failed to remove existing node %s: %s", id, err)
			}
		}
	}
	return common.JoinNode(k.raft, id, address)
}

func (k *KVStore) Leader() (raft.ServerAddress, raft.ServerID) {
	return k.raft.LeaderWithID()
}

type fsmSnapshot struct {
	store map[string]string
}

func (fs *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	storeBytes, err := proto.Marshal(&pb.KVFSMSnapshot{
		KvStore: fs.store,
	})
	if err != nil {
		sink.Cancel()
		return err
	}
	if _, err := sink.Write(storeBytes); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (fs *fsmSnapshot) Release() {}

type KVFsm KVStore

func NewKVFsm(store *map[string]string) *KVFsm {
	return &KVFsm{
		store: store,
	}
}

func (f *KVFsm) Apply(raftLog *raft.Log) interface{} {
	entry := &pb.KVFSMLogEntry{}
	if err := proto.Unmarshal(raftLog.Data, entry); err != nil {
		return err
	}
	log.Printf("Applying log entry: %v", entry)
	f.mutex.Lock()
	defer f.mutex.Unlock()
	switch entry.Operation {
	case SET:
		(*f.store)[entry.Key] = *entry.Value
		log.Printf("Key: %s, Value: %s", entry.Key, (*f.store)[entry.Key])
	case DELETE:
		delete(*f.store, entry.Key)
	}
	return nil
}

func (f *KVFsm) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{store: *f.store}, nil
}

func (f *KVFsm) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()
	snapshotData, err := io.ReadAll(snapshot)
	if err != nil {
		return err
	}
	snapshotObj := &pb.KVFSMSnapshot{}
	if err := proto.Unmarshal(snapshotData, snapshotObj); err != nil {
		return err
	}
	f.store = &snapshotObj.KvStore
	return nil
}