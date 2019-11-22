package whispertypes

import (
	"context"

	protocol "github.com/status-im/status-go/protocol/types"
)

// NewMessage represents a new whisper message that is posted through the RPC.
type NewMessage struct {
	SymKeyID   string    `json:"symKeyID"`
	PublicKey  []byte    `json:"pubKey"`
	SigID      string    `json:"sig"`
	TTL        uint32    `json:"ttl"`
	Topic      TopicType `json:"topic"`
	Payload    []byte    `json:"payload"`
	Padding    []byte    `json:"padding"`
	PowTime    uint32    `json:"powTime"`
	PowTarget  float64   `json:"powTarget"`
	TargetPeer string    `json:"targetPeer"`
}

// Message is the RPC representation of a whisper message.
type Message struct {
	Sig       []byte    `json:"sig,omitempty"`
	TTL       uint32    `json:"ttl"`
	Timestamp uint32    `json:"timestamp"`
	Topic     TopicType `json:"topic"`
	Payload   []byte    `json:"payload"`
	Padding   []byte    `json:"padding"`
	PoW       float64   `json:"pow"`
	Hash      []byte    `json:"hash"`
	Dst       []byte    `json:"recipientPublicKey,omitempty"`
	P2P       bool      `json:"bool,omitempty"`
}

// Criteria holds various filter options for inbound messages.
type Criteria struct {
	SymKeyID     string      `json:"symKeyID"`
	PrivateKeyID string      `json:"privateKeyID"`
	Sig          []byte      `json:"sig"`
	MinPow       float64     `json:"minPow"`
	Topics       []TopicType `json:"topics"`
	AllowP2P     bool        `json:"allowP2P"`
}

// PublicWhisperAPI provides the whisper RPC service that can be
// use publicly without security implications.
type PublicWhisperAPI interface {
	// AddPrivateKey imports the given private key.
	AddPrivateKey(ctx context.Context, privateKey protocol.HexBytes) (string, error)
	// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
	GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error)
	// DeleteKeyPair removes the key with the given key if it exists.
	DeleteKeyPair(ctx context.Context, key string) (bool, error)

	// Post posts a message on the Whisper network.
	// returns the hash of the message in case of success.
	Post(ctx context.Context, req NewMessage) ([]byte, error)

	// NewMessageFilter creates a new filter that can be used to poll for
	// (new) messages that satisfy the given criteria.
	NewMessageFilter(req Criteria) (string, error)
	// GetFilterMessages returns the messages that match the filter criteria and
	// are received between the last poll and now.
	GetFilterMessages(id string) ([]*Message, error)
}
