package main

import (
	"context"
	"fmt"
	"github.com/CapyDevelop/avatar_service/internal/config"
	"github.com/CapyDevelop/avatar_service/internal/repository/psql"
	pb_avatar "github.com/CapyDevelop/avatar_service_grpc/avatar_go"
	pb_storage "github.com/CapyDevelop/storage_service_grpc/storage_go"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
)

const avatarPath string = "https://capyavatars.storage.yandexcloud.net/avatar/%s/%s"
const defaultAvatar string = "https://capyavatars.storage.yandexcloud.net/avatar/default/default.webp"

type StorageServiceClient interface {
	Put(ctx context.Context, opts ...grpc.CallOption) (pb_storage.StorageService_PutClient, error)
}

type server struct {
	pb_avatar.UnimplementedAvatarServiceServer
	conn          *grpc.ClientConn
	db            psql.Postgres
	storageClient StorageServiceClient
}

func (s *server) SetAvatar(stream pb_avatar.AvatarService_SetAvatarServer) error {
	println("Here")
	var uuid string

	streamToServer, err := s.storageClient.Put(context.Background())
	if err != nil {
		return stream.SendAndClose(&pb_avatar.SetAvatarResponse{
			Status:      1,
			Description: "Cant open connection to Storage service",
		})
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stream.SendAndClose(&pb_avatar.SetAvatarResponse{
				Status:      1,
				Description: "Error while send data to storage service",
			})
		}

		uuid = req.Uuid

		if err := streamToServer.Send(&pb_storage.PutRequest{
			Data:     req.Data,
			Filename: req.Filename,
			Uuid:     req.Uuid,
		}); err != nil {
			fmt.Println("Error transmit")
		}
	}

	resp, err := streamToServer.CloseAndRecv()
	if err != nil {
		return stream.SendAndClose(&pb_avatar.SetAvatarResponse{
			Status:      1,
			Description: "Error receive answer from storage service",
		})
	}

	err = s.db.InsertAvatar(uuid, resp.Filename)
	if err != nil {
		return stream.SendAndClose(&pb_avatar.SetAvatarResponse{
			Status:      1,
			Description: "Error while send data to db",
		})
	}
	return stream.SendAndClose(&pb_avatar.SetAvatarResponse{
		Status:      resp.Status,
		Description: resp.Description,
		Avatar:      fmt.Sprintf(avatarPath, uuid, resp.Filename),
	})
}

func (s *server) GetAvatar(ctx context.Context, in *pb_avatar.GetAvatarRequest) (*pb_avatar.GetAvatarResponse, error) {
	filename, err := s.db.GetLastAvatar(in.Uuid)
	if err != nil {
		return &pb_avatar.GetAvatarResponse{
			Status:      1,
			Description: "Can't get Avatar. See log",
		}, nil
	}
	return &pb_avatar.GetAvatarResponse{
		Avatar:      fmt.Sprintf(avatarPath, in.Uuid, filename),
		Status:      0,
		Description: "",
	}, nil
}

func main() {
	cfg := config.MustLoad()
	fmt.Println(cfg)
	lis, err := net.Listen("tcp", ":1212")
	if err != nil {
		log.Fatalf("faild to listen")
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", cfg.Transport.Hostname, cfg.Transport.Port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Println("Error")
	}
	defer conn.Close()

	storageClient := pb_storage.NewStorageServiceClient(conn)

	s := grpc.NewServer()
	db, err := psql.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("Cannot connect to db, %v", err)
	}
	pb_avatar.RegisterAvatarServiceServer(s, &server{
		conn:          conn,
		db:            db,
		storageClient: storageClient,
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("cannot serve")
	}
}
