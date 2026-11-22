package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// The Stackyard fire endpoint delivers a schedule's Input to its target.
func TestSchedulerStage2FireDeliversToSQS(t *testing.T) {
	srv, ts := newSchedulerTestServer(t)
	if _, err := srv.sqs.CreateQueue("fire-q", nil); err != nil {
		t.Fatal(err)
	}
	once := strings.Replace(schedulerBody("at(2030-01-01T00:00:00)", schedulerSQSTarget("fire-q")), `{"ScheduleExpression"`, `{"ActionAfterCompletion":"DELETE","ScheduleExpression"`, 1)
	schedulerCall(t, ts, http.MethodPost, "/schedules/once", once, http.StatusOK)
	out := schedulerCall(t, ts, http.MethodPost, "/_stackyard/scheduler/schedules/default/once/fire", "", http.StatusOK)
	if out["MessageId"] == nil {
		t.Fatalf("fire: %v", out)
	}
	msgs, err := srv.sqs.ReceiveMessages("fire-q", 10)
	if err != nil || len(msgs) != 1 || msgs[0].Body != `{"hello":1}` {
		t.Fatalf("delivered: %v %v", msgs, err)
	}
	// A one-time schedule set to DELETE is gone after it fires.
	schedulerCallError(t, ts, http.MethodGet, "/schedules/once", "", http.StatusNotFound)

	disabled := strings.Replace(schedulerBody("rate(1 day)", schedulerSQSTarget("fire-q")), `{"ScheduleExpression"`, `{"State":"DISABLED","ScheduleExpression"`, 1)
	schedulerCall(t, ts, http.MethodPost, "/schedules/off", disabled, http.StatusOK)
	schedulerCallError(t, ts, http.MethodPost, "/_stackyard/scheduler/schedules/default/off/fire", "", http.StatusConflict)

	missing := schedulerBody("rate(1 day)", schedulerSQSTarget("no-such-queue"))
	schedulerCall(t, ts, http.MethodPost, "/schedules/missing", missing, http.StatusOK)
	schedulerCallError(t, ts, http.MethodPost, "/_stackyard/scheduler/schedules/default/missing/fire", "", http.StatusNotFound)

	lambda := schedulerBody("rate(1 day)", fmt.Sprintf(`{"Arn":"arn:aws:lambda:us-east-1:123456789012:function:f","RoleArn":%q}`, schedulerTestRole))
	schedulerCall(t, ts, http.MethodPost, "/schedules/fn", lambda, http.StatusOK)
	schedulerCallError(t, ts, http.MethodPost, "/_stackyard/scheduler/schedules/default/fn/fire", "", http.StatusBadRequest)
}

func TestSchedulerStage2FireDeliversToEventBridge(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	body := schedulerBody("rate(1 day)", fmt.Sprintf(`{"Arn":"arn:aws:events:us-east-1:123456789012:event-bus/default","RoleArn":%q,"Input":"{\"a\":1}","EventBridgeParameters":{"DetailType":"tick","Source":"app.scheduler"}}`, schedulerTestRole))
	schedulerCall(t, ts, http.MethodPost, "/schedules/ev", body, http.StatusOK)
	if out := schedulerCall(t, ts, http.MethodPost, "/_stackyard/scheduler/schedules/default/ev/fire", "", http.StatusOK); out["EventId"] == nil {
		t.Fatalf("fire event: %v", out)
	}
}
