// Генератор нагрузки на API (демо Prometheus / Grafana): POST /items с валидными UUID из сидов.
// Поддерживает несколько паттернов QPS во времени и не рвёт последние запросы из‑за deadline всего прогона.
//
// Примеры:
//
//	go run ./cmd/load -pattern steady -duration 2m -qps 8
//	go run ./cmd/load -pattern mixed -duration 30m -workers 6
//	go run ./cmd/load -pattern wave -qps-min 2 -qps-max 18 -wave-period 3m
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
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

type patternKind string

const (
	patternSteady patternKind = "steady"
	patternWave   patternKind = "wave"
	patternSteps  patternKind = "steps"
	patternBurst  patternKind = "burst"
	patternMixed  patternKind = "mixed"
)

func main() {
	baseURL := flag.String("url", "http://localhost:8081", "base URL API (без завершающего /)")
	duration := flag.Duration("duration", 30*time.Minute, "длительность прогона")
	pattern := flag.String("pattern", string(patternMixed), "steady | wave | steps | burst | mixed")
	qps := flag.Float64("qps", 6, "целевой QPS для steady (и «база» для подсказок в логах)")
	qpsMin := flag.Float64("qps-min", 2, "нижняя граница QPS (wave/steps/burst/mixed)")
	qpsMax := flag.Float64("qps-max", 18, "верхняя граница QPS")
	wavePeriod := flag.Duration("wave-period", 4*time.Minute, "период синусоиды для pattern=wave")
	stepInterval := flag.Duration("step-interval", 90*time.Second, "длительность ступени low/high для pattern=steps")
	burstOff := flag.Duration("burst-off", 4*time.Minute+30*time.Second, "низкий QPS в pattern=burst")
	burstOn := flag.Duration("burst-on", 30*time.Second, "всплеск QPS в pattern=burst")
	workers := flag.Int("workers", 4, "число параллельных воркеров")
	timeout := flag.Duration("timeout", 15*time.Second, "таймаут одного HTTP-запроса")
	authToken := flag.String("auth-token", envOrDefault("LOAD_AUTH_TOKEN", "dev-token"), "Bearer token for protected API routes")
	skipHealth := flag.Bool("skip-health", false, "не вызывать GET /health перед нагрузкой")
	logEvery := flag.Duration("log-every", 15*time.Second, "как часто печатать текущий целевой QPS (0 = отключить)")
	flag.Parse()

	*baseURL = strings.TrimSuffix(strings.TrimSpace(*baseURL), "/")
	if *workers < 1 {
		log.Fatal("workers must be >= 1")
	}
	if *qps <= 0 {
		log.Fatal("qps must be > 0")
	}
	if *qpsMin <= 0 || *qpsMax < *qpsMin {
		log.Fatal("need 0 < qps-min <= qps-max")
	}
	pk := patternKind(strings.ToLower(strings.TrimSpace(*pattern)))
	switch pk {
	case patternSteady, patternWave, patternSteps, patternBurst, patternMixed:
	default:
		log.Fatalf("unknown -pattern %q (steady|wave|steps|burst|mixed)", *pattern)
	}
	if pk == patternWave && *wavePeriod < time.Second {
		log.Fatal("wave-period must be >= 1s")
	}
	if pk == patternSteps && *stepInterval < time.Second {
		log.Fatal("step-interval must be >= 1s")
	}
	if pk == patternBurst && (*burstOff < time.Second || *burstOn < time.Second) {
		log.Fatal("burst-off and burst-on must be >= 1s")
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

	start := time.Now()
	endAt := start.Add(*duration)

	var milliQPS int64
	setMilliQPS(&milliQPS, resolveQPS(pk, start, start, *qps, *qpsMin, *qpsMax, *wavePeriod, *stepInterval, *burstOff, *burstOn))

	stopSched := make(chan struct{})
	var schedWG sync.WaitGroup
	schedWG.Add(1)
	go func() {
		defer schedWG.Done()
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		var lastLog time.Time
		for {
			select {
			case <-stopSched:
				return
			case <-tick.C:
				now := time.Now()
				if now.After(endAt) {
					return
				}
				q := resolveQPS(pk, start, now, *qps, *qpsMin, *qpsMax, *wavePeriod, *stepInterval, *burstOff, *burstOn)
				setMilliQPS(&milliQPS, q)
				if *logEvery > 0 && (lastLog.IsZero() || now.Sub(lastLog) >= *logEvery) {
					log.Printf("target qps ≈ %.2f (%s), elapsed %s / %s", q, pk, now.Sub(start).Round(time.Second), (*duration).Round(time.Second))
					lastLog = now
				}
			}
		}
	}()

	var okCount, errCount int64
	var wg sync.WaitGroup

	log.Printf("load: pattern=%s duration=%s workers=%d qps[steady]=%.1f range=[%.1f,%.1f]",
		pk, *duration, *workers, *qps, *qpsMin, *qpsMax)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*1_000_003))
			for {
				now := time.Now()
				if now.After(endAt) {
					return
				}
				q := getMilliQPS(&milliQPS)
				sleep := workerSleep(*workers, q)

				reqCtx, cancel := context.WithTimeout(context.Background(), *timeout)
				err := postTransaction(reqCtx, client, *baseURL, *authToken, rnd, &okCount, &errCount)
				cancel()
				if err != nil {
					log.Printf("worker %d: %v", workerID, err)
				}

				now = time.Now()
				if now.After(endAt) {
					return
				}
				rem := time.Until(endAt)
				if sleep > rem {
					time.Sleep(rem)
					return
				}
				time.Sleep(sleep)
			}
		}(i)
	}

	wg.Wait()
	close(stopSched)
	schedWG.Wait()

	ok := atomic.LoadInt64(&okCount)
	bad := atomic.LoadInt64(&errCount)
	fmt.Printf("\nload finished: ok=%d errors=%d pattern=%s (Grafana / Prometheus)\n", ok, bad, pk)
	if bad > 0 {
		os.Exit(1)
	}
}

