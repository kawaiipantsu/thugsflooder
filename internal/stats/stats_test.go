package stats

import "testing"

func TestRecordSentAggregates(t *testing.T) {
	s := New()
	s.RecordSent("udp", "127.0.0.1", 100)
	s.RecordSent("udp", "127.0.0.1", 50)
	s.RecordSent("tcp", "10.0.0.5", 200)
	s.RecordBlocked()
	s.RecordDropped()
	s.RecordDropped()

	snap := s.Snapshot()
	if snap.Sent != 3 {
		t.Errorf("Sent = %d, want 3", snap.Sent)
	}
	if snap.Bytes != 350 {
		t.Errorf("Bytes = %d, want 350", snap.Bytes)
	}
	if snap.Blocked != 1 {
		t.Errorf("Blocked = %d, want 1", snap.Blocked)
	}
	if snap.Dropped != 2 {
		t.Errorf("Dropped = %d, want 2", snap.Dropped)
	}
	if snap.ActiveHosts != 2 {
		t.Errorf("ActiveHosts = %d, want 2", snap.ActiveHosts)
	}
	if snap.PerProto["udp"] != 2 || snap.PerProto["tcp"] != 1 {
		t.Errorf("PerProto = %v, want udp=2 tcp=1", snap.PerProto)
	}
}

func TestTickRollsRateHistory(t *testing.T) {
	s := New()
	for i := 0; i < 10; i++ {
		s.RecordSent("udp", "127.0.0.1", 1)
	}
	pps := s.Tick()
	if pps <= 0 {
		t.Errorf("Tick() = %v, want > 0 after 10 sends", pps)
	}

	snap := s.Snapshot()
	if len(snap.RateSamples) != 1 {
		t.Errorf("RateSamples len = %d, want 1", len(snap.RateSamples))
	}
	if snap.CurrentPPS != pps {
		t.Errorf("CurrentPPS = %v, want %v", snap.CurrentPPS, pps)
	}
}
