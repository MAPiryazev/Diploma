// Лёгкая нагрузка на API для демо метрик (Prometheus / Grafana): POST /items с валидными UUID из сидов.
//
// Пример: go run ./cmd/load -url http://localhost:8081 -duration 45s -qps 8 -workers 4
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// UUID из migrations/002_init_data.sql (Michael + счёт + категория + провайдер).
const (
	defaultUserID        = "11111111-1111-1111-1111-111111111111"
	defaultFromAccountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	defaultCategoryID    = "44444444-4444-4444-4444-444444444444"
	defaultProviderID    = "77777777-7777-7777-7777-777777777777"
)

var extSeq int64

type createTxBody struct {
	UserID        string  `json:"user_id"`
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	FromAccountID *string `json:"from_account_id"`
	ToAccountID   *string `json:"to_account_id"`
	ProviderID    *string `json:"provider_id"`
	CategoryID    *string `json:"category_id"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	Description   *string `json:"description"`
	ExternalID    *string `json:"external_id"`
	OccurredAt    string  `json:"occurred_at"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:8081", "base URL API (без завершающего /)")
	duration := flag.Duration("duration", 30*time.Second, "длительность нагрузки, напр. 30s или 2m")
	qps := flag.Float64("qps", 6, "целевая суммарная частота запросов/сек (по всем workers)")
	workers := flag.Int("workers", 4, "число параллельных воркеров")
	timeout := flag.Duration("timeout", 15*time.Second, "таймаут одного HTTP-запроса")
	skipHealth := flag.Bool("skip-health", false, "не вызывать GET /health перед нагрузкой")
	flag.Parse()

	*baseURL = strings.TrimSuffix(strings.TrimSpace(*baseURL), "/")
	if *workers < 1 {
		log.Fatal("workers must be >= 1")
	}
	if *qps <= 0 {
		log.Fatal("qps must be > 0")
	}

	client := &http.Client{Timeout: *timeout}

	if !*skipHealth {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		if err := pingHealth(ctx, client, *baseURL); err != nil {
			cancel()
			log.Fatalf("health check: %v", err)
		}
		cancel()
		log.Printf("health OK, base URL %s", *baseURL)
	}

	// Задержка между запросами в одном воркере, чтобы суммарно получить ~qps.
	sleep := time.Duration(float64(time.Second) * float64(*workers) / *qps)
	if sleep < time.Millisecond {
		sleep = time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var okCount, errCount int64
	var wg sync.WaitGroup

	log.Printf("load: duration=%s qps≈%.1f workers=%d sleep/worker≈%s", *duration, *qps, *workers, sleep)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*1_000_003))
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if err := postTransaction(ctx, client, *baseURL, rnd, &okCount, &errCount); err != nil {
					log.Printf("worker %d: %v", workerID, err)
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(sleep):
				}
			}
		}(i)
	}

	wg.Wait()
	ok := atomic.LoadInt64(&okCount)
	bad := atomic.LoadInt64(&errCount)
	fmt.Printf("\nload finished: ok=%d errors=%d (смотри Grafana / Prometheus)\n", ok, bad)
	if bad > 0 {
		os.Exit(1)
	}
}

func pingHealth(ctx context.Context, client *http.Client, base string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET /health: %s: %s", resp.Status, string(body))
	}
	return nil
}

func postTransaction(ctx context.Context, client *http.Client, base string, rnd *rand.Rand, okCount, errCount *int64) error {
	from := defaultFromAccountID
	cat := defaultCategoryID
	prov := defaultProviderID
	ext := fmt.Sprintf("smoke-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&extSeq, 1))
	// уникальный external_id под (provider_id, external_id)
	amount := fmt.Sprintf("%.2f", 5.0+rnd.Float64()*200.0)
	desc := "load test"
	body := createTxBody{
		UserID:        defaultUserID,
		Amount:        amount,
		Currency:      "RUB",
		FromAccountID: &from,
		ToAccountID:   nil,
		ProviderID:    &prov,
		CategoryID:    &cat,
		Type:          "expense",
		Status:        "done",
		Description:   &desc,
		ExternalID:    &ext,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
	}

	raw, err := json.Marshal(body)
	if err != nil {
		atomic.AddInt64(errCount, 1)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/items", bytes.NewReader(raw))
	if err != nil {
		atomic.AddInt64(errCount, 1)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		atomic.AddInt64(errCount, 1)
		return err
	}
	defer resp.Body.Close()
	slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode == http.StatusCreated {
		atomic.AddInt64(okCount, 1)
		return nil
	}

	atomic.AddInt64(errCount, 1)
	return fmt.Errorf("POST /items: %s body=%s", resp.Status, string(slurp))
}
