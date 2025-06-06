package agents

import (
	"strings"
)

type PlannerAgent struct{}

func (p *PlannerAgent) Run(objective string) []string {
	// Dummy logic — later call core.Execute() with planner prompt
	if strings.Contains(strings.ToLower(objective), "tool") {
		return []string{
			"Design UI layout",
			"Define data structure",
			"Generate backend service",
			"Write unit tests",
			"Deploy to render",
		}
	}

	return []string{
		"Break down the project requirements",
		"Generate initial scaffolding",
		"Assign subtasks to specialized agents",
	}
}
