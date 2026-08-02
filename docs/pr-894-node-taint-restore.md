#### What type of PR is this?
Bug fix

#### What this PR does / why we need it:

Following the official demo
([QoS Interference Detection And Active Avoidance](https://gocrane.io/zh-cn/docs/tutorials/colocation-with-enhanced-qos/qos-interference-detection-and-active-avoidance.zh/)),
the `EnsuranceAnalyzedPressure:PreferNoSchedule` node taint added by an
`AvoidanceAction` (e.g. `disablescheduling`) is **never removed** after the
metric drops back below the watermark — users expect it to be cleared once
`coolDownSeconds` has elapsed.

**Root cause — two cooperating defects in `pkg/ensurance/analyzer/analyzer.go`:**

1. **One-shot restore.** `computeActionContext` fired `Restored` exactly
   once: it reset the trigger counter at the very moment the restore count
   reached `restoreThreshold`. The cool-down was therefore evaluated only at
   that single instant. With the usual defaults (`restoreThreshold: 2`,
   probe interval ~10s), that instant arrives ~20s after the usage recovers —
   almost always *inside* the `coolDownSeconds` window (60s in the reporter's
   YAML), so the merge step kept the node disabled, and since `Restored`
   never fired again, the taint stayed forever.
2. **Cool-down never elapses.** While waiting out the cool-down,
   `mergeSchedulingActions` kept calling `ToggleScheduleSetting(true)`, which
   refreshed `lastTriggeredTime` on *every* analysis round — the cool-down
   start moved forward each round, so `now > lastTriggeredTime +
   coolDownSeconds` could never become true.

**Fix (no API changes):**

- `computeActionContext`: the restore now stays **pending** while the usage
  remains below the watermark — the trigger/restore counters are no longer
  reset when `Restored` first fires, so the cool-down is re-evaluated on
  every analysis round.
- `mergeSchedulingActions`: when the cool-down finally elapses and the
  restore is executed (`ToBeDisable=false, ToBeRestore=true`), the counters
  are cleared so the rule stops firing and can trigger again on the next
  interference.
- `lastTriggeredTime` is now refreshed **only** by real trigger rounds
  (`ac.Triggered`), so the cool-down always starts from the last actual
  breach.
- New `recordRestoreDecision` audit log: one structured line per scheduling
  decision (`action`, `rule`, `nodeQOS`, `direction=enable|disable`,
  `coolDown`, `lastTriggered`, `now`), so the restore path is observable in
  production logs without a debugger.

After `coolDownSeconds` elapses, `ScheduleExecutor.Restore` runs and removes
the taint plus the node condition, exactly as the reporter expected.

**Verification:**

- New unit tests in `pkg/ensurance/analyzer/analyzer_test.go` (all pass,
  `go test ./pkg/ensurance/...` green, full-repo `go build ./...` green):
  - `TestComputeActionContext_RestoreStaysPendingUntilExecuted` — core
    regression: restore stays pending until executed, stops after, and the
    rule re-triggers on the next interference.
  - `TestMergeSchedulingActions_CoolDownNotExtendedWhileRestoring` —
    cool-down clock is not refreshed while restoring; restore executes after
    the cool-down and clears the counters.
  - `TestMergeSchedulingActions_SingleRestoredPastCooldownEnables` — the
    reporter's exact scenario (one NodeQOS `watermark1`, one action
    `disablescheduling` with `coolDownSeconds: 60`).
  - `TestMergeSchedulingActions_TriggeredAfterCooldownDisables` — a
    concurrently triggered rule still keeps the node disabled.
  - `TestMergeSchedulingActions_EmptyContextsEnables` — no matching rules
    still means scheduling is restored.
- Manual e2e (reporter's setup): apply the `AvoidanceAction` + `NodeQOS`
  from the issue, stress CPU above `cpu_total_usage: 3000` → taint appears;
  stop the load → after ~`restoreThreshold` rounds + `coolDownSeconds: 60`,
  `kubectl describe node` shows the `EnsuranceAnalyzedPressure` taint removed.

#### Which issue(s) this PR fixes:
<!--
*Automatically closes linked issue when PR is merged.
Usage: `Fixes #<issue number>`, or `Fixes (paste link of issue)`.
-->
Fixes #894

#### Special notes for your reviewer:

- **Design note — no `break` after the restore branch.** An earlier variant
  of this patch short-circuited `mergeSchedulingActions` after a past-cooldown
  restore; it was rejected because that skipped later contexts in the same
  round and violated the existing "any `Triggered` keeps the node disabled"
  contract. `TestMergeSchedulingActions_TriggeredAfterCooldownDisables`
  guards this ordering.
- **Restore event timing changes slightly.** `Restored` now fires on every
  analysis round while it is pending (previously exactly once). The k8s
  event/log spam is unchanged: the `RestoreTriggered` event is still recorded
  only on the triggered→restored transition, and the warning log fires only
  on the transition round.
- **Throttle restore is also re-evaluated during the cool-down** (same code
  path). `ThrottleExecutor.Restore` is idempotent once pods are restored to
  the watermark, so the extra rounds are no-ops; this also fixes the same
  "restore lost if inside cool-down" class of bug for throttle-up.
- **`lastTriggeredTime` remains a single per-node timestamp** (shared across
  rules), as before this PR. With multiple NodeQOS rules targeting the same
  node, the cool-down starts from the last trigger of *any* rule; making it
  per-rule would be a larger refactor and is left as follow-up.
- If the executor's node update itself fails (API conflict after retries),
  the taint can still be left behind — `execute()` only logs the error
  (pre-existing `TODO: if it failed in action, how to retry`). Also left as
  follow-up.
