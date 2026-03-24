package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fresp/StatusForge/configs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var sqliteDB *sql.DB

type EngineStatus struct {
	Engine           string `json:"engine"`
	SetupDone        bool   `json:"setupDone"`
	MongoConnected   bool   `json:"mongoConnected"`
	SQLiteConnected  bool   `json:"sqliteConnected"`
	RuntimeSupported bool   `json:"runtimeSupported"`
}

func Initialize(cfg *configs.Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.DBEngine)) {
	case "":
		return nil
	case "mongodb":
		if cfg.MongoURI == "" || cfg.MongoDBName == "" {
			return errors.New("mongodb configuration is incomplete")
		}
		if err := ConnectMongo(cfg.MongoURI, cfg.MongoDBName); err != nil {
			return err
		}
		return nil
	case "sqlite":
		db, err := ConnectSQLite(cfg.SQLitePath)
		if err != nil {
			return err
		}
		sqliteDB = db
		return nil
	default:
		return fmt.Errorf("unsupported DB_ENGINE: %s", cfg.DBEngine)
	}
}

func GetSQLiteDB() *sql.DB {
	return sqliteDB
}

func ValidateMongoConnection(uri, dbName string) error {
	if strings.TrimSpace(uri) == "" || strings.TrimSpace(dbName) == "" {
		return errors.New("mongo uri and db name are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)

	if err := client.Database(dbName).Client().Ping(ctx, nil); err != nil {
		return err
	}

	return nil
}

func ValidateSQLitePath(path string) error {
	_, err := ConnectSQLite(path)
	if err != nil {
		return err
	}
	if sqliteDB != nil {
		_ = sqliteDB.Close()
		sqliteDB = nil
	}
	return nil
}

func BuildStatus(cfg *configs.Config) EngineStatus {
	engine := strings.ToLower(strings.TrimSpace(cfg.DBEngine))
	status := EngineStatus{
		Engine:          engine,
		SetupDone:       cfg.SetupDone,
		MongoConnected:  GetDB() != nil,
		SQLiteConnected: GetSQLiteDB() != nil,
	}

	status.RuntimeSupported = status.MongoConnected || status.SQLiteConnected
	return status
}

func ensureSQLiteParent(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("sqlite path is required")
	}

	dir := filepath.Dir(cleanPath)
	dir = strings.TrimSpace(dir)

	// 🔥 HARDEN FIX
	if dir == "." || dir == "./" || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
