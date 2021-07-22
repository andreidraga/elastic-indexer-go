package data

import (
	"time"
)

// ValidatorsPublicKeys is a structure containing fields for validators public keys
type ValidatorsPublicKeys struct {
	PublicKeys []string `json:"publicKeys"`
}

// Response is a structure that holds response from Kibana
type Response struct {
	Error  interface{} `json:"error,omitempty"`
	Status int         `json:"status"`
}

// ValidatorRatingInfo is a structure containing validator rating information
type ValidatorRatingInfo struct {
	PublicKey string  `json:"-"`
	Rating    float32 `json:"rating"`
}

// RoundInfo is a structure containing block signers and shard id
type RoundInfo struct {
	Index            uint64        `json:"round"`
	SignersIndexes   []uint64      `json:"signersIndexes"`
	BlockWasProposed bool          `json:"blockWasProposed"`
	ShardId          uint32        `json:"shardId"`
	Timestamp        time.Duration `json:"timestamp"`
}

// EpochInfo holds the information about epoch
type EpochInfo struct {
	AccumulatedFees string `json:"accumulatedFees"`
	DeveloperFees   string `json:"developerFees"`
}
