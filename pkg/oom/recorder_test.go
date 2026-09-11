package oom

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gocrane/crane/pkg/known"
)

// TestUpdateOOMRecordTrimsOnEveryWrite reproduces the bug where the record count cap
// was only enforced by a 12h ticker, letting the ConfigMap grow without bound between
// ticks. It fails on the old code (all records kept) and passes once updateOOMRecord
// enforces OOMRecordMaxNumber on every write.
func TestUpdateOOMRecordTrimsOnEveryWrite(t *testing.T) {
	const maxNumber = 5
	const writes = 10

	r := &PodOOMRecorder{
		Client:             fakeClient.NewClientBuilder().Build(),
		OOMRecordMaxNumber: maxNumber,
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < writes; i++ {
		record := OOMRecord{
			Namespace: "default",
			Pod:       fmt.Sprintf("pod-%d", i),
			Container: "app",
			Memory:    resource.MustParse("1Gi"),
			OOMAt:     base.Add(time.Duration(i) * time.Minute),
		}
		saved, err := r.GetOOMRecord()
		if err != nil {
			t.Fatalf("GetOOMRecord() failed: %v", err)
		}
		if err := r.updateOOMRecord(record, saved); err != nil {
			t.Fatalf("updateOOMRecord() failed: %v", err)
		}
	}

	configMap := &v1.ConfigMap{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: known.CraneSystemNamespace, Name: ConfigMapOOMRecordName}, configMap); err != nil {
		t.Fatalf("Get ConfigMap failed: %v", err)
	}

	var stored []OOMRecord
	if err := json.Unmarshal([]byte(configMap.Data[ConfigMapDataOOMRecord]), &stored); err != nil {
		t.Fatalf("unmarshal stored records failed: %v", err)
	}

	if len(stored) != maxNumber {
		t.Fatalf("expected ConfigMap to be capped at %d records after every write, got %d", maxNumber, len(stored))
	}

	// The newest record must be kept, not dropped in favor of an older one.
	last := fmt.Sprintf("pod-%d", writes-1)
	found := false
	for _, rec := range stored {
		if rec.Pod == last {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected newest record %q to survive trimming, stored=%+v", last, stored)
	}
}

func TestTrimOOMRecordsByCount(t *testing.T) {
	records := make([]OOMRecord, 0, 5)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		records = append(records, OOMRecord{Pod: fmt.Sprintf("pod-%d", i), OOMAt: base.Add(time.Duration(i) * time.Minute)})
	}

	trimmed := trimOOMRecordsByCount(records, 2)
	if len(trimmed) != 2 {
		t.Fatalf("expected 2 records, got %d", len(trimmed))
	}
	if trimmed[0].Pod != "pod-3" || trimmed[1].Pod != "pod-4" {
		t.Fatalf("expected the newest records to be kept, got %+v", trimmed)
	}

	// No trimming needed when already within the limit.
	untouched := trimOOMRecordsByCount(records, 10)
	if len(untouched) != 5 {
		t.Fatalf("expected no trimming, got %d records", len(untouched))
	}
}

func TestTrimOOMRecordsToByteSize(t *testing.T) {
	records := make([]OOMRecord, 0, 50)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 50; i++ {
		records = append(records, OOMRecord{
			Namespace: "default",
			Pod:       fmt.Sprintf("some-fairly-long-pod-name-%d", i),
			Container: "app",
			Memory:    resource.MustParse("1Gi"),
			OOMAt:     base.Add(time.Duration(i) * time.Minute),
		})
	}

	fullData, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	maxBytes := len(fullData) / 2

	trimmed, data, err := trimOOMRecordsToByteSize(records, maxBytes)
	if err != nil {
		t.Fatalf("trimOOMRecordsToByteSize() failed: %v", err)
	}
	if len(data) > maxBytes {
		t.Fatalf("expected marshaled data to fit within %d bytes, got %d", maxBytes, len(data))
	}
	if len(trimmed) == 0 || len(trimmed) >= len(records) {
		t.Fatalf("expected trimming to drop some but not all records, got %d of %d", len(trimmed), len(records))
	}
	// Newest record must survive.
	if trimmed[len(trimmed)-1].Pod != records[len(records)-1].Pod {
		t.Fatalf("expected newest record to survive, got %+v", trimmed[len(trimmed)-1])
	}

	// Nothing to trim when already within budget.
	untouched, _, err := trimOOMRecordsToByteSize(records, len(fullData))
	if err != nil {
		t.Fatalf("trimOOMRecordsToByteSize() failed: %v", err)
	}
	if len(untouched) != len(records) {
		t.Fatalf("expected no trimming, got %d records", len(untouched))
	}

	// Even when a single record doesn't fit maxBytes on its own, the returned records
	// and marshaled data must stay in sync (never pair a shrunk record list with the
	// full, oversized data, which would still trip the Kubernetes object-size limit).
	tiny, tinyData, err := trimOOMRecordsToByteSize(records, 1)
	if err != nil {
		t.Fatalf("trimOOMRecordsToByteSize() failed: %v", err)
	}
	if len(tiny) != 1 {
		t.Fatalf("expected the single newest record to be kept as a floor, got %d records", len(tiny))
	}
	wantData, err := json.Marshal(tiny)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(tinyData) != string(wantData) {
		t.Fatalf("returned data does not match returned records: data=%s records=%s", tinyData, wantData)
	}
}
