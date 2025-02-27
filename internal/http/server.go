package httpserver

import (
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"io"
	"log/slog"
	"net/http"
)

type Request struct {
	TestNum int32  `json:"testNum"`
	Result  bool   `json:"result"`
	Msg     string `json:"msg"`
}

type Server struct {
	addr          string
	kafkaProducer *kafka.Producer
	kafkaTopic    string
}

type TestResponse struct {
	TestReq TestRequest `json:"test_req"`
	Status  bool        `json:"status"`
}

type TestRequest struct {
	SourceID   uint `json:"source_id"`
	TestNumber uint `json:"test_number"`
}

func NewServer(addr string, kafkaProducer *kafka.Producer, kafkaTopic string) *Server {
	return &Server{
		addr:          addr,
		kafkaProducer: kafkaProducer,
		kafkaTopic:    kafkaTopic,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/test", s.handleTest)
	slog.Info("Starting server", "address", s.addr)
	return http.ListenAndServe(s.addr, nil)
}

func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading body", "error", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var requestData TestResponse
	if err := json.Unmarshal(body, &requestData); err != nil {
		slog.Error("JSON decode error", "error", err, "body", string(body))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(requestData)
	if err != nil {
		slog.Error("Ошибка сериализации в sendToKafka", "error", err)
		return
	}

	err = s.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &s.kafkaTopic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data received and logged"))
}
