// Scenario load runner for the protected API.
//
// Examples:
//
//	go run ./cmd/load -duration 2m -qps 8 -profile balanced
//	go run ./cmd/load -profile negative -duration 30s
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
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultUserID        = "11111111-1111-1111-1111-111111111111"
	defaultFromAccountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	defaultCategoryID    = "44444444-4444-4444-4444-444444444444"
	defaultProviderID    = "77777777-7777-7777-7777-777777777777"
	wrongUserID          = "22222222-2222-2222-2222-222222222222"
)

var extSeq int64

type txBody struct {
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
	ExternalID    *string `json:"external_id,omitempty"`
	OccurredAt    string  `json:"occurred_at"`
}

type transaction struct {
	ID string `json:"id"`
}

type apiResponse[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Count   int    `json:"count"`
}

type scenario struct {
	name   string
	weight int
	run    func(context.Context, *runner, *rand.Rand) result
}

type result struct {
	scenario string
	expected int
	actual   int
	duration time.Duration
	err      error
}

type runner struct {
	baseURL   string
	authToken string
	client    *http.Client
	pool      txPool
	stats     stats
}

type txPool struct {
	mu  sync.Mutex
	ids []string
}

func (p *txPool) add(id string) {
	if id == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, id)
}

func (p *txPool) pick(rnd *rand.Rand) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.ids) == 0 {
		return "", false
	}
	return p.ids[rnd.Intn(len(p.ids))], true
}

func (p *txPool) pop(rnd *rand.Rand) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.ids) == 0 {
		return "", false
	}
	i := rnd.Intn(len(p.ids))
	id := p.ids[i]
	p.ids[i] = p.ids[len(p.ids)-1]
	p.ids = p.ids[:len(p.ids)-1]
	return id, true
}

type stats struct {
	mu      sync.Mutex
	rows    map[string]*statRow
	started time.Time
}

type statRow struct {
	ok        int
	bad       int
	statuses  map[string]int
	durations []time.Duration
}

func (s *stats) init() {
	s.started = time.Now()
	s.rows = make(map[string]*statRow)
}

func (s *stats) record(res result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.rows[res.scenario]
	if row == nil {
		row = &statRow{statuses: make(map[string]int)}
		s.rows[res.scenario] = row
	}
	if res.err == nil && res.actual == res.expected {
		row.ok++
	} else {
		row.bad++
	}
	status := fmt.Sprintf("%d", res.actual)
	if res.err != nil {
		status = "error"
	}
	row.statuses[status]++
	if res.duration > 0 {
		row.durations = append(row.durations, res.duration)
	}
}

func (s *stats) print(profile string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.rows))
	for name := range s.rows {
		names = append(names, name)
	}
	sort.Strings(names)

	var totalOK, totalBad int
	fmt.Printf("\nload finished: profile=%s elapsed=%s\n", profile, time.Since(s.started).Round(time.Second))
	fmt.Println("scenario             ok    errors  p50     p95     p99     statuses")
	for _, name := range names {
		row := s.rows[name]
		totalOK += row.ok
		totalBad += row.bad
		p50, p95, p99 := quantiles(row.durations)
		fmt.Printf("%-20s %5d %7d  %-7s %-7s %-7s %s\n",
			name, row.ok, row.bad, p50, p95, p99, formatStatuses(row.statuses))
	}
	fmt.Printf("total: ok=%d errors=%d\n", totalOK, totalBad)
}

