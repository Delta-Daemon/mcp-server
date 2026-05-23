package resources

import "testing"

func TestParseStationURI(t *testing.T) {
	id, err := parseStationURI("deltadaemon://stations/KLAX")
	if err != nil {
		t.Fatal(err)
	}
	if id != "KLAX" {
		t.Fatalf("got %q", id)
	}
	_, err = parseStationURI("deltadaemon://docs/overview")
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}
