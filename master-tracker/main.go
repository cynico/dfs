package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	clientGRPC "internal/grpc/client" // Import the client generated package
	nodesGRPC "internal/grpc/nodes"   // Import the nodes generated package
	"internal/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	SERVER_PORT        = 8080
	REPLICATION_PERIOD = 10
	UNALIVE_PERIOD     = 5
)

var (
	// Key is nodes' UUID
	nodes map[string]*DataKeeperNode

	// Key is a file name
	// Value is a map:
	// key is the node's UUID where the file is present,
	// value is the path on that node, where the file is stored.
	files map[string]File

	// A list of files that recently failed to persist in the system
	// Used to notify the client with failure when it polls for file status
	// And garbage collected every period
	failedFiles []string
)

func (u *FileServer) Upload(ctx context.Context, ur *clientGRPC.UploadRequest) (*clientGRPC.UploadResponse, error) {

	// TODO: he order of elements in a map is non-deterministic, but there should be a better
	// guarantee of sufficient load-balancing.
	// Finding a node with available space and returning it
	requestedSpace := ur.GetFileSize()
	for _, node := range nodes {
		if node.freeSpace > requestedSpace {
			return &clientGRPC.UploadResponse{
				Ipv4Address: node.ipAddress,
				Port:        node.clientPort,
			}, nil
		}
	}

	return nil, fmt.Errorf("no available nodes with sufficient space")
}

func (u *FileServer) Download(ctx context.Context, dr *clientGRPC.DownloadRequest) (*clientGRPC.DownloadResponse, error) {

	file, ok := files[dr.GetFileName()]
	if !ok {
		return &clientGRPC.DownloadResponse{}, status.Errorf(codes.NotFound, "no such file exists")
	}

	nodesList := make([]*clientGRPC.Node, len(file.Nodes))
	for nodeUUID := range file.Nodes {
		node, ok := nodes[nodeUUID]
		if !ok {
			continue
		}
		nodesList = append(nodesList, &clientGRPC.Node{
			Ipv4Address: node.ipAddress,
			Port:        node.clientPort,
		})
	}

	return &clientGRPC.DownloadResponse{
		Nodes:    nodesList,
		FileSize: file.Size,
	}, nil
}

func (u *FileServer) List(context.Context, *emptypb.Empty) (*clientGRPC.ListResponse, error) {

	f := make([]string, len(files))
	for fileName := range files {
		f = append(f, fileName)
	}

	return &clientGRPC.ListResponse{
		Files: f,
	}, nil
}

func (u *FileServer) CheckIfExists(ctx context.Context, dr *clientGRPC.DownloadRequest) (*emptypb.Empty, error) {

	failed := false
	for i, failedFile := range failedFiles {
		if failedFile == dr.FileName {
			failed = true
			failedFiles = util.RemoveElementFromSlice(failedFiles, i)
		}
	}

	if failed {
		return nil, status.Errorf(codes.Internal, "node failed to persist file, try again")
	}

	if _, ok := files[dr.GetFileName()]; ok {
		return &emptypb.Empty{}, nil
	} else {
		return nil, status.Errorf(codes.NotFound, "requested file does not exist")
	}
}

func (n *NodeRegistrarServer) RegisterNode(ctx context.Context, nr *nodesGRPC.NodeRegistrationRequest) (*emptypb.Empty, error) {

	// Get the ip address of the node
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("error extracting peer info from the context")
	}

	uuid := nr.GetUuid()
	_, ok = nodes[uuid]
	if !ok {
		nodes[uuid] = &DataKeeperNode{}
	}

	nodes[uuid].ipAddress = strings.Split(peer.Addr.String(), ":")[0]
	nodes[uuid].clientPort = nr.GetClientPort()
	nodes[uuid].clusterPort = nr.GetClusterPort()
	nodes[uuid].freeSpace = nr.GetFreeSpace()
	nodes[uuid].lastHeartbeatTime = time.Now()

	if nr.GetFiles() != nil {
		for fileName, fileInfo := range nr.GetFiles() {

			_, ok = files[fileName]
			if !ok {
				files[fileName] = File{
					Nodes: make(map[string]string),
					Size:  fileInfo.GetFileSize(),
				}
			}
			files[fileName].Nodes[uuid] = fileInfo.GetPath()
		}
	}

	// Connecting to the node on the port used for inter-cluster communications
	nodeConn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodes[uuid].ipAddress, nodes[uuid].clusterPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	nodes[uuid].conn = nodeConn

	return &emptypb.Empty{}, nil
}

func (u *HeartbeatTrackerServer) HeartbeatTrack(ctx context.Context, h *nodesGRPC.Heartbeat) (*emptypb.Empty, error) {

	// Get the ip address of the node
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return &emptypb.Empty{}, fmt.Errorf("error extracting peer info from the context")
	}

	uuid := h.GetUuid()
	_, ok = nodes[uuid]
	if !ok {
		return &emptypb.Empty{}, &util.NoSuchNodeExistsError{UUID: uuid}
	}

	nodes[uuid].lastHeartbeatTime = time.Now()

	currentIpAddress := strings.Split(peer.Addr.String(), ":")[0]
	if currentIpAddress != nodes[uuid].ipAddress {
		nodes[uuid].ipAddress = currentIpAddress
	}

	log.Printf("Received heartbeat from node with id %s at: %s\n", uuid, peer.Addr.String())

	return &emptypb.Empty{}, nil
}

