package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var RDB *redis.Client
var ctx = context.Background()

func Connect(cfg *config.Config) error {
	logrus.WithFields(logrus.Fields{
		"host": cfg.Redis.Host,
		"port": cfg.Redis.Port,
		"db": cfg.Redis.DB,
	}).Info("Connecting to Redis")

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	start := time.Now()
	pong, err := RDB.Ping(ctx).Result()
	latency := time.Since(start)
	
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to Redis")
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"response": pong,
		"latency": latency,
	}).Info("Redis connected successfully")
	
	return nil
}

func Set(key string, value interface{}, expiration time.Duration) error {
	logrus.WithFields(logrus.Fields{
		"key": key,
		"expiration": expiration,
	}).Debug("Setting cache key")

	jsonValue, err := json.Marshal(value)
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to marshal cache value")
		return err
	}

	err = RDB.Set(ctx, key, jsonValue, expiration).Err()
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to set cache key")
		return err
	}

	logrus.WithField("key", key).Trace("Cache key set successfully")
	return nil
}

func Get(key string, dest interface{}) error {
	logrus.WithField("key", key).Debug("Getting cache key")

	val, err := RDB.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			logrus.WithField("key", key).Debug("Cache key not found")
		} else {
			logrus.WithError(err).WithField("key", key).Error("Failed to get cache key")
		}
		return err
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		logrus.WithError(err).WithField("key", key).Error("Failed to unmarshal cache value")
		return err
	}

	logrus.WithField("key", key).Trace("Cache key retrieved successfully")
	return nil
}

func Delete(key string) error {
	return RDB.Del(ctx, key).Err()
}

func Exists(key string) bool {
	result := RDB.Exists(ctx, key).Val()
	return result > 0
}

func Close() error {
	return RDB.Close()
}

// 生成缓存键
func ProjectKey(id uint) string {
	return fmt.Sprintf("project:%d", id)
}

func ProjectListKey() string {
	return "projects:list"
}

func HostsKey(projectID uint) string {
	return fmt.Sprintf("project:%d:hosts", projectID)
}

func VariablesKey(projectID uint) string {
	return fmt.Sprintf("project:%d:variables", projectID)
}

func TasksKey(projectID uint) string {
	return fmt.Sprintf("project:%d:tasks", projectID)
}
