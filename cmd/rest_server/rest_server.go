package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	pb "todo/gen/proto"

	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func runHttpServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterToDoServiceHandlerFromEndpoint(ctx, mux, "localhost:8080", opts)
	if err != nil {
		log.Fatalf("Can not register mux server: %v", err)
	}

	return http.ListenAndServe(":8081", mux)
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := runHttpServer(); err != nil {
		glog.Fatal(err)
	}
}
