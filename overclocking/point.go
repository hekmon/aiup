package overclocking

// VFPoint represents a single voltage-frequency point with all components explicit.
// This structure is designed for AI agent consumption - all values are in MHz except voltage.
//
// Key insight: OffsetMHz is the CORE overclocking value that gets set/modified.
// EffectiveFreqMHz = BaseFreqMHz + OffsetMHz
type VFPoint struct {
	VoltageMV   float64 `json:"voltage_mv"`    // Voltage point (e.g., 850.0 mV)
	BaseFreqMHz float64 `json:"base_freq_mhz"` // Hardware base frequency at this voltage
	OffsetMHz   float64 `json:"offset_mhz"`    // Overclock offset applied (THE CORE VALUE)
}

func (vfp VFPoint) EffectiveFreqMHz() float64 {
	return vfp.BaseFreqMHz + vfp.OffsetMHz
}
