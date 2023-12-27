package types_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const testMaxDepth = 3

func createTestClaims() (types.Claim, types.Claim, types.Claim, types.Claim) {
	// root & middle are from the trace "abcdexyz"
	// top & bottom are from the trace  "abcdefgh"
	root := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: types.NewPosition(0, common.Big0),
		},
		// Root types.Claim has no parent
	}
	top := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: types.NewPosition(1, common.Big0),
		},
		ContractIndex:       1,
		ParentContractIndex: root.ContractIndex,
	}
	middle := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: types.NewPosition(2, big.NewInt(2)),
		},
		ContractIndex:       2,
		ParentContractIndex: top.ContractIndex,
	}

	bottom := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: types.NewPosition(3, big.NewInt(4)),
		},
		ContractIndex:       3,
		ParentContractIndex: middle.ContractIndex,
	}

	return root, top, middle, bottom
}

func TestIsDuplicate(t *testing.T) {
	root, top, middle, bottom := createTestClaims()
	g := types.NewGameState([]types.Claim{root, top}, testMaxDepth)

	// Root + Top should be duplicates
	require.True(t, g.IsDuplicate(root))
	require.True(t, g.IsDuplicate(top))

	// Middle + Bottom should not be a duplicate
	require.False(t, g.IsDuplicate(middle))
	require.False(t, g.IsDuplicate(bottom))
}

func TestGame_Claims(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	expected := []types.Claim{root, top, middle, bottom}
	g := types.NewGameState(expected, testMaxDepth)

	// Validate types.Claim pairs.
	actual := g.Claims()
	require.ElementsMatch(t, expected, actual)
}

func TestGame_DefendsParent(t *testing.T) {
	tests := []struct {
		name          string
		claimGIndex   *big.Int
		parentGIndex  *big.Int
		defendsParent bool
	}{
		{
			name:          "LeftChildDoesntDefend",
			claimGIndex:   big.NewInt(2),
			parentGIndex:  big.NewInt(1),
			defendsParent: false,
		},
		{
			name:          "RightChildDoesntDefend",
			claimGIndex:   big.NewInt(3),
			parentGIndex:  big.NewInt(1),
			defendsParent: false,
		},
		{
			name:          "GrandchildDoesntDefend",
			claimGIndex:   big.NewInt(4),
			parentGIndex:  big.NewInt(1),
			defendsParent: false,
		},
		{
			name:          "SecondGrandchildDoesntDefend",
			claimGIndex:   big.NewInt(5),
			parentGIndex:  big.NewInt(1),
			defendsParent: false,
		},
		{
			name:          "RightLeftChildDefends",
			claimGIndex:   big.NewInt(6),
			parentGIndex:  big.NewInt(1),
			defendsParent: true,
		},
		{
			name:          "SubThirdChildDefends",
			claimGIndex:   big.NewInt(7),
			parentGIndex:  big.NewInt(1),
			defendsParent: true,
		},
		{
			name:          "RootDoesntDefend",
			claimGIndex:   big.NewInt(0),
			parentGIndex:  nil,
			defendsParent: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := []types.Claim{}
			if test.parentGIndex != nil {
				claims = append(claims, types.Claim{
					ClaimData: types.ClaimData{
						Position: types.NewPositionFromGIndex(test.parentGIndex),
					},
					ContractIndex: len(claims),
				})
			}
			claims = append(claims, types.Claim{
				ClaimData: types.ClaimData{
					Position: types.NewPositionFromGIndex(test.claimGIndex),
				},
				ContractIndex:       len(claims),
				ParentContractIndex: 0,
			})
			/*log.Printf("test=%+v", test.name)
			log.Printf("claimGIndex=%s", test.claimGIndex.String())
			if test.parentGIndex != nil {
				log.Printf("parentGIndex=%s", test.parentGIndex.String())
			} else {
				log.Printf("parentGIndex=nil")
			}
			log.Printf("claims=%+v", claims)*/
			game := types.NewGameState(claims, testMaxDepth)
			require.Equal(t, test.defendsParent, game.DefendsParent(claims[len(claims)-1]))
		})
	}
}
