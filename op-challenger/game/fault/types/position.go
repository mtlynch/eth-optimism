package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPositionDepthTooSmall = errors.New("position depth is too small")
)

type PositionDepth uint64

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        PositionDepth
	indexAtDepth *big.Int
}

func NewPosition(depth PositionDepth, indexAtDepth *big.Int) Position {
	return Position{
		depth:        depth,
		indexAtDepth: indexAtDepth,
	}
}

func NewPositionFromGIndex(x *big.Int) Position {
	depth := bigMSB(x)
	withoutMSB := new(big.Int).Not(new(big.Int).Lsh(big.NewInt(1), uint(depth)))
	indexAtDepth := new(big.Int).And(x, withoutMSB)
	return NewPosition(depth, indexAtDepth)
}

func (p Position) String() string {
	return fmt.Sprintf("Position(depth: %v, indexAtDepth: %v)", p.depth, p.indexAtDepth)
}

func (p Position) MoveRight() Position {
	return Position{
		depth:        p.depth,
		indexAtDepth: new(big.Int).Add(p.indexAtDepth, big.NewInt(1)),
	}
}

// RelativeToAncestorAtDepth returns a new position for a subtree.
// [ancestor] is the depth of the subtree root node.
func (p Position) RelativeToAncestorAtDepth(ancestor PositionDepth) (Position, error) {
	if ancestor > p.depth {
		return Position{}, ErrPositionDepthTooSmall
	}
	newPosDepth := p.depth - ancestor
	nodesAtDepth := 1 << newPosDepth
	newIndexAtDepth := new(big.Int).Mod(p.indexAtDepth, big.NewInt(int64(nodesAtDepth)))
	return NewPosition(newPosDepth, newIndexAtDepth), nil
}

func (p Position) Depth() PositionDepth {
	return p.depth
}

func (p Position) IndexAtDepth() *big.Int {
	if p.indexAtDepth == nil {
		return common.Big0
	}
	return p.indexAtDepth
}

func (p Position) IsRootPosition() bool {
	return p.depth == 0 && common.Big0.Cmp(p.indexAtDepth) == 0
}

func (p Position) lshIndex(amount PositionDepth) *big.Int {
	return new(big.Int).Lsh(p.IndexAtDepth(), uint(amount))
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
func (p Position) TraceIndex(maxDepth PositionDepth) *big.Int {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	rd := maxDepth - p.depth
	rhs := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(rd)), big.NewInt(1))
	ti := new(big.Int).Or(p.lshIndex(rd), rhs)
	return ti
}

// move returns a new position at the left or right child.
func (p Position) move(right bool) Position {
	return Position{
		depth:        p.depth + 1,
		indexAtDepth: new(big.Int).Or(p.lshIndex(1), big.NewInt(int64(boolToInt(right)))),
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func (p Position) parentIndexAtDepth() *big.Int {
	return new(big.Int).Div(p.IndexAtDepth(), big.NewInt(2))
}

func (p Position) RightOf(parent Position) bool {
	return p.parentIndexAtDepth().Cmp(parent.IndexAtDepth()) != 0
}

// parent return a new position that is the parent of this Position.
func (p Position) parent() Position {
	return Position{
		depth:        p.depth - 1,
		indexAtDepth: p.parentIndexAtDepth(),
	}
}

// Attack creates a new position which is the attack position of this one.
func (p Position) Attack() Position {
	return p.move(false)
}

// Defend creates a new position which is the defend position of this one.
func (p Position) Defend() Position {
	return p.parent().move(true).move(false)
}

func (p Position) Print(maxDepth PositionDepth) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.indexAtDepth, p.depth, p.TraceIndex(maxDepth))
}

func (p Position) ToGIndex() *big.Int {
	return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), uint(p.depth)), p.IndexAtDepth())
}

// bigMSB returns the index of the most significant bit
func bigMSB(x *big.Int) PositionDepth {
	if x.Cmp(big.NewInt(0)) == 0 {
		return 0
	}
	out := PositionDepth(0)
	for ; x.Cmp(big.NewInt(0)) != 0; out++ {
		x = new(big.Int).Rsh(x, 1)
	}
	return out - 1
}
