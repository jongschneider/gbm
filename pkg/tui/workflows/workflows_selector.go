package workflows

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
)

// SelectWorkflowType creates and returns a Selector field for choosing a workflow type.
// The user can choose between Feature, Bug, Hotfix, or Merge workflows.
//
// Returns a configured Selector implementing the Field interface.
func SelectWorkflowType() tui.Field {
	return fields.NewSelector(
		"workflow_type",
		"Select Workflow Type",
		[]fields.Option{
			{Label: "Feature", Value: "feature"},
			{Label: "Bug", Value: "bug"},
			{Label: "Hotfix", Value: "hotfix"},
			{Label: "Merge", Value: "merge"},
		},
	)
}
