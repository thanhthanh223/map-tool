package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

var rdCluster *redis.ClusterClient
var rd *redis.Client
var ctx = context.Background()
var prefix = ""

const (
	redisHashProvincePolygon = "geo_polygon:tinh_tp"
	redisHashWardPolygon     = "geo_polygon:phuong_xa"
)

func InitRedis() {
	clusterEnv := os.Getenv("REDIS_CLUSTER")
	password := os.Getenv("REDIS_PASSWORD")

	if strings.TrimSpace(clusterEnv) != "" {
		addrs := []string{}
		for _, p := range strings.Split(clusterEnv, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				addrs = append(addrs, p)
			}
		}
		if len(addrs) > 0 {
			rdCluster = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:    addrs,
				Password: password,
			})
		}
	}

	if rdCluster == nil { // fallback to single instance
		addr := os.Getenv("REDIS_ADDRESS") // e.g., localhost:6379
		if strings.TrimSpace(addr) != "" {
			rd = redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: password,
			})
		}
	}

	prefix = os.Getenv("REDIS_PREFIX")
	if len(prefix) > 0 {
		prefix = prefix + ":"
	}
}

func Close() {
	if rdCluster != nil {
		_ = rdCluster.Close()
	}
	if rd != nil {
		_ = rd.Close()
	}

}

func Set(key string, val any, exp time.Duration) error {
	if rdCluster != nil {
		return rdCluster.Set(ctx, prefix+key, val, exp).Err()
	}
	return rd.Set(ctx, prefix+key, val, exp).Err()
}

func Get(key string) (string, error) {
	if rdCluster != nil {
		return rdCluster.Get(ctx, prefix+key).Result()
	}
	return rd.Get(ctx, prefix+key).Result()
}

func GetDel(key string) (string, error) {
	if rdCluster != nil {
		val, err := rdCluster.Get(ctx, prefix+key).Result()
		if err == nil {
			err = rdCluster.Del(ctx, prefix+key).Err()
			return val, err
		}
		return val, err
	}
	return rd.GetDel(ctx, prefix+key).Result()
}

func Del(key string) error {
	if rdCluster != nil {
		return rdCluster.Del(ctx, prefix+key).Err()
	}
	return rd.Del(ctx, prefix+key).Err()
}

func HMSet(key string, val any) error {
	if rdCluster != nil {
		return rdCluster.HMSet(ctx, prefix+key, val).Err()
	}
	return rd.HMSet(ctx, prefix+key, val).Err()
}

func HMGet[E any](key string, field string) (*E, error) {
	var values []interface{}
	var err error
	if rdCluster != nil {
		values, err = rdCluster.HMGet(ctx, prefix+key, field).Result()
	} else {
		values, err = rd.HMGet(ctx, prefix+key, field).Result()
	}
	if err != nil || values[0] == nil {
		return nil, err
	}
	var e E
	err = json.Unmarshal([]byte(values[0].(string)), &e)
	return &e, err
}

func HSet(key string, hKey any, val any) error {
	if rdCluster != nil {
		return rdCluster.HSet(ctx, prefix+key, hKey, val).Err()
	}
	return rd.HSet(ctx, prefix+key, hKey, val).Err()
}

func HGet[E any](key string, hkey string) (*E, error) {
	var value string
	if rdCluster != nil {
		value = rdCluster.HGet(ctx, prefix+key, hkey).Val()
	} else {
		value = rd.HGet(ctx, prefix+key, hkey).Val()
	}
	var e E
	err := json.Unmarshal([]byte(value), &e)
	return &e, err
}

func HGetAll[E any](key string) (map[string]E, error) {
	var value map[string]string
	if rdCluster != nil {
		value = rdCluster.HGetAll(ctx, prefix+key).Val()
	} else {
		value = rd.HGetAll(ctx, prefix+key).Val()
	}
	m := map[string]E{}
	for k, v := range value {
		var e E
		err := json.Unmarshal([]byte(v), &e)
		if err != nil {
			log.Printf("Loi khi parse value tu redis: %s", err.Error())
			return nil, err
		}
		m[k] = e
	}
	return m, nil
}

func HDel(key string, hKey string) error {
	if rdCluster != nil {
		return rdCluster.HDel(ctx, prefix+key, hKey).Err()
	}
	return rd.HDel(ctx, prefix+key, hKey).Err()
}

func Incr(key string) (int64, error) {
	if rdCluster != nil {
		return rdCluster.Incr(ctx, prefix+key).Result()
	}
	return rd.Incr(ctx, prefix+key).Result()
}

func SetNX(key string, val any, exp time.Duration) (bool, error) {
	if rdCluster != nil {
		return rdCluster.SetNX(ctx, prefix+key, val, exp).Result()
	}
	return rd.SetNX(ctx, prefix+key, val, exp).Result()
}

