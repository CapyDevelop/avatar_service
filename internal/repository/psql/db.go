package psql

import (
	"database/sql"
	"fmt"
	"github.com/CapyDevelop/avatar_service/internal/config"
)

const defaultAvatar string = "https://capyavatars.storage.yandexcloud.net/avatar/default/default.webp"

type Postgres interface {
	InsertAvatar(uuid, filename string) error
	GetLastAvatar(uuid string) (string, error)
}

type postgres struct {
	db *sql.DB
}

func (p *postgres) InsertAvatar(uuid, filename string) error {
	sqlQuery := fmt.Sprintf("INSERT INTO avatar (uuid, filename) VALUES ($1, $2)")
	fmt.Println(sqlQuery)
	_, err := p.db.Exec(sqlQuery, uuid, filename)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (p *postgres) GetLastAvatar(uuid string) (string, error) {
	var filename string
	err := p.db.QueryRow("SELECT filename FROM avatar WHERE uuid=$1 ORDER BY id DESC LIMIT 1", uuid).Scan(&filename)
	if err != nil {
		fmt.Println(err)
		return defaultAvatar, nil
	}
	return filename, nil
}

func NewPostgres(cfg *config.Config) (Postgres, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Hostname, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Can not open db connection", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("Can not ping db")
		return nil, err
	}

	fmt.Println("Successfully connection to DB")
	return &postgres{db: db}, nil
}
