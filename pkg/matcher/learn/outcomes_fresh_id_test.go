package learn

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
	"time"
)

// Rows in the shapes api-server's T2 records left set aside: inserts of ids
// the application mints each run (app_name_reservations' appId, an
// entity_versions entity_id, testSuiteEvents' and app_config_audits' _id, a
// v4 UUID each) and a cluster's access key (32 random bytes in standard
// base64).

// A fresh UUID's 16-bit group stands, as a piece of another value, at an
// earlier place keploy cannot read whole for it: the recording's text holds
// it inside another id, and round F's holds round F's group glued inside a
// longer word. The application copies an id whole, and the place shows
// nothing of this id's 32-bit and 48-bit groups in either run, so it cannot
// be where the id came from: the group is chance there, and the id fresh,
// its line naming the place. The id's long words still judge the place: one
// that holds the recorded id's 32-bit group too stays unknown. A short word
// that is no piece of one id (its long neighbour stands in another token),
// and a place that shows the piece itself (the recorded group in the
// recording and round F's in round F, at one field), read as before.
func TestOutcomeAPieceOfAFreshIdAtAPlaceShowingNothingOfItsLongWordsIsChance(t *testing.T) {
	const (
		rec, live = "1111aaaa-2222-48d1-9444-444455556666", "9999cccc-7777-4a2e-8888-000011112222"
		r, f      = "48d1", "4a2e"
	)
	changed := func(mock, recText, liveText string) Earlier {
		return oCarried(mock, "body.note", recText, liveText, 2*time.Second, time.Second, 1)
	}
	chance := changed("mock-45", "owner bb7ed9ee-670e-48d1-982f-b54b2b73982f", "owner 0c1d2e3f-4a5b-4c7d-8e9f-4a2e0b1c2d3e")
	ds := Decide(oInput("body.docs.msgs.0.appId", rec, live, []Earlier{chance}, nil))
	for _, d := range ds {
		if d.Outcome != OutcomeFresh {
			t.Errorf("a fresh UUID beside a place holding a group of it inside another id: %s; want every word fresh", oDescribe([]Decision{d}))
		}
	}
	d := oWord(t, ds, r, f)
	costFresh(t, "a 16-bit group at a place that shows nothing of the id's long words", d)
	if !strings.Contains(d.Line, "mock-45") || !strings.Contains(d.Line, "piece of an id") {
		t.Errorf("the line must name mock-45 and say why it is chance: %q", d.Line)
	}

	again := changed("mock-44", "lead 70e1c2d3-48d1-4f00-8abc-0123456789ab", "lead 1a2b3c4d-4c7d-4e8f-9a0b-4a2e00000000")
	d = oWord(t, Decide(oInput("body.docs.msgs.0.appId", rec, live, []Earlier{again, chance}, nil)), r, f)
	costFresh(t, "a 16-bit group at two places that show nothing of the id's long words", d)
	if !strings.Contains(d.Line, "mock-44") || !strings.Contains(d.Line, "mock-45") {
		t.Errorf("the line must name both places the group is chance at: %q", d.Line)
	}

	long := changed("mock-46", "owner 1111aaaa and bb7ed9ee-670e-48d1-982f-b54b2b73982f", "owner 0c1d2e3f-4a5b-4c7d-8e9f-4a2e0b1c2d3e")
	ds = Decide(oInput("body.docs.msgs.0.appId", rec, live, []Earlier{long}, nil))
	if d := oWord(t, ds, "1111aaaa", "9999cccc"); d.Outcome != OutcomeRefused {
		t.Errorf("the recorded id's 32-bit group at the place: %s; want refused", oDescribe([]Decision{d}))
	}
	oRefused(t, "a 16-bit group where the place holds the id's 32-bit group too", oWord(t, ds, r, f), RefusalUnknown, "mock-46")

	other := changed("mock-47", "qty 48d1 ok", "qty 4a2e0b1c ok")
	ds = Decide(oInput("body.note", "qty 48d1 ref 1111aaaa2222", "qty 4a2e ref 9999cccc7777", []Earlier{other}, nil))
	oRefused(t, "a short word whose long neighbour stands in another token", oWord(t, ds, r, f), RefusalUnknown, "mock-47")

	shown := oCarried("mock-48", "body.code", r, f, 2*time.Second, time.Second, 1)
	ds = Decide(oInput("body.docs.msgs.0.appId", rec, live, []Earlier{shown}, nil))
	if d := oWord(t, ds, r, f); d.Outcome != OutcomeValue || d.From.Mock != "mock-48" {
		t.Errorf("a place that shows the group at one field in both runs: %s; want a Value bound at mock-48", oDescribe([]Decision{d}))
	}
}