func (f *FileTrackerServer) TrackFile(ctx context.Context, fon *nodesGRPC.FileOnNode) (*emptypb.Empty, error) {

	if fon.GetFailed() {
		failedFiles = append(failedFiles, fon.FileName)
		return &emptypb.Empty{}, nil
	}

	_, ok := files[fon.GetFileName()]
	if !ok {
		files[fon.GetFileName()] = File{
			Nodes: make(map[string]string),
			Size:  fon.GetFileInfo().GetFileSize(),
		}
	}

	files[fon.GetFileName()].Nodes[fon.GetUuid()] = fon.GetFileInfo().GetPath()
	return &emptypb.Empty{}, nil
}

func replicate() {

	// Remove all dead nodes from the nodes map
	unaliveNodes()

	for fileName, file := range files {

		fileAliveNodes := make(map[string]interface{}, len(file.Nodes))
		for uuid := range file.Nodes {
			// If this node is alive, append it to the list
			if _, ok := nodes[uuid]; ok {
				fileAliveNodes[uuid] = struct{}{}
			} else {
				// If this node is not alive, remove it from the list of nodes
				// the file is stored on
				delete(files[fileName].Nodes, uuid)
			}
		}

		// If it is present on 3 or more alive nodes, skip it
		if len(fileAliveNodes) >= 3 {
			continue
		}

		// The number of nodes we need to replicate the file to
		// so that we have 3 replicas
		neededNodes := 3 - len(fileAliveNodes)

		// Get a random node to be the source node
		var sourceNode *DataKeeperNode
		for uuid := range fileAliveNodes {
			sourceNode = nodes[uuid]
			break
		}

		for uuid, node := range nodes {

			if _, ok := fileAliveNodes[uuid]; ok {
				continue
			}

			// If this node does not have enough space for the file, skip to the next
			if node.freeSpace < file.Size {
				continue
			}

			// TODO: SIGSEGV
			/*
				panic: runtime error: invalid memory address or nil pointer dereference
				[signal SIGSEGV: segmentation violation code=0x1 addr=0x38 pc=0x7f202a]

				goroutine 774 [running]:
				main.replicate()
				dfs/master-tracker/main.go:228 +0x40a
				created by main.main.func1 in goroutine 19
				dfs/master-tracker/main.go:286 +0x89
			*/
			_, err := nodesGRPC.NewNodeReplicationClient(sourceNode.conn).InitiateReplication(context.Background(), &nodesGRPC.InitiateReplicationRequest{
				FileName:        fileName,
				DestIpv4Address: node.ipAddress,
				DestPort:        node.clusterPort,
			})
			if err != nil {
				log.Println(err.Error())
				continue
			}

			// TODO: modify this to point to the actual file path
			file.Nodes[uuid] = ""

			neededNodes -= 1
			if neededNodes == 0 {
				break
			}
		}
	}
}

func unaliveNodes() {
	for uuid, node := range nodes {
		if time.Since(node.lastHeartbeatTime) > 2*time.Second {
			delete(nodes, uuid)
			node.conn.Close()
		}
	}
}

func main() {

	nodes = make(map[string]*DataKeeperNode)
	files = make(map[string]File)

	lis, err := net.Listen("tcp4", fmt.Sprintf(":%d", SERVER_PORT))
	if err != nil {
		log.Fatalf("failed to listen on %d: %s", SERVER_PORT, err.Error())
	}
	defer lis.Close()

	gs := grpc.NewServer()
	clientGRPC.RegisterFileServerServer(gs, &FileServer{})
	nodesGRPC.RegisterNodeRegistrarServer(gs, &NodeRegistrarServer{})
	nodesGRPC.RegisterHeartbeatTrackerServer(gs, &HeartbeatTrackerServer{})
	nodesGRPC.RegisterFileTrackerServer(gs, &FileTrackerServer{})

	// Creating a ticker that ticks every 10 second.
	// This triggers the go routine that sends a heartbeat to the master node
	replicationTicker := time.NewTicker(REPLICATION_PERIOD * time.Second)
	unalivingTicker := time.NewTicker(UNALIVE_PERIOD * time.Second)
	defer replicationTicker.Stop()
	defer unalivingTicker.Stop()

	go func() {
		for {
			select {
			case <-replicationTicker.C:
				// Garbage collecting the list of failed files
				failedFiles = []string{}
				go replicate()
			case <-unalivingTicker.C:
				go unaliveNodes()
			}
		}
	}()

	log.Printf("Listening on port %d", SERVER_PORT)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err.Error())
	}
}