func setMilliQPS(p *int64, qps float64) {
	qps = clampQPS(qps)
	atomic.StoreInt64(p, int64(qps*1000+0.5))
}

func getMilliQPS(p *int64) float64 {
	return float64(atomic.LoadInt64(p)) / 1000
}

func clampQPS(q float64) float64 {
	const minQ = 0.25
	const maxQ = 500
	if q < minQ {
		return minQ
	}
	if q > maxQ {
		return maxQ
	}
	return q
}

func workerSleep(workers int, qps float64) time.Duration {
	if qps <= 0 {
		qps = 0.25
	}
	s := time.Duration(float64(time.Second) * float64(workers) / qps)
	if s < time.Millisecond {
		s = time.Millisecond
	}
	return s
}

func resolveQPS(
	p patternKind,
	start, now time.Time,
	qps, qpsMin, qpsMax float64,
	wavePeriod, stepInterval, burstOff, burstOn time.Duration,
) float64 {
	elapsed := now.Sub(start)
	switch p {
	case patternSteady:
		return qps
	case patternWave:
		if wavePeriod <= 0 {
			return qps
		}
		t := elapsed.Seconds()
		phase := 2 * math.Pi * t / wavePeriod.Seconds()
		mid := (qpsMax + qpsMin) / 2
		amp := (qpsMax - qpsMin) / 2
		return mid + amp*math.Sin(phase)
	case patternSteps:
		if stepInterval <= 0 {
			return qpsMin
		}
		step := int(elapsed / stepInterval)
		if step%2 == 0 {
			return qpsMin
		}
		return qpsMax
	case patternBurst:
		cycle := burstOff + burstOn
		if cycle <= 0 {
			return qpsMin
		}
		pos := elapsed % cycle
		if pos < burstOff {
			return qpsMin
		}
		return qpsMax
	case patternMixed:
		return mixedQPS(elapsed, qpsMin, qpsMax)
	default:
		return qps
	}
}

// mixedQPS: цикл ~10 мин — спокойные окна, плавные наборы/сбросы, «пик».
func mixedQPS(elapsed time.Duration, qpsMin, qpsMax float64) float64 {
	const cycle = 10 * time.Minute
	t := elapsed % cycle
	low := qpsMin + (qpsMax-qpsMin)*0.2
	high := qpsMax
	switch {
	case t < 3*time.Minute:
		return low
	case t < 4*time.Minute:
		frac := float64(t-3*time.Minute) / float64(time.Minute)
		return low + (high-low)*frac
	case t < 6*time.Minute:
		return high
	case t < 7*time.Minute:
		frac := float64(t-6*time.Minute) / float64(time.Minute)
		return high - (high-low)*frac
	default:
		return low
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

func postTransaction(ctx context.Context, client *http.Client, base string, authToken string, rnd *rand.Rand, okCount, errCount *int64) error {
	from := defaultFromAccountID
	cat := defaultCategoryID
	prov := defaultProviderID
	ext := fmt.Sprintf("smoke-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&extSeq, 1))
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
	if strings.TrimSpace(authToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(authToken))
	}

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

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
