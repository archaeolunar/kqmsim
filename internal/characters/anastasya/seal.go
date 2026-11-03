package anastasya

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player"
)

// Sealing
// A special status applied to characters when Anastasya uses her Elemental Burst.
// Characters under the Sealing status cannot be switched onto the field.
// The initial duration of Sealing is the same as the duration of the
// corresponding Elemental Burst. Certain methods can reduce the remaining duration
// of Sealing.
const (
	annihilationReadyKey = "anastasya-annihilation-ready"
	triumphReadyKey      = "anastasya-triumph-ready"

	// ASSUMPTION: "for a certain period after Sealing ends" is not specified.
	unsealWindow = 10 * 60
	// Glory is redistributed until 8s after the Sealing ends.
	gloryShareDur = 8 * 60

	// ASSUMPTION: reduction amounts are not specified in the leak.
	// Benevolence: Stellar Glimmer DMG dealt by nearby party members reduces
	// Anastasya's own Sealing by 1s, at most once every 1s.
	selfSealReduction    = 1 * 60
	selfSealReductionICD = 1 * 60
	selfSealReductionKey = "anastasya-self-seal-reduction-icd"
	// Glory: every Normal Attack Anastasya performs reduces the Sealing of other
	// party members by 1s.
	othersSealReduction = 1 * 60

	triumphHitmark = 12
)

func (c *char) isSealed(idx int) bool {
	return c.Core.Player.ByIndex(idx).StatusIsActive(player.SealedStatusKey)
}

func (c *char) sealInit() {
	c.Core.Events.Subscribe(event.OnCharacterSwap, func(args ...any) {
		prev := args[0].(int)
		next := args[1].(int)

		// Benevolence: When Anastasya switches to the off-field, she seals herself.
		if prev == c.Index() && c.StatusIsActive(burstNewWorldKey) && !c.selfSealedBurst {
			c.selfSealedBurst = true
			c.seal(c.Index(), burstNewWorldDur)
		}

		// switching Anastasya onto the field after her Sealing ends triggers
		// Azure Star Decree - Triumph.
		if next == c.Index() && c.StatusIsActive(triumphReadyKey) {
			c.DeleteStatus(triumphReadyKey)
			c.triumph(1, "Azure Star Decree - Triumph")
		}

		if next != c.Index() {
			c.c6OnSwap()
		}
	}, "anastasya-seal-swap")

	// Benevolence: Whenever nearby party members deal Stellar Glimmer reaction DMG,
	// the remaining duration of her Sealing is reduced.
	c.Core.Events.Subscribe(event.OnEnemyDamage, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellar() {
			return
		}
		if !c.isSealed(c.Index()) {
			return
		}
		if c.StatusIsActive(selfSealReductionKey) {
			return
		}
		c.AddStatus(selfSealReductionKey, selfSealReductionICD, false)
		c.reduceSeal(c.Index(), selfSealReduction)
	}, "anastasya-self-seal-reduction")
}

// seal applies the Sealing status on the character at idx.
func (c *char) seal(idx, dur int) {
	if dur <= 0 {
		return
	}
	c.setSeal(idx, dur)
	c.Core.Log.NewEvent("character sealed", glog.LogCharacterEvent, idx).
		Write("by", c.Base.Key.String()).
		Write("duration", dur)
}

// setSeal (re)sets the remaining Sealing duration of the character at idx.
func (c *char) setSeal(idx, dur int) {
	ch := c.Core.Player.ByIndex(idx)
	c.sealSrc[idx]++
	src := c.sealSrc[idx]
	c.sealEnd[idx] = c.Core.F + dur
	ch.AddStatus(player.SealedStatusKey, dur, false)
	c.Core.Tasks.Add(func() {
		if c.sealSrc[idx] != src {
			return
		}
		c.onSealEnd(idx)
	}, dur)
}

func (c *char) reduceSeal(idx, amt int) {
	if !c.isSealed(idx) {
		return
	}
	rem := c.sealEnd[idx] - c.Core.F - amt
	c.Core.Log.NewEvent("sealing reduced", glog.LogCharacterEvent, idx).
		Write("amount", amt).
		Write("remaining", max(rem, 0))
	if rem > 0 {
		c.setSeal(idx, rem)
		return
	}
	c.sealSrc[idx]++
	c.Core.Player.ByIndex(idx).DeleteStatus(player.SealedStatusKey)
	c.onSealEnd(idx)
}

func (c *char) onSealEnd(idx int) {
	c.Core.Log.NewEvent("sealing ended", glog.LogCharacterEvent, idx)
	c.gloryShareUntil[idx] = c.Core.F + gloryShareDur

	if idx == c.Index() {
		// For a certain period after Sealing ends, switching Anastasya onto the
		// field triggers Azure Star Decree - Triumph.
		c.AddStatus(triumphReadyKey, unsealWindow, false)
		return
	}

	// For a certain period after Sealing ends, initially tapping the
	// corresponding character triggers Azure Star Decree - Annihilation.
	c.Core.Player.ByIndex(idx).AddStatus(annihilationReadyKey, unsealWindow, false)
	c.Core.Tasks.Add(c.checkGloryUnfreeze, unsealWindow+1)

	c.c6OnOthersSealEnd()
}

// reduceOthersSealOnAttack: Whenever Anastasya performs a Normal Attack, the
// remaining duration of the other party members' Sealing is reduced.
func (c *char) reduceOthersSealOnAttack() {
	for _, ch := range c.Core.Player.Chars() {
		if ch.Index() == c.Index() {
			continue
		}
		c.reduceSeal(ch.Index(), othersSealReduction)
	}
}

// annihilationReady returns the index of the first character that can trigger
// Azure Star Decree - Annihilation, or -1.
func (c *char) annihilationReady() int {
	for _, ch := range c.Core.Player.Chars() {
		if ch.Index() == c.Index() {
			continue
		}
		if ch.StatusIsActive(annihilationReadyKey) {
			return ch.Index()
		}
	}
	return -1
}

func (c *char) consumeAnnihilation(idx int) {
	c.Core.Player.ByIndex(idx).DeleteStatus(annihilationReadyKey)
}

// checkGloryUnfreeze ends the Glory freeze once no party member is sealed or
// waiting to unseal anymore (ASSUMPTION: the leak only says the 3rd stage of
// Annihilation ends the freeze; this is a fallback when not all stages are used).
func (c *char) checkGloryUnfreeze() {
	if !c.gloryFrozen {
		return
	}
	for _, ch := range c.Core.Player.Chars() {
		if ch.Index() == c.Index() {
			continue
		}
		if ch.StatusIsActive(player.SealedStatusKey) || ch.StatusIsActive(annihilationReadyKey) {
			return
		}
	}
	c.unfreezeGlory()
}

// Azure Star Decree - Triumph
// Deals AoE Cryo DMG to nearby opponents considered DMG from the Stellar-Conduct
// reaction (converted to the corresponding Stellar Glimmer reaction in Radiance).
func (c *char) triumph(mult float64, abil string) {
	c.Core.Tasks.Add(func() {
		lvl := c.TalentLvlBurst()
		ai := c.stellarAI(abil, triumphSSC[lvl]*mult, triumphSSw[lvl]*mult)
		ai.ICDTag = attacks.ICDTagNone
		ai.Element = attributes.Cryo
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 7)
		c.Core.QueueAttack(ai, ap, 0, 0)
	}, triumphHitmark)
}
