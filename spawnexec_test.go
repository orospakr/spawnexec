package spawnexec

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestCommand tests the basic Command function
func TestCommand(t *testing.T) {
	cmd := Command("echo", "hello")
	if cmd.Path == "" {
		t.Error("Path should not be empty")
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "echo" || cmd.Args[1] != "hello" {
		t.Errorf("Args = %v, want [echo hello]", cmd.Args)
	}
}

// TestRun tests the Run method with a simple command
func TestRun(t *testing.T) {
	cmd := Command("true")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

// TestRunFail tests that Run returns an error for failing commands
func TestRunFail(t *testing.T) {
	cmd := Command("false")
	err := cmd.Run()
	if err == nil {
		t.Error("Run() error = nil, want non-nil")
	}
	if _, ok := err.(*ExitError); !ok {
		t.Errorf("Run() error type = %T, want *ExitError", err)
	}
}

// TestOutput tests the Output method
func TestOutput(t *testing.T) {
	cmd := Command("echo", "hello")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v, want nil", err)
	}
	want := "hello\n"
	if string(out) != want {
		t.Errorf("Output() = %q, want %q", out, want)
	}
}

// TestOutputMultipleArgs tests Output with multiple arguments
func TestOutputMultipleArgs(t *testing.T) {
	cmd := Command("echo", "hello", "world")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v, want nil", err)
	}
	want := "hello world\n"
	if string(out) != want {
		t.Errorf("Output() = %q, want %q", out, want)
	}
}

// TestCombinedOutput tests the CombinedOutput method
func TestCombinedOutput(t *testing.T) {
	cmd := Command("sh", "-c", "echo stdout; echo stderr >&2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("CombinedOutput() error = %v, want nil", err)
	}
	if !strings.Contains(string(out), "stdout") || !strings.Contains(string(out), "stderr") {
		t.Errorf("CombinedOutput() = %q, want to contain both stdout and stderr", out)
	}
}

// TestEnv tests that environment variables are passed correctly
func TestEnv(t *testing.T) {
	cmd := Command("sh", "-c", "echo $TEST_VAR")
	cmd.Env = append(os.Environ(), "TEST_VAR=hello_world")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v, want nil", err)
	}
	want := "hello_world\n"
	if string(out) != want {
		t.Errorf("Output() = %q, want %q", out, want)
	}
}

// TestDir tests that the working directory is set correctly
func TestDir(t *testing.T) {
	if !hasChdir() {
		t.Skip("chdir not supported on this platform")
	}

	tmpDir := t.TempDir()
	cmd := Command("pwd")
	cmd.Dir = tmpDir
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v, want nil", err)
	}
	// Resolve symlinks because macOS /tmp is a symlink to /private/tmp
	got := strings.TrimSpace(string(out))
	wantResolved, _ := filepath.EvalSymlinks(tmpDir)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("pwd output = %q (resolved: %q), want %q (resolved: %q)", got, gotResolved, tmpDir, wantResolved)
	}
}

