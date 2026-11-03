package anastasya_test

import (
	"context"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/pkg/gcs/ast"
	"github.com/genshinsim/gcsim/pkg/simulator"
)

const team = `
anastasya char lvl=90/90 cons=%CONS% talent=9,9,9;
anastasya add weapon="favoniuswarbow" refine=3 lvl=90/90;
anastasya add set="noblesseoblige" count=4;
anastasya add stats hp=4780 atk=311 atk%=0.466 cryo%=0.466 cr=0.311;
anastasya add stats atk%=0.4 cr=0.3 cd=0.6 er=0.2;

valeriy char lvl=90/90 cons=0 talent=9,9,9;
valeriy add weapon="harbingerofdawn" refine=5 lvl=90/90;
valeriy add set="gladiatorsfinale" count=4;
valeriy add stats hp=4780 atk=311 atk%=0.466 electro%=0.466 cr=0.311;

sucrose char lvl=90/90 cons=6 talent=9,9,9;
sucrose add weapon="sacrificialfragments" refine=5 lvl=90/90;
sucrose add set="viridescentvenerer" count=4;
sucrose add stats hp=4780 atk=311 em=561;

bennett char lvl=90/90 cons=6 talent=9,9,9;
bennett add weapon="favoniussword" refine=5 lvl=90/90;
bennett add set="noblesseoblige" count=4;
bennett add stats hp=4780 atk=311 er=0.518 pyro%=0.466 cr=0.311;

options iteration=5 duration=60 swap_delay=12;
target lvl=100 resist=0.1 radius=2 pos=0,2.4;
energy every interval=480,720 amount=1;
active anastasya;
`

const benevolenceRotation = `
while 1 {
  anastasya skill;
  anastasya burst;
  valeriy skill, burst;
  sucrose skill, attack:2;
  bennett skill, burst;
  valeriy attack:4, skill;
  anastasya attack:4;
}
`

const gloryRotation = `
while 1 {
  anastasya skill[hold=1];
  anastasya burst;
  anastasya attack:12;
  anastasya skill;
  anastasya skill;
  anastasya skill;
  valeriy skill, burst;
  sucrose skill, attack:2;
  bennett skill, burst;
  valeriy attack:4, skill;
}
`

func runSim(t *testing.T, cons, rotation string) map[string]float64 {
	t.Helper()
	cfg := strings.ReplaceAll(team, "%CONS%", cons) + rotation
	file := ast.NewFile()
	simcfg, gcsl, err := simulator.Parse(file, cfg)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	seeds := make([]int64, simcfg.Settings.Iterations)
	for i := range seeds {
		seeds[i] = int64(i + 1)
	}
	res, err := simulator.RunWithSeededConfig(context.Background(), file, cfg, simcfg, gcsl, seeds)
	if err != nil {
		t.Fatalf("sim error: %v", err)
	}
	sources := map[string]float64{}
	for k, v := range res.Statistics.SourceDps[0].Sources {
		if v.Mean != nil {
			sources[k] = *v.Mean
		}
	}
	return sources
}

func hasSource(sources map[string]float64, prefix string) bool {
	for k, v := range sources {
		if strings.HasPrefix(k, prefix) && v > 0 {
			return true
		}
	}
	return false
}

func TestBenevolence(t *testing.T) {
	for _, cons := range []string{"0", "6"} {
		sources := runSim(t, cons, benevolenceRotation)
		for _, want := range []string{
			"Judgment of the Pole Star (Tap)",
			"Azure Star Crown",
			"To a New World",
			"Azure Star Crown (Burst)",
			"Azure Star Decree - Triumph",
		} {
			if !hasSource(sources, want) {
				t.Errorf("c%v: expected damage from %q, got %v", cons, want, sources)
			}
		}
		if cons == "6" && !hasSource(sources, "Azure Star Decree - Triumph (C6)") {
			t.Errorf("c6: expected C6 Triumph damage, got %v", sources)
		}
	}
}

func TestGlory(t *testing.T) {
	for _, cons := range []string{"0", "6"} {
		sources := runSim(t, cons, gloryRotation)
		for _, want := range []string{
			"Judgment of the Pole Star (Hold)",
			"Sacred Spear",
			"When the Eternal Night Gives Way to Dawn",
			"Azure Star Decree - Annihilation (1st Stage)",
			"Azure Star Decree - Annihilation (2nd Stage)",
			"Azure Star Decree - Annihilation (3rd Stage)",
		} {
			if !hasSource(sources, want) {
				t.Errorf("c%v: expected damage from %q, got %v", cons, want, sources)
			}
		}
	}
}
