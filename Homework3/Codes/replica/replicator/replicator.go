package replicator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"replica/config"
	"replica/store"
)

type ReplicateResult struct {
	PeerID  string
	Success bool
	Error   error
}

type Replicator struct {
	peers      []config.Peer
	delayMs    int
	httpClient *http.Client
}

func New(peers []config.Peer, delayMs int) *Replicator {
	return &Replicator{
		peers:   peers,
		delayMs: delayMs,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (r *Replicator) SendAsync(entry *store.Entry) {
	for _, peer := range r.peers {
		go func(p config.Peer) {
			if err := r.send(p, entry); err != nil {
				log.Printf("[replicator] async send to %s failed: %v", p.ID, err)
			}
		}(peer)
	}
}

func (r *Replicator) SendQuorum(entry *store.Entry) error {
	total := len(r.peers) + 1
	quorum := total/2 + 1
	needed := quorum - 1 

	if needed <= 0 {
		return nil
	}

	type result struct {
		peerID string
		ok     bool
		err    error
	}

	ch := make(chan result, len(r.peers))

	for _, peer := range r.peers {
		go func(p config.Peer) {
			err := r.send(p, entry)
			ch <- result{peerID: p.ID, ok: err == nil, err: err}
		}(peer)
	}

	acks := 0
	errors := []error{}

	for i := 0; i < len(r.peers); i++ {
		res := <-ch
		if res.ok {
			acks++
			log.Printf("[replicator] quorum ack from %s (%d/%d needed)", res.peerID, acks, needed)
			if acks >= needed {
				go func() {
					remaining := len(r.peers) - i - 1
					for j := 0; j < remaining; j++ {
						<-ch
					}
				}()
				return nil
			}
		} else {
			errors = append(errors, fmt.Errorf("%s: %w", res.peerID, res.err))
			log.Printf("[replicator] quorum nack from %s: %v", res.peerID, res.err)
		}
	}

	return fmt.Errorf("quorum not reached: got %d/%d acks, errors: %v", acks, needed, errors)
}

func (r *Replicator) send(peer config.Peer, entry *store.Entry) error {
	if r.delayMs > 0 {
		time.Sleep(time.Duration(r.delayMs) * time.Millisecond)
	}

	body, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	url := peer.Address + "/internal/replicate"
	resp, err := r.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("HTTP POST to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned status %d", peer.ID, resp.StatusCode)
	}
	return nil
}
