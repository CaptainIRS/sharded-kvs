package capture

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/hashicorp/go-msgpack/v2/codec"
	"github.com/hashicorp/raft"
)

const (
	rpcAppendEntries uint8 = iota
	rpcRequestVote
	rpcInstallSnapshot
	rpcTimeoutNow
)

type tcpStreamFactory struct{}

type tcpStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
}

func (t *tcpStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	tstream := &tcpStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
	}
	go tstream.run()
	return &tstream.r
}

func (t *tcpStream) run() {
	r := bufio.NewReader(&t.r)
	for {
		rpcType, err := r.ReadByte()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Printf("Failed to read RPC type: %v", err)
			return
		}
		dec := codec.NewDecoder(r, &codec.MsgpackHandle{})
		switch rpcType {
		case rpcAppendEntries:
			var req raft.AppendEntriesRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode AppendEntriesRequest: %v", err)
				return
			}
			if len(req.Entries) == 0 {
				log.Printf("Sending heartbeat to %s", t.net.Dst())
			} else {
				log.Printf("Sending AppendEntriesRequest %+v to %s", req, t.net.Dst())
			}
		case rpcRequestVote:
			var req raft.RequestVoteRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode RequestVoteRequest: %v", err)
				return
			}
			log.Printf("Sending RequestVoteRequest %+v to %s", req, t.net.Dst())
		case rpcInstallSnapshot:
			var req raft.InstallSnapshotRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode InstallSnapshotRequest: %v", err)
				return
			}

			log.Printf("Sending InstallSnapshotRequest %+v to %s", req, t.net.Dst())
		case rpcTimeoutNow:
			var req raft.TimeoutNowRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode TimeoutNowRequest: %v", err)
				return
			}
			log.Printf("Sending TimeoutNowRequest %+v to %s", req, t.net.Dst())
		default:
			log.Printf("Unknown RPC type: %d", rpcType)
		}
	}
}

func RunPacketCapture(ctx context.Context, srcIp, dstPort string) {
	sourceHandle, err := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatal("Failed to create packet capture handle")
		return
	}
	defer sourceHandle.Close()

	if err := sourceHandle.SetBPFFilter(fmt.Sprintf("src host %s and tcp dst port %s", srcIp, dstPort)); err != nil {
		log.Fatal("Failed to set BPF filter")
		return
	}

	streamFactory := &tcpStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	packetSource := gopacket.NewPacketSource(sourceHandle, sourceHandle.LinkType())
	packetChan := packetSource.Packets()

	ticker := time.Tick(time.Minute)
	for {
		select {
		case <-ctx.Done():
			log.Print("Packet capture context cancelled, exiting")
			return
		case p := <-packetChan:
			if tcpLayer := p.Layer(layers.LayerTypeTCP); tcpLayer != nil {
				tcp, _ := tcpLayer.(*layers.TCP)
				assembler.AssembleWithTimestamp(p.NetworkLayer().NetworkFlow(), tcp, time.Now())
			}
		case <-ticker:
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		}
	}
}