// TestStdinPipe tests stdin pipe functionality
func TestStdinPipe(t *testing.T) {
	cmd := Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Write to stdin
	go func() {
		io.WriteString(stdin, "hello from pipe")
		stdin.Close()
	}()

	// Read from stdout
	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("ReadAll() error = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	want := "hello from pipe"
	if string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestStdoutPipe tests stdout pipe functionality
func TestStdoutPipe(t *testing.T) {
	cmd := Command("echo", "hello")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("ReadAll() error = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	want := "hello\n"
	if string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestStderrPipe tests stderr pipe functionality
func TestStderrPipe(t *testing.T) {
	cmd := Command("sh", "-c", "echo error >&2")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	out, err := io.ReadAll(stderr)
	if err != nil {
		t.Errorf("ReadAll() error = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	want := "error\n"
	if string(out) != want {
		t.Errorf("stderr = %q, want %q", out, want)
	}
}

// TestStdinReader tests using a Reader for Stdin
func TestStdinReader(t *testing.T) {
	cmd := Command("cat")
	cmd.Stdin = strings.NewReader("hello from reader")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v", err)
	}
	want := "hello from reader"
	if string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestStdoutWriter tests using a Writer for Stdout
func TestStdoutWriter(t *testing.T) {
	var buf bytes.Buffer
	cmd := Command("echo", "hello")
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	want := "hello\n"
	if buf.String() != want {
		t.Errorf("stdout = %q, want %q", buf.String(), want)
	}
}

// TestStderrWriter tests using a Writer for Stderr
func TestStderrWriter(t *testing.T) {
	var buf bytes.Buffer
	cmd := Command("sh", "-c", "echo error >&2")
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	want := "error\n"
	if buf.String() != want {
		t.Errorf("stderr = %q, want %q", buf.String(), want)
	}
}

// TestContext tests that context cancellation kills the process
func TestContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := CommandContext(ctx, "sleep", "10")
	err := cmd.Run()
	if err == nil {
		t.Error("Run() error = nil, want context deadline exceeded or killed")
	}
}

// TestContextCancel tests that canceling context kills the process
func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := CommandContext(ctx, "sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the process a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait should return an error
	err := cmd.Wait()
	if err == nil {
		t.Error("Wait() error = nil, want error after context cancel")
	}
}

// TestLookPath tests the LookPath function
func TestLookPath(t *testing.T) {
	path, err := LookPath("echo")
	if err != nil {
		t.Errorf("LookPath(echo) error = %v, want nil", err)
	}
	if path == "" {
		t.Error("LookPath(echo) = empty, want non-empty path")
	}
}

// TestLookPathNotFound tests LookPath with non-existent command
func TestLookPathNotFound(t *testing.T) {
	_, err := LookPath("this-command-definitely-does-not-exist-xyz123")
	if err == nil {
		t.Error("LookPath() error = nil, want error")
	}
}

// TestCommandNotFound tests running a non-existent command
func TestCommandNotFound(t *testing.T) {
	cmd := Command("this-command-definitely-does-not-exist-xyz123")
	err := cmd.Run()
	if err == nil {
		t.Error("Run() error = nil, want error")
	}
}

// TestStartTwice tests that Start cannot be called twice
func TestStartTwice(t *testing.T) {
	cmd := Command("echo", "hello")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer cmd.Wait()

	err := cmd.Start()
	if err == nil {
		t.Error("second Start() error = nil, want error")
	}
}

// TestWaitWithoutStart tests that Wait fails if Start wasn't called
func TestWaitWithoutStart(t *testing.T) {
	cmd := Command("echo", "hello")
	err := cmd.Wait()
	if err == nil {
		t.Error("Wait() without Start() error = nil, want error")
	}
}

// TestWaitTwice tests that Wait cannot be called twice
func TestWaitTwice(t *testing.T) {
	cmd := Command("echo", "hello")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	err := cmd.Wait()
	if err == nil {
		t.Error("second Wait() error = nil, want error")
	}
}

// TestExitCode tests that ExitError contains the correct exit code
func TestExitCode(t *testing.T) {
	cmd := Command("sh", "-c", "exit 42")
	err := cmd.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want ExitError")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("error type = %T, want *ExitError", err)
	}
	if exitErr.ExitCode() != 42 {
		t.Errorf("ExitCode() = %d, want 42", exitErr.ExitCode())
	}
}

// TestProcessState tests that ProcessState is populated after Wait
func TestProcessState(t *testing.T) {
	cmd := Command("echo", "hello")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if cmd.ProcessState == nil {
		t.Error("ProcessState = nil, want non-nil")
	}
	if !cmd.ProcessState.Exited() {
		t.Error("ProcessState.Exited() = false, want true")
	}
	if !cmd.ProcessState.Success() {
		t.Error("ProcessState.Success() = false, want true")
	}
	if cmd.ProcessState.ExitCode() != 0 {
		t.Errorf("ProcessState.ExitCode() = %d, want 0", cmd.ProcessState.ExitCode())
	}
}

// TestString tests the String method
func TestString(t *testing.T) {
	cmd := Command("echo", "hello", "world")
	s := cmd.String()
	// String should contain the command path and arguments
	if !strings.Contains(s, "echo") || !strings.Contains(s, "hello") || !strings.Contains(s, "world") {
		t.Errorf("String() = %q, want to contain echo, hello, world", s)
	}
}

// TestEnviron tests the Environ method
func TestEnviron(t *testing.T) {
	cmd := Command("echo")
	env := cmd.Environ()
	if len(env) == 0 {
		t.Error("Environ() = empty, want non-empty (should inherit os.Environ)")
	}

	// Test with custom env
	cmd2 := Command("echo")
	cmd2.Env = []string{"FOO=bar"}
	env2 := cmd2.Environ()
	if len(env2) != 1 || env2[0] != "FOO=bar" {
		t.Errorf("Environ() = %v, want [FOO=bar]", env2)
	}
}

// TestLargeOutput tests that large output is handled correctly
func TestLargeOutput(t *testing.T) {
	// Generate a large output (1MB)
	size := 1024 * 1024
	cmd := Command("sh", "-c", "dd if=/dev/zero bs=1024 count=1024 2>/dev/null | tr '\\0' 'x'")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("Output() error = %v", err)
	}
	if len(out) != size {
		t.Errorf("output length = %d, want %d", len(out), size)
	}
}

// TestOutputAlreadySet tests that Output fails if Stdout is already set
func TestOutputAlreadySet(t *testing.T) {
	cmd := Command("echo", "hello")
	cmd.Stdout = os.Stdout
	_, err := cmd.Output()
	if err == nil {
		t.Error("Output() with Stdout already set error = nil, want error")
	}
}

// TestCombinedOutputAlreadySet tests that CombinedOutput fails if Stdout/Stderr is already set
func TestCombinedOutputAlreadySet(t *testing.T) {
	cmd := Command("echo", "hello")
	cmd.Stdout = os.Stdout
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("CombinedOutput() with Stdout already set error = nil, want error")
	}

	cmd2 := Command("echo", "hello")
	cmd2.Stderr = os.Stderr
	_, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		t.Error("CombinedOutput() with Stderr already set error = nil, want error")
	}
}

// TestStdinPipeAlreadySet tests that StdinPipe fails if Stdin is already set
func TestStdinPipeAlreadySet(t *testing.T) {
	cmd := Command("cat")
	cmd.Stdin = strings.NewReader("hello")
	_, err := cmd.StdinPipe()
	if err == nil {
		t.Error("StdinPipe() with Stdin already set error = nil, want error")
	}
}

// TestStdoutPipeAlreadySet tests that StdoutPipe fails if Stdout is already set
func TestStdoutPipeAlreadySet(t *testing.T) {
	cmd := Command("echo", "hello")
	cmd.Stdout = os.Stdout
	_, err := cmd.StdoutPipe()
	if err == nil {
		t.Error("StdoutPipe() with Stdout already set error = nil, want error")
	}
}

// TestStderrPipeAlreadySet tests that StderrPipe fails if Stderr is already set
func TestStderrPipeAlreadySet(t *testing.T) {
	cmd := Command("echo", "hello")
	cmd.Stderr = os.Stderr
	_, err := cmd.StderrPipe()
	if err == nil {
		t.Error("StderrPipe() with Stderr already set error = nil, want error")
	}
}

// TestPipeAfterStart tests that pipe methods fail after Start
func TestPipeAfterStart(t *testing.T) {
	cmd := Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer cmd.Wait()

	_, err := cmd.StdinPipe()
	if err == nil {
		t.Error("StdinPipe() after Start() error = nil, want error")
	}

	_, err = cmd.StdoutPipe()
	if err == nil {
		t.Error("StdoutPipe() after Start() error = nil, want error")
	}

	_, err = cmd.StderrPipe()
	if err == nil {
		t.Error("StderrPipe() after Start() error = nil, want error")
	}
}

// BenchmarkRunEcho benchmarks running a simple echo command
func BenchmarkRunEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := Command("echo", "hello")
		cmd.Run()
	}
}

