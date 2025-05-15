package zmultifield

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/go-redis/redis/v8"
)

// MultiFieldSet manages a Redis sorted set with multiple fields packed into a single score.
type MultiFieldSet struct {
	fields        []*multiField
	name          string
	client        redis.UniversalClient
	defaultZScore *big.Int
}

// MultiFieldSetOptions defines options for creating a new MultiFieldSet.
type MultiFieldSetOptions struct {
	Name   string
	Fields []Field
	Client redis.UniversalClient
}

// New creates a new MultiFieldSet instance.
func New(opts MultiFieldSetOptions) (*MultiFieldSet, error) {
	if opts.Name == "" {
		return nil, errors.New("name is required")
	}

	if len(opts.Fields) == 0 {
		return nil, errors.New("at least one field is required")
	}

	if opts.Client == nil {
		return nil, errors.New("Redis client is required")
	}

	// Create multiFields from Fields
	multiFields := make([]*multiField, len(opts.Fields))
	for i, f := range opts.Fields {
		multiFields[i] = newMultiField(f)
	}

	// Calculate total shifts and set indices
	var totalShifts uint64 = 0
	for i := len(multiFields) - 1; i >= 0; i-- {
		multiFields[i].setIndex(i, totalShifts)
		totalShifts += multiFields[i].bits
	}

	// Initialize MultiFieldSet
	mfs := &MultiFieldSet{
		fields: multiFields,
		name:   opts.Name,
		client: opts.Client,
	}

	// Calculate default zscore
	defaultScores := make([]*big.Int, len(multiFields))
	for i, field := range multiFields {
		defaultScores[i] = field.defaultScore()
	}
	mfs.defaultZScore = mfs.scoresToZScore(defaultScores)

	return mfs, nil
}

// GetName returns the name of the sorted set.
func (mfs *MultiFieldSet) GetName() string {
	return mfs.name
}

// GetFieldsInfo returns information about all fields.
func (mfs *MultiFieldSet) GetFieldsInfo() []FieldInfo {
	info := make([]FieldInfo, len(mfs.fields))
	for i, field := range mfs.fields {
		info[i] = field.getInfo()
	}
	return info
}

// scoresToZScore combines individual field scores into a single zscore.
func (mfs *MultiFieldSet) scoresToZScore(scores []*big.Int) *big.Int {
	zscore := big.NewInt(0)
	for i, score := range scores {
		// Shift the score by the field's shift value
		shiftedScore := new(big.Int).Lsh(score, uint(mfs.fields[i].shiftValue))
		// Add to the zscore
		zscore.Add(zscore, shiftedScore)
	}
	return zscore
}

// extractFieldScore extracts a field's score from a zscore.
func (mfs *MultiFieldSet) extractFieldScore(field *multiField, zscore *big.Int) *big.Int {
	if zscore == nil {
		return field.defaultScore()
	}

	// Apply mask to isolate the field bits
	fieldScore := new(big.Int).And(zscore, field.mask)
	// Shift right to get the actual value
	fieldScore.Rsh(fieldScore, uint(field.shiftValue))
	return fieldScore
}

// getFieldScores extracts all field scores from a zscore.
func (mfs *MultiFieldSet) getFieldScores(zscore *big.Int) []*big.Int {
	scores := make([]*big.Int, len(mfs.fields))
	for i, field := range mfs.fields {
		scores[i] = mfs.extractFieldScore(field, zscore)
	}
	return scores
}

