package cache

import "testing"

func TestDeletePrefix(t *testing.T) {
	c, err := New(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"line_annotations:1:1", "line_annotations:1:2", "line_annotations:12:1", "story:1"} {
		if err := c.Set(k, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if n := c.DeletePrefix("line_annotations:1:"); n != 2 {
		t.Errorf("DeletePrefix removed %d, want 2", n)
	}
	for k, want := range map[string]bool{
		"line_annotations:1:1": false, "line_annotations:1:2": false,
		"line_annotations:12:1": true, "story:1": true,
	} {
		if _, err := c.Get(k); (err == nil) != want {
			t.Errorf("%s present = %v, want %v", k, err == nil, want)
		}
	}
}

func TestStoryAccessKeyChangesWithEpoch(t *testing.T) {
	kb := NewKeyBuilder()
	before := kb.StoryAccess("u", 1)
	BumpAccessEpoch()
	if after := kb.StoryAccess("u", 1); after == before {
		t.Errorf("key %q unchanged after BumpAccessEpoch", after)
	}
}
