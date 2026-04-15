package plc

import "testing"

func TestParseSiemensAddressDB(t *testing.T) {
	addr, err := parseSiemensAddress("DB1.DBW20")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if addr.area != siemensAreaDB || addr.db != 1 || addr.pos != 20 {
		t.Fatalf("unexpected parse result: %#v", addr)
	}
}

func TestParseSiemensAddressMemory(t *testing.T) {
	addr, err := parseSiemensAddress("M100")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if addr.area != siemensAreaM || addr.pos != 100 {
		t.Fatalf("unexpected parse result: %#v", addr)
	}
}

func TestParseSiemensAddressInvalid(t *testing.T) {
	if _, err := parseSiemensAddress("D100"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestParseSiemensAddressOutOfRange(t *testing.T) {
	if _, err := parseSiemensAddress("DB70000.DBW1"); err == nil {
		t.Fatalf("expected out-of-range db number error")
	}
	if _, err := parseSiemensAddress("M70000"); err == nil {
		t.Fatalf("expected out-of-range memory address error")
	}
}
