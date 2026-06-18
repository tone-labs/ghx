package cli

import (
	"testing"

	"github.com/cbuchan/ghx/internal/model"
)

func fixture() *model.PR {
	return &model.PR{
		Reviews: []model.Review{
			{Author: "human", IsBot: false, State: "CHANGES_REQUESTED"},
			{Author: "bot[bot]", IsBot: true, State: "APPROVED"},
		},
		Conversation: []model.Comment{
			{Author: "human", IsBot: false},
			{Author: "bot[bot]", IsBot: true},
		},
		Threads: []model.Thread{
			{ID: "open-human", IsResolved: false, Comments: []model.Comment{{Author: "human"}}},
			{ID: "resolved-human", IsResolved: true, Comments: []model.Comment{{Author: "human"}}},
			{ID: "open-outdated-bot", IsResolved: false, IsOutdated: true, Comments: []model.Comment{{Author: "bot[bot]", IsBot: true}}},
		},
	}
}

func threadIDs(pr *model.PR) []string {
	var ids []string
	for _, t := range pr.Threads {
		ids = append(ids, t.ID)
	}
	return ids
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFilterDefaultUnresolved(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-human", "open-outdated-bot"}) {
		t.Errorf("default should drop resolved, got %v", got)
	}
}

func TestFilterAllIncludesResolved(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{all: true}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-human", "resolved-human", "open-outdated-bot"}) {
		t.Errorf("--all should keep resolved, got %v", got)
	}
}

func TestFilterHideOutdated(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{all: true, hideOutdated: true}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-human", "resolved-human"}) {
		t.Errorf("--hide-outdated should drop outdated, got %v", got)
	}
}

func TestFilterBots(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{all: true, bots: true}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-outdated-bot"}) {
		t.Errorf("--bots should keep only bot threads, got %v", got)
	}
	if len(pr.Reviews) != 1 || pr.Reviews[0].Author != "bot[bot]" {
		t.Errorf("--bots should filter reviews to bots, got %+v", pr.Reviews)
	}
	if len(pr.Conversation) != 1 || !pr.Conversation[0].IsBot {
		t.Errorf("--bots should filter conversation to bots, got %+v", pr.Conversation)
	}
}

func TestFilterHumans(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{all: true, humans: true}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-human", "resolved-human"}) {
		t.Errorf("--humans should drop bot threads, got %v", got)
	}
}

func TestFilterAuthorOverridesType(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{all: true, bots: true, author: "human"}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"open-human", "resolved-human"}) {
		t.Errorf("--author should override --bots, got %v", got)
	}
}

func TestFilterThreadDrillIn(t *testing.T) {
	pr := fixture()
	full := false
	commentFilter{thread: "resolved-human"}.apply(pr, &full)
	if got := threadIDs(pr); !eq(got, []string{"resolved-human"}) {
		t.Errorf("--thread should isolate one thread (even if resolved), got %v", got)
	}
	if !full {
		t.Error("--thread should force full output")
	}
	if pr.Reviews != nil || pr.Conversation != nil {
		t.Error("--thread should suppress reviews and conversation")
	}
}
