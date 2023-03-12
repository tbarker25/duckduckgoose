package main

import (
	"context"
	"fmt"
	"time"

	"github.com/shaj13/raft"
	duckduckgoose_pb "github.com/tbarker25/duckduckgoose/gen/go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type server struct {
	duckduckgoose_pb.UnimplementedDuckDuckGooseServer
}

func (s *server) GetRole(ctx context.Context, in *emptypb.Empty) (*wrapperspb.StringValue, error) {
	if node.Whoami() == node.Leader() {
		return &wrapperspb.StringValue{Value: "Goose"}, nil
	}

	return &wrapperspb.StringValue{Value: "Duck"}, nil
}

func (s *server) GetNode(ctx context.Context, in *duckduckgoose_pb.GetNodeRequest) (*duckduckgoose_pb.Node, error) {
	var nodeID uint64
	if _, err := fmt.Sscanf(in.Name, "nodes/%d", &nodeID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	member, ok := node.GetMemebr(nodeID)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Cannot find node with ID %q", nodeID)
	}

	return raftMemberToApiNode(member), nil
}

func (s *server) ListNodes(ctx context.Context, in *emptypb.Empty) (*duckduckgoose_pb.ListNodesResponse, error) {
	var resp duckduckgoose_pb.ListNodesResponse
	for _, m := range node.Members() {
		resp.Nodes = append(resp.Nodes, raftMemberToApiNode(m))
	}
	return &resp, nil
}

func (s *server) DeleteNode(ctx context.Context, in *duckduckgoose_pb.DeleteNodeRequest) (*emptypb.Empty, error) {
	var nodeID uint64
	if _, err := fmt.Sscanf(in.Name, "nodes/%d", &nodeID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if err := node.RemoveMember(ctx, nodeID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func raftMemberToApiNode(m raft.Member) *duckduckgoose_pb.Node {
	role := "Duck"
	if node.Leader() == m.ID() {
		role = "Goose"
	}
	return &duckduckgoose_pb.Node{
		Id:          m.ID(),
		Address:     m.Address(),
		ActiveSince: timestamppb.New(m.ActiveSince()),
		IsActive:    m.IsActive(),
		Role:        role,
	}
}
