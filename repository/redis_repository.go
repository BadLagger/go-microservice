package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Metric struct {
	DeviceID string  `json:"device_id"`
	Value    float64 `json:"value"`
	// При необходимости можно добавить Timestamp
}

type RedisRepository struct {
	client *redis.Client
	prefix string
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
		prefix: "metrics:", // Простой префикс для изоляции ключей
	}
}

// Сохраняет метрику в Redis LIST (для скользящего окна)
func (r *RedisRepository) PushMetric(ctx context.Context, metric Metric) error {
	// Сериализуем метрику в JSON
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal metric failed: %w", err)
	}

	// Используем LIST для хранения окна
	key := r.prefix + "rolling_window"

	// 1. Добавляем новое значение в конец списка
	if err := r.client.RPush(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("redis RPush failed: %w", err)
	}

	// 2. Обрезаем список до 50 элементов (окно)
	if err := r.client.LTrim(ctx, key, -50, -1).Err(); err != nil {
		return fmt.Errorf("redis LTrim failed: %w", err)
	}

	return nil
}

// Получает последние N метрик для вычислений
func (r *RedisRepository) GetLatestMetrics(ctx context.Context, limit int64) ([]Metric, error) {
	key := r.prefix + "rolling_window"

	// Получаем последние N элементов списка
	data, err := r.client.LRange(ctx, key, -limit, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis LRange failed: %w", err)
	}

	metrics := make([]Metric, 0, len(data))
	for _, item := range data {
		var metric Metric

		if err := json.Unmarshal([]byte(item), &metric); err != nil {
			// Пропускаем битые данные, но логируем
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// Опционально: простой метод для тестирования соединения
func (r *RedisRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
