package main

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:19092"),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record := &kgo.Record{Topic: "submission.created", Value: []byte("test")}
	results := client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		fmt.Printf("ProduceSync failed: %v\n", err)
	} else {
		fmt.Println("ProduceSync success!")
	}
}
