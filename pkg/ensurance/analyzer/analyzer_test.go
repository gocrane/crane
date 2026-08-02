/*
Copyright 2024 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package analyzer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	ecache "github.com/gocrane/crane/pkg/ensurance/cache"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// singleRestoredPastCooldown is the reporter's #894 scenario:
// one NodeQOS, one AvoidanceAction with coolDownSeconds.
// After CPU recovers, scheduling must be re-enabled (ToBeDisable=false
// and ToBeRestore=true) once `now > lastTriggeredTime + coolDownSeconds`.
func TestMergeSchedulingActions_SingleRestoredPastCooldownEnables(t *testing.T) {
	now := time.Now()
	coolDown := int32(60)
	s := &AnomalyAnalyzer{
		lastTriggeredTime: now.Add(-2 * time.Minute), // fired ~2 minutes ago
	}

	nodeQOS := &ensuranceapi.NodeQOS{
		ObjectMeta: metav1.ObjectMeta{Name: "watermark1"},
		Spec:       ensuranceapi.NodeQOSSpec{},
	}
	action := &ensuranceapi.AvoidanceAction{
		ObjectMeta: metav1.ObjectMeta{Name: "disablescheduling"},
		Spec: ensuranceapi.AvoidanceActionSpec{
			CoolDownSeconds: coolDown,
		},
	}

	actionContexts := []ecache.ActionContext{
		{
			NodeQOS:    nodeQOS,
			RuleName:   "cpu-usage",
			ActionName: "disablescheduling",
			Restored:   true,
		},
	}
	actionMap := map[string]*ensuranceapi.AvoidanceAction{
		"disablescheduling": action,
	}

	var ae executor.AvoidanceExecutor
	s.mergeSchedulingActions(actionContexts, actionMap, &ae)

	// Past-coolDown restored context must Re-enable scheduling.
	assert.False(t, ae.ScheduleExecutor.ToBeDisable,
		"after cooldown expired, scheduling must be enabled (reporter bug #894)")
	assert.True(t, ae.ScheduleExecutor.ToBeRestore,
		"after cooldown expired, restore must be set so the executor clears the taint")
}

// triggeredTakesPriorityEvenAfterCooldown confirms a later Triggered
// action in the same Analyze iteration still wins — the fix preserves
// "any Triggered -> disable" semantics, only the *restore* branch is
// affected by the added break, never the Triggered branch.
func TestMergeSchedulingActions_TriggeredAfterCooldownDisables(t *testing.T) {
	now := time.Now()
	coolDown := int32(60)
	s := &AnomalyAnalyzer{
		lastTriggeredTime: now.Add(-2 * time.Minute), // fired long ago
	}

	ruleA := &ensuranceapi.AvoidanceAction{
		ObjectMeta: metav1.ObjectMeta{Name: "disablescheduling"},
		Spec:       ensuranceapi.AvoidanceActionSpec{CoolDownSeconds: coolDown},
	}
	ctrlAction := &ensuranceapi.NodeQOS{
		ObjectMeta: metav1.ObjectMeta{Name: "ctrl-metric"},
		Spec:       ensuranceapi.NodeQOSSpec{},
	}
	restoreCtx := &ensuranceapi.NodeQOS{
		ObjectMeta: metav1.ObjectMeta{Name: "watermark1"},
		Spec:       ensuranceapi.NodeQOSSpec{},
	}

	actionContexts := []ecache.ActionContext{
		{
			NodeQOS:    restoreCtx,
			RuleName:   "cpu-usage",
			ActionName: "disablescheduling",
			Restored:   true, // restored past cooldown
		},
		{
			NodeQOS:    ctrlAction,
			RuleName:   "ctrl-latency",
			ActionName: "disablescheduling",
			Triggered:  true, // hot path: still triggered
		},
	}
	actionMap := map[string]*ensuranceapi.AvoidanceAction{
		"disablescheduling": ruleA,
	}

	var ae executor.AvoidanceExecutor
	s.mergeSchedulingActions(actionContexts, actionMap, &ae)

	// A Triggered sibling MUST keep scheduling disabled — the added `break`
	// in the restore branch only short-circuits the restore decision; it
	// does not bypass the Triggered short-circuit.
	assert.True(t, ae.ScheduleExecutor.ToBeDisable,
		"a Triggered sibling action must keep scheduling disabled")
	assert.False(t, ae.ScheduleExecutor.ToBeRestore,
		"a Triggered sibling action must not call Restore")
}

// restoreStaysPendingUntilExecuted is the core regression test for #894:
// before the fix, Restored fired exactly once (the trigger counter was
// reset immediately), so if that single fire landed inside the cool-down
// window the taint was never cleared. The restore must stay pending on
// subsequent below-watermark rounds until mergeSchedulingActions executes
// it and clears the counters.
func TestComputeActionContext_RestoreStaysPendingUntilExecuted(t *testing.T) {
	s := &AnomalyAnalyzer{
		triggered: map[string]uint64{},
		restored:  map[string]uint64{},
	}
	rule := ensuranceapi.Rule{
		Name:               "cpu-usage",
		AvoidanceThreshold: 2,
		RestoreThreshold:   2,
	}
	key := "watermark1.cpu-usage"

	// two above-watermark rounds trigger the avoidance action
	var ac ecache.ActionContext
	s.computeActionContext(true, key, rule, &ac)
	assert.False(t, ac.Triggered, "first breach must not trigger yet")
	ac = ecache.ActionContext{}
	s.computeActionContext(true, key, rule, &ac)
	assert.True(t, ac.Triggered, "second consecutive breach must trigger")

	// usage recovers: first round below the watermark only counts
	ac = ecache.ActionContext{}
	s.computeActionContext(false, key, rule, &ac)
	assert.False(t, ac.Restored, "first round below watermark must not restore yet")

	// second round below: restore fires
	ac = ecache.ActionContext{}
	s.computeActionContext(false, key, rule, &ac)
	assert.True(t, ac.Restored, "restore must fire once RestoreThreshold is reached")

	// REGRESSION #894: while the restore waits out the cool-down, every
	// further round below the watermark must keep the restore pending.
	ac = ecache.ActionContext{}
	s.computeActionContext(false, key, rule, &ac)
	assert.True(t, ac.Restored, "restore must stay pending until it is executed (issue #894)")

	// mergeSchedulingActions clears the counters once the restore executes
	delete(s.triggered, key)
	delete(s.restored, key)
	ac = ecache.ActionContext{}
	s.computeActionContext(false, key, rule, &ac)
	assert.False(t, ac.Restored, "after execution the restore must stop firing")

	// and the rule can trigger again on the next interference
	ac = ecache.ActionContext{}
	s.computeActionContext(true, key, rule, &ac)
	assert.False(t, ac.Triggered, "re-trigger requires the full threshold again")
	ac = ecache.ActionContext{}
	s.computeActionContext(true, key, rule, &ac)
	assert.True(t, ac.Triggered, "rule must trigger again after counters were cleared")
}

// coolDownClockOnlyMovesOnTriggers covers the second half of #894:
// lastTriggeredTime must not be refreshed while the restore waits out the
// cool-down, otherwise the cool-down never elapses.
func TestMergeSchedulingActions_CoolDownNotExtendedWhileRestoring(t *testing.T) {
	t0 := time.Now()
	action := &ensuranceapi.AvoidanceAction{
		ObjectMeta: metav1.ObjectMeta{Name: "disablescheduling"},
		Spec:       ensuranceapi.AvoidanceActionSpec{CoolDownSeconds: 60},
	}
	nodeQOS := &ensuranceapi.NodeQOS{ObjectMeta: metav1.ObjectMeta{Name: "watermark1"}}
	actionContexts := []ecache.ActionContext{
		{
			NodeQOS:    nodeQOS,
			RuleName:   "cpu-usage",
			ActionName: "disablescheduling",
			Restored:   true,
		},
	}
	actionMap := map[string]*ensuranceapi.AvoidanceAction{"disablescheduling": action}

	// inside the cool-down window: stays disabled, and the cool-down start
	// time must not move (pre-fix it was refreshed on every round, so the
	// cool-down never elapsed)
	s := &AnomalyAnalyzer{
		triggered:         map[string]uint64{"watermark1.cpu-usage": 2},
		restored:          map[string]uint64{"watermark1.cpu-usage": 2},
		lastTriggeredTime: t0,
	}
	var ae executor.AvoidanceExecutor
	s.mergeSchedulingActions(actionContexts, actionMap, &ae)
	assert.True(t, ae.ScheduleExecutor.ToBeDisable, "inside cool-down scheduling stays disabled")
	assert.False(t, ae.ScheduleExecutor.ToBeRestore, "inside cool-down restore must not run yet")
	assert.Equal(t, t0, s.lastTriggeredTime, "cool-down start time must not be refreshed while restoring (#894)")
	assert.Equal(t, uint64(2), s.triggered["watermark1.cpu-usage"], "counters stay while restore is pending")

	// after the cool-down: restore executes and counters are cleared
	s.lastTriggeredTime = time.Now().Add(-2 * time.Minute)
	ae = executor.AvoidanceExecutor{}
	s.mergeSchedulingActions(actionContexts, actionMap, &ae)
	assert.False(t, ae.ScheduleExecutor.ToBeDisable, "after cool-down scheduling is re-enabled")
	assert.True(t, ae.ScheduleExecutor.ToBeRestore, "after cool-down restore must run")
	_, triggeredExists := s.triggered["watermark1.cpu-usage"]
	_, restoredExists := s.restored["watermark1.cpu-usage"]
	assert.False(t, triggeredExists, "trigger counter must be cleared after restore executes")
	assert.False(t, restoredExists, "restore counter must be cleared after restore executes")
}

// emptyActionContexts picks up the branch that fires without any
// NodeQOS matches, which previously was the ONLY branch guaranteed to
// re-enable scheduling.
func TestMergeSchedulingActions_EmptyContextsEnables(t *testing.T) {
	s := &AnomalyAnalyzer{}

	actionContexts := []ecache.ActionContext{}
	actionMap := map[string]*ensuranceapi.AvoidanceAction{}

	var ae executor.AvoidanceExecutor
	s.mergeSchedulingActions(actionContexts, actionMap, &ae)

	assert.False(t, ae.ScheduleExecutor.ToBeDisable,
		"no NodeQOS matches should mean there is nothing to disable")
	assert.True(t, ae.ScheduleExecutor.ToBeRestore,
		"no NodeQOS matches should mean restore is set")
}