// A cluster's access key: 32 random bytes in standard base64, 44 characters
// with one '=' of padding. Its '+' and '/' are letters of that alphabet, and
// stand elsewhere in round F, so the runs' words do not pair and word by
// word replay could never write it: the key is one value, fresh at its first
// send and bound where an earlier place shows it whole. Two alphabets mixed,
// a length no whole group of four, another padding in each run, and a run
// that decodes to text are no such key.
func TestOutcomeAStandardBase64KeyIsOneValue(t *testing.T) {
	const (
		rec  = "q3+Zb7Lr0Tf/9XcVw2Kd8Hn+Pa4Me6Gs1Jy5Ru0Io7U="
		live = "Xk2Nw9/Ft4Qz+Lm8Hc3Vb6Dp1Sg7Ja0Ey5Ro2Ui9Tn+="
	)
	if len(rec) != 44 || len(live) != 44 {
		t.Fatalf("the keys are %d and %d characters; want 44", len(rec), len(live))
	}
	ds := Decide(oInput("body.docs.msgs.0.access_key", rec, live, nil, nil))
	if len(ds) != 1 || ds[0].Outcome != OutcomeFresh || ds[0].At.Word != -1 || ds[0].Recorded != rec || ds[0].Full != live {
		t.Errorf("an access key whose '+' and '/' moved: %s; want one fresh value, the field whole", oDescribe(ds))
	}
	ds = Decide(oInput("body.cmd.filter.access_key", rec, live, oSent(rec, live), nil))
	if len(ds) != 1 || ds[0].Outcome != OutcomeValue || ds[0].From.Mock != "mock-0" {
		t.Errorf("a later send of the key: %s; want a Value bound at the first send", oDescribe(ds))
	}
	text := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	for _, tc := range []struct{ name, r, f string }{
		{"two alphabets mixed", rec, strings.Replace(live, "/", "_", 1)},
		{"no whole group of four", rec[:43], live[:43]},
		{"another padding in each run", rec[:42] + "==", live},
		{"a text in standard base64", text(`{"name":"Иван Петров"}+`), text(`{"name":"Пётр Сидоров"}/`)},
	} {
		ds := Decide(oInput("body.docs.msgs.0.access_key", tc.r, tc.f, nil, nil))
		if slices.ContainsFunc(ds, func(d Decision) bool { return d.Outcome == OutcomeFresh && d.At.Word < 0 }) {
			t.Errorf("FALSE GREEN? %s: %s; want no one fresh value", tc.name, oDescribe(ds))
		}
	}
}

// The id is the token the piece stands in, and its long words the changed
// words of 32 bits or more in both runs inside that token in both runs. A
// place that holds a 16-bit group inside another id, beside nothing of
// those, is chance only for such a piece; anywhere else it stays unknown,
// refused naming it: a short word that stands in no id token (a '/' ends
// one), a long word outside its token (before or after it), a neighbour or
// a word of fewer than 32 bits in one run, and a place that holds the id's
// one long word. (A word inside the token in one run only would need the
// text between the words to change: changed-shape, decided before.)
func TestOutcomeOnlyAPieceOfOneIdIsChanceBesideNothingOfItsLongWords(t *testing.T) {
	changed := func(mock, recText, liveText string) Earlier {
		return oCarried(mock, "body.note", recText, liveText, 2*time.Second, time.Second, 1)
	}
	inside := changed("mock-45", "owner bb7ed9ee-670e-48d1-982f-b54b2b73982f", "owner 0c1d2e3f-4a5b-4c7d-8e9f-4a2e0b1c2d3e")
	for _, tc := range []struct {
		what, rec, live, r, f string
		place                 Earlier
	}{
		{"a piece in no id token", "48d1/1111aaaa2222", "4a2e/9999cccc7777", "48d1", "4a2e", inside},
		{"the long word after the piece's token", "ab-48d1 ref 1111aaaa2222", "ab-4a2e ref 9999cccc7777", "48d1", "4a2e", inside},
		{"the long word before the piece's token", "1111aaaa2222 ref ab-48d1", "9999cccc7777 ref ab-4a2e", "48d1", "4a2e", inside},
		{"a neighbour of 32 bits in round F only", "11112222-48d1", "9999cccc-4a2e", "48d1", "4a2e", inside},
		{"a word of 32 bits in round F only", "11112222-aaaabbbbcccc", "9999cccc-ddddeeeeffff", "11112222", "9999cccc",
			changed("mock-45", "owner 0000ffff-11112222-abcd", "owner 9999cccc0b1c2d3e")},
		{"a place that holds the id's one long word", "ab12cd34-48d1", "ef56ab78-4a2e", "48d1", "4a2e",
			changed("mock-45", "owner 0000ffff-ab12cd34-48d1-0000", "owner 0c1d2e3f-4a5b-4c7d-8e9f-4a2e0b1c2d3e")},
	} {
		d := oWord(t, Decide(oInput("body.ref", tc.rec, tc.live, []Earlier{tc.place}, nil)), tc.r, tc.f)
		oRefused(t, tc.what, d, RefusalUnknown, "mock-45")
	}
}
