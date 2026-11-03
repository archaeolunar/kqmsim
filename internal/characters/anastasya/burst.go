package anastasya

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

// ASSUMPTION: frames are placeholders.
var (
	burstNewWorldFrames []int
	burstDawnFrames     []int
)

const (
	burstNewWorldHitmark = 95
	burstDawnHitmark     = 88

	burstCD = 15 * 60

	burstNewWorldKey = "anastasya-to-a-new-world"
	burstNewWorldDur = 20 * 60
	burstDawnKey     = "anastasya-eternal-night"
	burstDawnDur     = 14 * 60

	burstStellarBonusKey = "anastasya-burst-stellar-bonus"
)

func init() {
	burstNewWorldFrames = frames.InitAbilSlice(112)
	burstNewWorldFrames[action.ActionSwap] = 104

	burstDawnFrames = frames.InitAbilSlice(106)
	burstDawnFrames[action.ActionAttack] = 98
}

func (c *char) Burst(p map[string]int) (action.Info, error) {
	if c.isGlory() {
		return c.burstDawn(), nil
	}
	return c.burstNewWorld(), nil
}

// To a New World
// Can be used when Anastasya possesses the "Benevolence" Blessing.
// Deals AoE Cryo DMG and freezes the duration of the Azure Star Crown.
// During this time, Stellar Glimmer reaction DMG dealt by all nearby party members
// is increased, and the Azure Star Crown deals one additional instance of AoE
// Cryo DMG considered DMG from the Stellar-Conduct reaction.
// When Anastasya switches to the off-field, she also seals herself.
func (c *char) burstNewWorld() action.Info {
	c.QueueCharTask(func() {
		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			Abil:       "To a New World",
			AttackTag:  attacks.AttackTagElementalBurst,
			ICDTag:     attacks.ICDTagNone,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeDefault,
			Element:    attributes.Cryo,
			Durability: 50,
			Mult:       burstNewWorld[c.TalentLvlBurst()],
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 8)
		c.Core.QueueAttack(ai, ap, 0, 0)

		c.AddStatus(burstNewWorldKey, burstNewWorldDur, false)
		c.selfSealedBurst = false
		c.freezeCrown(burstNewWorldDur)

		for _, char := range c.Core.Player.Chars() {
			char.AddReactBonusMod(character.ReactBonusMod{
				Base: modifier.NewBase(burstStellarBonusKey, burstNewWorldDur),
				Amount: func(ai info.AttackInfo) float64 {
					if !ai.AttackTag.IsStellar() {
						return 0
					}
					return burstStellarBonus
				},
			})
		}
	}, burstNewWorldHitmark)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(5)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstNewWorldFrames),
		AnimationLength: burstNewWorldFrames[action.InvalidAction],
		CanQueueAfter:   burstNewWorldFrames[action.ActionSwap],
		State:           action.BurstState,
	}
}

// When the Eternal Night Gives Way to Dawn
// Can be used when Anastasya possesses the "Glory" Blessing.
// Deals AoE Cryo DMG considered DMG from the Stellar-Conduct reaction, and freezes
// the duration of the Blessing. Anastasya seals up to three other party members.
func (c *char) burstDawn() action.Info {
	c.QueueCharTask(func() {
		lvl := c.TalentLvlBurst()
		ai := c.stellarAI("When the Eternal Night Gives Way to Dawn", burstDawnSSC[lvl], burstDawnSSw[lvl])
		ai.Mult *= c.c2GloryMult()
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 8)
		c.Core.QueueAttack(ai, ap, 0, 0)

		c.AddStatus(burstDawnKey, burstDawnDur, false)
		c.freezeGlory()
		c.annihilationStage = 0

		for _, char := range c.Core.Player.Chars() {
			if char.Index() == c.Index() {
				continue
			}
			c.seal(char.Index(), burstDawnDur)
		}
	}, burstDawnHitmark)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(5)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstDawnFrames),
		AnimationLength: burstDawnFrames[action.InvalidAction],
		CanQueueAfter:   burstDawnFrames[action.ActionAttack],
		State:           action.BurstState,
	}
}

func (c *char) freezeGlory() {
	if !c.isGlory() || c.gloryFrozen {
		return
	}
	c.gloryFrozen = true
	c.gloryFrozenRemaining = c.StatusDuration(gloryBlessingKey)
	c.AddStatus(gloryBlessingKey, gloryFrozenDur, true)
	c.Core.Log.NewEvent("anastasya glory blessing frozen", glog.LogCharacterEvent, c.Index()).
		Write("remaining", c.gloryFrozenRemaining)
}

func (c *char) unfreezeGlory() {
	if !c.gloryFrozen {
		return
	}
	c.gloryFrozen = false
	if c.isGlory() {
		c.AddStatus(gloryBlessingKey, c.gloryFrozenRemaining, true)
	}
	c.Core.Log.NewEvent("anastasya glory blessing unfrozen", glog.LogCharacterEvent, c.Index()).
		Write("remaining", c.gloryFrozenRemaining)
}
