//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Windows Console Pseudoterminal (ConPTY) backend.
//
// Uses the real Win32 pseudo-console API:
//
//	CreatePseudoConsole / ResizePseudoConsole / ClosePseudoConsole
//
// and launches the child via CreateProcessW with the
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attribute. If ConPTY is unavailable
// (e.g. Windows < 10.0.17763), Start falls back to standard pipes and reports
// the backend truthfully as "standard pipes" (no resize / no raw mode).

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
	createUnicodeEnvironment         = 0x00000400
	stillActive                      = 0x00000103 // 259
)

var (
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole = modKernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole = modKernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole  = modKernel32.NewProc("ClosePseudoConsole")

	procCreatePipe                        = modKernel32.NewProc("CreatePipe")
	procCreateProcessW                    = modKernel32.NewProc("CreateProcessW")
	procInitializeProcThreadAttributeList = modKernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = modKernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList     = modKernel32.NewProc("DeleteProcThreadAttributeList")
	procCloseHandle                       = modKernel32.NewProc("CloseHandle")
	procWaitForSingleObject               = modKernel32.NewProc("WaitForSingleObject")
	procGetExitCodeProcess                = modKernel32.NewProc("GetExitCodeProcess")
	procTerminateProcess                  = modKernel32.NewProc("TerminateProcess")
)

type coord struct {
	X int16
	Y int16
}

type startupInfo struct {
	Cb              uint32
	LpReserved      *uint16
	LpDesktop       *uint16
	LpTitle         *uint16
	DwX             uint32
	DwY             uint32
	DwXSize         uint32
	DwYSize         uint32
	DwXCountChars   uint32
	DwYCountChars   uint32
	DwFillAttribute uint32
	DwFlags         uint32
	WShowWindow     uint16
	CbReserved2     uint16
	LpReserved2     *byte
	HStdInput       uintptr
	HStdOutput      uintptr
	HStdError       uintptr
}

type startupInfoEx struct {
	StartupInfo     startupInfo
	LpAttributeList uintptr
}

type processInformation struct {
	Process   uintptr
	Thread    uintptr
	ProcessId uint32
	ThreadId  uint32
}

type windowsBackend struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	inPipe    io.WriteCloser
	rPipe     io.ReadCloser
	hProcess  uintptr
	pid       int
	hPC       uintptr
	isConPTY  bool
	mechanism string
}

func newPlatformBackend() Backend {
	return &windowsBackend{}
}

