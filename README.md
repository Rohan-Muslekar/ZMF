# ZMultiField

ZMultiField is a Go library that allows you to create and manage multi-field sorted sets in Redis. It enables you to store multiple values (fields) as a single score in Redis sorted sets through clever bit manipulation.

## Features

- Store multiple fields in a single Redis sorted set score
- Support for both ascending and descending sort orders
- Incremental and replacement update types
- Efficient bit packing to maximize storage in Redis scores
- Flexible retrieval of scores by field or for all fields
- Support for range queries and pagination

## Installation

```bash
go get github.com/Rohan-Muslekar/ZMultiField
```

## Requirements

- Go 1.24 or later
- go-redis/redis v8 or later

## Usage

### Basic Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/Rohan-Muslekar/ZMultiField"
	"log"
)

func main() {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	ctx := context.Background()
	
	// Define fields for a gaming leaderboard
	fields := []zmultifield.Field{
		{
			Name:       "points",
			Sort:       zmultifield.Descending, // Higher points are better
			MaxValue:   1000000,
			UpdateType: zmultifield.Incremental,
		},
		{
			Name:       "kills",
			Sort:       zmultifield.Descending, // Higher kills are better
			MaxValue:   10000,
			UpdateType: zmultifield.Incremental,
		},
		{
			Name:       "deaths",
			Sort:       zmultifield.Ascending, // Lower deaths are better
			MaxValue:   10000,
			UpdateType: zmultifield.Incremental,
		},
	}
	
	// Create a new multi-field set
	mfs, err := zmultifield.New(zmultifield.MultiFieldSetOptions{
		Name:   "game:leaderboard",
		Fields: fields,
		Client: rdb,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-field set: %v", err)
	}
	
	// Add a player with initial scores
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 100,
		"kills":  5,
		"deaths": 2,
	}, "player1")
	if err != nil {
		log.Fatalf("Failed to add player: %v", err)
	}
	
	// Update a player's scores
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 50,  // Add 50 points
		"kills":  3,   // Add 3 kills
		"deaths": 1,   // Add 1 death
	}, "player1")
	if err != nil {
		log.Fatalf("Failed to update player: %v", err)
	}
	
	// Get a player's scores
	scores, err := mfs.GetScores(ctx, "player1")
	if err != nil {
		log.Fatalf("Failed to get player scores: %v", err)
	}
	
	fmt.Println("Player scores:")
	for _, score := range scores {
		fmt.Printf("%s: %v\n", score.Name, score.Score)
	}
	
	// Get top 10 players
	topPlayers, err := mfs.GetTopMembers(ctx, 10)
	if err != nil {
		log.Fatalf("Failed to get top players: %v", err)
	}
	
	fmt.Println("\nTop players:")
	for i, player := range topPlayers {
		fmt.Printf("%d. %s\n", i+1, player.Member)
		for _, score := range player.Scores {
			fmt.Printf("   %s: %v\n", score.Name, score.Score)
		}
	}
}
```

### Creating a Leaderboard Extension

You can easily extend the ZMultiField library to create more specialized implementations like a gaming leaderboard:

```go
package leaderboard

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/Rohan-Muslekar/ZMultiField"
)

// GameLeaderboard extends ZMultiField for gaming leaderboards
type GameLeaderboard struct {
	mfs *zmultifield.MultiFieldSet
}

// NewGameLeaderboard creates a new game leaderboard
func NewGameLeaderboard(ctx context.Context, client redis.UniversalClient, name string) (*GameLeaderboard, error) {
	// Define fields specific to a game leaderboard
	fields := []zmultifield.Field{
		{
			Name:       "points",
			Sort:       zmultifield.Descending,
			MaxValue:   1000000,
			UpdateType: zmultifield.Incremental,
		},
		{
			Name:       "wins",
			Sort:       zmultifield.Descending,
			MaxValue:   10000,
			UpdateType: zmultifield.Incremental,
		},
		{
			Name:       "playtime",
			Sort:       zmultifield.Descending,
			MaxValue:   1000000, // in minutes
			UpdateType: zmultifield.Incremental,
		},
	}
	
	mfs, err := zmultifield.New(zmultifield.MultiFieldSetOptions{
		Name:   name,
		Fields: fields,
		Client: client,
	})
	
	if err != nil {
		return nil, err
	}
	
	return &GameLeaderboard{mfs: mfs}, nil
}

// AddPoints adds points to a player
func (gl *GameLeaderboard) AddPoints(ctx context.Context, player string, points float64) error {
	_, err := gl.mfs.IncreaseScore(ctx, map[string]float64{"points": points}, player)
	return err
}

// AddWin adds a win to a player
func (gl *GameLeaderboard) AddWin(ctx context.Context, player string) error {
	_, err := gl.mfs.IncreaseScore(ctx, map[string]float64{"wins": 1}, player)
	return err
}

// AddPlaytime adds playtime minutes to a player
func (gl *GameLeaderboard) AddPlaytime(ctx context.Context, player string, minutes float64) error {
	_, err := gl.mfs.IncreaseScore(ctx, map[string]float64{"playtime": minutes}, player)
	return err
}

// GetTopPlayers gets the top n players
func (gl *GameLeaderboard) GetTopPlayers(ctx context.Context, limit int64) ([]zmultifield.MemberScores, error) {
	return gl.mfs.GetTopMembers(ctx, limit)
}

// GetPlayerRank gets a player's rank
func (gl *GameLeaderboard) GetPlayerRank(ctx context.Context, player string) (int64, error) {
	return gl.mfs.GetRank(ctx, player)
}
```

## How It Works

ZMultiField allocates a specific number of bits for each field based on its maximum value. These fields are then combined using bitwise operations to create a single score value that can be stored in Redis sorted sets.

For example, with three fields:
- Field A: 10 bits (values 0-1023)
- Field B: 8 bits (values 0-255)
- Field C: 6 bits (values 0-63)

The fields are arranged from most significant to least significant bits:
- Field A: bits 14-23
- Field B: bits 6-13
- Field C: bits 0-5

This arrangement ensures that Redis sorts primarily by Field A, then by Field B for ties, and finally by Field C.

The library handles the complexity of packing and unpacking these bit-encoded scores, as well as handling ascending vs. descending sort orders by inverting values appropriately.

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
