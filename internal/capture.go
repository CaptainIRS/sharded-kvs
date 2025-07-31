package capture

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/hashicorp/go-msgpack/v2/codec"
	"github.com/hashicorp/raft"
	"google.golang.org/protobuf/proto"
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

func getHostFromIp(ip string) string {
	host, err := net.LookupAddr(ip)
	if err != nil {
		return ip
	}
	return host[0]
}

func replaceIpsWithHostnames(s string) string {
	return ipRegex.ReplaceAllStringFunc(s, func(ip string) string {
		if hostname, ok := hostnameCache[ip]; ok {
			return hostname
		}
		hostname := getHostFromIp(ip)
		if strings.HasSuffix(hostname, ".kvs.svc.localho.st.") {
			hostname = strings.Replace(hostname, ".kvs.svc.localho.st.", "", 1)
		}
		hostnameCache[ip] = hostname
		return hostname
	})
}

var hostnameCache = map[string]string{}
var ipRegex = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

func appendEntriesToString(r raft.AppendEntriesRequest) string {
	logString := "  Term: " + fmt.Sprintf("%d\n", r.Term)
	logString += "  Leader: " + fmt.Sprintf("%s\n", replaceIpsWithHostnames(string(r.Leader)))
	logString += "  PrevLogEntry: " + fmt.Sprintf("%d\n", r.PrevLogEntry)
	logString += "  PrevLogTerm: " + fmt.Sprintf("%d\n", r.PrevLogTerm)
	logString += "  LeaderCommitIndex: " + fmt.Sprintf("%d\n", r.LeaderCommitIndex)
	logString += "  Entries:\n"
	for i, entry := range r.Entries {
		logString += "    Entry " + fmt.Sprintf("%d:\n", i)
		logString += "      Index: " + fmt.Sprintf("%d\n", entry.Index)
		logString += "      Term: " + fmt.Sprintf("%d\n", entry.Term)
		logString += "      Type: " + fmt.Sprintf("%s\n", entry.Type.String())
		switch entry.Type {
		case raft.LogCommand:
			kvLogEntry := &pb.KVFSMLogEntry{}
			if err := proto.Unmarshal(entry.Data, kvLogEntry); err != nil {
				fmt.Printf("Failed to unmarshal KVFSMLogEntry: %v", err)
			}
			logString += "      Data: " + fmt.Sprintf("%s\n", kvLogEntry.String())
		case raft.LogConfiguration:
			config := raft.DecodeConfiguration(entry.Data)
			logString += "      Data:\n"
			for i, server := range config.Servers {
				logString += "        Server " + fmt.Sprintf("%d:\n", i)
				logString += "          ID: " + fmt.Sprintf("%s\n", server.ID)
				logString += "          Address: " + fmt.Sprintf("%s\n", string(server.Address))
				logString += "          Suffrage: " + fmt.Sprintf("%s\n", server.Suffrage.String())
			}
		}
	}
	return logString
}

func requestVoteToString(r raft.RequestVoteRequest) string {
	logString := "  Term: " + fmt.Sprintf("%d\n", r.Term)
	logString += "  Candidate: " + fmt.Sprintf("%s\n", replaceIpsWithHostnames(string(r.Candidate)))
	logString += "  LastLogIndex: " + fmt.Sprintf("%d\n", r.LastLogIndex)
	logString += "  LastLogTerm: " + fmt.Sprintf("%d\n", r.LastLogTerm)
	logString += "  LeadershipTransfer: " + fmt.Sprintf("%t\n", r.LeadershipTransfer)
	return logString
}

func installSnapshotToString(r raft.InstallSnapshotRequest) string {
	config := raft.DecodeConfiguration(r.Configuration)
	logString := "  Term: " + fmt.Sprintf("%d\n", r.Term)
	logString += "  Leader: " + fmt.Sprintf("%s\n", replaceIpsWithHostnames(string(r.Leader)))
	logString += "  SnapshotVersion: " + fmt.Sprintf("%d\n", r.SnapshotVersion)
	logString += "  LastLogIndex: " + fmt.Sprintf("%d\n", r.LastLogIndex)
	logString += "  LastLogTerm: " + fmt.Sprintf("%d\n", r.LastLogTerm)
	logString += "  Configuration: " + fmt.Sprintf("%v\n", config)
	logString += "  ConfigurationIndex: " + fmt.Sprintf("%d\n", r.ConfigurationIndex)
	logString += "  Size: " + fmt.Sprintf("%d\n", r.Size)
	return logString
}

func (t *tcpStream) run() {
	r := bufio.NewReader(&t.r)
	for {
		rpcType, err := r.ReadByte()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Printf("Failed to read RPC type: %v", err)
			continue
		}
		dec := codec.NewDecoder(r, &codec.MsgpackHandle{})
		switch rpcType {
		case rpcAppendEntries:
			var req raft.AppendEntriesRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode AppendEntriesRequest: %v", err)
				continue
			}
			if len(req.Entries) == 0 {
				log.Printf("Sending heartbeat to %s", replaceIpsWithHostnames(t.net.Dst().String()))
			} else {
				log.Printf("Sending AppendEntriesRequest to %s:\n%s", replaceIpsWithHostnames(t.net.Dst().String()), appendEntriesToString(req))
			}
		case rpcRequestVote:
			var req raft.RequestVoteRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode RequestVoteRequest: %v", err)
				continue
			}
			log.Printf("Sending RequestVoteRequest to %s:\n%s", replaceIpsWithHostnames(t.net.Dst().String()), requestVoteToString(req))
		case rpcInstallSnapshot:
			var req raft.InstallSnapshotRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode InstallSnapshotRequest: %v", err)
				continue
			}
			log.Printf("Sending InstallSnapshotRequest to %s:\n%s", replaceIpsWithHostnames(t.net.Dst().String()), installSnapshotToString(req))
		case rpcTimeoutNow:
			var req raft.TimeoutNowRequest
			if err := dec.Decode(&req); err != nil {
				log.Printf("Failed to decode TimeoutNowRequest: %v", err)
				continue
			}
			log.Printf("Sending TimeoutNowRequest %s", replaceIpsWithHostnames(t.net.Dst().String()))
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
		// if err := sourceHandle.SetBPFFilter(fmt.Sprintf("(dst host %s and tcp src port %s) or (src host %s and tcp dst port %s)", srcIp, dstPort, srcIp, dstPort)); err != nil {
		log.Fatal("Failed to set BPF filter")
		return
	}

	streamFactory := &tcpStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	packetSource := gopacket.NewPacketSource(sourceHandle, sourceHandle.LinkType())
	packetChan := packetSource.Packets()

	ticker := time.NewTicker(time.Minute)
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
		case <-ticker.C:
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		}
	}
}
