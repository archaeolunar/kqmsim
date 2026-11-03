package anastasya

import (
	"strings"

	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	c1Atk = 0.15
	c1EM  = 60

	c2BenevolenceKey = "anastasya-c2-benevolence"
	c2BenevolenceDur = 30 * 60
	c2GloryKey       = "anastasya-c2-glory"
	c2GloryDur       = 15 * 60
	c2StellarCD      = 0.6
	c2Elevation      = 0.05

	c6BenevolenceKey     = "anastasya-c6-benevolence"
	c6BenevolenceSwapKey = "anastasya-c6-benevolence-swap"
	c6GloryKey           = "anastasya-c6-glory"
	c6GloryTapKey        = "anastasya-c6-glory-tap"
	c6ICDKey             = "anastasya-c6-icd"
	c6Dur                = 30 * 60
	c6ICD                = 15 * 60
	c6Elevation          = 0.10
	c6SelfStellarBonus   = 0.20
	c6GloryAbil          = "Azure Star Decree - Annihilation (C6)"
)

// C1: To the Years Gone By, To the Heart That Never Ceased Burning
// Glory additionally increases the ATK of Cryo characters by 15% and the
// Elemental Mastery of characters of other Elemental Types by 60 (added in a1Init
// so the Glory instance math is shared). The Glory redistribution changes are
// handled in gloryUnits.

// C2: To the Harsh Winter, To Those Who Share Its Bitterness
// After using the Elemental Skill, depending on the Blessing:
//   - Benevolence: CRIT DMG of Stellar Glimmer reaction DMG dealt by all party
//     members is increased by 60% for 30s.
//   - Glory: Anastasya's Normal Attacks, When the Eternal Night Gives Way to Dawn
//     and Azure Star Decree - Annihilation deal 125% of their original DMG for 15s.
//
// Additionally, Stellar Glimmer reaction DMG dealt by all nearby party members is
// elevated by 5%.
func (c *char) c2Init() {
	if c.Base.Cons < 2 {
		return
	}

	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellar() {
			return
		}
		atk.Info.Elevation += c.stellarElevation()

		// crit for direct Stellar Glimmer DMG is rolled from the snapshot
		if atk.Info.AttackTag.IsStellarDirect() && c.StatusIsActive(c2BenevolenceKey) {
			atk.Snapshot.Stats[attributes.CD] += c2StellarCD
		}
	}, "anastasya-c2-c6-elevation")

	// Stellar Swirl reaction contributions roll crit individually
	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellarReact() {
			return
		}
		if !c.StatusIsActive(c2BenevolenceKey) {
			return
		}
		atk.Snapshot.Stats[attributes.CD] += c2StellarCD
	}, "anastasya-c2-reaction-cd")
}

func (c *char) stellarElevation() float64 {
	e := 0.0
	if c.Base.Cons >= 2 {
		e += c2Elevation
	}
	if c.StatusIsActive(c6BenevolenceKey) {
		e += c6Elevation
	}
	return e
}

func (c *char) c2OnSkill(glory bool) {
	if c.Base.Cons < 2 {
		return
	}
	if glory {
		c.AddStatus(c2GloryKey, c2GloryDur, true)
		return
	}
	c.AddStatus(c2BenevolenceKey, c2BenevolenceDur, true)
}

func (c *char) c2GloryMult() float64 {
	if c.Base.Cons < 2 || !c.StatusIsActive(c2GloryKey) {
		return 1
	}
	return 1.25
}

// C4: To the Tyrant, Let the Song of Defiance Be Sung
// Enhances Glory: The effects that increase CRIT DMG, ATK, and Elemental Mastery
// are increased to 130% of their original values.
func (c *char) c4GloryMult() float64 {
	if c.Base.Cons < 4 {
		return 1
	}
	return 1.3
}

