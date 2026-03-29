package wt

import "testing"

func TestPersistentPubSubHistory(t *testing.T) {
	pps := NewPersistentPubSub(10)

	pps.PublishPersistent("news", []byte("msg1"))
	pps.PublishPersistent("news", []byte("msg2"))
	pps.PublishPersistent("news", []byte("msg3"))

	if pps.HistoryLen("news") != 3 {
		t.Errorf("expected 3, got %d", pps.HistoryLen("news"))
	}
	if pps.HistoryLen("empty") != 0 {
		t.Errorf("expected 0 for empty topic, got %d", pps.HistoryLen("empty"))
	}
}

func TestPersistentPubSubClear(t *testing.T) {
	pps := NewPersistentPubSub(10)

	pps.PublishPersistent("news", []byte("msg"))
	pps.ClearHistory("news")

	if pps.HistoryLen("news") != 0 {
		t.Errorf("expected 0 after clear, got %d", pps.HistoryLen("news"))
	}
}

func TestPersistentPubSubOverflow(t *testing.T) {
	pps := NewPersistentPubSub(3)

	for i := range 5 {
		pps.PublishPersistent("topic", []byte{byte(i)})
	}

	if pps.HistoryLen("topic") != 3 {
		t.Errorf("expected 3 (capacity), got %d", pps.HistoryLen("topic"))
	}
}
