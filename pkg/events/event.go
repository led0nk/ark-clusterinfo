package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.GetMeterProvider().Meter("github.com/led0nk/ark-overseer/internal/events")

type EventHandler interface {
	HandleEvent(context.Context, EventMessage)
}

type EventManager struct {
	logger     *slog.Logger
	subscriber map[uuid.UUID]chan EventMessage
	mu         sync.RWMutex
}

type EventMessage struct {
	Type    string
	Payload interface{}
}

func NewEventManager() *EventManager {
	return &EventManager{
		logger:     slog.Default().WithGroup("event"),
		subscriber: make(map[uuid.UUID]chan EventMessage),
	}
}

func (e *EventManager) Subscribe(name string) (uuid.UUID, <-chan EventMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()

	id, err := uuid.NewUUID()
	if err != nil {
		return uuid.Nil, nil
	}

	ch := make(chan EventMessage, 5)
	e.subscriber[id] = ch

	e.logger.Info("service subscribed to eventManager", "service id", id, "service name", name)
	return id, ch
}

func (e *EventManager) Unsubscribe(id uuid.UUID, name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := e.subscriber[id]
	close(ch)
	delete(e.subscriber, id)
	e.logger.Info("service unsubscribed to eventManager", "service id", id, "service name", name)
}

func (e *EventManager) Publish(emsg EventMessage) {
	ctx := context.Background()
	eventCtr, err := meter.Int64Counter(
		"eventCtr",
		metric.WithDescription("number of events published"),
	)
	if err != nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for subscriber, ch := range e.subscriber {
		select {
		case ch <- emsg:
			eventCtr.Add(ctx, 1)
			e.logger.Debug("publish eventMessage", "debug", "publish", subscriber.String(), fmt.Sprintf("%s", emsg))
		default:
			e.logger.Debug("channel blocked, skipping subscriber", "subscriber", subscriber.String())
		}
	}
}

func (e *EventManager) StartListening(ctx context.Context, handler EventHandler, serviceName string, onSubscribe func()) {
	id, ch := e.Subscribe(serviceName)
	if id == uuid.Nil {
		return
	}
	defer e.Unsubscribe(id, serviceName)

	onSubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			handler.HandleEvent(ctx, event)
		}
	}
}
