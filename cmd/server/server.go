package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/golang/protobuf/ptypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "todo/gen/proto"
)

const (
	port = ":8080"
)

var (
	db       *mongo.Client
	tododb   *mongo.Collection
	mongoCtx context.Context
)

type ToDoServiceServer struct {
	pb.UnimplementedToDoServiceServer
}

type TodoItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title,omitempty"`
	Description string             `bson:"description,omitempty"`
	Insert_at   time.Time          `bson:"insertat,omitempty"`
	Update_at   time.Time          `bson:"updateat,omitempty"`
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
	log.Println(data.GetTitle())
	toDoItem := TodoItem{
		Title:       data.GetTitle(),
		Description: data.GetDescription(),
		Insert_at:   insertAt,
		Update_at:   updateAt,
	}

	result, err := tododb.InsertOne(mongoCtx, toDoItem)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
	resultID := result.InsertedID.(primitive.ObjectID)

	return &pb.CreateResponse{Api: api, Id: resultID.Hex()}, nil
}

func (s *ToDoServiceServer) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	api := req.GetApi()
	if api != "1" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid api verison: %v", api))
	}
	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert object ID from the read request: %v", err))
	}

	data := tododb.FindOne(ctx, bson.M{"_id": objID})
	result := TodoItem{}
	if err := data.Decode(&result); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find todo item with id %s: %v", req.GetId(), err))
	}

	respone := &pb.ReadResponse{
		Api: api,
		ToDo: &pb.ToDo{
			Id:          objID.Hex(),
			Title:       result.Title,
			Description: result.Description,
			InsertAt:    timestamppb.New(result.Insert_at),
			UpdateAt:    timestamppb.New(result.Update_at),
		},
	}
	return respone, nil
}

func (s *ToDoServiceServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	api := req.GetApi()
	if api != "1" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid api verison: %v", api))
	}
	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert object ID from the read request: %v", err))
	}
	_, err = tododb.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find/delete todo item with id %s: %v", req.GetId(), err))
	}
	return &pb.DeleteResponse{Api: req.GetApi(), Deleted: fmt.Sprintf("Successfully deleted %v", req.GetId())}, nil
}

func (s *ToDoServiceServer) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	api := req.GetApi()
	if api != "1" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid api verison: %v", api))
	}
	objID, err := primitive.ObjectIDFromHex(req.ToDo.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert object ID from the read request: %v", err))
	}

	insertAt, err := ptypes.Timestamp(req.ToDo.GetInsertAt())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid insert time: %v", err))
	}
	updateAt, err := ptypes.Timestamp(req.ToDo.GetUpdateAt())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update time: %v", err))
	}

	update := TodoItem{
		Title:       req.ToDo.GetTitle(),
		Description: req.ToDo.GetDescription(),
		Insert_at:   insertAt,
		Update_at:   updateAt,
	}
	_, err = tododb.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update time: %v", err))
	}
	log.Println(objID)
	return &pb.UpdateResponse{Api: req.GetApi(), Updated: fmt.Sprintf("Successfully updated %v", req.ToDo.GetId())}, nil
}

func (s *ToDoServiceServer) ReadAll(req *pb.ReadAllRequest, stream pb.ToDoService_ReadAllServer) error {
	data := TodoItem{}
	cursor, err := tododb.Find(context.Background(), bson.M{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&data)
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data to TodoItem: %v", err))
		}
		if err := stream.Send(&pb.ReadAllResponse{
			ToDo: &pb.ToDo{
				Id:          data.ID.Hex(),
				Title:       data.Title,
				Description: data.Description,
				InsertAt:    timestamppb.New(data.Insert_at),
				UpdateAt:    timestamppb.New(data.Update_at),
			},
		}); err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not send data: %v", err))
		}
	}
	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown cursor error: %v", err))
	}
	return nil
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

	// init mongodb client
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
