package nvvf

// ClkDomainInfo represents information about a GPU clock domain.
//
// Clock domains are different clock regions on the GPU that can be
// independently controlled. This includes:
//   - Graphics clock (GPU core)
//   - Memory clock (VRAM)
//   - Processor clock
//   - Video clock
//
// The MinOffsetKHz and MaxOffsetKHz values indicate the safe operating
// range for frequency offsets in each domain.
type ClkDomainInfo struct {
	Domain       ClkDomain // Domain identifier
	Flags        uint32    // Domain flags
	MinOffsetKHz int32     // Minimum allowed offset in kHz
	MaxOffsetKHz int32     // Maximum allowed offset in kHz
}
