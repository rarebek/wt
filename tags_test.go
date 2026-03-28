package wt

import "testing"

func TestTagsTagUntag(t *testing.T) {
	tags := NewTags()

	tags.Tag("s1", "premium")
	tags.Tag("s1", "admin")
	tags.Tag("s2", "premium")

	if !tags.HasTag("s1", "premium") {
		t.Error("s1 should have premium tag")
	}
	if !tags.HasTag("s1", "admin") {
		t.Error("s1 should have admin tag")
	}
	if tags.HasTag("s2", "admin") {
		t.Error("s2 should not have admin tag")
	}

	if tags.Count("premium") != 2 {
		t.Errorf("expected 2 premium, got %d", tags.Count("premium"))
	}

	tags.Untag("s1", "premium")
	if tags.HasTag("s1", "premium") {
		t.Error("s1 should not have premium after untag")
	}
	if tags.Count("premium") != 1 {
		t.Errorf("expected 1 premium after untag, got %d", tags.Count("premium"))
	}
}

func TestTagsUntagAll(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "a")
	tags.Tag("s1", "b")
	tags.Tag("s1", "c")

	tags.UntagAll("s1")

	if len(tags.TagsForSession("s1")) != 0 {
		t.Error("expected no tags after UntagAll")
	}
}

func TestTagsSessionsWithTag(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "vip")
	tags.Tag("s2", "vip")
	tags.Tag("s3", "vip")

	vips := tags.SessionsWithTag("vip")
	if len(vips) != 3 {
		t.Errorf("expected 3 vips, got %d", len(vips))
	}
}

func TestTagsAllTags(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "a")
	tags.Tag("s1", "b")
	tags.Tag("s2", "c")

	all := tags.AllTags()
	if len(all) != 3 {
		t.Errorf("expected 3 tags, got %d", len(all))
	}
}

func TestTagsEmpty(t *testing.T) {
	tags := NewTags()

	if tags.HasTag("nonexistent", "tag") {
		t.Error("should be false for nonexistent")
	}
	if tags.Count("empty") != 0 {
		t.Error("should be 0")
	}
	if len(tags.SessionsWithTag("nothing")) != 0 {
		t.Error("should be empty")
	}
}
