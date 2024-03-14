package main

import (
	clientGRPC "internal/grpc/client"
	nodesGRPC "internal/grpc/nodes"
	"time"

	"google.golang.org/grpc"
)

type FileServer struct {
	clientGRPC.UnimplementedFileServerServer
}

type HeartbeatTrackerServer struct {
	nodesGRPC.UnimplementedHeartbeatTrackerServer
}

type NodeRegistrarServer struct {
	nodesGRPC.UnimplementedNodeRegistrarServer
}

type FileTrackerServer struct {
	nodesGRPC.UnimplementedFileTrackerServer
}

type DataKeeperNode struct {
	// The IP Address of the node
	ipAddress string

	// Last heartbeat time
	lastHeartbeatTime time.Time

	// Port for client communications
	clientPort int32

	// Port for inter-cluster communications
	clusterPort int32

	// Free space available at that node
	freeSpace int64

	// Inter-cluster GRPC connection
	conn *grpc.ClientConn
}

type File struct {

	// Nodes where the file resides.
	// Key: node UUID
	// Value: path on the node
	Nodes map[string]string

	// Size of the file
	Size int64
}
