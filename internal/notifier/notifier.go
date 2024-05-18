package notifier

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/led0nk/ark-clusterinfo/internal"
	"github.com/led0nk/ark-clusterinfo/internal/model"
)

type Notifier struct {
	sStore     internal.ServerStore
	mu         sync.RWMutex
	logger     *slog.Logger
	subscriber map[uuid.UUID]chan string
}

func NewNotifier(s internal.ServerStore) *Notifier {
	return &Notifier{
		sStore:     s,
		logger:     slog.Default().WithGroup("notifier"),
		subscriber: make(map[uuid.UUID]chan string),
	}
}

func (n *Notifier) Create(ctx context.Context, srv *model.Server) (*model.Server, error) {
	n.notify("create")
	return n.sStore.Create(ctx, srv)
}

func (n *Notifier) Delete(ctx context.Context, id uuid.UUID) error {
	n.notify("delete")
	return n.sStore.Delete(ctx, id)
}

func (n *Notifier) GetByID(ctx context.Context, id uuid.UUID) (*model.Server, error) {
	n.notify("get by id")
	return n.sStore.GetByID(ctx, id)
}

func (n *Notifier) GetByName(ctx context.Context, name string) (*model.Server, error) {
	n.notify("get by name")
	return n.sStore.GetByName(ctx, name)
}

func (n *Notifier) List(ctx context.Context) ([]*model.Server, error) {
	//n.notify("list")
	return n.sStore.List(ctx)
}

func (n *Notifier) Update(ctx context.Context, srv *model.Server) error {
	//(n.notify("update")
	return n.sStore.Update(ctx, srv)
}

func (n *Notifier) notify(method string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, ch := range n.subscriber {
		select {
		case ch <- method:
		default:
		}
	}
}

func (n *Notifier) Subscribe() (uuid.UUID, <-chan string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	id, err := uuid.NewUUID()
	if err != nil {
		n.logger.Error("failed to create uuid", "error", err)
		return uuid.Nil, nil
	}
	ch := make(chan string)
	n.subscriber[id] = ch
	n.logger.Info("new notifier subscribed", "notifier id", id)
	return id, ch
}

func (n *Notifier) Unsubscribe(id uuid.UUID) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ch := n.subscriber[id]
	close(ch)
	delete(n.subscriber, id)
	n.logger.Info("notifier unsubscribed", "notifier id", id)
}

func (n *Notifier) Run(gofunc func(context.Context), ctx context.Context) {
	id, signal := n.Subscribe()
	defer n.Unsubscribe(id)
	for {
		notification := <-signal
		n.logger.InfoContext(ctx, "targets were updated", "type", notification)
		//go obs.ManageScraper(ctx)
		go gofunc(ctx)
	}
}
