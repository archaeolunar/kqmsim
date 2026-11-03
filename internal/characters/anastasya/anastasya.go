// Package anastasya implements Anastasya, a 5* Cryo bow user from Snezhnaya.
//
// The kit is implemented from a pre-release leak. Base stats, frames, particle
// counts and a number of mechanic values (Sealing reduction amounts, unsealing
// windows, A4 Polestar Field bonus magnitude) were not in the leak and are
// placeholders; they are marked with "ASSUMPTION" comments.
package anastasya

import (
	"errors"
	"fmt"
	"slices"

	tmpl "github.com/genshinsim/gcsim/internal/template/character"
	"github.com/genshinsim/gcsim/pkg/core"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/keys"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/gcs/validation"
)

func init() {
	core.RegisterCharFunc(keys.Anastasya, NewChar)
	paramsFor := map[action.Action][]string{
		action.ActionAttack: {"travel"},
		action.ActionAim:    {"hold", "travel", "weakspot"},
		action.ActionSkill:  {"hold"},
	}
	validation.RegisterCharParamValidationFunc(keys.Anastasya, func(a action.Action, keys []string) error {
		valid, ok := paramsFor[a]
		if !ok {
			return nil
		}
		for _, v := range keys {
			if !slices.Contains(valid, v) {
				return fmt.Errorf("key %v is invalid for action %v", v, a)
			}
		}
		return nil
	})
}

const normalHitNum = 5

type char struct {
	*tmpl.Character

	// Azure Star Crown (tap skill)
	crownSrc    int
	crownExpiry int

	// Glory blessing freeze (Eternal Night burst)
	gloryFrozen          bool
	gloryFrozenRemaining int

	// Sealing
	sealSrc         [info.MaxChars]int
	sealEnd         [info.MaxChars]int
	gloryShareUntil [info.MaxChars]int
	selfSealedBurst bool

	// Azure Star Decree - Annihilation
	annihilationStage int
}

func NewChar(s *core.Core, w *character.CharWrapper, _ info.CharacterProfile) error {
	c := char{}
	c.Character = tmpl.NewWithWrapper(s, w)

	c.EnergyMax = 50
	c.NormalHitNum = normalHitNum
	c.SkillCon = 3
	c.BurstCon = 5

	c.crownSrc = -1
	for i := range c.gloryShareUntil {
		c.gloryShareUntil[i] = -1
	}

	w.Character = &c

	return nil
}

func (c *char) Init() error {
	c.stellarInit()
	c.sealInit()
	c.a1Init()
	c.a4Init()
	c.c2Init()
	c.c6Init()
	return nil
}

func (c *char) ActionReady(a action.Action, p map[string]int) (bool, action.Failure) {
	// unsealing Elemental Skill replaces tapping the skill and ignores its cooldown
	if a == action.ActionSkill && p["hold"] == 0 && c.annihilationReady() >= 0 {
		return true, action.NoFailure
	}
	return c.Character.ActionReady(a, p)
}

func (c *char) NextQueueItemIsValid(k keys.Char, a action.Action, p map[string]int) error {
	if c.isGlory() {
		switch a {
		case action.ActionAim, action.ActionCharge:
			return errors.New("anastasya cannot perform charged attacks under the Glory blessing")
		}
	}
	return c.Character.NextQueueItemIsValid(k, a, p)
}

func (c *char) Condition(fields []string) (any, error) {
	switch fields[0] {
	case "glory":
		return c.isGlory(), nil
	case "benevolence":
		return !c.isGlory(), nil
	case "crown":
		return c.crownActive(), nil
	case "annihilation":
		return c.annihilationReady() >= 0, nil
	case "triumph":
		return c.StatusIsActive(triumphReadyKey), nil
	case "sealed":
		return c.isSealed(c.Index()), nil
	default:
		return c.Character.Condition(fields)
	}
}

func (c *char) AnimationStartDelay(k info.AnimationDelayKey) int {
	if k == info.AnimationXingqiuN0StartDelay {
		return 10
	}
	return c.Character.AnimationStartDelay(k)
}