func main() {
	baseURL := flag.String("url", "http://localhost:8081", "base API URL without trailing slash")
	duration := flag.Duration("duration", 30*time.Minute, "load duration")
	profile := flag.String("profile", envOrDefault("LOAD_PROFILE", "balanced"), "smoke | balanced | stress | negative")
	qps := flag.Float64("qps", 8, "target total requests per second")
	workers := flag.Int("workers", 6, "parallel workers")
	timeout := flag.Duration("timeout", 15*time.Second, "single request timeout")
	authToken := flag.String("auth-token", envOrDefault("LOAD_AUTH_TOKEN", "dev-token"), "Bearer token for protected routes")
	skipHealth := flag.Bool("skip-health", false, "skip GET /health before the run")
	logEvery := flag.Duration("log-every", 15*time.Second, "progress log interval, 0 disables logs")
	flag.Parse()

	*baseURL = strings.TrimSuffix(strings.TrimSpace(*baseURL), "/")
	if *workers < 1 {
		log.Fatal("workers must be >= 1")
	}
	if *qps <= 0 {
		log.Fatal("qps must be > 0")
	}
	scenarios, err := scenariosFor(*profile)
	if err != nil {
		log.Fatal(err)
	}

	r := &runner{
		baseURL:   *baseURL,
		authToken: strings.TrimSpace(*authToken),
		client:    &http.Client{Timeout: *timeout},
	}
	r.stats.init()

	if !*skipHealth {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		if err := r.health(ctx); err != nil {
			cancel()
			log.Fatalf("health check: %v", err)
		}
		cancel()
		log.Printf("health OK, base URL %s", *baseURL)
	}

	endAt := time.Now().Add(*duration)
	var wg sync.WaitGroup
	log.Printf("load: profile=%s duration=%s workers=%d qps=%.1f", *profile, *duration, *workers, *qps)

	stopProgress := make(chan struct{})
	go progressLogger(stopProgress, *logEvery, endAt, *profile, *qps)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*1_000_003))
			sleep := workerSleep(*workers, *qps)
			for time.Now().Before(endAt) {
				reqCtx, cancel := context.WithTimeout(context.Background(), *timeout)
				sc := chooseScenario(scenarios, rnd)
				res := sc.run(reqCtx, r, rnd)
				if res.scenario == "" {
					res.scenario = sc.name
				}
				cancel()
				r.stats.record(res)
				if res.err != nil {
					log.Printf("%s: %v", sc.name, res.err)
				}

				if rem := time.Until(endAt); rem < sleep {
					time.Sleep(rem)
					return
				}
				time.Sleep(sleep)
			}
		}(i)
	}

	wg.Wait()
	close(stopProgress)
	r.stats.print(*profile)

	var errors int
	r.stats.mu.Lock()
	for _, row := range r.stats.rows {
		errors += row.bad
	}
	r.stats.mu.Unlock()
	if errors > 0 {
		os.Exit(1)
	}
}

func scenariosFor(profile string) ([]scenario, error) {
	create := scenario{"create", 35, runCreate}
	list := scenario{"list", 20, runList}
	get := scenario{"get_by_id", 15, runGet}
	update := scenario{"update", 10, runUpdate}
	analytics := scenario{"analytics", 10, runAnalytics}
	del := scenario{"delete", 5, runDelete}
	replay := scenario{"idempotency_replay", 3, runIdempotencyReplay}
	conflict := scenario{"idempotency_conflict", 1, runIdempotencyConflict}
	forbidden := scenario{"forbidden", 1, runForbidden}
	unauthorized := scenario{"unauthorized", 1, runUnauthorized}

	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "smoke":
		return []scenario{create, list, get, analytics}, nil
	case "balanced", "":
		return []scenario{create, list, get, update, analytics, del, replay}, nil
	case "stress":
		create.weight = 45
		list.weight = 20
		get.weight = 15
		update.weight = 10
		analytics.weight = 8
		del.weight = 2
		return []scenario{create, list, get, update, analytics, del}, nil
	case "negative":
		replay.weight = 25
		conflict.weight = 35
		forbidden.weight = 20
		unauthorized.weight = 20
		return []scenario{replay, conflict, forbidden, unauthorized}, nil
	default:
		return nil, fmt.Errorf("unknown -profile %q (smoke|balanced|stress|negative)", profile)
	}
}

func chooseScenario(scenarios []scenario, rnd *rand.Rand) scenario {
	total := 0
	for _, sc := range scenarios {
		total += sc.weight
	}
	n := rnd.Intn(total)
	for _, sc := range scenarios {
		if n < sc.weight {
			return sc
		}
		n -= sc.weight
	}
	return scenarios[len(scenarios)-1]
}

