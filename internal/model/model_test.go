package model

import "testing"

func TestIsBotActor(t *testing.T) {
	cases := []struct {
		login, typename string
		want            bool
	}{
		{"dependabot[bot]", "Bot", true},
		{"copilot-pull-request-reviewer", "Bot", true}, // typename wins even without suffix
		{"renovate[bot]", "User", true},                // suffix fallback when typename misreports
		{"alice", "User", false},
		{"", "", false},
	}
	for _, c := range cases {
		if got := IsBotActor(c.login, c.typename); got != c.want {
			t.Errorf("IsBotActor(%q,%q)=%v want %v", c.login, c.typename, got, c.want)
		}
	}
}

func TestThreadStarter(t *testing.T) {
	if (Thread{}).Starter() != nil {
		t.Fatal("empty thread should have nil starter")
	}
	th := Thread{Comments: []Comment{{Author: "a"}, {Author: "b"}}}
	if s := th.Starter(); s == nil || s.Author != "a" {
		t.Fatalf("starter = %+v, want author a", s)
	}
}
