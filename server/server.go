package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	pb "todo/todolist"

	"github.com/golang/protobuf/ptypes"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port = ":8080"
)

var db *mongo.Client
var tododb *mongo.Collection
var mongoCtx context.Context

type ToDoServiceServer struct {
	pb.UnimplementedToDoServiceServer
}

type TodoItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	title       string             `bson:"title"`
	description string             `bson:"description"`
	insert_at   time.Time          `bson:"insertat"`
	update_at   time.Time          `bson:"updateat"`
}

func (s *ToDoServiceServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	api := req.GetApi()
	if api != "1" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid api verison: %v", api))
	}

	data := req.GetToDo()

	insertAt, err := ptypes.Timestamp(data.GetInsertAt())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid insert time: %v", err))
	}
	updateAt, err := ptypes.Timestamp(data.GetUpdateAt())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update time: %v", err))
	}
	toDoItem := TodoItem{
		title:       data.GetTitle(),
		description: data.GetDescription(),
		insert_at:   insertAt,
		update_at:   updateAt,
	}
	result, err := tododb.InsertOne(ctx, toDoItem)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
	resultID := result.InsertedID.(primitive.ObjectID)

	return &pb.CreateResponse{Api: api, Id: resultID.Hex()}, nil

}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Printf("starting server at port: %v", port)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("can not listen on port: %v", err)
	}

	opt := []grpc.ServerOption{}
	s := grpc.NewServer(opt...)
	pb.RegisterToDoServiceServer(s, &ToDoServiceServer{})

	//init mongodb client
	fmt.Println("Connecting to MongoDB ...")

	mongoCtx = context.Background()
	db, err = mongo.Connect(mongoCtx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Cannot connect MongoDB: %v", err)
	}

	tododb = db.Database("mydb").Collection("Todo")

	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Fail to serve: %v", err)
		}
	}()

	fmt.Println("Server succesfully started on port", port)

	c := make(chan os.Signal)

	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("\nStopping the server...")
	s.Stop()
	listener.Close()
	fmt.Println("Closing MongoDB connection")
	db.Disconnect(mongoCtx)
	fmt.Println("Done.")
}
