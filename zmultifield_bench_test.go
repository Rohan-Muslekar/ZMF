package zmultifield

import (
	"math/big"
	"testing"
)

// Benchmark scoresToZScore to see how fast we can combine multiple fields into one score
func BenchmarkScoresToZScore(b *testing.B) {
	// Create test fields
	fields := []Field{
		{
			Name:       "field1",
			Sort:       Descending,
			MaxValue:   100000,
			UpdateType: Incremental,
		},
		{
			Name:       "field2",
			Sort:       Ascending,
			MaxValue:   50000,
			UpdateType: Incremental,
		},
		{
			Name:       "field3",
			Sort:       Descending,
			MaxValue:   1000,
			UpdateType: Incremental,
		},
	}

	// Create a MultiFieldSet with mock Redis client
	mfs, err := New(MultiFieldSetOptions{
		Name:   "benchmark",
		Fields: fields,
		Client: newMockRedisClient(),
	})

	if err != nil {
		b.Fatalf("Failed to create MultiFieldSet: %v", err)
	}

	// Create test scores
	scores := []*big.Int{
		big.NewInt(50000),  // field1
		big.NewInt(25000),  // field2
		big.NewInt(500),    // field3
	}

	// Reset timer for fair benchmarking
	b.ResetTimer()
	
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		mfs.scoresToZScore(scores)
	}
}

// Benchmark extractFieldScore to see how fast we can extract field scores from a combined score
func BenchmarkExtractFieldScore(b *testing.B) {
	// Create test fields
	fields := []Field{
		{
			Name:       "field1",
			Sort:       Descending,
			MaxValue:   100000,
			UpdateType: Incremental,
		},
		{
			Name:       "field2",
			Sort:       Ascending,
			MaxValue:   50000,
			UpdateType: Incremental,
		},
		{
			Name:       "field3",
			Sort:       Descending,
			MaxValue:   1000,
			UpdateType: Incremental,
		},
	}

	// Create a MultiFieldSet with mock Redis client
	mfs, err := New(MultiFieldSetOptions{
		Name:   "benchmark",
		Fields: fields,
		Client: newMockRedisClient(),
	})

	if err != nil {
		b.Fatalf("Failed to create MultiFieldSet: %v", err)
	}

	// Create test scores and calculate zscore
	scores := []*big.Int{
		big.NewInt(50000),  // field1
		big.NewInt(25000),  // field2
		big.NewInt(500),    // field3
	}
	zscore := mfs.scoresToZScore(scores)

	// Reset timer for fair benchmarking
	b.ResetTimer()
	
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		for _, field := range mfs.fields {
			mfs.extractFieldScore(field, zscore)
		}
	}
}

// Benchmark zscoreToAllFieldScores to see how fast we can convert a zscore to all field scores
func BenchmarkZScoreToAllFieldScores(b *testing.B) {
	// Create test fields
	fields := []Field{
		{
			Name:       "field1",
			Sort:       Descending,
			MaxValue:   100000,
			UpdateType: Incremental,
		},
		{
			Name:       "field2",
			Sort:       Ascending,
			MaxValue:   50000,
			UpdateType: Incremental,
		},
		{
			Name:       "field3",
			Sort:       Descending,
			MaxValue:   1000,
			UpdateType: Incremental,
		},
	}

	// Create a MultiFieldSet with mock Redis client
	mfs, err := New(MultiFieldSetOptions{
		Name:   "benchmark",
		Fields: fields,
		Client: newMockRedisClient(),
	})

	if err != nil {
		b.Fatalf("Failed to create MultiFieldSet: %v", err)
	}

	// Create test scores and calculate zscore
	scores := []*big.Int{
		big.NewInt(50000),  // field1
		big.NewInt(25000),  // field2
		big.NewInt(500),    // field3
	}
	zscore := mfs.scoresToZScore(scores)

	// Reset timer for fair benchmarking
	b.ResetTimer()
	
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		mfs.zscoreToAllFieldScores(zscore)
	}
}
