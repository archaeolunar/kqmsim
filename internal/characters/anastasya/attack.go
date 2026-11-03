package anastasya

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// ASSUMPTION: all frames are placeholders until the character can be frame counted.
var (
	attackFrames   [][]int
	attackHitmarks = [][]int{{15}, {14}, {18, 24}, {22, 26, 30}, {30}}

	spearFrames   [][]int
	spearHitmarks = [][]int{{12}, {12, 20}, {18}, {16, 22, 28}, {18, 24, 32}}
	spearRadius   = []float64{2.5, 2.5, 3, 3, 3.5}
)

func init() {
	attackFrames = make([][]int, normalHitNum)
	attackFrames[0] = frames.InitNormalCancelSlice(15, 22)
	attackFrames[1] = frames.InitNormalCancelSlice(14, 24)
	attackFrames[2] = frames.InitNormalCancelSlice(24, 32)
	attackFrames[3] = frames.InitNormalCancelSlice(30, 38)
	attackFrames[4] = frames.InitNormalCancelSlice(30, 58)

	spearFrames = make([][]int, normalHitNum)
	spearFrames[0] = frames.InitNormalCancelSlice(12, 22)
	spearFrames[1] = frames.InitNormalCancelSlice(20, 30)
	spearFrames[2] = frames.InitNormalCancelSlice(18, 30)
	spearFrames[3] = frames.InitNormalCancelSlice(28, 40)
	spearFrames[4] = frames.InitNormalCancelSlice(32, 60)
}

func (c *char) Attack(p map[string]int) (action.Info, error) {
	// every Normal Attack reduces the Sealing of other party members (Glory burst)
	c.reduceOthersSealOnAttack()

	if c.isGlory() {
		return c.spearAttack()
	}

	travel, ok := p["travel"]
	if !ok {
		travel = 10
	}

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       fmt.Sprintf("Normal %v", c.NormalCounter),
		AttackTag:  attacks.AttackTagNormal,
		ICDTag:     attacks.ICDTagNone,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypePierce,
		Element:    attributes.Physical,
		Durability: 25,
		Mult:       attack[c.NormalCounter][c.TalentLvlAttack()] * c.c2GloryMult(),
	}

	for _, hitmark := range attackHitmarks[c.NormalCounter] {
		c.Core.QueueAttack(
			ai,
			combat.NewBoxHit(
				c.Core.Combat.Player(),
				c.Core.Combat.PrimaryTarget(),
				info.Point{Y: -0.5},
				0.1,
				1,
			),
			hitmark,
			hitmark+travel,
		)
	}

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, attackFrames),
		AnimationLength: attackFrames[c.NormalCounter][action.InvalidAction],
		CanQueueAfter:   attackHitmarks[c.NormalCounter][len(attackHitmarks[c.NormalCounter])-1],
		State:           action.NormalAttackState,
	}, nil
}

// Hold: Anastasya's Normal Attacks are converted into up to five consecutive
// attacks with the Sacred Spear of Protection - Degomeno, dealing AoE Cryo DMG
// that cannot be overridden by other Elemental Infusions.
// Radiance - Stellar Glimmer: The final instance of DMG dealt by each hit is
// instead converted into AoE Cryo DMG considered DMG from the corresponding
// Stellar Glimmer reaction.
func (c *char) spearAttack() (action.Info, error) {
	n := c.NormalCounter
	hitmarks := spearHitmarks[n]
	ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 1}, spearRadius[n])

	for i, hitmark := range hitmarks {
		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			Abil:       fmt.Sprintf("Sacred Spear %v", n),
			AttackTag:  attacks.AttackTagNormal,
			ICDTag:     attacks.ICDTagNormalAttack,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeSpear,
			Element:    attributes.Cryo,
			Durability: 25,
			Mult:       spear[n][c.TalentLvlSkill()],
		}
		if i == len(hitmarks)-1 {
			c.convertToStellar(&ai, spearSSC[n][c.TalentLvlSkill()], spearSSw[n][c.TalentLvlSkill()])
		}
		ai.Mult *= c.c2GloryMult()
		c.Core.QueueAttack(ai, ap, hitmark, hitmark)
	}

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, spearFrames),
		AnimationLength: spearFrames[n][action.InvalidAction],
		CanQueueAfter:   hitmarks[len(hitmarks)-1],
		State:           action.NormalAttackState,
	}, nil
}