// GetFieldByName returns a field by name or nil if not found.
func (mfs *MultiFieldSet) GetFieldByName(name string) *multiField {
	for _, field := range mfs.fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

// IncreaseScore increases the score for specified fields of a member.
func (mfs *MultiFieldSet) IncreaseScore(ctx context.Context, fields map[string]float64, member string) (*big.Int, error) {
	// Get current scores
	currentZScore, err := mfs.client.ZScore(ctx, mfs.name, member).Result()
	if err == redis.Nil {
		// Member doesn't exist, use default scores
		scores := make([]*big.Int, len(mfs.fields))
		for i, field := range mfs.fields {
			scores[i] = field.defaultScore()
		}

		// Update scores
		for fieldName, incValue := range fields {
			field := mfs.GetFieldByName(fieldName)
			if field == nil {
				return nil, fmt.Errorf("field %s not found", fieldName)
			}

			inc := new(big.Int).SetInt64(int64(incValue))
			inc.Mul(inc, field.multiplier)

			if field.UpdateType == Incremental {
				scores[field.position].Add(scores[field.position], inc)
			} else if field.UpdateType == Replace {
				scores[field.position] = new(big.Int).Add(field.defaultScore(), inc)
			} else {
				return nil, errors.New("unknown update type")
			}

			// Check range
			if scores[field.position].Sign() < 0 || scores[field.position].Cmp(field.maxAbsolute) > 0 {
				return nil, fmt.Errorf("score %v out of range for field %s", scores[field.position], field.Name)
			}
		}

		// Calculate new zscore
		finalZScore := mfs.scoresToZScore(scores)

		// Update in Redis
		_, err = mfs.client.ZAdd(ctx, mfs.name, &redis.Z{
			Score:  float64(finalZScore.Int64()),
			Member: member,
		}).Result()

		if err != nil {
			return nil, err
		}

		return finalZScore, nil
	} else if err != nil {
		return nil, err
	}

	// Member exists, update scores
	currentBigZScore := new(big.Int).SetInt64(int64(currentZScore))
	scores := mfs.getFieldScores(currentBigZScore)

	// Update scores
	for fieldName, incValue := range fields {
		field := mfs.GetFieldByName(fieldName)
		if field == nil {
			return nil, fmt.Errorf("field %s not found", fieldName)
		}

		inc := new(big.Int).SetInt64(int64(incValue))
		inc.Mul(inc, field.multiplier)

		if field.UpdateType == Incremental {
			scores[field.position].Add(scores[field.position], inc)
		} else if field.UpdateType == Replace {
			scores[field.position] = new(big.Int).Add(field.defaultScore(), inc)
		} else {
			return nil, errors.New("unknown update type")
		}

		// Check range
		if scores[field.position].Sign() < 0 || scores[field.position].Cmp(field.maxAbsolute) > 0 {
			return nil, fmt.Errorf("score %v out of range for field %s", scores[field.position], field.Name)
		}
	}

	// Calculate new zscore
	finalZScore := mfs.scoresToZScore(scores)

	// Update in Redis
	_, err = mfs.client.ZAdd(ctx, mfs.name, &redis.Z{
		Score:  float64(finalZScore.Int64()),
		Member: member,
	}).Result()

	if err != nil {
		return nil, err
	}

	return finalZScore, nil
}

// GetRank returns the rank of a member in the sorted set.
func (mfs *MultiFieldSet) GetRank(ctx context.Context, member string) (int64, error) {
	return mfs.client.ZRank(ctx, mfs.name, member).Result()
}

// GetScores returns all field scores for a member.
func (mfs *MultiFieldSet) GetScores(ctx context.Context, member string) ([]fieldScore, error) {
	zscoreStr, err := mfs.client.ZScore(ctx, mfs.name, member).Result()
	if err == redis.Nil {
		// Member doesn't exist, return default scores
		scores := make([]fieldScore, len(mfs.fields))
		for i, field := range mfs.fields {
			scores[i] = fieldScore{
				Name:  field.Name,
				Score: field.defaultScore(),
			}
		}
		return scores, nil
	} else if err != nil {
		return nil, err
	}

	zscore := new(big.Int).SetInt64(int64(zscoreStr))
	return mfs.zscoreToAllFieldScores(zscore), nil
}

// zscoreToAllFieldScores converts a zscore to a slice of field scores.
func (mfs *MultiFieldSet) zscoreToAllFieldScores(zscore *big.Int) []fieldScore {
	scores := make([]fieldScore, len(mfs.fields))
	for i, field := range mfs.fields {
		fieldVal := mfs.extractFieldScore(field, zscore)

		// Reverse calculation for descending fields for display
		if field.Sort == Descending {
			fieldVal = new(big.Int).Sub(field.maxAbsolute, fieldVal)
		}

		scores[i] = fieldScore{
			Name:  field.Name,
			Score: fieldVal,
		}
	}
	return scores
}

// GetScoreForField returns the score for a specific field of a member.
func (mfs *MultiFieldSet) GetScoreForField(ctx context.Context, fieldName string, member string) (*big.Int, error) {
	field := mfs.GetFieldByName(fieldName)
	if field == nil {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}

	zscoreStr, err := mfs.client.ZScore(ctx, mfs.name, member).Result()
	if err == redis.Nil {
		// Member doesn't exist, return default score
		return field.defaultScore(), nil
	} else if err != nil {
		return nil, err
	}

	zscore := new(big.Int).SetInt64(int64(zscoreStr))
	fieldVal := mfs.extractFieldScore(field, zscore)

	// Reverse calculation for descending fields for display
	if field.Sort == Descending {
		fieldVal = new(big.Int).Sub(field.maxAbsolute, fieldVal)
	}

	return fieldVal, nil
}

// GetMembers returns members with their scores from the sorted set.
func (mfs *MultiFieldSet) GetMembers(ctx context.Context, limit, offset int64) ([]MemberScores, error) {
	results, err := mfs.client.ZRangeWithScores(ctx, mfs.name, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	members := make([]MemberScores, len(results))
	for i, z := range results {
		zscore := new(big.Int).SetInt64(int64(z.Score))
		members[i] = MemberScores{
			Member: z.Member.(string),
			Scores: mfs.zscoreToAllFieldScores(zscore),
		}
	}

	return members, nil
}

// GetTopMembers returns the top n members from the sorted set.
func (mfs *MultiFieldSet) GetTopMembers(ctx context.Context, limit int64) ([]MemberScores, error) {
	return mfs.GetMembers(ctx, limit, 0)
}

// GetMembersInRange returns members with scores within a range.
func (mfs *MultiFieldSet) GetMembersInRange(ctx context.Context, limit, offset int64, min, max string) ([]MemberScores, error) {
	// Convert strings to Redis range format
	if min == "" {
		min = "-inf"
	}
	if max == "" {
		max = "+inf"
	}

	opt := &redis.ZRangeBy{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  limit,
	}

	results, err := mfs.client.ZRangeByScoreWithScores(ctx, mfs.name, opt).Result()
	if err != nil {
		return nil, err
	}

	members := make([]MemberScores, len(results))
	for i, z := range results {
		zscore := new(big.Int).SetInt64(int64(z.Score))
		members[i] = MemberScores{
			Member: z.Member.(string),
			Scores: mfs.zscoreToAllFieldScores(zscore),
		}
	}

	return members, nil
}

// ResetMember resets a member's score to the default values.
func (mfs *MultiFieldSet) ResetMember(ctx context.Context, member string) error {
	_, err := mfs.client.ZAdd(ctx, mfs.name, &redis.Z{
		Score:  float64(mfs.defaultZScore.Int64()),
		Member: member,
	}).Result()
	return err
}

// GetCountInRange returns the count of members with scores within a range.
func (mfs *MultiFieldSet) GetCountInRange(ctx context.Context, min, max string) (int64, error) {
	return mfs.client.ZCount(ctx, mfs.name, min, max).Result()
}

// MaxScoreWithFields calculates the maximum zscore for given field limits.
func (mfs *MultiFieldSet) MaxScoreWithFields(limits map[string]float64) (*big.Int, error) {
	scores := make([]*big.Int, len(mfs.fields))

	// Initialize with default scores
	for i, field := range mfs.fields {
		scores[i] = field.defaultScore()
	}

	// Apply limits
	for fieldName, limit := range limits {
		field := mfs.GetFieldByName(fieldName)
		if field == nil {
			return nil, fmt.Errorf("field %s not found", fieldName)
		}

		inc := new(big.Int).SetInt64(int64(limit))
		inc.Mul(inc, field.multiplier)

		scores[field.position] = new(big.Int).Add(field.defaultScore(), inc)
	}

	return mfs.scoresToZScore(scores), nil
}

// CalculateScoresFromZScore is a utility method that converts a zscore to human-readable field scores
// This is useful for testing and debugging without needing Redis
func (mfs *MultiFieldSet) CalculateScoresFromZScore(zscore *big.Int) map[string]*big.Int {
	result := make(map[string]*big.Int)

	for _, field := range mfs.fields {
		fieldVal := mfs.extractFieldScore(field, zscore)

		// Reverse calculation for descending fields for display
		if field.Sort == Descending {
			fieldVal = new(big.Int).Sub(field.maxAbsolute, fieldVal)
		}

		result[field.Name] = fieldVal
	}

	return result
}
