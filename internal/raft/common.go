package raft

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type serverAddressProvider struct {
	raft.ServerAddressProvider
}

func (s serverAddressProvider) ServerAddr(id raft.ServerID) (raft.ServerAddress, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", string(id))
	if err != nil {
		return "", err
	}
	return raft.ServerAddress(tcpAddr.String()), nil
}

var raftTimeout = 10 * time.Second

func SetupRaft(dir, id, address string, shouldBootstrap bool, fsm raft.FSM) (*raft.Raft, error) {

	log.Printf("Creating Raft store")
	store, err := raftboltdb.NewBoltStore(path.Join(dir, "bolt"))
	if err != nil {
		return nil, fmt.Errorf("could not create bolt store: %s", err)
	}

	log.Printf("Creating Raft snapshot store")
	snapshots, err := raft.NewFileSnapshotStore(path.Join(dir, "snapshot"), 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not create snapshot store: %s", err)
	}

	log.Printf("Resolving address: %s", address)
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("could not resolve address: %s", err)
	}

	log.Printf("Creating Raft instance at %s", tcpAddr)

	transport, err := raft.NewTCPTransportWithConfig(
		address,
		tcpAddr,
		&raft.NetworkTransportConfig{
			MaxPool:               10,
			Timeout:               raftTimeout,
			ServerAddressProvider: serverAddressProvider{},
			Logger:                hclog.New(&hclog.LoggerOptions{DisableTime: true}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create tcp transport: %s", err)
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)
	config.LogLevel = "DEBUG"
	config.Logger = hclog.New(&hclog.LoggerOptions{DisableTime: true})
	config.HeartbeatTimeout = 5 * time.Second
	config.ElectionTimeout = 5 * time.Second
	config.LeaderLeaseTimeout = 1 * time.Second

	log.Printf("Creating Raft instance")
	r, err := raft.NewRaft(config, fsm, store, store, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("could not create raft instance: %s", err)
	}

	if shouldBootstrap {
		log.Printf("Bootstrapping Raft cluster")
		if err := r.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(id),
					Address: transport.LocalAddr(),
				},
			},
		}).Error(); err != nil {
			return nil, fmt.Errorf("could not bootstrap cluster: %s", err)
		}
	}

	log.Printf("Raft setup complete")

	return r, nil
}

func ShutdownRaft(r *raft.Raft) error {
	log.Printf("Shutting down Raft")
	if r.State() == raft.Leader {
		log.Printf("Demoting leader")
		if err := r.LeadershipTransfer().Error(); err != nil {
			return fmt.Errorf("could not demote leader: %s", err)
		}
		if err := r.Snapshot().Error(); err != nil {
			return fmt.Errorf("could not snapshot: %s", err)
		}
		log.Printf("Leader demoted")
	}
	return nil
}

func JoinReplica(r *raft.Raft, id, address string) error {
	if r.State() != raft.Leader {
		return nil
	}
	err := r.AddVoter(raft.ServerID(id), raft.ServerAddress(address), 0, 0).Error()
	if err != nil {
		return fmt.Errorf("could not add voter: %s", err)
	}
	log.Printf("Replica %s joined the cluster", id)
	return nil
}
