// Package zmultifield provides functionality to create and manage multi-field sorted sets in Redis.
// It can be used for various applications, including leaderboards, where multiple values need to
// be stored as a single score in Redis sorted sets.
package zmultifield

import (
	"math/big"
	"math/bits"
)

// SortOrder defines the sorting order for a field.
type SortOrder int

const (
	// Ascending indicates a field should be sorted in ascending order (smaller values first).
	Ascending SortOrder = 1
	// Descending indicates a field should be sorted in descending order (larger values first).
	Descending SortOrder = -1
)

// UpdateType defines how a field's value should be updated when using incremental functions.
type UpdateType int

const (
	// Incremental indicates a field's value should be incremented/decremented by the given value.
	Incremental UpdateType = 1
	// Replace indicates a field's value should be replaced with the given value.
	Replace UpdateType = 2
)

// Field defines the properties for a single field within a multi-field sorted set.
type Field struct {
	Name       string
	Sort       SortOrder
	MaxValue   float64
	UpdateType UpdateType
}

// FieldInfo provides detailed information about a field's properties and bit allocation.
type FieldInfo struct {
	Name         string
	Sort         SortOrder
	UpdateType   string
	MaxValue     float64
	Bits         uint64
	ShiftValue   uint64
	Mask         *big.Int
	IsMain       bool
	DefaultScore *big.Int
	Position     int
	MaxAbsolute  *big.Int
}

// fieldScore represents a field's name and its score.
type fieldScore struct {
	Name  string
	Score *big.Int
}

// MemberScores represents a member and its scores for all fields.
type MemberScores struct {
	Member string
	Scores []fieldScore
}

// BitCount returns the number of bits required to represent a value.
func BitCount(n float64) uint64 {
	if n <= 0 {
		return 1
	}
	return uint64(bits.Len64(uint64(n)))
}

// MaxBin returns a big.Int with n bits set to 1.
func MaxBin(n uint64) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}

	result := new(big.Int).Lsh(big.NewInt(1), uint(n))
	return new(big.Int).Sub(result, big.NewInt(1))
}
