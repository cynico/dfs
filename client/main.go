package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	// Import the client generated package
	clientGRPC "internal/grpc/client" // Import the nodes generated package
	"internal/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const MASTER_ADDR = "localhost:8080"

func connectToNode(upload bool, ipv4Address string, port int32) (net.Conn, error) {

	// Connecting to the the data keeper node
	nodeConn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", ipv4Address, port))
	if err != nil {
		return nil, err
	}
	// First, inform the node that we will be doing an upload operation.
	if err := binary.Write(nodeConn, binary.LittleEndian, upload); err != nil {
		return nil, err
	}

	return nodeConn, nil
}

func upload(filePath string, fsClient clientGRPC.FileServerClient) {

	fileSize, err := util.CheckFile(filePath)
	if err != nil {
		log.Fatalf("error checking the file: %s\n", err.Error())
	}

	res, err := fsClient.Upload(context.Background(), &clientGRPC.UploadRequest{
		FileSize: fileSize,
	})
	if err != nil {
		log.Fatalf("error sending upload request: %s", err.Error())
	}
	log.Printf("Got the following info: %s %d\n", res.GetIpv4Address(), res.GetPort())

	// Connecting to the the data keeper node
	nodeConn, err := connectToNode(true, res.GetIpv4Address(), res.GetPort())
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer nodeConn.Close()

	// Send the file name, then the file.
	s := strings.Split(filePath, "/")
	fileName := s[len(s)-1]

	util.SendString(fileName, nodeConn)
	util.SendFile(filePath, int64(fileSize), nodeConn)

	// Poll for confirmation from the master
	for {
		_, err = fsClient.CheckIfExists(context.Background(), &clientGRPC.DownloadRequest{FileName: fileName})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				time.Sleep(1 * time.Second)
				continue
			} else {
				log.Fatalf("error checking status of uploaded file: %s\n", err.Error())
			}
		}

		break
	}
}

func download(dirPath, fileName string, fsClient clientGRPC.FileServerClient) {

	res, err := fsClient.Download(context.Background(), &clientGRPC.DownloadRequest{
		FileName: fileName,
	})
	if err != nil {
		log.Fatalf("error getting nodes to download the file %s from: %s\n", fileName, err.Error())
	}

	nodes := res.GetNodes()
	downloaded := false
	log.Println("List of nodes to try: ", nodes)

	for _, node := range nodes {

		// Connecting to the the data keeper node
		nodeConn, err := connectToNode(false, node.Ipv4Address, node.Port)
		if err != nil {
			continue
		}
		defer nodeConn.Close()

		// Send the requested file name
		util.SendString(fileName, nodeConn)

		// Receive the file
		fullFilePath := fmt.Sprintf("%s/%s", dirPath, fileName)
		fileSize, err := util.ReceiveFile(fullFilePath, nodeConn)
		if err != nil {
			continue
		} else if fileSize != res.GetFileSize() {
			log.Printf("received partial file from node at %s:%d. received %d out of %d bytes\n",
				node.Ipv4Address, node.Port, fileSize, res.GetFileSize())
			continue
		}

		// Break from the loop after successful download
		downloaded = true
		break
	}

	if !downloaded {
		log.Fatalf("no node available to download the requested file from")
	}
}

func list(fsClient clientGRPC.FileServerClient) {

	res, err := fsClient.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("error getting file list: %s\n", err.Error())
	}

	log.Println("List of files: \n", res.GetFiles())
}

func main() {

	// Verifying supplied args
	var u, d, l bool
	var filePath, fileName string

	flag.BoolVar(&u, "upload", false, "A flag indicating the requested operation is file upload. Mutually-exclusive with other operations")
	flag.BoolVar(&d, "download", false, "A flag indicating the requested opertion is file download. Mutually-exclusive with other operations")
	flag.BoolVar(&l, "list", false, "A flag indicating the requested opertion is list files. Mutually-exclusive with other operations")
	flag.StringVar(&filePath, "path", "", "Path of the file to upload (mandatory), or path to download the file at (optional, defaults to the current directory)")
	flag.StringVar(&fileName, "file-name", "", "Name of the file to download. Must be present if -download is specified")
	flag.Parse()

	if (util.CountTrue(u, l, d) != 1) || (u && filePath == "") || (d && fileName == "") {
		flag.Usage()
		os.Exit(1)
	}

	// Connect to the master
	conn, err := grpc.Dial(MASTER_ADDR, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error dialing the master node: %s\n", err.Error())
	}
	defer conn.Close()

	// Instantiating the client for the uploader
	fsClient := clientGRPC.NewFileServerClient(conn)

	if u {
		upload(filePath, fsClient)
	} else if d {
		if filePath == "" {
			filePath = "."
		}
		download(filePath, fileName, fsClient)
	} else if l {
		list(fsClient)
	}
}
