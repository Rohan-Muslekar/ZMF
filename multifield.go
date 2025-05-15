package zmultifield

import (
	"math"
	"math/big"
)

// multiField represents a field with bit allocation information for use in multi-field sets.
type multiField struct {
	Field
	bits        uint64
	position    int
	shiftValue  uint64
	mask        *big.Int
	isMain      bool
	maxAbsolute *big.Int
	multiplier  *big.Int // 1 for ascending, -1 for descending
}

// newMultiField creates a new multiField from a Field definition.
func newMultiField(f Field) *multiField {
	mf := &multiField{
		Field:     f,
		position:  0,
		shiftValue: 0,
	}

	// Determine the number of bits needed
	if math.IsInf(float64(f.MaxValue), 1) {
		mf.bits = 53 // Max safe integer bits in JavaScript (we'll use the same limit)
		mf.isMain = true
	} else {
		mf.bits = BitCount(f.MaxValue)
		mf.isMain = false
	}

	// Set the mask
	mf.mask = MaxBin(mf.bits)
	
	// Set max absolute value
	if mf.isMain {
		// Similar to JavaScript's MAX_SAFE_INTEGER
		mf.maxAbsolute = new(big.Int).SetUint64(1<<53 - 1)
	} else {
		mf.maxAbsolute = new(big.Int).Set(mf.mask)
	}

	// Set multiplier based on sort order
	if f.Sort == Descending {
		mf.multiplier = big.NewInt(-1)
	} else {
		mf.multiplier = big.NewInt(1)
	}

	return mf
}

// defaultScore returns the default score for the field based on sort order.
func (mf *multiField) defaultScore() *big.Int {
	if mf.Sort == Descending {
		return new(big.Int).Set(mf.maxAbsolute)
	}
	return big.NewInt(0)
}

// updateTypeName returns the string representation of the update type.
func (mf *multiField) updateTypeName() string {
	switch mf.UpdateType {
	case Incremental:
		return "INCREMENTAL"
	case Replace:
		return "REPLACE"
	default:
		return "UNKNOWN"
	}
}

// setIndex sets the position and bit shift values for the field.
func (mf *multiField) setIndex(position int, shiftValue uint64) {
	mf.position = position
	mf.shiftValue = shiftValue

	// Shift the mask
	mf.mask = new(big.Int).Lsh(mf.mask, uint(shiftValue))

	// Adjust for main field
	if mf.isMain {
		// If main field and mask exceeds 53 bits, adjust
		if mf.mask.BitLen() > 53 {
			// Get the 53 least significant bits
			mf.mask = MaxBin(53)
			mf.mask = new(big.Int).Lsh(mf.mask, uint(shiftValue))
		}
		
		mf.bits = 53 - mf.shiftValue
		mf.maxAbsolute = MaxBin(mf.bits)
	}
}

// getInfo returns the field information.
func (mf *multiField) getInfo() FieldInfo {
	return FieldInfo{
		Name:        mf.Name,
		Sort:        mf.Sort,
		UpdateType:  mf.updateTypeName(),
		MaxValue:    mf.MaxValue,
		Bits:        mf.bits,
		ShiftValue:  mf.shiftValue,
		Mask:        mf.mask,
		IsMain:      mf.isMain,
		DefaultScore: mf.defaultScore(),
		Position:    mf.position,
		MaxAbsolute: mf.maxAbsolute,
	}
}
