package relayer

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"

	_ "github.com/lib/pq"
)

type RelayerDB struct {
	mu        sync.Mutex
	db        *sql.DB
	useMem    bool
	memNonces map[string]string            // nonceHex -> state (e.g. "pending", "confirmed", "stuck")
	memVotes  map[string]map[string][]byte // nonceHex -> relayerAddress -> signature
	checkpoints map[string]uint64
	memLocks    map[string]LockEvent
	memBurns    map[string]BurnEvent
}

func NewRelayerDB(connStr string) (*RelayerDB, error) {
	if connStr == "" || connStr == "memory" {
		return &RelayerDB{
			useMem:    true,
			memNonces: make(map[string]string),
			memVotes:  make(map[string]map[string][]byte),
			checkpoints: make(map[string]uint64),
			memLocks:  make(map[string]LockEvent),
			memBurns:  make(map[string]BurnEvent),
		}, nil
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	rdb := &RelayerDB{
		db:        db,
		useMem:    false,
		memNonces: make(map[string]string),
		memVotes:  make(map[string]map[string][]byte),
		checkpoints: make(map[string]uint64),
		memLocks:  make(map[string]LockEvent),
		memBurns:  make(map[string]BurnEvent),
	}

	if err := rdb.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return rdb, nil
}

func (r *RelayerDB) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *RelayerDB) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS nonces (
		nonce TEXT PRIMARY KEY,
		state TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS votes (
		nonce TEXT,
		relayer TEXT,
		signature BYTEA,
		PRIMARY KEY(nonce, relayer)
	);
	CREATE TABLE IF NOT EXISTS checkpoints (
		key TEXT PRIMARY KEY,
		block_num BIGINT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS lock_events (
		nonce TEXT PRIMARY KEY,
		user_address TEXT NOT NULL,
		amount BIGINT NOT NULL,
		cosmos_recipient TEXT NOT NULL,
		block_number BIGINT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS burn_events (
		nonce TEXT PRIMARY KEY,
		sender TEXT NOT NULL,
		bsc_recipient TEXT NOT NULL,
		amount BIGINT NOT NULL,
		block_height BIGINT NOT NULL
	);
	`
	_, err := r.db.Exec(query)
	return err
}

func (r *RelayerDB) SetNonceState(nonceHex string, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		r.memNonces[nonceHex] = state
		return nil
	}

	_, err := r.db.Exec(`
		INSERT INTO nonces (nonce, state) VALUES ($1, $2)
		ON CONFLICT (nonce) DO UPDATE SET state = EXCLUDED.state`,
		nonceHex, state,
	)
	return err
}

func (r *RelayerDB) GetNonceState(nonceHex string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		return r.memNonces[nonceHex], nil
	}

	var state string
	err := r.db.QueryRow("SELECT state FROM nonces WHERE nonce = $1", nonceHex).Scan(&state)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return state, err
}

func (r *RelayerDB) AddVote(nonceHex string, relayer string, sig []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		if r.memVotes[nonceHex] == nil {
			r.memVotes[nonceHex] = make(map[string][]byte)
		}
		r.memVotes[nonceHex][relayer] = sig
		return len(r.memVotes[nonceHex]), nil
	}

	_, err := r.db.Exec(`
		INSERT INTO votes (nonce, relayer, signature) VALUES ($1, $2, $3)
		ON CONFLICT (nonce, relayer) DO UPDATE SET signature = EXCLUDED.signature`,
		nonceHex, relayer, sig,
	)
	if err != nil {
		return 0, err
	}

	var count int
	err = r.db.QueryRow("SELECT COUNT(*) FROM votes WHERE nonce = $1", nonceHex).Scan(&count)
	return count, err
}

func (r *RelayerDB) GetVotes(nonceHex string) ([][]byte, []string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		votesMap := r.memVotes[nonceHex]
		var sigs [][]byte
		var relayers []string
		for rel, sig := range votesMap {
			sigs = append(sigs, sig)
			relayers = append(relayers, rel)
		}
		return sigs, relayers, nil
	}

	rows, err := r.db.Query("SELECT relayer, signature FROM votes WHERE nonce = $1", nonceHex)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var sigs [][]byte
	var relayers []string
	for rows.Next() {
		var relayer string
		var sig []byte
		if err := rows.Scan(&relayer, &sig); err != nil {
			return nil, nil, err
		}
		sigs = append(sigs, sig)
		relayers = append(relayers, relayer)
	}
	return sigs, relayers, nil
}

func (r *RelayerDB) SaveCheckpoint(key string, blockNum uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		r.checkpoints[key] = blockNum
		return nil
	}

	_, err := r.db.Exec(`
		INSERT INTO checkpoints (key, block_num) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET block_num = EXCLUDED.block_num`,
		key, blockNum,
	)
	return err
}

func (r *RelayerDB) GetCheckpoint(key string) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		return r.checkpoints[key], nil
	}

	var blockNum uint64
	err := r.db.QueryRow("SELECT block_num FROM checkpoints WHERE key = $1", key).Scan(&blockNum)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return blockNum, err
}

func (r *RelayerDB) SaveLockEvent(event LockEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nonceHex := fmt.Sprintf("%x", event.Nonce)
	if r.useMem {
		r.memLocks[nonceHex] = event
		return nil
	}

	_, err := r.db.Exec(`
		INSERT INTO lock_events (nonce, user_address, amount, cosmos_recipient, block_number)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nonce) DO NOTHING`,
		nonceHex, event.User, event.Amount, event.CosmosRecipient, event.BlockNumber,
	)
	return err
}

func (r *RelayerDB) GetLockEvent(nonceHex string) (*LockEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		event, ok := r.memLocks[nonceHex]
		if !ok {
			return nil, nil
		}
		return &event, nil
	}

	var user, recipient string
	var amount, blockNum uint64
	err := r.db.QueryRow(`
		SELECT user_address, amount, cosmos_recipient, block_number 
		FROM lock_events WHERE nonce = $1`, nonceHex,
	).Scan(&user, &amount, &recipient, &blockNum)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, err
	}

	return &LockEvent{
		User:            user,
		Amount:          amount,
		CosmosRecipient: recipient,
		Nonce:           nonce,
		BlockNumber:     blockNum,
	}, nil
}

func (r *RelayerDB) SaveBurnEvent(event BurnEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nonceHex := fmt.Sprintf("%x", event.Nonce)
	if r.useMem {
		r.memBurns[nonceHex] = event
		return nil
	}

	_, err := r.db.Exec(`
		INSERT INTO burn_events (nonce, sender, bsc_recipient, amount, block_height)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nonce) DO NOTHING`,
		nonceHex, event.Sender, event.BscRecipient, event.Amount, event.BlockHeight,
	)
	return err
}

func (r *RelayerDB) GetBurnEvent(nonceHex string) (*BurnEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useMem {
		event, ok := r.memBurns[nonceHex]
		if !ok {
			return nil, nil
		}
		return &event, nil
	}

	var sender, recipient string
	var amount, blockHeight uint64
	err := r.db.QueryRow(`
		SELECT sender, bsc_recipient, amount, block_height 
		FROM burn_events WHERE nonce = $1`, nonceHex,
	).Scan(&sender, &recipient, &amount, &blockHeight)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, err
	}

	return &BurnEvent{
		Sender:       sender,
		BscRecipient: recipient,
		Amount:       amount,
		Nonce:        nonce,
		BlockHeight:  blockHeight,
	}, nil
}

