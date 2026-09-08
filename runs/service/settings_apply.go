package service

import (
	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/core"
	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/settings"
	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/task"
)

// applyRunSettings fills fields the caller left unset with values resolved from
// settings. An explicit value in the request always wins, so this only ever fills
// gaps, and a setting that is INHERIT or UNSET contributes nothing.
func applyRunSettings(spec *task.RunSpec, resolved *settings.Settings) {
	if spec == nil {
		return
	}

	if spec.GetQueue() == "" && resolved.GetRun().GetDefaultQueue().GetState() ==
		settings.SettingState_SETTING_STATE_VALUE {
		spec.Queue = resolved.GetRun().GetDefaultQueue().GetStringValue()
	}

	// The proto defines 0 as unset for this field, so there is no explicit request for
	// "unlimited" to override. The settings validator bounds the value to 0 or
	// [2, MaxUint32], so narrowing to uint32 cannot overflow.
	if concurrency := resolved.GetRun().GetMaxActionConcurrency(); spec.GetMaxActionConcurrency() == 0 &&
		concurrency.GetState() == settings.SettingState_SETTING_STATE_VALUE {
		spec.MaxActionConcurrency = uint32(concurrency.GetIntValue())
	}
}

// applyTaskSettings stamps the resolved pod template name onto a task that did not
// name one itself. The task's own choice always wins, and only a VALUE-state setting
// carrying a non-empty name is applied, since a wrong name fails the task at pod
// build time rather than falling back.
func applyTaskSettings(spec *task.TaskSpec, resolved *settings.Settings) {
	template := spec.GetTaskTemplate()
	if template.GetMetadata().GetPodTemplateName() != "" {
		return
	}

	podTemplate := resolved.GetPodTemplateName()
	if podTemplate.GetState() != settings.SettingState_SETTING_STATE_VALUE || podTemplate.GetStringValue() == "" {
		return
	}

	if template.Metadata == nil {
		template.Metadata = &core.TaskMetadata{}
	}
	template.Metadata.PodTemplateName = podTemplate.GetStringValue()
}