// BenchmarkOutputEcho benchmarks Output with echo
func BenchmarkOutputEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := Command("echo", "hello")
		cmd.Output()
	}
}

// BenchmarkRunTrue benchmarks running /bin/true (minimal work)
func BenchmarkRunTrue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := Command("true")
		cmd.Run()
	}
}

// TestProcessPid tests that Process.Pid is set correctly
func TestProcessPid(t *testing.T) {
	cmd := Command("sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if cmd.Process == nil {
		t.Fatal("Process = nil after Start")
	}
	if cmd.Process.Pid <= 0 {
		t.Errorf("Process.Pid = %d, want > 0", cmd.Process.Pid)
	}

	cmd.Wait()
}

// TestProcessKill tests killing a process
func TestProcessKill(t *testing.T) {
	cmd := Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the process time to start
	time.Sleep(50 * time.Millisecond)

	if err := cmd.Process.Kill(); err != nil {
		t.Errorf("Kill() error = %v", err)
	}

	err := cmd.Wait()
	if err == nil {
		t.Error("Wait() after Kill() error = nil, want error")
	}
}

// TestIsDarwin reports whether we're running the posix_spawn implementation
func TestIsDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Log("Running on darwin - using posix_spawn implementation")
	} else {
		t.Log("Running on", runtime.GOOS, "- using os/exec fallback")
	}
}

// TestStdinEmptyEOF tests that EOF is properly sent when stdin is empty
func TestStdinEmptyEOF(t *testing.T) {
	// Use 'cat' which should exit immediately when receiving EOF on empty stdin
	cmd := Command("cat")
	cmd.Stdin = strings.NewReader("")

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() with empty stdin error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Command with empty stdin timed out - EOF may not have been sent")
	}
}

// TestStdinZeroLengthReader tests stdin with a zero-length io.Reader
func TestStdinZeroLengthReader(t *testing.T) {
	cmd := Command("cat")
	cmd.Stdin = bytes.NewReader([]byte{})

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() with zero-length reader error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Command with zero-length stdin timed out")
	}
}

