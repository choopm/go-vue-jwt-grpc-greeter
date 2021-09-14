package greeter

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/choopm/go-vue-jwt-grpc-greeter/api/services/greeterservice"
	"github.com/choopm/go-vue-jwt-grpc-greeter/pkg/jwthelper"
	"gitlab.0pointer.org/choopm/grpchelpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GreeterService struct {
	greeterservice.UnimplementedGreeterServiceServer
	jwtSecret string
}

// New returns a Service to be registered at a grpc.Server
func New(jwtSecret string) *GreeterService {
	return &GreeterService{jwtSecret: jwtSecret}
}

// Start embedded grpc server and register ourself
func (s *GreeterService) Start(bindAddress, jwtSecret string, certFile string, keyFile string) {
	lis, err := net.Listen("tcp", bindAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	ser, err := grpchelpers.NewServer(jwtSecret, certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	greeterservice.RegisterGreeterServiceServer(ser, s)
	log.Printf("server listening at %v", lis.Addr())
	if err := ser.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *GreeterService) Hello(ctx context.Context, in *greeterservice.HelloRequest) (*greeterservice.HelloResponse, error) {
	claims := jwthelper.GetClaimsFromContext(s.jwtSecret, ctx)
	if claims == nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing jwt token, can't identify")
	}

	time.Sleep(1 * time.Second)
	if claims.Username == in.GetName() {
		return &greeterservice.HelloResponse{Greeting: "Hello " + in.GetName()}, nil
	} else {
		return &greeterservice.HelloResponse{Greeting: "Hello " + in.GetName() + ", real name: " + claims.Username}, nil
	}
}
