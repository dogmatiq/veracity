package discovery

import (
	"context"
	"time"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/veracity/discovery/internal/discoverypb"
	"github.com/dogmatiq/veracity/persistence"
	"google.golang.org/protobuf/proto"
)

// DefaultTTL is the default TTL for advertisements to live without being
// refreshed.
const DefaultTTL = 10 * time.Second

type Observer interface {
	PeerOnline(id string, data []byte)
	PeerOffline(id string)
}

type Discoverer struct {
	NodeID   string
	Observer Observer
	Journal  persistence.Journal
	TTL      time.Duration
	Logger   logging.Logger

	prevID              []byte
	peers               map[string]time.Time
	higherPriorityPeers int
}

func (d *Discoverer) Run(ctx context.Context) error {
	defer d.expireAll()

	ttl := d.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	pollInterval := 1 * time.Second
	truncateInterval := 30 * time.Second

	nextAdvertise := time.Now()
	nextTruncate := time.Now().Add(truncateInterval)

	for {

		if err := linger.SleepX(
			ctx,
			linger.FullJitter,
			100*time.Millisecond,
		); err != nil {
			return err
		}

		d.expire()

		if err := d.read(ctx); err != nil {
			return err
		}

		now := time.Now()

		if !now.Before(nextAdvertise) {
			expiresAt := now.Add(ttl)

			ok, err := d.write(ctx, expiresAt)
			if err != nil {
				return err
			}

			if !ok {
				continue
			}

			nextAdvertise = now.Add(ttl / 2)
		}

		if !now.Before(nextTruncate) {
			if err := d.truncate(ctx); err != nil {
				return err
			}

			nextTruncate = now.Add(truncateInterval)
		}

		nextPoll := time.Now().Add(pollInterval)

		if err := linger.SleepUntil(
			ctx,
			linger.Earliest(
				nextPoll,
				nextAdvertise,
			),
		); err != nil {
			return err
		}
	}
}

// read processes the advertisements on the journal.
func (d *Discoverer) read(ctx context.Context) error {
	r, err := d.Journal.NewReader(
		ctx,
		d.prevID,
		persistence.JournalReaderOptions{
			SkipTruncated: true,
		},
	)
	if err != nil {
		return err
	}
	defer r.Close()

	for {
		id, data, ok, err := r.Next(ctx)
		if err != nil {
			return err
		}

		if !ok {
			break
		}

		m := &discoverypb.NodeAdvertisement{}
		if err := proto.Unmarshal(data, m); err != nil {
			return err
		}

		d.prevID = id

		if m.NodeId == d.NodeID {
			continue
		}

		if m.ExpiresAt < uint64(time.Now().Unix()) {
			logging.Debug(d.Logger, "skipped record '%s' for node %s, advertisement is expired", d.prevID, m.NodeId)
			continue
		}

		d.track(m)
	}

	return nil
}

// write appends an advertisement for this node to the journal.
func (d *Discoverer) write(ctx context.Context, expiresAt time.Time) (bool, error) {
	m := &discoverypb.NodeAdvertisement{
		NodeId:    d.NodeID,
		ExpiresAt: uint64(expiresAt.Unix()),
	}
	data, err := proto.Marshal(m)
	if err != nil {
		return false, err
	}

	id, ok, err := d.Journal.Append(ctx, d.prevID, data)
	if err != nil {
		return false, err
	}

	if !ok {
		logging.Debug(d.Logger, "optimistic concurrency failure writing advertisement after '%s'", d.prevID)
		return false, nil
	}

	d.prevID = id

	return true, nil
}

// track beings tracking a node as a peer, or updates its expiry timestamp if it
// is already known.
func (d *Discoverer) track(m *discoverypb.NodeAdvertisement) {
	if d.peers == nil {
		d.peers = map[string]time.Time{}
	}

	_, known := d.peers[m.NodeId]
	expiresAt := time.Unix(int64(m.ExpiresAt), 0)
	d.peers[m.NodeId] = expiresAt

	if !known {
		d.Observer.PeerOnline(m.NodeId, m.Data)

		if m.NodeId < d.NodeID {
			d.higherPriorityPeers++
		}
	}
}

// expire marks any peers that are beyond their expiry time as offline.
func (d *Discoverer) expire() {
	now := time.Now()

	for id, expiresAt := range d.peers {
		if expiresAt.Before(now) {
			delete(d.peers, id)
			d.Observer.PeerOffline(id)

			if id < d.NodeID {
				d.higherPriorityPeers--
			}
		}
	}
}

// expireAll marks all peers as offline.
func (d *Discoverer) expireAll() {
	for id := range d.peers {
		d.Observer.PeerOffline(id)
	}

	d.peers = nil
}

func (d *Discoverer) truncate(ctx context.Context) error {
	if d.higherPriorityPeers > 0 {
		return nil
	}

	r, err := d.Journal.NewReader(
		ctx,
		nil,
		persistence.JournalReaderOptions{
			SkipTruncated: true,
		},
	)
	if err != nil {
		return err
	}
	defer r.Close()

	var truncate bool
	var keepID []byte

	for {
		id, data, ok, err := r.Next(ctx)
		if err != nil {
			return err
		}

		if !ok {
			break
		}

		m := &discoverypb.NodeAdvertisement{}
		if err := proto.Unmarshal(data, m); err != nil {
			return err
		}

		if m.ExpiresAt < uint64(time.Now().Unix()) {
			truncate = true
			continue
		}

		keepID = id
		break
	}

	if truncate && keepID != nil {
		logging.Debug(d.Logger, "truncating records before '%s'", keepID)
		return d.Journal.Truncate(ctx, keepID)
	}

	return nil
}
