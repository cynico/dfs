package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	nodesGRPC "internal/grpc/nodes" // Import the nodes generated package
	"internal/util"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const MASTER_ADDR = "localhost:8080"

var (
	homeDir     string
	clientPort  int64
	clusterPort int64
	id          string
)

type NodeReplicationServer struct {
	nodesGRPC.UnimplementedNodeReplicationServer
}

func (nr *NodeReplicationServer) ReplicateFile(ctx context.Context, irr *nodesGRPC.ReplicationRequest) (*nodesGRPC.ReplicationResponse, error) {

	var port int64
	lis := listenOnPort(&port)

	// Start the replication in another goroutine
	go func() {
		c, err := lis.Accept()
		if err != nil {
			return
		}
		util.ReceiveFile(fmt.Sprintf("%s/%s", homeDir, irr.GetFileName()), c)

		log.Printf("Successfuly stored replicated file at: %s\n", fmt.Sprintf("%s/%s", homeDir, irr.GetFileName()))
	}()

	return &nodesGRPC.ReplicationResponse{
		Port: int32(port),
	}, nil

}

// This is asynchronous replication
// It is only synchronous insofar the connection used for replication is established
func (nr *NodeReplicationServer) InitiateReplication(ctx context.Context, rr *nodesGRPC.InitiateReplicationRequest) (*emptypb.Empty, error) {

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", rr.GetDestIpv4Address(), rr.GetDestPort()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error while connecting to destination node: %s", err.Error())
	}

	res, err := nodesGRPC.NewNodeReplicationClient(conn).ReplicateFile(context.Background(), &nodesGRPC.ReplicationRequest{
		FileName: rr.GetFileName(),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting a destination port from the node to stream replicated file to")
	}

	streamingConn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", rr.GetDestIpv4Address(), res.GetPort()))
	if err != nil {
		return nil, fmt.Errorf("error establishing TCP streaming connection to the destination node")
	}

	filePath := fmt.Sprintf("%s/%s", homeDir, rr.GetFileName())
	fileSize, err := util.CheckFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting file %s: %s", filePath, err.Error())
	}

	go util.SendFile(filePath, fileSize, streamingConn)

	return &emptypb.Empty{}, nil
}

func getFreeSpace() (int64, error) {

	var stat syscall.Statfs_t

	err := syscall.Statfs(homeDir, &stat)
	if err != nil {
		return 0, err
	}

	return int64(stat.Bfree) * stat.Bsize, nil
}

func setupHomeDir() error {

	// Create our home directory where we will store the files
	if homeDir == "" {

		var err error
		homeDir, err = os.MkdirTemp("", "data-keeper-node")
		if err != nil {
			return fmt.Errorf("error creating our home directory at: %s", homeDir)
		}

		log.Printf("No home directory provided. Assuming clean start at new home directory: %s\n", homeDir)

		// Generating a unique id for ourselves and writing it to homeDir/uuid
		id = uuid.New().String()
		err = os.WriteFile(fmt.Sprintf("%s/uuid", homeDir), []byte(id), 0644)
		if err != nil {
			os.RemoveAll(homeDir)
			return fmt.Errorf("error writing our id at: %s/uuid", homeDir)
		}

	} else {

		s, err := os.Stat(homeDir)
		if err != nil {
			return err
		}
		if !s.IsDir() {
			return fmt.Errorf("provided home path is not a directory")
		}

		uuidFilePath := fmt.Sprintf("%s/uuid", homeDir)
		_, err = os.Stat(uuidFilePath)
		if err != nil {
			return err
		}

		// Reading the uuid file to get our uuid
		buf, err := os.ReadFile(uuidFilePath)
		if err != nil {
			return err
		}

		u, err := uuid.Parse(string(buf))
		if err != nil {
			return err
		}
		id = u.String()

		return nil
	}

	return nil
}

func listFiles() (map[string]*nodesGRPC.FileInfo, error) {

	entries, err := os.ReadDir(homeDir)
	if err != nil {
		return nil, err
	}

	files := make(map[string]*nodesGRPC.FileInfo, len(entries)-1)
	for _, e := range entries {
		if !e.IsDir() && e.Name() != "uuid" {

			fileSize, err := util.CheckFile(e.Name())
			if err != nil {
				continue
			}

			files[e.Name()] = &nodesGRPC.FileInfo{
				FileSize: fileSize,
				Path:     homeDir,
			}
		}
	}

	return files, nil
}

