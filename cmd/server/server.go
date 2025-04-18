package main

import (
	ctx "context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/anirudhsudhir/grpc-chat-app/grpc_api"
	"google.golang.org/grpc"
)

type ChatServer struct {
	pb.UnimplementedChatServerServer
	gRPCLocalPort int
}

func main() {
	chatServer := ChatServer{}
	flag.IntVar(&chatServer.gRPCLocalPort, "gRPCLocalPort", 8080, "port used by local gRPC server")

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

	log.Printf("gRPC server serving over port %d", s.gRPCLocalPort)
	if err := gRPCServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed to serve -> %+v", err)
	}
}

func (s *ChatServer) BroadcastMessage(ctx ctx.Context, req *pb.BroadcastRequest) (*pb.BroadcastResponse, error) {
	fmt.Printf("Received a BroadcastMessage RPC -> %q \n", req.Text)
	return &pb.BroadcastResponse{RequestReceived: true}, nil
}

func signalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Received SIGINT, exiting \n")
}
