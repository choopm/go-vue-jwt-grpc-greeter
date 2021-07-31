package greeter

import (
	"context"
	"log"
	"net"

	"gitlab.0pointer.org/choopm/greeter/api/services/greeterservice"
	"gitlab.0pointer.org/choopm/grpchelpers"
)

const (
	port = "127.0.0.1:50051"
)

type server struct {
	greeterservice.UnimplementedGreeterServiceServer
}

func (s *server) Hello(ctx context.Context, in *greeterservice.HelloRequest) (*greeterservice.HelloResponse, error) {
	log.Printf("Received: %v", in.GetName())
	return &greeterservice.HelloResponse{Greeting: "Hello " + in.GetName()}, nil
}

func StartServer(jwtSecret string, certFile string, keyFile string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s, err := grpchelpers.NewServer(jwtSecret, certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	greeterservice.RegisterGreeterServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
