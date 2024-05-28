package blist

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/led0nk/ark-clusterinfo/internal/model"
)

type Blacklist struct {
	filename  string
	blacklist map[uuid.UUID]*model.BlacklistPlayers
	mu        sync.Mutex
}

func NewBlacklist(filename string) (*Blacklist, error) {
	blacklist := &Blacklist{
		filename:  filename,
		blacklist: make(map[uuid.UUID]*model.BlacklistPlayers),
	}
	if err := blacklist.readJSON(); err != nil {
		return nil, err
	}
	return blacklist, nil
}

func (b *Blacklist) writeJSON() error {
	as_json, err := json.MarshalIndent(b.blacklist, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(b.filename, as_json, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (b *Blacklist) readJSON() error {
	if _, err := os.Stat(b.filename); os.IsNotExist(err) {
		err = b.writeJSON()
		if err != nil {
			return err
		}
	}
	data, err := os.ReadFile(b.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.blacklist)
}

func (b *Blacklist) Create(ctx context.Context, player *model.BlacklistPlayers) (*model.BlacklistPlayers, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if player.ID == uuid.Nil {
		player.ID = uuid.New()
	}

	b.blacklist[player.ID] = player
	if err := b.writeJSON(); err != nil {
		return nil, err
	}
	return player, nil
}

func (b *Blacklist) Delete(ctx context.Context, id uuid.UUID) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.blacklist, id)
	if err := b.writeJSON(); err != nil {
		return err
	}
	return nil
}

func (b *Blacklist) List(ctx context.Context) []*model.BlacklistPlayers {
	b.mu.Lock()
	defer b.mu.Unlock()

	blacklist := make([]*model.BlacklistPlayers, 0, len(b.blacklist))

	for _, player := range b.blacklist {
		blacklist = append(blacklist, player)
	}
	return blacklist
}
