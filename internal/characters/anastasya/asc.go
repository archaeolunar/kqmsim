package anastasya

import (
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	a1Key = "anastasya-a1-glory"
	a1Dur = 20 * 60
	a1CD  = 0.12
)

// Glory of the Radiant Night
// After Anastasya uses her Elemental Skill Judgment of the Pole Star, all party
// members gain Glory for 20s: CRIT DMG is increased by 12%.
// When a party member is Sealed, the Glory they possess is temporarily
// distributed equally among party members who are not Sealed until 8s after their
// Sealing ends.
func (c *char) a1Init() {
	if c.Base.Ascension < 1 {
		return
	}

	for _, ch := range c.Core.Player.Chars() {
		idx := ch.Index()
		isCryo := ch.Base.Element == attributes.Cryo
		m := make([]float64, attributes.EndStatType)
		ch.AddStatMod(character.StatMod{
			Base:         modifier.NewBase(a1Key, -1),
			AffectedStat: attributes.NoStat,
			Amount: func() []float64 {
				units := c.gloryUnits(idx)
				if units <= 0 {
					return nil
				}
				mult := units * c.c4GloryMult()
				clear(m)
				m[attributes.CD] = a1CD * mult
				if c.Base.Cons >= 1 {
					if isCryo {
						m[attributes.ATKP] = c1Atk * mult
					} else {
						m[attributes.EM] = c1EM * mult
					}
				}
				return m
			},
		})
	}
}

func (c *char) a1OnSkill() {
	if c.Base.Ascension < 1 {
		return
	}
	c.AddStatus(a1Key, a1Dur, true)
}

// gloryDistributing returns true if the Glory of the character at idx is currently
// being distributed to others.
func (c *char) gloryDistributing(idx int) bool {
	return c.isSealed(idx) || c.Core.F < c.gloryShareUntil[idx]
}

// gloryUnits returns how many instances of Glory the character at idx possesses.
func (c *char) gloryUnits(idx int) float64 {
	if !c.StatusIsActive(a1Key) {
		return 0
	}
	chars := c.Core.Player.Chars()

	units := 0.0
	// C1: distributing Glory equally will no longer cause the original
	// character's Glory to disappear.
	if !c.gloryDistributing(idx) || c.Base.Cons >= 1 {
		units = 1
	}
	if c.isSealed(idx) {
		return units
	}

	for _, other := range chars {
		j := other.Index()
		if j == idx || !c.gloryDistributing(j) {
			continue
		}
		// recipients: party members that are not Sealed (excluding the source)
		recipients := 0
		for _, r := range chars {
			if r.Index() != j && !c.isSealed(r.Index()) {
				recipients++
			}
		}
		if recipients == 0 {
			continue
		}
		share := 1.0
		// C1: When Anastasya is Sealed, the Glory granted to other characters is
		// increased to 300% of its original value.
		if j == c.Index() && c.Base.Cons >= 1 {
			share = 3
		}
		units += share / float64(recipients)
	}
	return units
}

// An Unfinished Requiem
// When Anastasya is in the party, Stellar Glimmer reactions triggered by all
// party members gain the following effects:
//   - Stellar-Conduct: Each time the Polestar Field grants a buff, its effects are
//     further increased. Additionally, when a Stellar Prism is created, the
//     Polestar Field immediately grants one instance of its buff.
//   - Stellar Swirl: Stellar Vortexes and Wandering Vortexes are upgraded to
//     Level 2 upon creation. This effect can trigger at most once every 1s.
//     Additionally, the base reaction multiplier of Stellar Swirl is increased by
//     15%.
//
// ASSUMPTIONS:
//   - "further increased" is modelled as +1 Polestar Field stack on every buff
//     refresh. The Stellar Prism part is not implemented (no generic Stellar Prism
//     exists in the sim).
//   - Stellar Vortex "Level 2" is modelled as starting the vortex at 3 stacks
//     (the first stack count that uses the larger detonation multiplier/radius).
//     Wandering Vortexes do not exist in the sim.
//   - The +15% base multiplier is applied multiplicatively to both the initial
//     Stellar Swirl hit and the vortex detonation.
func (c *char) a4Init() {
	if c.Base.Ascension < 4 {
		return
	}
	c.Core.Flags.Custom[reactable.PolestarFieldBonusStacksKey] = 1
	c.Core.Flags.Custom[reactable.StellarVortexMinStacksKey] = 3
	c.Core.Flags.Custom[reactable.StellarSwirlMultBonusKey] = 0.15
}
