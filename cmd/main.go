package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/CapyDevelop/avatar_service/internal/config"
	pb_avatar "github.com/CapyDevelop/avatar_service_grpc/avatar_go"
	pb_storage "github.com/CapyDevelop/storage_service_grpc/storage_go"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"strings"
)

const avatarPath string = "https://capyavatars.storage.yandexcloud.net/avatar/%s/%s"
const defaultAvatar string = "https://capyavatars.storage.yandexcloud.net/avatar/default/default.webp"

type StorageServiceClient interface {
	Put(ctx context.Context, opts ...grpc.CallOption) (pb_storage.StorageService_PutClient, error)
}

type server struct {
	pb_avatar.UnimplementedAvatarServiceServer
	conn          *grpc.ClientConn
	db            *sql.DB
	storageClient StorageServiceClient
}

func insertData(db *sql.DB, tableName string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	i := 1
	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	sqlQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	fmt.Println(sqlQuery)
	_, err := db.Exec(sqlQuery, values...)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func GetLastAvatar(db *sql.DB, uuid string) (string, error) {
	var filename string
	err := db.QueryRow("SELECT filename FROM avatar WHERE uuid=$1 ORDER BY id DESC LIMIT 1", uuid).Scan(&filename)
	if err != nil {
		fmt.Println(err)
		return defaultAvatar, nil
	}
	return filename, nil
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

	var data = map[string]interface{}{
		"uuid":     uuid,
		"filename": resp.Filename,
	}

	err = insertData(s.db, "avatar", data)
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
	filename, err := GetLastAvatar(s.db, in.Uuid)
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

func connectDB(cfg *config.Config) *sql.DB {
	fmt.Println("Try to connect to DB")
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Hostname, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Can not open db connection", err)
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("Can not ping db")
	}

	fmt.Println("Successfully connection to DB")
	return db
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
	db := connectDB(cfg)
	pb_avatar.RegisterAvatarServiceServer(s, &server{
		conn:          conn,
		db:            db,
		storageClient: storageClient,
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("cannot serve")
	}
}
