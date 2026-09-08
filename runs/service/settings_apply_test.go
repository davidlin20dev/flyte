package service

import (
	"testing"

	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/core"
	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/settings"
	"github.com/flyteorg/flyte/v2/gen/go/flyteidl2/task"
	"github.com/stretchr/testify/assert"
)

func runSettings(queue *settings.StringSetting, concurrency *settings.Int64Setting) *settings.Settings {
	return &settings.Settings{
		Run: &settings.RunSettings{DefaultQueue: queue, MaxActionConcurrency: concurrency},
	}
}

func podTemplateSettings(name *settings.StringSetting) *settings.Settings {
	return &settings.Settings{PodTemplateName: name}
}

// taskWithPodTemplate builds a task spec carrying the given pod template name. An empty
// name leaves the metadata block off entirely, which is what a task that never
// mentioned a pod template actually looks like.
func taskWithPodTemplate(name string) *task.TaskSpec {
	template := &core.TaskTemplate{}
	if name != "" {
		template.Metadata = &core.TaskMetadata{PodTemplateName: name}
	}
	return &task.TaskSpec{TaskTemplate: template}
}

func TestApplyRunSettings(t *testing.T) {
	tests := []struct {
		name            string
		spec            *task.RunSpec
		resolved        *settings.Settings
		wantQueue       string
		wantConcurrency uint32
	}{
		{
			name:      "empty queue takes the settings value",
			spec:      &task.RunSpec{},
			resolved:  runSettings(&settings.StringSetting{State: stateValue, StringValue: "fast-queue"}, nil),
			wantQueue: "fast-queue",
		},
		{
			name:      "an explicit queue wins over settings",
			spec:      &task.RunSpec{Queue: "user-queue"},
			resolved:  runSettings(&settings.StringSetting{State: stateValue, StringValue: "fast-queue"}, nil),
			wantQueue: "user-queue",
		},
		{
			name:      "a queue in INHERIT contributes nothing",
			spec:      &task.RunSpec{},
			resolved:  runSettings(&settings.StringSetting{State: stateInherit, StringValue: "fast-queue"}, nil),
			wantQueue: "",
		},
		{
			name:            "zero concurrency takes the settings value",
			spec:            &task.RunSpec{},
			resolved:        runSettings(nil, &settings.Int64Setting{State: stateValue, IntValue: 5}),
			wantConcurrency: 5,
		},
		{
			name:            "an explicit concurrency wins over settings",
			spec:            &task.RunSpec{MaxActionConcurrency: 3},
			resolved:        runSettings(nil, &settings.Int64Setting{State: stateValue, IntValue: 5}),
			wantConcurrency: 3,
		},
		{
			name:            "concurrency in UNSET contributes nothing",
			spec:            &task.RunSpec{},
			resolved:        runSettings(nil, &settings.Int64Setting{State: stateUnset, IntValue: 5}),
			wantConcurrency: 0,
		},
		{
			name:     "no settings at all",
			spec:     &task.RunSpec{},
			resolved: &settings.Settings{},
		},
		{
			name:     "nil spec does not panic",
			spec:     nil,
			resolved: runSettings(&settings.StringSetting{State: stateValue, StringValue: "fast-queue"}, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyRunSettings(tt.spec, tt.resolved)
			assert.Equal(t, tt.wantQueue, tt.spec.GetQueue())
			assert.Equal(t, tt.wantConcurrency, tt.spec.GetMaxActionConcurrency())
		})
	}
}

func TestApplyTaskSettings(t *testing.T) {
	tests := []struct {
		name     string
		spec     *task.TaskSpec
		resolved *settings.Settings
		want     string
	}{
		{
			name:     "a task naming no template takes the settings value",
			spec:     taskWithPodTemplate(""),
			resolved: podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: "gpu-template"}),
			want:     "gpu-template",
		},
		{
			name:     "an explicit template wins over settings",
			spec:     taskWithPodTemplate("task-template"),
			resolved: podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: "gpu-template"}),
			want:     "task-template",
		},
		{
			name:     "a template in INHERIT contributes nothing",
			spec:     taskWithPodTemplate(""),
			resolved: podTemplateSettings(&settings.StringSetting{State: stateInherit, StringValue: "gpu-template"}),
			want:     "",
		},
		{
			name:     "a template in UNSET contributes nothing",
			spec:     taskWithPodTemplate(""),
			resolved: podTemplateSettings(&settings.StringSetting{State: stateUnset, StringValue: "gpu-template"}),
			want:     "",
		},
		{
			name:     "a VALUE state carrying an empty name contributes nothing",
			spec:     taskWithPodTemplate(""),
			resolved: podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: ""}),
			want:     "",
		},
		{
			name:     "no settings at all",
			spec:     taskWithPodTemplate(""),
			resolved: &settings.Settings{},
			want:     "",
		},
		{
			name:     "nil spec does not panic",
			spec:     nil,
			resolved: podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: "gpu-template"}),
			want:     "",
		},
		{
			name:     "a spec with no template does not panic",
			spec:     &task.TaskSpec{},
			resolved: podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: "gpu-template"}),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyTaskSettings(tt.spec, tt.resolved)
			assert.Equal(t, tt.want, tt.spec.GetTaskTemplate().GetMetadata().GetPodTemplateName())
		})
	}
}

// TestApplyTaskSettings_PreservesOtherMetadata guards the allocation branch: a task
// that already has a metadata block must keep it, since that block also carries
// retries, timeouts and the cache version.
func TestApplyTaskSettings_PreservesOtherMetadata(t *testing.T) {
	spec := &task.TaskSpec{TaskTemplate: &core.TaskTemplate{
		Metadata: &core.TaskMetadata{DiscoveryVersion: "v1"},
	}}

	applyTaskSettings(spec, podTemplateSettings(&settings.StringSetting{State: stateValue, StringValue: "gpu-template"}))

	metadata := spec.GetTaskTemplate().GetMetadata()
	assert.Equal(t, "gpu-template", metadata.GetPodTemplateName())
	assert.Equal(t, "v1", metadata.GetDiscoveryVersion())
}
