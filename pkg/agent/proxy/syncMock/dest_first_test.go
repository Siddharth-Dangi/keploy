package manager

import (
	"fmt"
	"testing"
	"time"

	"go.keploy.io/server/v3/pkg/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// appOf names the app a mock goes to by its "app" metadata, "" when it has
// none: a node's resolver that cannot tell it.
func appOf(mk *models.Mock) string { return mk.Spec.Metadata["app"] }

// appStartupAt is a start-up mock (IsStartup) no test owns, requested at at
// on conn, that goes to app's recording ("" for one the node cannot tell).
func appStartupAt(conn, app string, at time.Time) *models.Mock {
	mk := startupAt(conn, at)
	if app != "" {
		if mk.Spec.Metadata == nil {
			mk.Spec.Metadata = map[string]string{}
		}
		mk.Spec.Metadata["app"] = app
	}
	return mk
}

// nodeManagers is a node's default manager, whose mocks go to the apps'
// recordings (SetDestinationOf), to a client that deletes the test cases it
// revokes, with its logs, and apps A's and B's own managers.
func nodeManagers(t *testing.T) (node, appA, appB *SyncMockManager, logs *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	node, appA, appB = New(zap.New(core)), New(nil), New(nil)
	node.SetRevokeCapable(true)
	node.SetDestinationOf(appOf)
	for _, m := range []*SyncMockManager{node, appA, appB} {
		m.SetOutputChannel(make(chan *models.Mock, 1))
	}
	return node, appA, appB, logs
}

// sendTo has the stream send app's test case named name over [start, end],
// through app's own manager and the node's, and fails the row if it was
// left out.
func sendTo(t *testing.T, node, own *SyncMockManager, app, name string, start, end time.Time) {
	t.Helper()
	if !SendTestCaseOf([]*SyncMockManager{own, node}, app, name, start, end, nil) {
		t.Fatalf("setup: %s's test case %s was left out", app, name)
	}
}

// A node's default manager takes the boot-time mocks of every app whose
// connections it cannot place yet, and notes every app's test cases. Each
// app's recording has its own first test case, which its replay serves its
// start-up tier from: app B's boot read, requested after app A's first test
// case and before B's own, is one any of B's tests may need. Its loss leaves
// out B's test cases from its request, with the shared loss's ERROR, though
// A's first test case came before it.
func TestAStartUpMockLostBeforeItsOwnAppsFirstTestCaseIsSharedThoughAnotherAppsCameFirst(t *testing.T) {
	node, appA, appB, logs := nodeManagers(t)
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appA, "A", "a-1", t0, t0.Add(time.Second))
	boot := appStartupAt("b-conn", "B", t0.Add(10*time.Second))
	if !node.servedToAnyTest(boot, "") {
		t.Fatal("B's boot read, before B's first test case, counts as a test's: A's first test case was read as B's")
	}
	node.dropOut(boot, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	if n := leftOutLines(logs); n != 1 {
		t.Fatalf("%d shared-loss ERRORs for B's lost boot read, want 1", n)
	}
	if SendTestCaseOf([]*SyncMockManager{appB, node}, "B", "b-1", t0.Add(20*time.Second), t0.Add(21*time.Second), nil) {
		t.Fatal("B's first test case, recorded after B's lost boot read, was kept")
	}
}

// Each app's first test case is its own: a start-up mock of A's requested
// after A's first test case is A's test's, while B has sent none (B's first
// is not A's), and once B has sent one, a start-up mock of B's requested
// after it is B's test's too. Neither loss leaves out test cases as shared.
func TestAStartUpMockAfterItsOwnAppsFirstTestCaseIsATestsWhileAnotherAppHasSentNone(t *testing.T) {
	node, appA, appB, logs := nodeManagers(t)
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appA, "A", "a-1", t0, t0.Add(time.Second))
	a := appStartupAt("a-conn", "A", t0.Add(5*time.Second))
	if node.servedToAnyTest(a, "") {
		t.Fatal("A's start-up mock after A's first test case counts as one any test may need while B has sent none")
	}
	node.dropOut(a, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	sendTo(t, node, appB, "B", "b-1", t0.Add(10*time.Second), t0.Add(11*time.Second))
	b := appStartupAt("b-conn", "B", t0.Add(15*time.Second))
	if node.servedToAnyTest(b, "") {
		t.Fatal("B's start-up mock after B's first test case counts as one any test may need")
	}
	node.dropOut(b, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	if n := leftOutLines(logs); n != 0 {
		t.Fatalf("%d shared-loss ERRORs for start-up mocks lost after their own app's first test case, want 0", n)
	}
	sendTo(t, node, appB, "B", "b-2", t0.Add(20*time.Second), t0.Add(21*time.Second))
}

// A start-up mock whose app the node cannot tell may be any app's, a new
// one's too, whose first test case is still to come: it counts as one any
// test may need, after every app's first test case, and after one sent with
// no app named.
func TestAStartUpMockOfAnAppTheNodeCannotTellIsOneAnyTestMayNeed(t *testing.T) {
	node, appA, appB, _ := nodeManagers(t)
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appA, "A", "a-1", t0, t0.Add(time.Second))
	sendTo(t, node, appB, "B", "b-1", t0.Add(2*time.Second), t0.Add(3*time.Second))
	if !node.SendingTestCase("no-app", t0.Add(4*time.Second), t0.Add(5*time.Second)) {
		t.Fatal("setup: a test case sent with no app named was left out")
	}
	if !node.servedToAnyTest(appStartupAt("x-conn", "", t0.Add(10*time.Second)), "") {
		t.Fatal("a start-up mock of an app the node cannot tell counts as a test's")
	}
}

// A manager whose mocks go to one recording (no SetDestinationOf), as an
// app's own does, notes a test case sent to a named app as that recording's:
// a start-up mock requested after it is a test's.
func TestAManagerOfOneRecordingNotesATestCaseSentToANamedAppAsItsOwn(t *testing.T) {
	own := New(nil)
	own.SetOutputChannel(make(chan *models.Mock, 1))
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, nil, own, "B", "b-1", t0, t0.Add(time.Second))
	if own.servedToAnyTest(appStartupAt("b-conn", "B", t0.Add(5*time.Second)), "") {
		t.Fatal("a start-up mock after the recording's first test case, sent to a named app, counts as one any test may need")
	}
	if f := own.firstSentOf(""); f == nil || !f.Equal(t0) {
		t.Fatalf("the recording's first test case is %v, want %v", f, t0)
	}
}

