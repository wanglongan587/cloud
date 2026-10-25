package controlgrpc

import (
	"github.com/wanglongan587/cloud/internal/controlpb"
	"github.com/wanglongan587/cloud/internal/core"
)

// workflowSessionObjects retains every declared attempt and its fixed upload location.
func workflowSessionObjects(values []*controlpb.WorkflowHistorySession) []core.Object {
	out := make([]core.Object, 0, len(values))
	for _, value := range values {
		out = append(out, core.Object{"nodeId": value.GetNodeId(), "attempt": int64(value.GetAttempt()), "executionId": value.GetExecutionId(), "historyKey": value.GetHistoryKey()})
	}
	return out
}

// workflowSessionMessages carries only the frozen declaration; the Controller cannot choose files.
func workflowSessionMessages(values []core.Object) []*controlpb.WorkflowHistorySession {
	out := make([]*controlpb.WorkflowHistorySession, 0, len(values))
	for _, value := range values {
		out = append(out, &controlpb.WorkflowHistorySession{NodeId: value.S("nodeId"), Attempt: uint32(value.N("attempt")), ExecutionId: value.S("executionId"), HistoryKey: value.S("historyKey")}) // #nosec G115 -- attempts originate from the bounded workflow retry counter.
	}
	return out
}

// workflowHistoryObjects preserves Node measurements separately from Cloud's later verification.
func workflowHistoryObjects(values []*controlpb.WorkflowHistoryObject) []core.Object {
	out := make([]core.Object, 0, len(values))
	for _, value := range values {
		out = append(out, core.Object{"nodeId": value.GetNodeId(), "attempt": int64(value.GetAttempt()), "executionId": value.GetExecutionId(), "history": storedObject(value.GetHistory())})
	}
	return out
}

// workflowHistoryMessages reproduces persisted evidence without narrowing its ownership metadata.
func workflowHistoryMessages(values []core.Object) []*controlpb.WorkflowHistoryObject {
	out := make([]*controlpb.WorkflowHistoryObject, 0, len(values))
	for _, value := range values {
		out = append(out, &controlpb.WorkflowHistoryObject{NodeId: value.S("nodeId"), Attempt: uint32(value.N("attempt")), ExecutionId: value.S("executionId"), History: storedMessage(value.O("history"))}) // #nosec G115 -- only declarations matching the bounded frozen attempt are persisted.
	}
	return out
}
