package anastasya

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// ASSUMPTION: frames and particle counts are placeholders.
var (
	skillTapFrames     []int
	skillHoldFrames    []int
	annihilationFrames [3][]int
)

const (
	skillTapHitmark  = 20
	skillHoldHitmark = 42

	skillCD = 15 * 60

	gloryBlessingKey = "anastasya-glory-blessing"
	gloryBlessingDur = 15 * 60
	// used while the Glory blessing is frozen by the Eternal Night burst
	gloryFrozenDur = 3600 * 60

	crownDur      = 20 * 60
	crownInterval = 2 * 60

	particleICDKey = "anastasya-particle-icd"
	particleCount  = 4 // ASSUMPTION
)

var annihilationHitmarks = [3][]int{
	{8, 14, 20, 26, 32, 42}, // 5x Elemental Skill DMG + Stellar-Conduct finisher
	{26},
	{36},
}

func init() {
	skillTapFrames = frames.InitAbilSlice(40)
	skillTapFrames[action.ActionSwap] = 32

	skillHoldFrames = frames.InitAbilSlice(64)
	skillHoldFrames[action.ActionAttack] = 56
	skillHoldFrames[action.ActionSwap] = 58

	annihilationFrames[0] = frames.InitAbilSlice(52)
	annihilationFrames[1] = frames.InitAbilSlice(40)
	annihilationFrames[2] = frames.InitAbilSlice(52)
}

func (c *char) isGlory() bool {
	return c.StatusIsActive(gloryBlessingKey)
}

func (c *char) crownActive() bool {
	return c.crownSrc >= 0 && c.Core.F < c.crownExpiry
}

func (c *char) Skill(p map[string]int) (action.Info, error) {
	hold := p["hold"]
	if hold == 0 {
		// tapping the skill while an unsealed character is waiting triggers the
		// special unsealing Elemental Skill instead
		if idx := c.annihilationReady(); idx >= 0 {
			return c.annihilation(idx), nil
		}
		return c.skillTap(), nil
	}
	return c.skillHold(), nil
}

// Tap: Grants the "Benevolence" Blessing and casts the Azure Star Crown.
func (c *char) skillTap() action.Info {
	// switching to Benevolence ends Glory
	c.DeleteStatus(gloryBlessingKey)
	c.gloryFrozen = false

	c.QueueCharTask(func() {
		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			Abil:       "Judgment of the Pole Star (Tap)",
			AttackTag:  attacks.AttackTagElementalArt,
			ICDTag:     attacks.ICDTagNone,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeDefault,
			Element:    attributes.Cryo,
			Durability: 25,
			Mult:       skillTap[c.TalentLvlSkill()],
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 6)
		c.Core.QueueAttack(ai, ap, 0, 0, c.particleCB)

		c.summonCrown()
	}, skillTapHitmark)

	c.c6OnTap()
	c.onSkillCast(false)
	c.SetCDWithDelay(action.ActionSkill, skillCD, skillTapHitmark-2)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillTapFrames),
		AnimationLength: skillTapFrames[action.InvalidAction],
		CanQueueAfter:   skillTapFrames[action.ActionSwap],
		State:           action.SkillState,
	}
}

// Hold: Grants the "Glory" Blessing; Normal Attacks are converted into Sacred
// Spear attacks and Charged Attacks cannot be performed.
func (c *char) skillHold() action.Info {
	c.QueueCharTask(func() {
		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			Abil:       "Judgment of the Pole Star (Hold)",
			AttackTag:  attacks.AttackTagElementalArt,
			ICDTag:     attacks.ICDTagNone,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeDefault,
			Element:    attributes.Cryo,
			Durability: 25,
			Mult:       skillHold[c.TalentLvlSkill()],
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 6)
		c.Core.QueueAttack(ai, ap, 0, 0, c.particleCB)

		c.gloryFrozen = false
		c.AddStatus(gloryBlessingKey, gloryBlessingDur, true)
	}, skillHoldHitmark)

	c.onSkillCast(true)
	c.SetCDWithDelay(action.ActionSkill, skillCD, skillHoldHitmark-2)

	return action.Info{
		Frames:          frames.NewAbilFunc(skillHoldFrames),
		AnimationLength: skillHoldFrames[action.InvalidAction],
		CanQueueAfter:   skillHoldFrames[action.ActionAttack],
		State:           action.SkillState,
	}
}

// onSkillCast handles effects that trigger on using Judgment of the Pole Star
// (not the unsealing skill).
func (c *char) onSkillCast(glory bool) {
	c.a1OnSkill()
	c.c2OnSkill(glory)
	c.c6OnSkill(glory)
}

func (c *char) particleCB(a info.AttackCB) {
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if c.StatusIsActive(particleICDKey) {
		return
	}
	c.AddStatus(particleICDKey, 0.5*60, true)
	c.Core.QueueParticle(c.Base.Key.String(), particleCount, attributes.Cryo, c.ParticleDelay)
}