// BackendMechanism reports the platform's primary terminal backend.
func BackendMechanism() string { return "ConPTY (CreatePseudoConsole)" }

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

	// 1. Create anonymous pipes used as the ConPTY transport.
	var hInRead, hInWrite uintptr
	r, _, e := procCreatePipe.Call(
		uintptr(unsafe.Pointer(&hInRead)),
		uintptr(unsafe.Pointer(&hInWrite)),
		0, 0)
	if r == 0 {
		return fmt.Errorf("CreatePipe failed (input): %v", e)
	}
	var hOutRead, hOutWrite uintptr
	r, _, e = procCreatePipe.Call(
		uintptr(unsafe.Pointer(&hOutRead)),
		uintptr(unsafe.Pointer(&hOutWrite)),
		0, 0)
	if r == 0 {
		_ = closeHandle(hInRead)
		_ = closeHandle(hInWrite)
		return fmt.Errorf("CreatePipe failed (output): %v", e)
	}

	// 2. Create the pseudo console. Size is passed by value as a packed COORD.
	sizeVal := *(*uint32)(unsafe.Pointer(&coord{X: int16(initialCols), Y: int16(initialRows)}))
	var hPC uintptr
	r, _, e = procCreatePseudoConsole.Call(
		uintptr(sizeVal), uintptr(hInRead), uintptr(hOutWrite), 0, uintptr(unsafe.Pointer(&hPC)))
	if r != 0 {
		// ConPTY unavailable: fall back to standard pipes, reported truthfully.
		childIn := os.NewFile(hInRead, "conpty-fallback-child-in")
		inFile := os.NewFile(hInWrite, "conpty-fallback-in")
		outFile := os.NewFile(hOutRead, "conpty-fallback-out")
		b.inPipe = inFile
		b.rPipe = outFile
		b.pid = 0
		b.isConPTY = false
		b.mechanism = "standard pipes (ConPTY unavailable)"
		_ = closeHandle(hOutWrite)
		cmd.Stdin = childIn
		cmd.Stdout = outFile
		cmd.Stderr = outFile
		if err := cmd.Start(); err != nil {
			_ = closeHandle(hInWrite)
			_ = closeHandle(hOutRead)
			if childIn != nil {
				_ = childIn.Close()
			}
			return fmt.Errorf("failed to start process (pipe fallback): %w", err)
		}
		b.pid = cmd.Process.Pid
		b.cmd = cmd
		return nil
	}

	// 3. Parent retains the client ends; ConPTY owns the server ends.
	_ = closeHandle(hInRead)
	_ = closeHandle(hOutWrite)

	// 4. Build the PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attribute list.
	var attrSize uintptr
	procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrSize)))
	if attrSize == 0 {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return errors.New("InitializeProcThreadAttributeList failed to size attribute list")
	}
	attrBuf := make([]byte, attrSize)
	r, _, e = procInitializeProcThreadAttributeList.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])), 1, 0, uintptr(unsafe.Pointer(&attrSize)))
	if r == 0 {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("InitializeProcThreadAttributeList failed: %v", e)
	}
	defer procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrBuf[0])))

	r, _, e = procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])),
		0,
		procThreadAttributePseudoConsole,
		uintptr(unsafe.Pointer(&hPC)),
		unsafe.Sizeof(hPC),
		0, 0)
	if r == 0 {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("UpdateProcThreadAttribute failed: %v", e)
	}

	// 5. Launch the child attached to the pseudo console.
	appName, err := syscall.UTF16PtrFromString(cmd.Path)
	if err != nil {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("invalid binary path %q: %w", cmd.Path, err)
	}
	commandLine, err := syscall.UTF16FromString(buildCommandLine(cmd.Args))
	if err != nil {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("invalid command line: %w", err)
	}
	envBlock := utf16EnvBlock(cmd.Environ())
	var dirPtr *uint16
	if cmd.Dir != "" {
		dirPtr, err = syscall.UTF16PtrFromString(cmd.Dir)
		if err != nil {
			_ = closeHandle(hInWrite)
			_ = closeHandle(hOutRead)
			_ = closePseudoConsole(hPC)
			return fmt.Errorf("invalid working directory %q: %w", cmd.Dir, err)
		}
	}

	siEx := startupInfoEx{}
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))
	siEx.LpAttributeList = uintptr(unsafe.Pointer(&attrBuf[0]))

	var pi processInformation
	r, _, e = procCreateProcessW.Call(
		uintptr(unsafe.Pointer(appName)),
		uintptr(unsafe.Pointer(&commandLine[0])),
		0, 0, 0,
		extendedStartupInfoPresent|createUnicodeEnvironment,
		uintptr(unsafe.Pointer(&envBlock[0])),
		uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&siEx)),
		uintptr(unsafe.Pointer(&pi)))
	if r == 0 {
		_ = closeHandle(hInWrite)
		_ = closeHandle(hOutRead)
		_ = closePseudoConsole(hPC)
		return fmt.Errorf("CreateProcessW failed: %v", e)
	}
	if pi.Thread != 0 {
		_ = closeHandle(pi.Thread)
	}

	b.inPipe = os.NewFile(hInWrite, "conpty-in")
	b.rPipe = os.NewFile(hOutRead, "conpty-out")
	b.hProcess = pi.Process
	b.pid = int(pi.ProcessId)
	b.hPC = hPC
	b.isConPTY = true
	b.mechanism = "ConPTY (CreatePseudoConsole)"

	// exec.Cmd must not try to manage the child itself on Windows.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Process = &os.Process{Pid: b.pid}
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
	if !b.isConPTY || b.hPC == 0 || rows <= 0 || cols <= 0 {
		return nil
	}
	sizeVal := *(*uint32)(unsafe.Pointer(&coord{X: int16(cols), Y: int16(rows)}))
	_, _, _ = procResizePseudoConsole.Call(b.hPC, uintptr(sizeVal))
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
	if b.hPC != 0 {
		_ = closePseudoConsole(b.hPC)
		b.hPC = 0
	}
	if b.hProcess != 0 {
		_ = closeHandle(b.hProcess)
		b.hProcess = 0
	}
	return nil
}

