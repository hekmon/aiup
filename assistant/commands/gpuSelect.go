package commands

import (
	"fmt"
	"time"

	"github.com/hekmon/aiup/overclocking"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type DiscoveryResult struct {
	GPUs     []list.Item
	Warnings []string
	Err      error
}

func GPUDiscovery(profilesDir string) tea.Cmd {
	return func() tea.Msg {
		var (
			results *overclocking.DiscoveryResult
			dr      DiscoveryResult
		)
		if results, dr.Err = overclocking.ScanGPUs(profilesDir); dr.Err == nil {
			dr.GPUs = make([]list.Item, len(results.GPUs))
			for idx, gpuInfos := range results.GPUs {
				dr.GPUs[idx] = GPUItem{
					GPUInfo: gpuInfos,
				}
			}
			dr.Warnings = results.Errors
			dr.Warnings = []string{
				"Test warning 1",
				"Test warning 2 over a much much longer line that will wrap around to the next line if the terminal is not wide enough to contain it all because we need to test that too, right?",
			}
		}
		dr.Err = fmt.Errorf("GPU discovery failed: TEST ERROR")
		time.Sleep(3 * time.Second)
		return dr
	}
}

type GPUItem struct {
	overclocking.GPUInfo
}

func (i GPUItem) Title() string { return i.FullDescription }
func (i GPUItem) Description() string {
	return fmt.Sprintf("PCIe %d:%d:%d",
		i.BusNumber,
		i.DeviceNumber,
		i.FunctionNumber,
	)
}
func (i GPUItem) FilterValue() string { return i.Title() }