// The Azure Star Crown follows the active character and deals two instances of
// AoE Cryo DMG to nearby opponents every 2s.
// Radiance - Stellar Glimmer: The second instance is instead converted into AoE
// Cryo DMG considered DMG from the corresponding Stellar Glimmer reaction.
func (c *char) summonCrown() {
	src := c.Core.F
	c.crownSrc = src
	c.crownExpiry = c.Core.F + crownDur
	// if the burst is active the crown's duration is frozen
	if c.StatusIsActive(burstNewWorldKey) {
		c.crownExpiry += c.StatusDuration(burstNewWorldKey)
	}
	c.Core.Tasks.Add(func() { c.crownTick(src) }, crownInterval)
}

func (c *char) crownTick(src int) {
	if c.crownSrc != src {
		return
	}
	if c.Core.F > c.crownExpiry {
		return
	}

	ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 5)
	lvl := c.TalentLvlSkill()

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Azure Star Crown",
		AttackTag:  attacks.AttackTagElementalArt,
		ICDTag:     attacks.ICDTagElementalArt,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       crown[lvl],
	}
	c.Core.QueueAttack(ai, ap, 0, 0)

	second := ai
	c.convertToStellar(&second, crownSSC[lvl], crownSSw[lvl])
	c.Core.QueueAttack(second, ap, 6, 6)

	// To a New World: one additional instance considered Stellar-Conduct DMG
	if c.StatusIsActive(burstNewWorldKey) {
		blvl := c.TalentLvlBurst()
		extra := c.stellarAI("Azure Star Crown (Burst)", burstCrownSSC[blvl], burstCrownSSw[blvl])
		c.Core.QueueAttack(extra, ap, 12, 12)
	}

	c.Core.Tasks.Add(func() { c.crownTick(src) }, crownInterval)
}

// freezeCrown keeps the crown from expiring for dur frames.
func (c *char) freezeCrown(dur int) {
	if !c.crownActive() {
		return
	}
	c.crownExpiry += dur
	c.Core.Log.NewEvent("azure star crown frozen", glog.LogCharacterEvent, c.Index()).Write("expiry", c.crownExpiry)
}

// Azure Star Decree - Annihilation: consists of three stages performed in
// sequence when initially tapping the corresponding character or tapping the
// Elemental Skill.
//
// In the sim this is triggered by "anastasya skill" while any party member is
// in its unsealing window.
func (c *char) annihilation(idx int) action.Info {
	c.consumeAnnihilation(idx)
	stage := c.annihilationStage
	c.annihilationStage = min(c.annihilationStage+1, 2)

	c.Core.Log.NewEvent("anastasya azure star decree - annihilation", glog.LogCharacterEvent, c.Index()).
		Write("stage", stage+1).
		Write("unsealed_char", idx)

	c.queueAnnihilationStage(stage, 1, false)

	if stage == 2 {
		// ends the freezing of the Blessing
		c.QueueCharTask(c.unfreezeGlory, annihilationHitmarks[2][0])
	}

	f := annihilationFrames[stage]
	return action.Info{
		Frames:          frames.NewAbilFunc(f),
		AnimationLength: f[action.InvalidAction],
		CanQueueAfter:   annihilationHitmarks[stage][len(annihilationHitmarks[stage])-1],
		State:           action.SkillState,
	}
}

// queueAnnihilationStage queues the attacks for stage (0 indexed), scaled by mult.
func (c *char) queueAnnihilationStage(stage int, mult float64, c6 bool) {
	lvl := c.TalentLvlBurst()
	mult *= c.c2GloryMult()

	switch stage {
	case 0:
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 1}, 4)
		for _, h := range annihilationHitmarks[0][:5] {
			ai := info.AttackInfo{
				ActorIndex: c.Index(),
				Abil:       "Azure Star Decree - Annihilation (1st Stage)",
				AttackTag:  attacks.AttackTagElementalArt,
				ICDTag:     attacks.ICDTagElementalArt,
				ICDGroup:   attacks.ICDGroupDefault,
				StrikeType: attacks.StrikeTypeSpear,
				Element:    attributes.Cryo,
				Durability: 25,
				Mult:       annihilation1[lvl] * mult,
			}
			c.QueueCharTask(func() { c.Core.QueueAttack(ai, ap, 0, 0) }, h)
		}
		ai := c.stellarAI("Azure Star Decree - Annihilation (1st Stage)", annihilation1SSC[lvl], annihilation1SSw[lvl])
		ai.Mult *= mult
		c.QueueCharTask(func() { c.Core.QueueAttack(ai, ap, 0, 0) }, annihilationHitmarks[0][5])
	case 1:
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 3}, 5)
		ai := c.stellarAI("Azure Star Decree - Annihilation (2nd Stage)", annihilation2SSC[lvl], annihilation2SSw[lvl])
		ai.Mult *= mult
		c.QueueCharTask(func() { c.Core.QueueAttack(ai, ap, 0, 0) }, annihilationHitmarks[1][0])
	case 2:
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.PrimaryTarget(), nil, 6)
		abil := "Azure Star Decree - Annihilation (3rd Stage)"
		if c6 {
			abil = c6GloryAbil
		}
		ai := c.stellarAI(abil, annihilation3SSC[lvl], annihilation3SSw[lvl])
		ai.Mult *= mult
		c.QueueCharTask(func() { c.Core.QueueAttack(ai, ap, 0, 0) }, annihilationHitmarks[2][0])
	}
}
