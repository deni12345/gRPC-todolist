package main

import (
	"context"
	"io"
	"log"
	"time"
	pb "todo/todolist"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

func ReadToDoItem(client pb.ToDoServiceClient, req *pb.ReadRequest) {
	log.Printf("Getting the todo item with id: %v", req.GetId())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	todoItem, err := client.Read(ctx, req)
	if err != nil {
		log.Fatalf("%v.Create(_) = _, %v: ", client, err)
	}
	log.Println(todoItem)
}

func ReadToDoItems(client pb.ToDoServiceClient, req *pb.ReadAllRequest) {
	log.Print("Getting all todo items")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ReadAll(ctx, req)
	if err != nil {
		log.Fatalf("%v.Create(_) = _, %v: ", client, err)
	}
	for {
		todoItem, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ReadAll(_) = _, %v", client, err)
		}
		log.Printf("Todo item: %v \n", todoItem.GetToDo())
	}
}

func createToDoItem(client pb.ToDoServiceClient, req *pb.CreateRequest) string {
	log.Printf("Creating the todo item with id: %v", req.ToDo.GetId())

	todoItem, err := client.Create(context.TODO(), req)
	if err != nil {
		log.Fatalf("%v.Create(_) = _, %v: ", client, err)
	}
	log.Printf("Blog created: %s\n", todoItem.GetId())
	return todoItem.GetId()
}

func updateToDoItem(client pb.ToDoServiceClient, req *pb.UpdateRequest) {
	log.Printf("Updating the todo item with id: %v", req.ToDo.GetId())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updatedItem, err := client.Update(ctx, req)
	if err != nil {
		log.Fatalf("%v.Update(_) = _, %v: ", client, err)
	}
	log.Print(updatedItem.GetUpdated())
}

func deleteToDoItem(client pb.ToDoServiceClient, req *pb.DeleteRequest) {
	log.Printf("Deleting the todo item with id: %v", req.GetId())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deletedItem, err := client.Delete(ctx, req)
	if err != nil {
		log.Fatalf("%v.Delete(_) = _, %v: ", client, err)
	}
	log.Print(deletedItem.GetDeleted())
}

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(":8080", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewToDoServiceClient(conn)

	t := time.Now().In(time.UTC)
	insert_at, _ := ptypes.TimestampProto(t)

	id := createToDoItem(client, &pb.CreateRequest{
		Api: "1",
		ToDo: &pb.ToDo{
			Title:       "demo task",
			Description: "this is just a demo",
			InsertAt:    insert_at,
			UpdateAt:    insert_at,
		},
	})

	ReadToDoItem(client, &pb.ReadRequest{
		Api: "1",
		Id:  id,
	})

	ReadToDoItems(client, &pb.ReadAllRequest{
		Api: "1",
	})

	updateToDoItem(client, &pb.UpdateRequest{
		Api: "1",
		ToDo: &pb.ToDo{
			Id:       id,
			Title:    "testing update",
			InsertAt: insert_at,
			UpdateAt: insert_at,
		},
	})

	// deleteToDoItem(client, &pb.DeleteRequest{
	// 	Api: "1",
	// 	Id:  id,
	// })
}
