package tradeid

import "testing"

func TestRoundtrip(t *testing.T) {
	s, e := Format(7, 42, 0)
	if e != nil || s != "7:42:0" {
		t.Fatalf("%q %v", s, e)
	}
	id, seq, ord, e := Parse(s)
	if e != nil || id != 7 || seq != 42 || ord != 0 {
		t.Fatalf("%d %d %d %v", id, seq, ord, e)
	}
}
func TestInvalid(t *testing.T) {
	for _, s := range []string{"t1:7:42:0", "07:42:0", "7:0:0", "7:42:-1", "7:42:00", "4294967296:42:0"} {
		if _, _, _, e := Parse(s); e == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}

func TestZeroNumericMarketIsValid(t *testing.T) {
	s, e := Format(0, 1, 0)
	if e != nil || s != "0:1:0" {
		t.Fatalf("%q %v", s, e)
	}
	if _, _, _, e = Parse(s); e != nil {
		t.Fatal(e)
	}
}
