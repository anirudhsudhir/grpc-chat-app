package main

import (
	"bufio"
	ctx "context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pb "github.com/anirudhsudhir/grpc-chat-app/grpc_api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatClient struct {
	pb.UnimplementedChatClientServer
	gRPCLocalPort int
	serverAddr    string
	gRPCClient    pb.ChatServerClient
	username      string
	ctx           ctx.Context
	cancelFunc    ctx.CancelCauseFunc
}

func main() {
	chatClient := ChatClient{}
	chatClient.ctx, chatClient.cancelFunc = ctx.WithCancelCause(ctx.Background())
	flag.IntVar(&chatClient.gRPCLocalPort, "gRPCLocalPort", 8081, "port used by local gRPC server")
	flag.StringVar(&chatClient.serverAddr, "serverAddr", "localhost:8080", "server address")

	go chatClient.startServer()

	conn, err := grpc.NewClient(chatClient.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to start gRPC connection to server at addr = %s -> %+v", chatClient.serverAddr, err)
	}
	chatClient.gRPCClient = pb.NewChatServerClient(conn)

	go chatClient.startChatLoop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("Received SIGINT, exiting")
	case <-chatClient.ctx.Done():
		if err := chatClient.ctx.Err(); err != nil {
			log.Println(err)
			log.Println("exiting")
		}
	}
}

func (s *ChatClient) startServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.gRPCLocalPort))
	if err != nil {
		log.Fatalf("failed to listen on %d -> %+v", &s.gRPCLocalPort, err)
	}

	gRPCServer := grpc.NewServer()
	pb.RegisterChatClientServer(gRPCServer, s)

	// log.Printf("gRPC server serving over port %d", s.gRPCServerPort)
	if err := gRPCServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed to serve -> %+v", err)
	}
}

func (s *ChatClient) SendMessage(ctx ctx.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	log.Printf("Received a SendMessage RPC")
	return &pb.SendResponse{RequestReceived: true}, nil
}

func (c *ChatClient) startChatLoop() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Enter a username: ")
	uname, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("error while reading from Stdin, aborting -> %+v", err)
		c.cancelFunc(err)
		return
	}
	c.username = strings.Trim(uname, "\n")

	fmt.Printf("Send a message to start chatting!\n")
	for {
		fmt.Printf("%s: ", c.username)
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("error while reading from Stdin, aborting -> %+v", err)
			c.cancelFunc(err)
			return
		}
		msg = strings.Trim(msg, "\n")

		ctx, cancFunc := ctx.WithTimeout(c.ctx, 3*time.Second)
		resp, err := c.gRPCClient.BroadcastMessage(ctx, &pb.BroadcastRequest{Text: c.username + ": " + msg})
		cancFunc()

		if err != nil {
			log.Printf("error while performing BroadcastMessage RPC -> %+v", err)
		} else if !resp.RequestReceived {
			log.Printf("failed to send %q successfully\n", msg)
		}
	}
}