func runCreate(ctx context.Context, r *runner, rnd *rand.Rand) result {
	body := newTxBody(rnd)
	var resp apiResponse[transaction]
	status, dur, err := r.doJSON(ctx, http.MethodPost, "/items", "", body, &resp, true, nil)
	if err == nil && status == http.StatusCreated {
		r.pool.add(resp.Data.ID)
	}
	return result{expected: http.StatusCreated, actual: status, duration: dur, err: err}
}

func runList(ctx context.Context, r *runner, _ *rand.Rand) result {
	path := "/items?" + userQuery(defaultUserID)
	status, dur, err := r.doJSON(ctx, http.MethodGet, path, "", nil, nil, true, nil)
	return result{expected: http.StatusOK, actual: status, duration: dur, err: err}
}

func runGet(ctx context.Context, r *runner, rnd *rand.Rand) result {
	id, ok := r.pool.pick(rnd)
	if !ok {
		res := runCreate(ctx, r, rnd)
		res.scenario = "create"
		return res
	}
	path := "/items/" + url.PathEscape(id) + "?" + userQuery(defaultUserID)
	status, dur, err := r.doJSON(ctx, http.MethodGet, path, "", nil, nil, true, nil)
	return result{expected: http.StatusOK, actual: status, duration: dur, err: err}
}

func runUpdate(ctx context.Context, r *runner, rnd *rand.Rand) result {
	id, ok := r.pool.pick(rnd)
	if !ok {
		res := runCreate(ctx, r, rnd)
		res.scenario = "create"
		return res
	}
	body := newTxBody(rnd)
	body.ExternalID = nil
	body.Description = stringPtr("load test updated")
	status, dur, err := r.doJSON(ctx, http.MethodPut, "/items/"+url.PathEscape(id), "", body, nil, true, nil)
	return result{expected: http.StatusOK, actual: status, duration: dur, err: err}
}

func runAnalytics(ctx context.Context, r *runner, _ *rand.Rand) result {
	to := time.Now().UTC()
	from := to.Add(-24 * time.Hour)
	q := url.Values{}
	q.Set("user_id", defaultUserID)
	q.Set("from", from.Format(time.RFC3339))
	q.Set("to", to.Format(time.RFC3339))
	status, dur, err := r.doJSON(ctx, http.MethodGet, "/analytics?"+q.Encode(), "", nil, nil, true, nil)
	return result{expected: http.StatusOK, actual: status, duration: dur, err: err}
}

func runDelete(ctx context.Context, r *runner, rnd *rand.Rand) result {
	id, ok := r.pool.pop(rnd)
	if !ok {
		res := runCreate(ctx, r, rnd)
		res.scenario = "create"
		return res
	}
	path := "/items/" + url.PathEscape(id) + "?" + userQuery(defaultUserID)
	status, dur, err := r.doJSON(ctx, http.MethodDelete, path, "", nil, nil, true, nil)
	return result{expected: http.StatusOK, actual: status, duration: dur, err: err}
}

