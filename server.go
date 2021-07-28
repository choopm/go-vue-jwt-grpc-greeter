package greeter

import (
	"context"
	"log"
	"net"

	pb "gitlab.0pointer.org/choopm/greeter/api/services/greeterservice"

	"gitlab.0pointer.org/choopm/grpchelpers"
)

const (
	port = "127.0.0.1:50051"
)

type server struct {
	pb.UnimplementedGreeterServiceServer
}

func (s *server) Hello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloResponse{Greeting: "Hello " + in.GetName()}, nil
}

func Start(bearerTokenFile string, certFile string, keyFile string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s, err := grpchelpers.NewServer(bearerTokenFile, certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	pb.RegisterGreeterServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