// C6: Behold, the Pure Dawn Shall Rise
// After using the Elemental Skill, depending on the Blessing:
//   - Benevolence: The next time another character is switched onto the field, an
//     effect equivalent to Azure Star Decree - Triumph is triggered, dealing 250%
//     of its original DMG. Additionally, Stellar Glimmer reaction DMG dealt by all
//     nearby party members is elevated by 10%.
//   - Glory: The next time the Elemental Skill is Tapped, an effect equivalent to
//     the third stage of Azure Star Decree - Annihilation is triggered, dealing
//     200% of its original DMG. If other characters are not Sealed, Anastasya's
//     attack is considered to possess the Glory of all party members.
//     Additionally, Stellar Glimmer reaction DMG dealt by Anastasya is increased
//     by 20%.
//
// These effects last for 30s and can be triggered once every 15s. The effect will
// be removed when the Sealing status of other party members ends.
func (c *char) c6Init() {
	if c.Base.Cons < 6 {
		return
	}

	c.AddReactBonusMod(character.ReactBonusMod{
		Base: modifier.NewBase(c6GloryKey+"-bonus", -1),
		Amount: func(ai info.AttackInfo) float64 {
			if !ai.AttackTag.IsStellar() || !c.StatusIsActive(c6GloryKey) {
				return 0
			}
			return c6SelfStellarBonus
		},
	})

	// "considered to possess the Glory of all party members"
	m := make([]float64, attributes.EndStatType)
	c.AddAttackMod(character.AttackMod{
		Base: modifier.NewBase(c6GloryKey+"-all-glory", -1),
		Amount: func(atk *info.AttackEvent, _ info.Target) []float64 {
			if !strings.HasPrefix(atk.Info.Abil, c6GloryAbil) {
				return nil
			}
			if c.Base.Ascension < 1 || !c.StatusIsActive(a1Key) {
				return nil
			}
			for _, ch := range c.Core.Player.Chars() {
				if c.isSealed(ch.Index()) {
					return nil
				}
			}
			extra := float64(len(c.Core.Player.Chars())) - c.gloryUnits(c.Index())
			if extra <= 0 {
				return nil
			}
			mult := extra * c.c4GloryMult()
			clear(m)
			m[attributes.CD] = a1CD * mult
			if c.Base.Cons >= 1 {
				m[attributes.ATKP] = c1Atk * mult
			}
			return m
		},
	})
}

func (c *char) c6OnSkill(glory bool) {
	if c.Base.Cons < 6 {
		return
	}
	if c.StatusIsActive(c6ICDKey) {
		return
	}
	c.AddStatus(c6ICDKey, c6ICD, true)
	if glory {
		c.AddStatus(c6GloryKey, c6Dur, true)
		c.AddStatus(c6GloryTapKey, c6Dur, true)
		return
	}
	c.AddStatus(c6BenevolenceKey, c6Dur, true)
	c.AddStatus(c6BenevolenceSwapKey, c6Dur, true)
}

// c6OnTap is called when the Elemental Skill is tapped (not as Annihilation).
func (c *char) c6OnTap() {
	if c.Base.Cons < 6 || !c.StatusIsActive(c6GloryTapKey) {
		return
	}
	c.DeleteStatus(c6GloryTapKey)
	c.queueAnnihilationStage(2, 2, true)
}

func (c *char) c6OnSwap() {
	if c.Base.Cons < 6 || !c.StatusIsActive(c6BenevolenceSwapKey) {
		return
	}
	c.DeleteStatus(c6BenevolenceSwapKey)
	c.triumph(2.5, "Azure Star Decree - Triumph (C6)")
}

func (c *char) c6OnOthersSealEnd() {
	if c.Base.Cons < 6 {
		return
	}
	c.DeleteStatus(c6BenevolenceKey)
	c.DeleteStatus(c6BenevolenceSwapKey)
	c.DeleteStatus(c6GloryKey)
	c.DeleteStatus(c6GloryTapKey)
}