func runIdempotencyReplay(ctx context.Context, r *runner, rnd *rand.Rand) result {
	key := fmt.Sprintf("load-replay-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&extSeq, 1))
	body := newTxBody(rnd)
	var first apiResponse[transaction]
	status, _, err := r.doJSON(ctx, http.MethodPost, "/items", key, body, &first, true, nil)
	if err != nil || status != http.StatusCreated {
		return result{expected: http.StatusCreated, actual: status, err: err}
	}
	r.pool.add(first.Data.ID)

	var second apiResponse[transaction]
	status, dur, err := r.doJSON(ctx, http.MethodPost, "/items", key, body, &second, true, nil)
	return result{expected: http.StatusCreated, actual: status, duration: dur, err: err}
}

func runIdempotencyConflict(ctx context.Context, r *runner, rnd *rand.Rand) result {
	key := fmt.Sprintf("load-conflict-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&extSeq, 1))
	body := newTxBody(rnd)
	var first apiResponse[transaction]
	status, _, err := r.doJSON(ctx, http.MethodPost, "/items", key, body, &first, true, nil)
	if err != nil || status != http.StatusCreated {
		return result{expected: http.StatusCreated, actual: status, err: err}
	}
	r.pool.add(first.Data.ID)

	changed := body
	changed.Amount = fmt.Sprintf("%.2f", 500.0+rnd.Float64()*200.0)
	status, dur, err := r.doJSON(ctx, http.MethodPost, "/items", key, changed, nil, true, nil)
	return result{expected: http.StatusConflict, actual: status, duration: dur, err: err}
}

func runForbidden(ctx context.Context, r *runner, _ *rand.Rand) result {
	status, dur, err := r.doJSON(ctx, http.MethodGet, "/items?"+userQuery(wrongUserID), "", nil, nil, true, nil)
	return result{expected: http.StatusForbidden, actual: status, duration: dur, err: err}
}

func runUnauthorized(ctx context.Context, r *runner, _ *rand.Rand) result {
	status, dur, err := r.doJSON(ctx, http.MethodGet, "/items?"+userQuery(defaultUserID), "", nil, nil, false, nil)
	return result{expected: http.StatusUnauthorized, actual: status, duration: dur, err: err}
}

func (r *runner) health(ctx context.Context) error {
	status, _, err := r.doJSON(ctx, http.MethodGet, "/health", "", nil, nil, false, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET /health: status %d", status)
	}
	return nil
}

func (r *runner) doJSON(
	ctx context.Context,
	method string,
	path string,
	idempotencyKey string,
	body any,
	out any,
	withAuth bool,
	headers map[string]string,
) (int, time.Duration, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, 0, err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, reader)
	if err != nil {
		return 0, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth && r.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.authToken)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	dur := time.Since(start)
	if err != nil {
		return 0, dur, err
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return resp.StatusCode, dur, readErr
	}
	if out != nil && len(raw) > 0 && resp.StatusCode < 500 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, dur, fmt.Errorf("decode response: %w: body=%s", err, string(raw))
		}
	}
	return resp.StatusCode, dur, nil
}

func newTxBody(rnd *rand.Rand) txBody {
	from := defaultFromAccountID
	cat := defaultCategoryID
	prov := defaultProviderID
	desc := "load test"
	ext := fmt.Sprintf("load-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&extSeq, 1))
	amount := 5.0 + rnd.Float64()*200.0
	if rnd.Intn(10) == 0 {
		amount = 1200.0 + rnd.Float64()*300.0
	}
	return txBody{
		UserID:        defaultUserID,
		Amount:        fmt.Sprintf("%.2f", amount),
		Currency:      "RUB",
		FromAccountID: &from,
		ProviderID:    &prov,
		CategoryID:    &cat,
		Type:          "expense",
		Status:        "done",
		Description:   &desc,
		ExternalID:    &ext,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
	}
}

func userQuery(userID string) string {
	q := url.Values{}
	q.Set("user_id", userID)
	return q.Encode()
}

func stringPtr(value string) *string {
	return &value
}

func workerSleep(workers int, qps float64) time.Duration {
	sleep := time.Duration(float64(time.Second) * float64(workers) / qps)
	if sleep < time.Millisecond {
		return time.Millisecond
	}
	return sleep
}

func progressLogger(stop <-chan struct{}, every time.Duration, endAt time.Time, profile string, qps float64) {
	if every <= 0 {
		return
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			if time.Now().After(endAt) {
				return
			}
			log.Printf("load progress: profile=%s target_qps=%.1f remaining=%s", profile, qps, time.Until(endAt).Round(time.Second))
		}
	}
}

func quantiles(values []time.Duration) (string, string, string) {
	if len(values) == 0 {
		return "-", "-", "-"
	}
	cp := append([]time.Duration(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return fmtDuration(percentile(cp, 0.50)), fmtDuration(percentile(cp, 0.95)), fmtDuration(percentile(cp, 0.99))
}

func percentile(values []time.Duration, q float64) time.Duration {
	if len(values) == 1 {
		return values[0]
	}
	i := int(q * float64(len(values)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(values) {
		i = len(values) - 1
	}
	return values[i]
}

func fmtDuration(d time.Duration) string {
	if d >= time.Second {
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

func formatStatuses(statuses map[string]int) string {
	keys := make([]string, 0, len(statuses))
	for key := range statuses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, statuses[key]))
	}
	return strings.Join(parts, ",")
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