// A node's default manager tags every mock it takes a start-up one: the
// requests and test cases it counted are any app's, which cannot say the
// start-up window of the app a mock goes to is past. A manager of one
// recording tags none past its window.
func TestANodesManagerTagsEveryMockAStartUpOne(t *testing.T) {
	for _, tc := range []struct {
		name  string
		dest  func(*models.Mock) string
		wantS bool
	}{{"the node's", appOf, true}, {"one recording's", nil, false}} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(nil)
			m.SetOutputChannel(make(chan *models.Mock, 1))
			m.SetDestinationOf(tc.dest)
			m.mu.Lock()
			m.firstReqSeen, m.resolvedTestCount = true, models.StartupMockTestCaseWindow
			m.mu.Unlock()
			mk := addMockTo("c1", false)
			m.AddMock(mk)
			if mk.TestModeInfo.IsStartup != tc.wantS {
				t.Fatalf("a mock taken past the start-up window is tagged start-up %v, want %v", mk.TestModeInfo.IsStartup, tc.wantS)
			}
		})
	}
}

// A revoke that moves one app's first test case re-reads only a start-up
// mock of that app's lost as a test's: A's first test case taken out leaves
// B's lost read, after B's first, a test's. B's first taken out re-reads it,
// once, and the node's first test case of A's is gone.
func TestARevokeOfOneAppsFirstTestCaseReReadsOnlyThatAppsStartUpLoss(t *testing.T) {
	node, appA, appB, logs := nodeManagers(t)
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appA, "A", "a-1", t0, t0.Add(time.Second))
	sendTo(t, node, appB, "B", "b-1", t0.Add(2*time.Second), t0.Add(3*time.Second))
	read := appStartupAt("b-conn", "B", t0.Add(5*time.Second))
	node.dropOut(read, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	if n := leftOutLines(logs); n != 0 {
		t.Fatalf("setup: B's read after B's first test case said %d shared losses", n)
	}
	node.RevokeTestCase("a-1")
	if f := node.firstSentOf("A"); f != nil {
		t.Fatalf("A's first test case is %v once its only one was revoked", f)
	}
	if n := len(rereadLines(logs)); n != 0 {
		t.Fatalf("revoking A's first test case re-read %d start-up losses; B's first did not move", n)
	}
	node.RevokeTestCase("b-1")
	got := rereadLines(logs)
	if len(got) != 1 || got[0]["left_out_from"] != read.Spec.ReqTimestampMock.Format(time.RFC3339Nano) {
		t.Fatalf("revoking B's first test case re-read %v, want B's read once", got)
	}
}

