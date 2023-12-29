package types

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPositionDepthTooSmall = errors.New("position depth is too small")
)

type Depth struct {
	val uint64
}

func NewDepth(d uint64) Depth {
	return Depth{d}
}

func (d Depth) IsRoot() bool {
	return d.val == 0
}

func (d Depth) IsOdd() bool {
	return d.val%2 == 1
}

func (d Depth) AsUint64() uint64 {
	return d.val
}

func (d Depth) AsUint() uint {
	if d.val > math.MaxUint {
		panic(fmt.Sprintf("cannot convert %d to uint", d.val))
	}
	return uint(d.val)
}

func (d Depth) MaxGIndex() *big.Int {
	// TODO: Check maxint64
	return new(big.Int).Lsh(big.NewInt(1), uint(d.val))
}

func (d Depth) OneLevelShallower() Depth {
	if d.IsRoot() {
		panic("can't go shallower than root depth")
	}
	return NewDepth(d.val - 1)
}

func (d Depth) OneLevelDeeper() Depth {
	if d.val == math.MaxUint64 {
		panic("already at deeepest possible level")
	}
	return NewDepth(d.val + 1)
}

func (d Depth) DeeperThan(other Depth) bool {
	return d.val > other.val
}

func (d Depth) RelativeDepth(ancestor Depth) Depth {
	if ancestor.DeeperThan(d) {
		panic("can't calculate relative depth when ancestor is deeper")
	}
	return NewDepth(ancestor.val - d.val)
}

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        Depth
	indexAtDepth *big.Int
}

func NewPosition(depth Depth, indexAtDepth *big.Int) Position {
	return Position{
		depth:        depth,
		indexAtDepth: indexAtDepth,
	}
}

func NewPositionFromGIndex(x *big.Int) Position {
	depth := bigMSB(x)
	withoutMSB := new(big.Int).Not(new(big.Int).Lsh(big.NewInt(1), depth.AsUint()))
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
func (p Position) RelativeToAncestorAtDepth(ancestor Depth) (Position, error) {
	if ancestor.DeeperThan(p.depth) {
		return Position{}, ErrPositionDepthTooSmall
	}
	newPosDepth := p.depth.RelativeDepth(ancestor)
	newIndexAtDepth := new(big.Int).Mod(p.indexAtDepth, newPosDepth.MaxGIndex())
	return NewPosition(newPosDepth, newIndexAtDepth), nil
}

func (p Position) Depth() Depth {
	return p.depth
}

func (p Position) IndexAtDepth() *big.Int {
	if p.indexAtDepth == nil {
		return common.Big0
	}
	return p.indexAtDepth
}

func (p Position) IsRootPosition() bool {
	return p.depth.IsRoot() && common.Big0.Cmp(p.indexAtDepth) == 0
}

func (p Position) lshIndex(amount Depth) *big.Int {
	return new(big.Int).Lsh(p.IndexAtDepth(), amount.AsUint())
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
func (p Position) TraceIndex(maxDepth Depth) *big.Int {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	rd := maxDepth.RelativeDepth(p.Depth())
	rhs := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), rd.AsUint()), big.NewInt(1))
	ti := new(big.Int).Or(p.lshIndex(rd), rhs)
	return ti
}

// move returns a new position at the left or right child.
func (p Position) move(right bool) Position {
	return Position{
		depth:        p.depth.OneLevelDeeper(),
		indexAtDepth: new(big.Int).Or(p.lshIndex(NewDepth(1)), big.NewInt(int64(boolToInt(right)))),
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
		depth:        p.depth.OneLevelShallower(),
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

func (p Position) Print(maxDepth Depth) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.indexAtDepth, p.depth, p.TraceIndex(maxDepth))
}

func (p Position) ToGIndex() *big.Int {
	d64 := p.depth.AsUint64()
	if d64 > math.MaxUint {
		panic(fmt.Sprintf("cannot convert %d to uint", d64))
	}
	d := uint(d64)

	return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), d), p.IndexAtDepth())
}

// bigMSB returns the index of the most significant bit
func bigMSB(x *big.Int) Depth {
	if x.Cmp(big.NewInt(0)) == 0 {
		return NewDepth(0)
	}
	out := uint64(0)
	for ; x.Cmp(big.NewInt(0)) != 0; out++ {
		x = new(big.Int).Rsh(x, 1)
	}
	return NewDepth(out - 1)
}