func Expire(key string, exp time.Duration) error {
	if rdCluster != nil {
		return rdCluster.Expire(ctx, prefix+key, exp).Err()
	}
	return rd.Expire(ctx, prefix+key, exp).Err()
}

func ExpireNX(key string, exp time.Duration) (bool, error) {
	if rdCluster != nil {
		return rdCluster.Expire(ctx, prefix+key, exp).Result()
	}
	return rd.Expire(ctx, prefix+key, exp).Result()
}

func HExists(key string, hkey string) (bool, error) {
	if rdCluster != nil {
		return rdCluster.HExists(ctx, prefix+key, hkey).Result()
	}
	return rd.HExists(ctx, prefix+key, hkey).Result()
}

func HKeys(key string) ([]string, error) {
	if rdCluster != nil {
		return rdCluster.HKeys(ctx, prefix+key).Result()
	}
	return rd.HKeys(ctx, prefix+key).Result()
}

func HValues(key string) ([]string, error) {
	if rdCluster != nil {
		return rdCluster.HVals(ctx, prefix+key).Result()
	}
	return rd.HVals(ctx, prefix+key).Result()
}

func HLen(key string) (int64, error) {
	if rdCluster != nil {
		return rdCluster.HLen(ctx, prefix+key).Result()
	}
	return rd.HLen(ctx, prefix+key).Result()
}

func HSetNX(key string, hKey string, val any) (bool, error) {
	if rdCluster != nil {
		return rdCluster.HSetNX(ctx, prefix+key, hKey, val).Result()
	}
	return rd.HSetNX(ctx, prefix+key, hKey, val).Result()
}

func HIncrBy(key string, hKey string, incr int64) (int64, error) {
	if rdCluster != nil {
		return rdCluster.HIncrBy(ctx, prefix+key, hKey, incr).Result()
	}
	return rd.HIncrBy(ctx, prefix+key, hKey, incr).Result()
}

func HIncrByFloat(key string, hKey string, incr float64) (float64, error) {
	if rdCluster != nil {
		return rdCluster.HIncrByFloat(ctx, prefix+key, hKey, incr).Result()
	}
	return rd.HIncrByFloat(ctx, prefix+key, hKey, incr).Result()
}

// Helper function để lưu struct vào HSET
func HSetStruct(key string, hKey string, data interface{}) error {
	// Convert struct thành JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Lưu JSON string vào HSET
	return HSet(key, hKey, string(jsonData))
}

// Helper function để lấy struct từ HSET
func HGetStruct[T any](key string, hKey string) (*T, error) {
	// Lấy JSON string từ HSET
	jsonStr, err := HGet[string](key, hKey)
	if err != nil {
		return nil, err
	}

	// Convert JSON string về struct
	var result T
	err = json.Unmarshal([]byte(*jsonStr), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Helper function để lưu nhiều struct vào HSET
func HMSetStruct(key string, data map[string]interface{}) error {
	// Convert tất cả struct thành JSON string
	jsonData := make(map[string]interface{})
	for k, v := range data {
		if jsonBytes, err := json.Marshal(v); err == nil {
			jsonData[k] = string(jsonBytes)
		} else {
			jsonData[k] = v // Giữ nguyên nếu không phải struct
		}
	}

	return HMSet(key, jsonData)
}

// Helper function để lấy tất cả struct từ HSET
func HGetAllStruct[T any](key string) (map[string]T, error) {
	// Lấy tất cả data dưới dạng string
	allData, err := HGetAll[string](key)
	if err != nil {
		return nil, err
	}

	// Convert từng JSON string về struct
	result := make(map[string]T)
	for k, v := range allData {
		var item T
		if err := json.Unmarshal([]byte(v), &item); err == nil {
			result[k] = item
		}
		// Skip nếu không parse được (có thể là string thường)
	}

	return result, nil
}

func GetAllKeyByPrefix(prefix string) ([]string, error) {
	var keys []string
	var mu sync.Mutex
	match := fmt.Sprintf("%s:*", prefix)

	_, ctx := errgroup.WithContext(ctx)

	if rdCluster != nil {
		err := rdCluster.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			iter := client.Scan(ctx, 0, match, 0).Iterator()
			for iter.Next(ctx) {
				mu.Lock()
				keys = append(keys, iter.Val())
				mu.Unlock()
			}

			if err := iter.Err(); err != nil {
				return fmt.Errorf("error scanning keys on a master node: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return keys, nil
	}

	if rd == nil {
		return nil, fmt.Errorf("redis is not initialized")
	}

	iter := rd.Scan(ctx, 0, match, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error scanning keys: %w", err)
	}
	return keys, nil
}
func Exists(key string) (bool, error) {
	if rdCluster != nil {
		exists, err := rdCluster.Exists(ctx, prefix+key).Result()
		return exists > 0, err
	}
	exists, err := rd.Exists(ctx, prefix+key).Result()
	return exists > 0, err
}
