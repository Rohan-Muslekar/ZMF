package examples

import (
	"context"

	"github.com/Rohan-Muslekar/ZMultiField"
	"github.com/go-redis/redis/v8"
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

// Player represents a player in the game
type Player struct {
	ID       string
	Points   int64
	Wins     int64
	Playtime int64
	Rank     int64
}

// GetPlayerInfo gets a player's information
func (gl *GameLeaderboard) GetPlayerInfo(ctx context.Context, playerID string) (*Player, error) {
	// Get scores
	scores, err := gl.mfs.GetScores(ctx, playerID)
	if err != nil {
		return nil, err
	}
	
	// Get rank
	rank, err := gl.mfs.GetRank(ctx, playerID)
	if err != nil {
		return nil, err
	}
	
	player := &Player{
		ID:   playerID,
		Rank: rank,
	}
	
	// Extract scores
	for _, score := range scores {
		scoreInt := score.Score.Int64()
		
		switch score.Name {
		case "points":
			player.Points = scoreInt
		case "wins":
			player.Wins = scoreInt
		case "playtime":
			player.Playtime = scoreInt
		}
	}
	
	return player, nil
}

// ExampleExtendedLeaderboard demonstrates how to use a custom leaderboard implementation
func ExampleExtendedLeaderboard() {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	ctx := context.Background()
	
	// Create a new game leaderboard
	gameLeaderboard, err := NewGameLeaderboard(ctx, rdb, "mygame:leaderboard")
	if err != nil {
		// Handle error
		return
	}
	
	// Add a player with points
	err = gameLeaderboard.AddPoints(ctx, "player1", 100)
	if err != nil {
		// Handle error
		return
	}
	
	// Add a win
	err = gameLeaderboard.AddWin(ctx, "player1")
	if err != nil {
		// Handle error
		return
	}
	
	// Add playtime
	err = gameLeaderboard.AddPlaytime(ctx, "player1", 30)
	if err != nil {
		// Handle error
		return
	}
	
	// Get player info
	player, err := gameLeaderboard.GetPlayerInfo(ctx, "player1")
	if err != nil {
		// Handle error
		return
	}
	
	// Use player info
	_ = player
}