func handleUpload(c net.Conn, ft nodesGRPC.FileTrackerClient) {

	// Setting a read deadline
	// err = clientConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	// if err != nil {
	// 	log.Printf("error setting read deadline for the connection: %s\n", err.Error())
	// 	clientConn.Close()
	// 	continue
	// }

	defer c.Close()

	fileBasename, _ := util.ReceiveString(c)
	filePath := fmt.Sprintf("%s/%s", homeDir, fileBasename)

	fileSize, err := util.ReceiveFile(filePath, c)
	if err != nil {
		os.Remove(filePath)
	}

	freeSpace, err := getFreeSpace()
	if err != nil {
		os.Remove(filePath)
	}

	if _, err = ft.TrackFile(context.Background(), &nodesGRPC.FileOnNode{
		Uuid:     id,
		FileName: fileBasename,
		FileInfo: &nodesGRPC.FileInfo{
			Path:     homeDir,
			FileSize: fileSize,
		},
		FreeSpace: freeSpace,
	}); err != nil {
		os.Remove(filePath)
	}

	log.Printf("Successfuly stored file at: %s\n", filePath)
}

func handleDownload(c net.Conn) {

	defer c.Close()

	// Read file name
	fileName, err := util.ReceiveString(c)
	if err != nil {
		log.Fatalln(err.Error())
	}
	filePath := fmt.Sprintf("%s/%s", homeDir, fileName)

	// Get file size
	fileSize, err := util.CheckFile(filePath)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// Send file
	err = util.SendFile(filePath, fileSize, c)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func registerNode(conn *grpc.ClientConn) error {

	// Registering the node with the master
	files, err := listFiles()
	if err != nil {
		return err
	}

	// Getting the free space of the filesystem at which our current home directory resides
	freeSpace, err := getFreeSpace()
	if err != nil {
		return err
	}

	c := nodesGRPC.NewNodeRegistrarClient(conn)
	_, err = c.RegisterNode(context.Background(), &nodesGRPC.NodeRegistrationRequest{
		ClientPort:  int32(clientPort),
		ClusterPort: int32(clusterPort),
		FreeSpace:   freeSpace,
		Uuid:        id,
		Files:       files,
	})
	if err != nil {
		return err
	}

	return nil
}

func listenOnPort(port *int64) net.Listener {

	// Listening on all ipv4 interfaces.
	// Specifying the port = 0 lets it pick a port
	lis, err := net.Listen("tcp4", ":0")
	// Getting the port it picked
	*port, _ = strconv.ParseInt(strings.Split(lis.Addr().String(), ":")[1], 10, 0)
	if err != nil {
		log.Fatalf("failed to listen on %d: %s\n", *port, err.Error())
	}

	return lis
}

func main() {

	// Parsing flags
	flag.StringVar(&homeDir, "home", "", "The path to the home directory for this node")
	flag.Parse()

	err := setupHomeDir()
	if err != nil {
		log.Fatalf("error setting up home directory: %s\n", err)
	}

	// Connect to the master
	conn, err := grpc.Dial(MASTER_ADDR, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error dialing the master node: %s\n", err.Error())
	}
	defer conn.Close()

	// Listening on 2 ports: one for client communications,
	// the other for inter-cluster communications (such as replication)
	clientLis := listenOnPort(&clientPort)
	clusterLis := listenOnPort(&clusterPort)

	defer clientLis.Close()
	defer clusterLis.Close()

	// Creating our GRPC server to listen on for inter-cluster communication
	gs := grpc.NewServer()
	nodesGRPC.RegisterNodeReplicationServer(gs, &NodeReplicationServer{})
	go func() {
		if err := gs.Serve(clusterLis); err != nil {
			log.Fatalf("failed to serve: %s", err.Error())
		}
	}()

	// Registering the node with the master
	err = registerNode(conn)
	if err != nil {
		log.Fatalf("error registering node: %s", err.Error())
	}

	// Instantiating the client for the heartbeats
	h := nodesGRPC.NewHeartbeatTrackerClient(conn)

	// Creating a ticker that ticks every 1 second.
	// This triggers the go routine that sends a heartbeat to the master node
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			_, err = h.HeartbeatTrack(context.Background(), &nodesGRPC.Heartbeat{
				Uuid: id,
			})
			if err != nil {
				// If for whatever reason we get back this error, we need to re-register
				// the node with the master. Plausible reasons may include temporary network
				// failures/disconnect.
				if _, ok := err.(*util.NoSuchNodeExistsError); ok {
					err = registerNode(conn)
					if err != nil {
						log.Fatalf(err.Error())
					}
				} else {
					log.Fatalf("error sending heartbeat: %s\n", err.Error())
				}
			}
		}
	}()

	// Creating a file tracker client, used to communicate with the master node
	// to confirm file uploads
	fileTrackerClient := nodesGRPC.NewFileTrackerClient(conn)

	// Listening on the TCP socket for client connections used for file transfers
	for {
		clientConn, err := clientLis.Accept()
		if err != nil {
			log.Printf("error accepting connection: %s\n", err.Error())
			continue
		}

		// Read whether this is an upload or a download request
		var upload bool
		binary.Read(clientConn, binary.LittleEndian, &upload)

		if upload {
			go handleUpload(clientConn, fileTrackerClient)
		} else {
			go handleDownload(clientConn)
		}
	}
}
