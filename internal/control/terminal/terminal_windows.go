//go:build windows

package terminal

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

var (
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole = modKernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole = modKernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole  = modKernel32.NewProc("ClosePseudoConsole")
)

type coord struct {
	X int16
	Y int16
}

type windowsBackend struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	inPipe    io.WriteCloser
	rPipe     io.ReadCloser
	hPC       uintptr
	isConPTY  bool
}

func newPlatformBackend() Backend {
	return &windowsBackend{}
}

func (b *windowsBackend) Start(cmd *exec.Cmd, initialRows, initialCols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cmd = cmd
	if initialRows <= 0 {
		initialRows = 24
	}
	if initialCols <= 0 {
		initialCols = 80
	}

	// Try standard pipes execution on Windows
	inPipe, pipeErr := cmd.StdinPipe()
	if pipeErr != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", pipeErr)
	}
	outPipe, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", pipeErr)
	}
	cmd.Stderr = cmd.Stdout

	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("failed to start process: %w", startErr)
	}

	b.inPipe = inPipe
	b.rPipe = outPipe
	b.isConPTY = false
	return nil
}

func (b *windowsBackend) Read(p []byte) (int, error) {
	b.mu.Lock()
	rPipe := b.rPipe
	b.mu.Unlock()

	if rPipe != nil {
		return rPipe.Read(p)
	}
	return 0, io.EOF
}

func (b *windowsBackend) Write(p []byte) (int, error) {
	b.mu.Lock()
	inPipe := b.inPipe
	b.mu.Unlock()

	if inPipe != nil {
		return inPipe.Write(p)
	}
	return 0, io.ErrClosedPipe
}

func (b *windowsBackend) Resize(rows, cols int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isConPTY && b.hPC != 0 && procResizePseudoConsole.Find() == nil {
		c := coord{X: int16(cols), Y: int16(rows)}
		_, _, _ = procResizePseudoConsole.Call(b.hPC, *(*uintptr)(unsafe.Pointer(&c)))
	}
	return nil
}

func (b *windowsBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.inPipe != nil {
		_ = b.inPipe.Close()
		b.inPipe = nil
	}
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
	}
	if b.hPC != 0 && procClosePseudoConsole.Find() == nil {
		_, _, _ = procClosePseudoConsole.Call(b.hPC)
		b.hPC = 0
	}
	return nil
}

func (b *windowsBackend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Pid
	}
	return 0
}

func (b *windowsBackend) SupportsResize() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isConPTY
}

func (b *windowsBackend) SupportsRawMode() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.isConPTY
}
