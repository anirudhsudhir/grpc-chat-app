package main

import (
	ctx "context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pb "github.com/anirudhsudhir/grpc-chat-app/grpc-api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatServer struct {
	pb.UnimplementedChatServerServer
	gRPCLocalPort int
	clients       *sync.Map
}

type Client struct {
	addr       string
	gRPCClient pb.ChatClientClient
}

func main() {
	chatServer := ChatServer{}
	chatServer.clients = &sync.Map{}

	flag.IntVar(&chatServer.gRPCLocalPort, "gRPCLocalPort", 8080, "port used by local gRPC server")
	flag.Parse()

	go chatServer.startServer()

	signalHandler()
}

func (s *ChatServer) startServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.gRPCLocalPort))
	if err != nil {
		log.Fatalf("failed to listen on %d -> %+v", &s.gRPCLocalPort, err)
	}

	gRPCServer := grpc.NewServer()
	pb.RegisterChatServerServer(gRPCServer, s)

	fmt.Printf("gRPC server running at localhost:%d\n", s.gRPCLocalPort)
	if err := gRPCServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed to serve -> %+v", err)
	}
}

func (s *ChatServer) BroadcastMessage(ctx ctx.Context, req *pb.BroadcastRequest) (*pb.BroadcastResponse, error) {
	fmt.Printf("Received a BroadcastMessage RPC with message %q from user %q \n", req.Text, req.Username)

	broadcastFunc := func(username any, client any) bool {
		if username.(string) == req.Username {
			return true
		}

		_, err := client.(Client).gRPCClient.SendMessage(ctx, &pb.SendRequest{Text: req.Text, Username: req.Username})
		if err != nil {
			log.Printf("failed to forward message %q to client with username %q and address %q -> %+v", req.Text, username, client.(Client).addr, err)
		}
		return true
	}
	s.clients.Range(broadcastFunc)

	return &pb.BroadcastResponse{RequestReceived: true}, nil
}

func (s *ChatServer) RegisterClient(ctx ctx.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	conn, err := grpc.NewClient(req.ClientAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("failed to connect to gRPC server at address %q -> %+x", req.ClientAddr, err)
		return nil, err
	}

	client := Client{
		addr:       req.ClientAddr,
		gRPCClient: pb.NewChatClientClient(conn),
	}
	s.clients.Store(req.Username, client)
	fmt.Printf("New client registered with username: %q at address: %s\n", req.Username, req.ClientAddr)
	return &pb.RegisterResponse{Registered: true}, nil
}

func signalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Received SIGINT, exiting \n")
}
