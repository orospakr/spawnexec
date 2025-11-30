package spawnexec_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/orospakr/spawnexec"
)

// Comparison benchmarks between spawnexec and os/exec

// BenchmarkSpawnExecRunTrue benchmarks spawnexec.Command("true").Run()
func BenchmarkSpawnExecRunTrue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("true")
		cmd.Run()
	}
}

// BenchmarkOsExecRunTrue benchmarks exec.Command("true").Run()
func BenchmarkOsExecRunTrue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("true")
		cmd.Run()
	}
}

// BenchmarkSpawnExecRunEcho benchmarks spawnexec.Command("echo").Run()
func BenchmarkSpawnExecRunEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("echo", "hello")
		cmd.Run()
	}
}

// BenchmarkOsExecRunEcho benchmarks exec.Command("echo").Run()
func BenchmarkOsExecRunEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("echo", "hello")
		cmd.Run()
	}
}

// BenchmarkSpawnExecOutput benchmarks spawnexec.Command("echo").Output()
func BenchmarkSpawnExecOutput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("echo", "hello")
		cmd.Output()
	}
}

// BenchmarkOsExecOutput benchmarks exec.Command("echo").Output()
func BenchmarkOsExecOutput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("echo", "hello")
		cmd.Output()
	}
}

// BenchmarkSpawnExecWithStdin benchmarks spawnexec with stdin
func BenchmarkSpawnExecWithStdin(b *testing.B) {
	input := "hello world"
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("cat")
		cmd.Stdin = strings.NewReader(input)
		cmd.Output()
	}
}

// BenchmarkOsExecWithStdin benchmarks os/exec with stdin
func BenchmarkOsExecWithStdin(b *testing.B) {
	input := "hello world"
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("cat")
		cmd.Stdin = strings.NewReader(input)
		cmd.Output()
	}
}

// BenchmarkSpawnExecWithEnv benchmarks spawnexec with custom environment
func BenchmarkSpawnExecWithEnv(b *testing.B) {
	env := []string{"FOO=bar", "BAZ=qux"}
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("true")
		cmd.Env = env
		cmd.Run()
	}
}

// BenchmarkOsExecWithEnv benchmarks os/exec with custom environment
func BenchmarkOsExecWithEnv(b *testing.B) {
	env := []string{"FOO=bar", "BAZ=qux"}
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("true")
		cmd.Env = env
		cmd.Run()
	}
}

// BenchmarkSpawnExecCombinedOutput benchmarks spawnexec CombinedOutput
func BenchmarkSpawnExecCombinedOutput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := spawnexec.Command("sh", "-c", "echo out; echo err >&2")
		cmd.CombinedOutput()
	}
}

// BenchmarkOsExecCombinedOutput benchmarks os/exec CombinedOutput
func BenchmarkOsExecCombinedOutput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("sh", "-c", "echo out; echo err >&2")
		cmd.CombinedOutput()
	}
}

// TestCompareOutputWithOsExec verifies that spawnexec produces identical output to os/exec
func TestCompareOutputWithOsExec(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"echo hello", []string{"echo", "hello"}},
		{"echo multiple args", []string{"echo", "hello", "world", "foo", "bar"}},
		{"printf", []string{"printf", "%s %d\n", "test", "42"}},
		{"env var", []string{"sh", "-c", "echo $HOME"}},
		{"exit status zero", []string{"true"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run with spawnexec
			spawnCmd := spawnexec.Command(tt.args[0], tt.args[1:]...)
			spawnOut, spawnErr := spawnCmd.Output()

			// Run with os/exec
			osCmd := exec.Command(tt.args[0], tt.args[1:]...)
			osOut, osErr := osCmd.Output()

			// Compare results
			if (spawnErr == nil) != (osErr == nil) {
				t.Errorf("error mismatch: spawnexec err = %v, os/exec err = %v", spawnErr, osErr)
			}

			if !bytes.Equal(spawnOut, osOut) {
				t.Errorf("output mismatch: spawnexec = %q, os/exec = %q", spawnOut, osOut)
			}
		})
	}
}

// TestCompareExitCodeWithOsExec verifies that exit codes match
func TestCompareExitCodeWithOsExec(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{"exit 0", []string{"sh", "-c", "exit 0"}, 0},
		{"exit 1", []string{"sh", "-c", "exit 1"}, 1},
		{"exit 42", []string{"sh", "-c", "exit 42"}, 42},
		{"exit 255", []string{"sh", "-c", "exit 255"}, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run with spawnexec
			spawnCmd := spawnexec.Command(tt.args[0], tt.args[1:]...)
			spawnErr := spawnCmd.Run()

			// Run with os/exec
			osCmd := exec.Command(tt.args[0], tt.args[1:]...)
			osErr := osCmd.Run()

			// Extract exit codes
			var spawnCode, osCode int
			if spawnErr != nil {
				if ee, ok := spawnErr.(*spawnexec.ExitError); ok {
					spawnCode = ee.ExitCode()
				} else {
					t.Errorf("spawnexec unexpected error type: %T", spawnErr)
				}
			}
			if osErr != nil {
				if ee, ok := osErr.(*exec.ExitError); ok {
					spawnCode = ee.ExitCode()
				} else {
					t.Errorf("os/exec unexpected error type: %T", osErr)
				}
			}

			if spawnCode != tt.wantCode && osCode != tt.wantCode {
				t.Errorf("exit code = %d (spawnexec) / %d (os/exec), want %d", spawnCode, osCode, tt.wantCode)
			}
		})
	}
}

// TestCompareCombinedOutputWithOsExec verifies combined output matches
func TestCompareCombinedOutputWithOsExec(t *testing.T) {
	// Run with spawnexec
	spawnCmd := spawnexec.Command("sh", "-c", "echo stdout; echo stderr >&2")
	spawnOut, spawnErr := spawnCmd.CombinedOutput()

	// Run with os/exec
	osCmd := exec.Command("sh", "-c", "echo stdout; echo stderr >&2")
	osOut, osErr := osCmd.CombinedOutput()

	// Compare results
	if (spawnErr == nil) != (osErr == nil) {
		t.Errorf("error mismatch: spawnexec err = %v, os/exec err = %v", spawnErr, osErr)
	}

	// Both should contain stdout and stderr (order may vary)
	spawnHasStdout := bytes.Contains(spawnOut, []byte("stdout"))
	spawnHasStderr := bytes.Contains(spawnOut, []byte("stderr"))
	osHasStdout := bytes.Contains(osOut, []byte("stdout"))
	osHasStderr := bytes.Contains(osOut, []byte("stderr"))

	if spawnHasStdout != osHasStdout || spawnHasStderr != osHasStderr {
		t.Errorf("output content mismatch: spawnexec has stdout=%v stderr=%v, os/exec has stdout=%v stderr=%v",
			spawnHasStdout, spawnHasStderr, osHasStdout, osHasStderr)
	}
}
