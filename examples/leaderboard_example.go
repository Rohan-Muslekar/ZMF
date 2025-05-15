// Package examples demonstrates how to use the ZMultiField library.
package examples

import (
	"context"
	"fmt"

	"github.com/Rohan-Muslekar/ZMultiField"
	"github.com/go-redis/redis/v8"
)

// This example shows how to create a simple gaming leaderboard.
func ExampleLeaderboard() {
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
		fmt.Printf("Failed to create multi-field set: %v\n", err)
		return
	}
	
	// Add a player with initial scores
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 100,
		"kills":  5,
		"deaths": 2,
	}, "player1")
	if err != nil {
		fmt.Printf("Failed to add player: %v\n", err)
		return
	}
	
	// Update a player's scores
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 50,  // Add 50 points
		"kills":  3,   // Add 3 kills
		"deaths": 1,   // Add 1 death
	}, "player1")
	if err != nil {
		fmt.Printf("Failed to update player: %v\n", err)
		return
	}
	
	// Get a player's scores
	scores, err := mfs.GetScores(ctx, "player1")
	if err != nil {
		fmt.Printf("Failed to get player scores: %v\n", err)
		return
	}
	
	fmt.Println("Player scores:")
	for _, score := range scores {
		fmt.Printf("%s: %v\n", score.Name, score.Score)
	}
	
	// Add more players
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 200,
		"kills":  10,
		"deaths": 5,
	}, "player2")
	if err != nil {
		fmt.Printf("Failed to add player2: %v\n", err)
		return
	}
	
	_, err = mfs.IncreaseScore(ctx, map[string]float64{
		"points": 150,
		"kills":  8,
		"deaths": 3,
	}, "player3")
	if err != nil {
		fmt.Printf("Failed to add player3: %v\n", err)
		return
	}
	
	// Get top 10 players
	topPlayers, err := mfs.GetTopMembers(ctx, 10)
	if err != nil {
		fmt.Printf("Failed to get top players: %v\n", err)
		return
	}
	
	fmt.Println("\nTop players:")
	for i, player := range topPlayers {
		fmt.Printf("%d. %s\n", i+1, player.Member)
		for _, score := range player.Scores {
			fmt.Printf("   %s: %v\n", score.Name, score.Score)
		}
	}
	
	// Get player ranks
	rank, err := mfs.GetRank(ctx, "player1")
	if err != nil {
		fmt.Printf("Failed to get player1 rank: %v\n", err)
		return
	}
	fmt.Printf("\nPlayer1 rank: %d\n", rank)
	
	// Reset a player
	err = mfs.ResetMember(ctx, "player3")
	if err != nil {
		fmt.Printf("Failed to reset player3: %v\n", err)
		return
	}
	fmt.Println("\nReset player3 to default scores")
	
	// Get count of players
	count, err := mfs.GetCountInRange(ctx, "-inf", "+inf")
	if err != nil {
		fmt.Printf("Failed to get player count: %v\n", err)
		return
	}
	fmt.Printf("\nTotal players: %d\n", count)
}
