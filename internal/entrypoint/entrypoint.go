package entrypoint

import (
	"UserService/internal/config"
	httpserver "UserService/internal/http"
	"bytes"
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
)

type (
	// Entrypoint entrypoint methods.
	Entrypoint interface {
		Run() error
	}

	entrypoint struct {
		cfg           *config.Config
		logger        *slog.Logger
		kafkaProducer *kafka.Producer
		mu            sync.Mutex
		startTime     time.Time
		eventCount    int
	}
)

type Test struct {
	SourceID   uint   `json:"source_id"`
	TestNumber uint   `json:"test_number"`
	Time       string `json:"time"`
}

func New(cfg *config.Config) (Entrypoint, error) {
	ep := &entrypoint{cfg: cfg}

	var err error

	ep.logger = SetupLogger(ep.cfg.Env)

	ep.logger.Info("UserService starts", slog.String("env", cfg.Env))

	ep.kafkaProducer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": cfg.KafkaProducer.Broker})
	if err != nil {
		ep.logger.Error("Ошибка создания Kafka producer", "error", err)
		return nil, err
	}
	ep.logger.Info("Connected tt Kafka")

	return ep, nil
}

func (e *entrypoint) Run() error {
	e.logger.Info("Запуск сервиса")
	defer e.kafkaProducer.Close()

	rand.Seed(time.Now().UnixNano())
	e.startTime = time.Now()

	wg := sync.WaitGroup{}
	wg.Add(int(e.cfg.Sources))

	for i := range e.cfg.Sources {
		go func(i uint) {
			defer wg.Done()
			test := Test{
				SourceID:   i,
				TestNumber: 0,
				Time:       time.Now().Format(time.RFC3339),
			}
			for {
				test.TestNumber++
				currentLambda := e.calculateLambda()
				e.logger.Info("Текущая λ", "lambda", currentLambda)

				e.sendTest(test)

				sleepTime := poissonDelay(currentLambda)
				e.logger.Info("Следующий запрос через", "duration", sleepTime)

				time.Sleep(sleepTime)
			}
		}(i)
	}

	server := httpserver.NewServer(":8081", e.kafkaProducer, e.cfg.Topic)
	if err := server.Start(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
	server.Start()

	wg.Wait()
	e.logger.Info("Сервис закончил работу")
	return nil
}

func (e *entrypoint) calculateLambda() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	elapsedTime := time.Since(e.startTime).Seconds() // Время наблюдения Tn
	if elapsedTime == 0 {
		return 0.5 // Если программа только запустилась, используем 0.5 по умолчанию
	}
	return float64(e.eventCount) / elapsedTime
}

// Функция генерации случайного интервала (по Пуассону)
func poissonDelay(lambda float64) time.Duration {
	if lambda <= 0 {
		lambda = 0.5 // Предохранитель от деления на ноль
	}
	r := rand.Float64()
	tau := -math.Log(r) / lambda
	return time.Duration(tau * float64(time.Second))
}

// sendToKafka сериализует данные теста и отправляет их в Kafka.
func (e *entrypoint) sendToKafka(test Test) {
	data, err := json.Marshal(test)
	if err != nil {
		slog.Error("Ошибка сериализации в sendToKafka", "error", err)
		return
	}

	err = e.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &e.cfg.KafkaProducer.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)

	if err != nil {
		e.logger.Error("Ошибка отправки в Kafka", "error", err)
	} else {
		e.logger.Info("Данные успешно отправлены в Kafka", "topic", e.cfg.KafkaProducer.Topic)
	}
}

func (e *entrypoint) sendTest(test Test) {
	data, err := json.Marshal(test)
	if err != nil {
		e.logger.Error("Ошибка сериализации теста", "error", err)
		return
	}

	resp, err := http.Post("http://localhost:8082/test", "application/json", bytes.NewBuffer(data))
	if err != nil {
		e.logger.Error("Ошибка отправки POST запроса", "error", err)
		return
	}
	defer resp.Body.Close()

	e.logger.Info("Тест успешно отправлен", "status", resp.Status)

	e.sendToKafka(test)

	e.mu.Lock()
	e.eventCount++
	e.mu.Unlock()
}

func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}
