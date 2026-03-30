package commands

import (
	"github.com/hekmon/aiup/overclocking"

	tea "charm.land/bubbletea/v2"
)

type CurrentState struct {
	State *overclocking.CurrentStateResult
	Error error
}

func GetCurrentState(gpuID int, profilePath string) tea.Cmd {
	return func() tea.Msg {
		var cs CurrentState
		cs.State, cs.Error = overclocking.GetCurrentState(gpuID, profilePath)
		return cs
	}
}

// GetCurrentState is meant to be ticked by tea
func GetCurrentStateRaw(gpuID int, profilePath string) (cs CurrentState) {
	cs.State, cs.Error = overclocking.GetCurrentState(gpuID, profilePath)
	// cs.State.FanMode = "tick ok"
	return
}
