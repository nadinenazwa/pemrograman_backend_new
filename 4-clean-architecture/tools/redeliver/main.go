package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"api-students-db/app/model"
	"api-students-db/config"
	"api-students-db/database"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "ID outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "Jumlah salinan task yang dikirim")
	flag.Parse()

	if *outboxID <= 0 {
		log.Fatal("-outbox harus berupa angka positif")
	}

	config.LoadEnv()
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(context.Background(),
		"SELECT event_type, payload FROM outbox WHERE id = $1", *outboxID).Scan(&eventType, &payload)
	if err != nil {
		log.Fatalf("gagal mengambil outbox %d: %v", *outboxID, err)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("gagal marshal job: %v", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	now := time.Now().UnixNano()
	for i := 1; i <= *copies; i++ {
		taskID := fmt.Sprintf("redeliver-%d-%d-%d", *outboxID, now, i)
		task := asynq.NewTask(model.TaskSendNotification, payloadBytes,
			asynq.TaskID(taskID),
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)
		info, err := client.EnqueueContext(context.Background(), task)
		if err != nil {
			log.Fatalf("gagal enqueue salinan ke-%d: %v", i, err)
		}
		fmt.Printf("Berhasil enqueue salinan ke-%d (task_id: %s)\n", i, info.ID)
	}
	fmt.Printf("Selesai: %d salinan task berhasil dimasukkan ke Redis\n", *copies)
}