func (b *windowsBackend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pid
}

func (b *windowsBackend) Wait() error {
	b.mu.Lock()
	h := b.hProcess
	cmd := b.cmd
	isConPTY := b.isConPTY
	b.mu.Unlock()
	if !isConPTY {
		if cmd != nil {
			return cmd.Wait()
		}
		return nil
	}
	if h == 0 {
		return nil
	}
	_, _, _ = procWaitForSingleObject.Call(h, ^uintptr(0)) // INFINITE
	b.mu.Lock()
	if b.rPipe != nil {
		_ = b.rPipe.Close()
		b.rPipe = nil
	}
	b.mu.Unlock()
	var code uint32
	_, _, _ = procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&code)))
	if code == stillActive {
		return errors.New("process still active")
	}
	return nil
}

func (b *windowsBackend) Signal(sig os.Signal) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.isConPTY {
		// Interrupt (Ctrl+C) is delivered to the pseudo console via the input pipe.
		if sig == os.Interrupt && b.inPipe != nil {
			_, _ = b.inPipe.Write([]byte{0x03})
			return nil
		}
		return nil
	}
	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Signal(sig)
	}
	return nil
}

func (b *windowsBackend) Kill() error {
	b.mu.Lock()
	h := b.hProcess
	cmd := b.cmd
	isConPTY := b.isConPTY
	b.mu.Unlock()
	if isConPTY {
		if h == 0 {
			return nil
		}
		r, _, e := procTerminateProcess.Call(h, 1)
		if r == 0 {
			return e
		}
		return nil
	}
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
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

func (b *windowsBackend) Mechanism() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.mechanism == "" {
		return "unknown"
	}
	return b.mechanism
}

func closePseudoConsole(h uintptr) error {
	r, _, e := procClosePseudoConsole.Call(h)
	if r == 0 {
		return e
	}
	return nil
}

func closeHandle(h uintptr) error {
	if h == 0 {
		return nil
	}
	r, _, e := procCloseHandle.Call(h)
	if r == 0 {
		return e
	}
	return nil
}

// buildCommandLine joins argv with Windows quoting rules.
func buildCommandLine(args []string) string {
	var sb strings.Builder
	for i, a := range args {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(escapeArg(a))
	}
	return sb.String()
}

// escapeArg quotes an argument following Windows CreateProcess conventions.
func escapeArg(s string) string {
	if len(s) == 0 {
		return `""`
	}
	hasSpace := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\', ' ', '\t':
			hasSpace = true
		}
	}
	if !hasSpace {
		return s
	}
	var sb strings.Builder
	sb.WriteByte('"')
	i := 0
	for i < len(s) {
		backslashes := 0
		for i < len(s) && s[i] == '\\' {
			backslashes++
			i++
		}
		if i == len(s) {
			for j := 0; j < backslashes*2; j++ {
				sb.WriteByte('\\')
			}
			break
		}
		if s[i] == '"' {
			for j := 0; j < backslashes*2+1; j++ {
				sb.WriteByte('\\')
			}
			sb.WriteByte('"')
			i++
		} else {
			for j := 0; j < backslashes; j++ {
				sb.WriteByte('\\')
			}
			sb.WriteByte(s[i])
			i++
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// utf16EnvBlock builds a double-null-terminated UTF-16 environment block.
func utf16EnvBlock(env []string) []uint16 {
	var block []uint16
	for _, e := range env {
		block = append(block, syscall.StringToUTF16(e)...)
		block = append(block, 0)
	}
	block = append(block, 0)
	if len(block) == 0 {
		block = append(block, 0)
	}
	return block
}
