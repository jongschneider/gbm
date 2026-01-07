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
		tui.FieldKeyWorkflowType,
		"Select Workflow Type",
		[]fields.Option{
			{Label: "Feature", Value: tui.WorkflowTypeFeature},
			{Label: "Bug", Value: tui.WorkflowTypeBug},
			{Label: "Hotfix", Value: tui.WorkflowTypeHotfix},
			{Label: "Merge", Value: tui.WorkflowTypeMerge},
		},
	)
}
