package types

import (
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPositionDepthTooSmall = errors.New("position depth is too small")
)

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        int
	indexAtDepth *big.Int
}

func NewPosition(depth int, indexAtDepth *big.Int) (Position, error) {
	log.Printf("creating new position with depth=%d, index=%s", depth, indexAtDepth) // DEBUG
	if depth < 0 {
		return Position{}, fmt.Errorf("position depth must be non-negative, got %d", depth)
	}
	if indexAtDepth == nil || indexAtDepth.Cmp(common.Big0) < 0 {
		return Position{}, fmt.Errorf("invalid indexAtDepth for position, got %s", indexAtDepth)
	}
	bigDepth := big.NewInt(int64(depth))
	depthToPowerOfTwo := bigDepth.Exp(big.NewInt(2), bigDepth, nil)
	maxIndex := depthToPowerOfTwo.Sub(depthToPowerOfTwo, big.NewInt(1))
	if indexAtDepth.Cmp(maxIndex) > 0 {
		return Position{}, fmt.Errorf("for depth of %d, expected maximum index of %s for position, got %s", depth, maxIndex, indexAtDepth)
	}

	return Position{
		depth:        depth,
		indexAtDepth: indexAtDepth,
	}, nil
}

// NewPositionFromGIndex creates a new Position given a generalized index,
// numbered like the following:
//
//			 1
//		    /  \
//	      2     3
//		 / \   / \
//		4   5 6   7
//
// See ../../../../specs/fault-dispute-game.md#game-tree
func NewPositionFromGIndex(x *big.Int) (Position, error) {
	depth := bigMSB(x)
	withoutMSB := new(big.Int).Not(new(big.Int).Lsh(big.NewInt(1), uint(depth)))
	indexAtDepth := new(big.Int).And(x, withoutMSB)
	return NewPosition(depth, indexAtDepth)
}

func (p Position) Equal(other Position) bool {
	return p.Depth() == other.Depth() && p.IndexAtDepth().Cmp(other.IndexAtDepth()) == 0
}

func (p Position) String() string {
	return fmt.Sprintf("Position(depth: %v, indexAtDepth: %v)", p.depth, p.indexAtDepth)
}

func (p Position) LeftChild() (Position, error) {
	log.Printf("getting left child of %s", p)                                           // DEBUG
	log.Printf("new index is %s", new(big.Int).Or(p.lshIndex(1), big.NewInt(int64(0)))) // DEBUG
	return NewPosition(p.depth+1, new(big.Int).Or(p.lshIndex(1), big.NewInt(int64(0))))
}

func (p Position) RightChild() (Position, error) {
	return NewPosition(p.depth+1, new(big.Int).Or(p.lshIndex(1), big.NewInt(int64(1))))
}

// RelativeToAncestorAtDepth returns a new position for a subtree.
// [ancestor] is the depth of the subtree root node.
func (p Position) RelativeToAncestorAtDepth(ancestor uint64) (Position, error) {
	if ancestor > uint64(p.depth) {
		return Position{}, ErrPositionDepthTooSmall
	}
	newPosDepth := uint64(p.depth) - ancestor
	nodesAtDepth := 1 << newPosDepth
	newIndexAtDepth := new(big.Int).Mod(p.indexAtDepth, big.NewInt(int64(nodesAtDepth)))
	log.Printf("relative to %s, creating new position at depth %d with depth %d and index %s", p, ancestor, newPosDepth, newIndexAtDepth)
	return NewPosition(int(newPosDepth), newIndexAtDepth)
}

func (p Position) Depth() int {
	return p.depth
}

func (p Position) IndexAtDepth() *big.Int {
	return p.indexAtDepth
}

func (p Position) IsRootPosition() bool {
	return p.depth == 0 && common.Big0.Cmp(p.indexAtDepth) == 0
}

func (p Position) lshIndex(amount int) *big.Int {
	return new(big.Int).Lsh(p.IndexAtDepth(), uint(amount))
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
func (p Position) TraceIndex(maxDepth int) *big.Int {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	rd := maxDepth - p.depth
	rhs := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(rd)), big.NewInt(1))
	ti := new(big.Int).Or(p.lshIndex(rd), rhs)
	return ti
}

func (p Position) parentIndexAtDepth() *big.Int {
	return new(big.Int).Div(p.IndexAtDepth(), big.NewInt(2))
}

func (p Position) RightOf(parent Position) bool {
	return p.parentIndexAtDepth().Cmp(parent.IndexAtDepth()) != 0
}

// parent return a new position that is the parent of this Position.
func (p Position) parent() (Position, error) {
	return NewPosition(p.depth-1, p.parentIndexAtDepth())
}

// Attack creates a new position which is the attack position of this one.
func (p Position) Attack() (Position, error) {
	return p.LeftChild()
}

// Defend creates a new position which is the defend position of this one.
func (p Position) Defend() (Position, error) {
	parent, err := p.parent()
	if err != nil {
		return Position{}, err
	}
	rc, err := parent.RightChild()
	if err != nil {
		return Position{}, err
	}
	return rc.LeftChild()
}

func (p Position) Print(maxDepth int) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.indexAtDepth, p.depth, p.TraceIndex(maxDepth))
}

func (p Position) ToGIndex() *big.Int {
	return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), uint(p.depth)), p.IndexAtDepth())
}

// bigMSB returns the index of the most significant bit
func bigMSB(x *big.Int) int {
	if x.Cmp(big.NewInt(0)) == 0 {
		return 0
	}
	out := 0
	for ; x.Cmp(big.NewInt(0)) != 0; out++ {
		x = new(big.Int).Rsh(x, 1)
	}
	return out - 1
}
