package controlgrpc

import (
	"reflect"
	"testing"

	"github.com/wanglongan587/cloud/internal/controlpb"
	"github.com/wanglongan587/cloud/internal/core"
)

// The new declaration is lossless through the same codec used by claim and RecordDispatch.
func TestWorkflowHistoryCodecPreservesAttemptAndLocations(t *testing.T) {
	sessions := []core.Object{
		{"nodeId": "a", "attempt": int64(1), "executionId": "exec-1", "historyKey": "run/attempt-1.jsonl"},
		{"nodeId": "a", "attempt": int64(2), "executionId": "exec-2", "historyKey": "run/attempt-2.jsonl"},
	}
	if got := workflowSessionObjects(workflowSessionMessages(sessions)); !reflect.DeepEqual(got, sessions) {
		t.Fatalf("session round trip: %v", got)
	}
	histories := []*controlpb.WorkflowHistoryObject{{NodeId: "a", Attempt: 2, ExecutionId: "exec-2",
		History: &controlpb.StoredObject{Key: "run/attempt-2.jsonl", Size: 17, Sha256: "deadbeef"}}}
	if got := workflowHistoryMessages(workflowHistoryObjects(histories)); !reflect.DeepEqual(got, histories) {
		t.Fatalf("history round trip: %v", got)
	}
}
