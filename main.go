package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/shaj13/raft"
	"github.com/shaj13/raft/transport"
	"github.com/shaj13/raft/transport/raftgrpc"
	duckduckgoose_pb "github.com/tbarker25/duckduckgoose/gen/go/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/soheilhy/cmux"
)

var (
	raftAddrFlag         = flag.String("raft_addr", "", "raft server address")
	apiAddrFlag          = flag.String("api_addr", "", "api server address")
	stateDirFlag         = flag.String("state_dir", "", "raft state directory (WAL, Snapshots)")
	joinAddrFlag         = flag.String("join_addr", "", "join an existing cluster with address")
	bootstrapClusterFlag = flag.Bool("bootstrap_cluster", false, "bootstrap a new cluster")
)

var (
	node *raft.Node
)

func main() {
	flag.Parse()
	validateFlags()

	startRaftNode()
	startRaftGrpcServer()
	startApi()
}

func validateFlags() {
	switch {
	case *raftAddrFlag == "":
		log.Fatalln("'raft_addr' flag must be set")

	case *apiAddrFlag == "":
		log.Fatalln("'api_addr' flag must be set")

	case *stateDirFlag == "":
		log.Fatalln("'state_dir' flag must be set")

	case *joinAddrFlag == "" != *bootstrapClusterFlag:
		log.Fatalln("exactly one of 'bootstrap_cluster' or 'join_addr' flag must be set")
	}
}

func startRaftNode() {
	var (
		opts      []raft.Option
		startOpts []raft.StartOption
	)

	startOpts = append(startOpts, raft.WithAddress(*raftAddrFlag))
	opts = append(opts, raft.WithStateDIR(*stateDirFlag))
	if *bootstrapClusterFlag {
		startOpts = append(startOpts, raft.WithFallback(
			raft.WithInitCluster(),
			raft.WithRestart(),
		))
	} else {
		startOpts = append(startOpts, raft.WithFallback(
			raft.WithJoin(*joinAddrFlag, time.Second),
			raft.WithRestart(),
		))
	}

	node = raft.NewNode(noopStateMachine{}, transport.GRPC, opts...)

	go func() {
		err := node.Start(startOpts...)
		if err != nil && err != raft.ErrNodeStopped {
			log.Fatal(err)
		}
	}()
}

func startRaftGrpcServer() {
	raftgrpc.Register(
		raftgrpc.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))
	raftServer := grpc.NewServer()
	raftgrpc.RegisterHandler(raftServer, node.Handler())

	go func() {
		lis, err := net.Listen("tcp", *raftAddrFlag)
		if err != nil {
			log.Fatal(err)
		}

		err = raftServer.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func startApi() {
	// set up grpc server
	grpcServer := grpc.NewServer()
	duckduckgoose_pb.RegisterDuckDuckGooseServer(grpcServer, &server{})

	// set up http server that proxies to grpc server
	mux := runtime.NewServeMux()
	err := duckduckgoose_pb.RegisterDuckDuckGooseHandlerFromEndpoint(
		context.Background(), mux, *apiAddrFlag, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Fatal(err)
	}
	httpServer := http.Server{Handler: mux}

	// Setup multiplexing:
	// - http2 -> grpcServer
	// - http1 -> httpServer
	l, err := net.Listen("tcp", *apiAddrFlag)
	if err != nil {
		log.Fatal(err)
	}
	m := cmux.New(l)
	go httpServer.Serve(m.Match(cmux.HTTP1Fast()))
	go grpcServer.Serve(m.Match(cmux.HTTP2()))
	if err := m.Serve(); err != nil {
		log.Fatal(err)
	}
}

type noopStateMachine struct{}

func (noopStateMachine) Apply(data []byte) {}

func (noopStateMachine) Snapshot() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (noopStateMachine) Restore(r io.ReadCloser) error {
	return r.Close()
}
