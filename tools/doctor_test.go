package tools

import (
	"os/exec"
	"testing"
)

// mirror_doctor.py is Python (standard library only, so a mirror admin needs no
// pip); its judgements are tested by its own unittest file, run here so the
// Go test suite covers it too.
func TestMirrorDoctor(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("no python3")
	}
	out, err := exec.Command("python3", "-m", "unittest", "mirror_doctor_test").CombinedOutput()
	if err != nil {
		t.Fatalf("python3 -m unittest mirror_doctor_test: %v\n%s", err, out)
	}
}
