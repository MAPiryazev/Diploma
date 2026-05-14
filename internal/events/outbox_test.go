package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeOutboxStore struct {
	batches          [][]OutboxMessage
	fetchCalls       int
	markPublishedIDs []string
	markFailedCalls  []fakeMarkFailedCall
}

type fakeMarkFailedCall struct {
	id         string
	errMessage string
	retryDelay time.Duration
}

func (s *fakeOutboxStore) FetchReady(_ context.Context, _ int) ([]OutboxMessage, error) {
	if s.fetchCalls >= len(s.batches) {
		s.fetchCalls++
		return nil, nil
	}

	batch := s.batches[s.fetchCalls]
	s.fetchCalls++
	return batch, nil
}

func (s *fakeOutboxStore) MarkPublished(_ context.Context, id string) error {
	s.markPublishedIDs = append(s.markPublishedIDs, id)
	return nil
}

func (s *fakeOutboxStore) MarkFailed(_ context.Context, id string, publishErr error, retryDelay time.Duration) error {
	call := fakeMarkFailedCall{
		id:         id,
		retryDelay: retryDelay,
	}
	if publishErr != nil {
		call.errMessage = publishErr.Error()
	}
	s.markFailedCalls = append(s.markFailedCalls, call)
	return nil
}

type fakeRawPublisher struct {
	calls []fakePublishCall
	err   error
}

type fakePublishCall struct {
	key     string
	payload string
}

func (p *fakeRawPublisher) PublishRaw(_ context.Context, key, payload []byte) error {
	p.calls = append(p.calls, fakePublishCall{
		key:     string(key),
		payload: string(payload),
	})
	return p.err
}

func TestOutboxRelayPublishReadyMarksPublishedOnSuccess(t *testing.T) {
	store := &fakeOutboxStore{
		batches: [][]OutboxMessage{
			{{
				ID:        "outbox-1",
				EventID:   "event-1",
				EventType: EventTypeTransactionCreated,
				Topic:     "transactions.events",
				Key:       "user-1",
				Payload:   []byte(`{"ok":true}`),
			}},
			nil,
		},
	}
	publisher := &fakeRawPublisher{}
	relay := NewOutboxRelay(store, publisher, OutboxRelayConfig{
		BatchSize:      10,
		PublishTimeout: time.Second,
		RetryDelay:     3 * time.Second,
	})

	relay.publishReady(context.Background())

	if len(publisher.calls) != 1 {
		t.Fatalf("publisher calls = %d, want 1", len(publisher.calls))
	}
	if publisher.calls[0].key != "user-1" {
		t.Fatalf("publish key = %q, want %q", publisher.calls[0].key, "user-1")
	}
	if len(store.markPublishedIDs) != 1 || store.markPublishedIDs[0] != "outbox-1" {
		t.Fatalf("markPublishedIDs = %#v", store.markPublishedIDs)
	}
	if len(store.markFailedCalls) != 0 {
		t.Fatalf("markFailedCalls = %#v, want none", store.markFailedCalls)
	}
}

func TestOutboxRelayPublishReadyMarksFailedOnPublishError(t *testing.T) {
	store := &fakeOutboxStore{
		batches: [][]OutboxMessage{
			{{
				ID:        "outbox-2",
				EventID:   "event-2",
				EventType: EventTypeTransactionUpdated,
				Topic:     "transactions.events",
				Key:       "user-2",
				Payload:   []byte(`{"ok":false}`),
			}},
			nil,
		},
	}
	publisher := &fakeRawPublisher{err: errors.New("kafka unavailable")}
	relay := NewOutboxRelay(store, publisher, OutboxRelayConfig{
		BatchSize:      10,
		PublishTimeout: time.Second,
		RetryDelay:     5 * time.Second,
	})

	relay.publishReady(context.Background())

	if len(store.markPublishedIDs) != 0 {
		t.Fatalf("markPublishedIDs = %#v, want none", store.markPublishedIDs)
	}
	if len(store.markFailedCalls) != 1 {
		t.Fatalf("markFailedCalls = %d, want 1", len(store.markFailedCalls))
	}
	call := store.markFailedCalls[0]
	if call.id != "outbox-2" {
		t.Fatalf("failed id = %q, want %q", call.id, "outbox-2")
	}
	if call.errMessage != "kafka unavailable" {
		t.Fatalf("failed errMessage = %q", call.errMessage)
	}
	if call.retryDelay != 5*time.Second {
		t.Fatalf("retryDelay = %s, want %s", call.retryDelay, 5*time.Second)
	}
}
