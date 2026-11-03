package anastasya

// PLACEHOLDER TALENT TABLES
//
// Anastasya has not been datamined yet. The leak only gives a single talent level
// per ability (Lv.1 for Normal Attack and Elemental Skill, Lv.10 for Elemental
// Burst). The full 15 level tables below are extrapolated from those values with
// the standard Genshin talent scaling curves:
//   - physicalScaling: Normal/Aimed/Plunge (physical style) scaling
//   - elementalScaling: Fully Charged Aimed Shot, Elemental Skill and Burst scaling
// Replace this file with pipeline output (zz_anastasya.dm.go) once datamine is
// available.

var (
	physicalScaling  = []float64{1, 1.0814, 1.1628, 1.2791, 1.3605, 1.4535, 1.5814, 1.7093, 1.8372, 1.9767, 2.1163, 2.2558, 2.3953, 2.5349, 2.6744}
	elementalScaling = []float64{1, 1.075, 1.15, 1.25, 1.325, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0, 2.125, 2.25, 2.375}
)

// fromLv1 builds a 15 level table from a Lv.1 value.
func fromLv1(v float64, scaling []float64) []float64 {
	out := make([]float64, len(scaling))
	for i, s := range scaling {
		out[i] = v * s
	}
	return out
}

// fromLv10 builds a 15 level table from a Lv.10 value (elemental scaling).
func fromLv10(v float64) []float64 {
	return fromLv1(v/elementalScaling[9], elementalScaling)
}

var (
	// Normal Attack: Long Winter's Dawn (Lv.1 values)
	// 3-Hit is 25.2% x2, 4-Hit is 18.3% x3
	attack = [][]float64{
		fromLv1(0.396, physicalScaling),
		fromLv1(0.403, physicalScaling),
		fromLv1(0.252, physicalScaling),
		fromLv1(0.183, physicalScaling),
		fromLv1(0.657, physicalScaling),
	}
	aim     = fromLv1(0.439, physicalScaling)
	fullaim = fromLv1(1.24, elementalScaling)

	// Elemental Skill: Judgment of the Pole Star (Lv.1 values)
	skillTap  = fromLv1(1.182, elementalScaling)
	skillHold = fromLv1(1.576, elementalScaling)
	crown     = fromLv1(0.64, elementalScaling)
	crownSSC  = fromLv1(0.40, elementalScaling)
	crownSSw  = fromLv1(0.60, elementalScaling)
	// Sacred Spear (Glory) normal attacks
	// 1-Hit x1, 2-Hit x2, 3-Hit x1, 4-Hit x3, 5-Hit x3
	spear = [][]float64{
		fromLv1(0.627, elementalScaling),
		fromLv1(0.328, elementalScaling),
		fromLv1(0.797, elementalScaling),
		fromLv1(0.294, elementalScaling),
		fromLv1(0.354, elementalScaling),
	}
	spearSSC = [][]float64{
		fromLv1(0.392, elementalScaling),
		fromLv1(0.411, elementalScaling),
		fromLv1(0.499, elementalScaling),
		fromLv1(0.544, elementalScaling),
		fromLv1(0.664, elementalScaling),
	}
	spearSSw = [][]float64{
		fromLv1(0.588, elementalScaling),
		fromLv1(0.616, elementalScaling),
		fromLv1(0.748, elementalScaling),
		fromLv1(0.816, elementalScaling),
		fromLv1(0.996, elementalScaling),
	}

	// Elemental Burst: To a New World (Lv.10 values)
	burstNewWorld = fromLv10(3.607)
	// Stellar Glimmer Reaction DMG Bonus: 50% at Lv.10. Assumed flat across levels.
	burstStellarBonus = 0.5
	burstCrownSSC     = fromLv10(0.202)
	burstCrownSSw     = fromLv10(0.302)
	triumphSSC        = fromLv10(4.008)
	triumphSSw        = fromLv10(6.012)
	burstDawnSSC      = fromLv10(5.611)
	burstDawnSSw      = fromLv10(8.417)
	annihilation1     = fromLv10(0.481) // x5
	annihilation1SSC  = fromLv10(1.202)
	annihilation1SSw  = fromLv10(1.804)
	annihilation2SSC  = fromLv10(1.603)
	annihilation2SSw  = fromLv10(2.405)
	annihilation3SSC  = fromLv10(2.004)
	annihilation3SSw  = fromLv10(3.006)
)
