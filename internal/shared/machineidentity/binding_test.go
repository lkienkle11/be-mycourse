package machineidentity

import "testing"

func TestBuildHybridMaterialFromParts(t *testing.T) {
	got := buildHybridMaterialFromParts("filesecret", machineFingerprint{
		MachineID:    "mid-1",
		HardwareUUID: "hw-1",
		Hostname:     "host-a",
		Platform:     "linux/amd64",
	})
	want := "v1|file:filesecret|mid:mid-1|hw:hw-1|host:host-a|plat:linux/amd64"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildHybridMaterialFromPartsDeterministic(t *testing.T) {
	fp := machineFingerprint{
		MachineID:    "a",
		HardwareUUID: "b",
		Hostname:     "c",
		Platform:     "darwin/arm64",
	}
	a := buildHybridMaterialFromParts("s", fp)
	b := buildHybridMaterialFromParts("s", fp)
	if a != b {
		t.Fatal("expected deterministic hybrid material")
	}
}
