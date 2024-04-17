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

func SetupRaft(dir, id, address string, isLeader bool, fsm raft.FSM) (*raft.Raft, error) {

	log.Printf("Creating Raft store")
	store, err := raftboltdb.NewBoltStore(path.Join(dir, "bolt"))
	if err != nil {
		return nil, fmt.Errorf("Could not create bolt store: %s", err)
	}

	log.Printf("Creating Raft snapshot store")
	snapshots, err := raft.NewFileSnapshotStore(path.Join(dir, "snapshot"), 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("Could not create snapshot store: %s", err)
	}

	log.Printf("Resolving address: %s", address)
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("Could not resolve address: %s", err)
	}

	log.Printf("Creating Raft instance at %s", tcpAddr)

	transport, err := raft.NewTCPTransportWithConfig(
		address,
		tcpAddr,
		&raft.NetworkTransportConfig{
			MaxPool:               10,
			Timeout:               time.Second * 10,
			ServerAddressProvider: serverAddressProvider{},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("Could not create tcp transport: %s", err)
	}

	raftCfg := raft.DefaultConfig()
	raftCfg.LocalID = raft.ServerID(id)
	raftCfg.LogLevel = "DEBUG"
	raftCfg.Logger = hclog.New(&hclog.LoggerOptions{
		DisableTime: true,
	})

	log.Printf("Creating Raft instance")
	r, err := raft.NewRaft(raftCfg, fsm, store, store, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("Could not create raft instance: %s", err)
	}

	if isLeader {
		log.Printf("Bootstrapping Raft cluster")
		r.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(id),
					Address: transport.LocalAddr(),
				},
			},
		})
	}

	log.Printf("Raft setup complete")

	return r, nil
}

func JoinNode(r *raft.Raft, id, address string) error {
	err := r.AddVoter(raft.ServerID(id), raft.ServerAddress(address), 0, 0).Error()
	if err != nil {
		return fmt.Errorf("Could not add voter: %s", err)
	}
	log.Printf("Node %s joined the cluster", id)
	return nil
}