// TestStdinLargeData tests that large stdin data is handled correctly
func TestStdinLargeData(t *testing.T) {
	// Generate 10MB of data
	size := 10 * 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	cmd := Command("cat")
	cmd.Stdin = bytes.NewReader(data)

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output() with large stdin error = %v", err)
	}

	if len(out) != size {
		t.Errorf("output length = %d, want %d", len(out), size)
	}

	if !bytes.Equal(out, data) {
		t.Error("output does not match input data")
	}
}

// TestStdinProcessReadsUntilEOF tests a process that explicitly reads until EOF
func TestStdinProcessReadsUntilEOF(t *testing.T) {
	// Use 'wc -c' which counts bytes and only exits after receiving EOF
	cmd := Command("wc", "-c")
	cmd.Stdin = strings.NewReader("hello world")

	done := make(chan error, 1)
	go func() {
		_, err := cmd.Output()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Output() with wc -c error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Process reading until EOF timed out - EOF may not have been sent")
	}
}

// TestStdinMultipleWrites tests multiple writes to stdin pipe
func TestStdinMultipleWrites(t *testing.T) {
	cmd := Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Write multiple chunks
	writes := []string{"first ", "second ", "third"}
	done := make(chan error, 1)
	go func() {
		for _, data := range writes {
			if _, err := io.WriteString(stdin, data); err != nil {
				done <- err
				return
			}
		}
		stdin.Close()
		done <- nil
	}()

	// Read output
	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("ReadAll() error = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	if err := <-done; err != nil {
		t.Errorf("Write goroutine error = %v", err)
	}

	want := "first second third"
	if string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// TestStdinBinaryData tests that binary data (including null bytes) is handled correctly
func TestStdinBinaryData(t *testing.T) {
	// Create binary data with null bytes and various byte values
	data := []byte{0, 1, 2, 3, 255, 254, 253, 0, 0, 128, 127}

	cmd := Command("cat")
	cmd.Stdin = bytes.NewReader(data)

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output() with binary data error = %v", err)
	}

	if !bytes.Equal(out, data) {
		t.Errorf("output = %v, want %v", out, data)
	}
}

// TestStdinSlowReader tests stdin with a slow-reading process
func TestStdinSlowReader(t *testing.T) {
	// Use a shell script that reads one byte at a time with small delays
	// This tests that the stdin pipe doesn't close prematurely
	script := `
while IFS= read -r -n1 char; do
	printf '%s' "$char"
done
printf '\n'
`
	cmd := Command("sh", "-c", script)
	cmd.Stdin = strings.NewReader("slow")

	done := make(chan []byte, 1)
	errChan := make(chan error, 1)
	go func() {
		out, err := cmd.Output()
		if err != nil {
			errChan <- err
			return
		}
		done <- out
	}()

	select {
	case err := <-errChan:
		t.Errorf("Output() with slow reader error = %v", err)
	case out := <-done:
		if string(out) != "slow\n" {
			t.Errorf("output = %q, want %q", out, "slow\n")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Slow reader test timed out")
	}
}

// TestStdinPipeCloseTiming tests that closing stdin pipe at the right time allows process to exit
func TestStdinPipeCloseTiming(t *testing.T) {
	cmd := Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Write some data
	io.WriteString(stdin, "data")

	// Close stdin immediately - this should send EOF to the process
	stdin.Close()

	// Wait should complete quickly since cat should exit after receiving EOF
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wait() after stdin close error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() timed out after stdin close - EOF may not have been properly sent")
	}
}

// TestStdinReaderEOFImmediate tests that a reader that returns EOF immediately works
func TestStdinReaderEOFImmediate(t *testing.T) {
	// Create a custom reader that returns EOF on first read
	eofReader := &immediateEOFReader{}

	cmd := Command("cat")
	cmd.Stdin = eofReader

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() with immediate EOF reader error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Command with immediate EOF reader timed out")
	}
}

// TestStdinLongRunningProcess tests stdin with a process that takes time to process input
func TestStdinLongRunningProcess(t *testing.T) {
	// Use a shell script that processes input then sleeps briefly
	script := `
cat
sleep 0.1
`
	cmd := Command("sh", "-c", script)
	cmd.Stdin = strings.NewReader("test data")

	done := make(chan error, 1)
	go func() {
		_, err := cmd.Output()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Output() with long-running process error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Long-running process test timed out")
	}
}

// TestStdinPipeWithNoWrite tests that a process exits when stdin pipe is created but never written to
func TestStdinPipeWithNoWrite(t *testing.T) {
	cmd := Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Close stdin immediately without writing - should send EOF
	stdin.Close()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wait() with no-write stdin error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Command with no-write stdin timed out")
	}
}

// immediateEOFReader is a helper that returns EOF on the first Read
type immediateEOFReader struct{}

func (r *immediateEOFReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
