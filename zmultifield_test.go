package zmultifield

import (
	"math/big"
	"testing"

	"github.com/go-redis/redis/v8"
)

// mockRedisClient is a mock implementation of the redis.UniversalClient interface
type mockRedisClient struct {
	redis.UniversalClient
}

// Create a mock implementation that satisfies the interface but does nothing
func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{}
}

func TestMaxBin(t *testing.T) {
	tests := []struct {
		bits     uint64
		expected string
	}{
		{1, "1"},
		{2, "3"},
		{3, "7"},
		{8, "255"},
		{10, "1023"},
	}

	for _, test := range tests {
		result := MaxBin(test.bits)
		expected := new(big.Int)
		expected.SetString(test.expected, 10)

		if result.Cmp(expected) != 0 {
			t.Errorf("MaxBin(%d) = %s, expected %s", test.bits, result.String(), expected.String())
		}
	}
}

func TestBitCount(t *testing.T) {
	tests := []struct {
		value    float64
		expected uint64
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 2},
		{7, 3},
		{8, 4},
		{15, 4},
		{16, 5},
		{255, 8},
		{256, 9},
		{1023, 10},
		{1024, 11},
	}

	for _, test := range tests {
		result := BitCount(test.value)
		if result != test.expected {
			t.Errorf("BitCount(%f) = %d, expected %d", test.value, result, test.expected)
		}
	}
}

func TestMultiField_SetIndex(t *testing.T) {
	field := newMultiField(Field{
		Name:       "test",
		Sort:       Ascending,
		MaxValue:   255,
		UpdateType: Incremental,
	})

	// Initial values
	if field.position != 0 {
		t.Errorf("Initial position = %d, expected 0", field.position)
	}
	if field.shiftValue != 0 {
		t.Errorf("Initial shiftValue = %d, expected 0", field.shiftValue)
	}

	// Set index
	field.setIndex(1, 10)

	// Check values
	if field.position != 1 {
		t.Errorf("After setIndex position = %d, expected 1", field.position)
	}
	if field.shiftValue != 10 {
		t.Errorf("After setIndex shiftValue = %d, expected 10", field.shiftValue)
	}

	// Check mask shift
	expectedMask := new(big.Int).Lsh(MaxBin(field.bits), 10)
	if field.mask.Cmp(expectedMask) != 0 {
		t.Errorf("After setIndex mask = %s, expected %s", field.mask.String(), expectedMask.String())
	}
}

func TestMultiField_DefaultScore(t *testing.T) {
	// Test ascending field
	ascField := newMultiField(Field{
		Name:       "asc",
		Sort:       Ascending,
		MaxValue:   255,
		UpdateType: Incremental,
	})

	if ascField.defaultScore().Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Ascending field defaultScore = %s, expected 0", ascField.defaultScore().String())
	}

	// Test descending field
	descField := newMultiField(Field{
		Name:       "desc",
		Sort:       Descending,
		MaxValue:   255,
		UpdateType: Incremental,
	})

	expectedDefault := MaxBin(descField.bits)
	if descField.defaultScore().Cmp(expectedDefault) != 0 {
		t.Errorf("Descending field defaultScore = %s, expected %s", descField.defaultScore().String(), expectedDefault.String())
	}
}

func TestScoresToZScore(t *testing.T) {
	// Create test fields
	fields := []Field{
		{
			Name:       "field1",
			Sort:       Descending,
			MaxValue:   100,
			UpdateType: Incremental,
		},
		{
			Name:       "field2",
			Sort:       Ascending,
			MaxValue:   50,
			UpdateType: Incremental,
		},
	}

	// Create a MultiFieldSet with mock Redis client
	mfs, err := New(MultiFieldSetOptions{
		Name:   "test",
		Fields: fields,
		Client: newMockRedisClient(),
	})

	if err != nil {
		t.Fatalf("Failed to create MultiFieldSet: %v", err)
	}

	// Test combining scores
	scores := []*big.Int{
		big.NewInt(20), // field1
		big.NewInt(30), // field2
	}

	// Calculate zscore
	zscore := mfs.scoresToZScore(scores)

	// Manual calculation
	// field1 gets 7 bits (0-127), field2 gets 6 bits (0-63)
	// field2 is at position 0, field1 is at position 1
	// 20 << 6 + 30 = 1310
	expectedZScore := big.NewInt(1310)

	if zscore.Cmp(expectedZScore) != 0 {
		t.Errorf("scoresToZScore() = %s, expected %s", zscore.String(), expectedZScore.String())
	}
}

func TestExtractFieldScore(t *testing.T) {
	// Create test fields
	fields := []Field{
		{
			Name:       "field1",
			Sort:       Descending,
			MaxValue:   100,
			UpdateType: Incremental,
		},
		{
			Name:       "field2",
			Sort:       Ascending,
			MaxValue:   50,
			UpdateType: Incremental,
		},
	}

	// Create a MultiFieldSet with mock Redis client
	mfs, err := New(MultiFieldSetOptions{
		Name:   "test",
		Fields: fields,
		Client: newMockRedisClient(),
	})

	if err != nil {
		t.Fatalf("Failed to create MultiFieldSet: %v", err)
	}

	// Create a test zscore: 20 << 6 + 30 = 1310
	zscore := big.NewInt(1310)

	// Extract field1 (position 1)
	field1Score := mfs.extractFieldScore(mfs.fields[0], zscore)
	expectedField1Score := big.NewInt(20)
	if field1Score.Cmp(expectedField1Score) != 0 {
		t.Errorf("extractFieldScore(field1) = %s, expected %s", field1Score.String(), expectedField1Score.String())
	}

	// Extract field2 (position 0)
	field2Score := mfs.extractFieldScore(mfs.fields[1], zscore)
	expectedField2Score := big.NewInt(30)
	if field2Score.Cmp(expectedField2Score) != 0 {
		t.Errorf("extractFieldScore(field2) = %s, expected %s", field2Score.String(), expectedField2Score.String())
	}
}
