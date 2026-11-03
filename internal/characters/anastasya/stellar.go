package anastasya

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	stellarBonusKey = "anastasya-stellar-bonus"

	stellarConductText = " (Stellar-Conduct)"
	stellarSwirlText   = " (Stellar Swirl)"
)

type radianceState int

const (
	radianceNone radianceState = iota
	radianceStellarConduct
	radianceStellarSwirl
)

// A character can only be affected by one Radiance state at a time, with
// Radiance - Stellar-Conduct taking priority.
func (c *char) getRadiance() radianceState {
	if c.StatusIsActive(reactable.PolestarFieldKey) {
		return radianceStellarConduct
	}
	if c.StatusIsActive(reactable.SswKey) {
		return radianceStellarSwirl
	}
	return radianceNone
}

// stellarAI returns a Cryo AoE attack that is considered Stellar Glimmer reaction
// DMG. For attacks that are Stellar-Conduct by default, the Stellar Swirl version is
// used when Anastasya is in Radiance - Stellar Swirl.
func (c *char) stellarAI(abil string, ssc, ssw float64) info.AttackInfo {
	ai := info.AttackInfo{
		ActorIndex:       c.Index(),
		Abil:             abil + stellarConductText,
		AttackTag:        attacks.AttackTagDirectStellarConduct,
		ICDTag:           attacks.ICDTagNone,
		ICDGroup:         attacks.ICDGroupDefault,
		StrikeType:       attacks.StrikeTypeDefault,
		Element:          attributes.Cryo,
		Durability:       0,
		Mult:             ssc,
		IgnoreDefPercent: 1,
	}
	if c.getRadiance() == radianceStellarSwirl {
		ai.Abil = abil + stellarSwirlText
		ai.AttackTag = attacks.AttackTagDirectStellarSwirl
		ai.Mult = ssw
	}
	return ai
}

// convertToStellar converts ai into the corresponding Stellar Glimmer DMG if
// Anastasya is in a Radiance state. Returns false if no conversion happened.
func (c *char) convertToStellar(ai *info.AttackInfo, ssc, ssw float64) bool {
	switch c.getRadiance() {
	case radianceStellarConduct:
		ai.Abil += stellarConductText
		ai.AttackTag = attacks.AttackTagDirectStellarConduct
		ai.Mult = ssc
	case radianceStellarSwirl:
		ai.Abil += stellarSwirlText
		ai.AttackTag = attacks.AttackTagDirectStellarSwirl
		ai.Mult = ssw
	default:
		return false
	}
	ai.ICDTag = attacks.ICDTagNone
	ai.ICDGroup = attacks.ICDGroupDefault
	ai.Durability = 0
	ai.IgnoreDefPercent = 1
	return true
}

// Stellar Benediction, Untainted Rebirth
// While within the Polestar Field / within 8s after a party member triggers a
// Stellar Swirl reaction nearby, Anastasya enters Radiance - Stellar-Conduct /
// Radiance - Stellar Swirl.
// When party members trigger Superconduct / Cryo Swirl reactions, these reactions
// are converted into Stellar-Conduct / Stellar Swirl reactions.
// Additionally, every 100 ATK increases the base DMG of these reactions dealt by
// party members by 0.7%, up to 14%.
func (c *char) stellarInit() {
	c.Core.Flags.Custom[reactable.StellarConductEnableKey] = 1
	c.Core.Flags.Custom[reactable.StellarSwirlEnableKey] = 1

	bonus := func() float64 {
		return min(c.TotalAtk()/100.0*0.007, 0.14)
	}

	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellarDirect() {
			return
		}
		b := bonus()
		if c.Core.Flags.LogDebug {
			c.Core.Log.NewEvent("anastasya adding stellar base damage", glog.LogCharacterEvent, c.Index()).Write("bonus", b)
		}
		atk.Info.BaseDmgBonus += b
	}, stellarBonusKey)

	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		b := bonus()
		if c.Core.Flags.LogDebug {
			c.Core.Log.NewEvent("anastasya adding stellar base damage", glog.LogCharacterEvent, c.Index()).Write("bonus", b)
		}
		atk.Info.BaseDmgBonus += b
	}, stellarBonusKey+"-reaction")
}