// One revoke can move two apps' first test cases past their lost start-up
// mocks: the earliest of them is re-read, so the span reaches the other's
// request too.
func TestARevokeMovingTwoAppsFirstsReReadsTheEarliestLoss(t *testing.T) {
	node, appA, appB, logs := nodeManagers(t)
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appA, "A", "a-1", t0.Add(10*time.Second), t0.Add(11*time.Second))
	sendTo(t, node, appB, "B", "b-1", t0.Add(14*time.Second), t0.Add(15*time.Second))
	for _, mk := range []*models.Mock{appStartupAt("a-conn", "A", t0.Add(12*time.Second)), appStartupAt("b-conn", "B", t0.Add(16*time.Second))} {
		node.dropOut(mk, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	}
	// A's call no test owns, over both first test cases, lost: it revokes
	// both.
	call := callAt("a-conn2", t0.Add(10500*time.Millisecond))
	call.Spec.ResTimestampMock = t0.Add(14500 * time.Millisecond)
	call.Spec.Metadata = map[string]string{"app": "A"}
	node.dropOut(call, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	got := rereadLines(logs)
	if len(got) != 1 || got[0]["left_out_from"] != t0.Add(12*time.Second).Format(time.RFC3339Nano) {
		t.Fatalf("the revoke of both first test cases re-read %v, want A's read, the earliest, once", got)
	}
	if SendTestCaseOf([]*SyncMockManager{appB, node}, "B", "b-2", t0.Add(14600*time.Millisecond), t0.Add(15500*time.Millisecond), nil) {
		t.Fatal("a test case between the two re-read losses was kept")
	}
}

// The per-app lists add up: a span's revokes count every app's, a loss says
// the latest end of a test case any app's list cannot name, and a revoke by
// name finds its test case in its own app's list, whatever order the apps
// are walked in.
func TestTheAppsSentListsAddUp(t *testing.T) {
	t0 := time.Now().Add(-time.Hour)
	for i := 0; i < 20; i++ {
		var s sentByDest
		for d := 0; d < 8; d++ {
			s.of(fmt.Sprint("app-", d)).add(fmt.Sprint("tc-", d), t0.Add(time.Duration(d)*time.Second), t0.Add(time.Duration(d)*time.Second+500*time.Millisecond))
		}
		s.of("app-3").add("", t0.Add(30*time.Second), t0.Add(40*time.Second))
		if got := s.unnamedReaching(t0.Add(35 * time.Second)); !got.Equal(t0.Add(40 * time.Second)) {
			t.Fatalf("a loss from t0+35s may reach an unnamed test case through %v, want t0+40s", got)
		}
		if n := s.revokeReaching(t0.Add(2*time.Second), time.Time{}, true, func(string) {}); n != 6 {
			t.Fatalf("a span from t0+2s revoked %d test cases of the apps' lists, want 6", n)
		}
		// A revoke by name takes the test case out of its own app's list,
		// whichever list is walked first.
		for d := 0; d < 2; d++ {
			app := fmt.Sprint("app-", d)
			s.forget(fmt.Sprint("tc-", d), true)
			if first, ok, _ := s.first(app); ok {
				t.Fatalf("%s's only test case revoked by name, its first is still %v", app, first)
			}
		}
	}
}

// A start-up mock lost as a test's of its app, whose app the node can no
// longer tell by the time its loss is kept, may be any app's: it is re-read
// at once as one any test may need, with the ERROR.
func TestAStartUpLossWhoseAppTheNodeLostTrackOfIsReReadAtOnce(t *testing.T) {
	node, _, appB, logs := nodeManagers(t)
	calls := 0
	node.SetDestinationOf(func(mk *models.Mock) string {
		calls++
		if calls == 1 {
			return "B"
		}
		return ""
	})
	t0 := time.Now().Add(-time.Hour)
	sendTo(t, node, appB, "B", "b-1", t0, t0.Add(time.Second))
	read := appStartupAt("b-conn", "B", t0.Add(5*time.Second))
	node.dropOut(read, "", node.revokeEpoch.Load(), string(MockDroppedOutputFull))
	if calls < 2 {
		t.Fatalf("setup: the node named the lost read's app %d times, want it named as B, then not told", calls)
	}
	got := rereadLines(logs)
	if len(got) != 1 || got[0]["left_out_from"] != read.Spec.ReqTimestampMock.Format(time.RFC3339Nano) {
		t.Fatalf("the read whose app the node lost track of re-read %v, want it once", got)
	}
}
