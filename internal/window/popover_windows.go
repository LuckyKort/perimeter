package window

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// The tray window on Windows: our own hidden popup with WebView2 inside (it
// ships with Windows 10/11). It is created at start and only shown on a click,
// so it opens instantly; it closes when it loses focus, like the system flyouts.

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	dwmapi               = windows.NewLazySystemDLL("dwmapi.dll")
	pRegisterClassEx     = user32.NewProc("RegisterClassExW")
	pCreateWindowEx      = user32.NewProc("CreateWindowExW")
	pDefWindowProc       = user32.NewProc("DefWindowProcW")
	pShowWindow          = user32.NewProc("ShowWindow")
	pSetWindowPos        = user32.NewProc("SetWindowPos")
	pGetMessage          = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessage     = user32.NewProc("DispatchMessageW")
	pPostThreadMessage   = user32.NewProc("PostThreadMessageW")
	pGetCursorPos        = user32.NewProc("GetCursorPos")
	pMonitorFromPoint    = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	pGetDpiForWindow     = user32.NewProc("GetDpiForWindow")
	pSetTimer            = user32.NewProc("SetTimer")
	pKillTimer           = user32.NewProc("KillTimer")
	pIsWindowVisible     = user32.NewProc("IsWindowVisible")
	pDwmSetWindowAttr    = dwmapi.NewProc("DwmSetWindowAttribute")
	pGetCurrentThreadID  = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentThreadId")
)

const (
	wsPopup        = 0x80000000
	wsExToolWindow = 0x00000080
	wsExTopmost    = 0x00000008
	csDropShadow   = 0x00020000
	swHide         = 0
	swShowNormal   = 5
	swpNoActivate  = 0x0010
	swpShowWindow  = 0x0040
	hwndTopmost    = ^uintptr(0) // -1
	wmSize         = 0x0005
	wmMove         = 0x0003
	wmActivate     = 0x0006
	wmClose        = 0x0010
	wmTimer        = 0x0113
	wmApp          = 0x8000
	waInactive     = 0
	popW           = 340
	gap            = 12
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}
type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}
type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type winPop struct {
	hwnd     uintptr
	web      *edge.Chromium
	thread   uintptr
	height   int // CSS pixels
	hiddenAt time.Time
	shownAt  time.Time
	fromTop  bool // the taskbar is at the top: the window hangs down from it

	mu    sync.Mutex
	queue []func()
}

var pop *winPop

func init() { runtime.LockOSThread() } // the window and its messages live on the main thread

// Popover runs the tray window until cmds closes (Kordon has exited).
func Popover(url string, cmds io.Reader) error {
	p := &winPop{height: 420}
	pop = p
	p.thread, _, _ = pGetCurrentThreadID.Call()
	if err := p.create(); err != nil {
		return err
	}
	if !p.web.Embed(p.hwnd) {
		return errors.New("WebView2 is not available")
	}
	if s, err := p.web.GetSettings(); err == nil {
		_ = s.PutAreDefaultContextMenusEnabled(false)
		_ = s.PutAreDevToolsEnabled(false)
		_ = s.PutIsZoomControlEnabled(false)
		_ = s.PutIsStatusBarEnabled(false)
	}
	if c2 := p.web.GetController().GetICoreWebView2Controller2(); c2 != nil {
		// No white flash before the page paints: the page's own background colour.
		bg := edge.COREWEBVIEW2_COLOR{A: 255, R: 0x14, G: 0x16, B: 0x1D}
		if lightTheme() {
			bg = edge.COREWEBVIEW2_COLOR{A: 255, R: 0xF7, G: 0xF8, B: 0xFB}
		}
		_ = c2.PutDefaultBackgroundColor(bg)
	}
	p.web.Resize()
	p.web.Navigate(url)

	go func() {
		sc := bufio.NewScanner(cmds)
		for sc.Scan() {
			switch c := sc.Text(); c {
			case "toggle", "show", "hide":
				p.dispatch(func() { p.command(c) })
			}
		}
		os.Exit(0)
	}()

	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return nil
		}
		if m.Message == wmApp && m.Hwnd == 0 {
			p.mu.Lock()
			q := p.queue
			p.queue = nil
			p.mu.Unlock()
			for _, f := range q {
				f()
			}
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (p *winPop) create() error {
	var inst windows.Handle
	_ = windows.GetModuleHandleEx(0, nil, &inst)
	cls, _ := windows.UTF16PtrFromString("KordonPopover")
	wc := wndClassEx{Style: csDropShadow, WndProc: windows.NewCallback(wndProc), Instance: inst, ClassName: cls}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	title, _ := windows.UTF16PtrFromString("Kordon")
	// Created off-screen and hidden: nothing flashes when Kordon starts.
	p.hwnd, _, _ = pCreateWindowEx.Call(wsExToolWindow|wsExTopmost, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)),
		wsPopup, uintptr(^uint32(31999)), uintptr(^uint32(31999)), popW, uintptr(p.height), 0, 0, uintptr(inst), 0)
	if p.hwnd == 0 {
		return errors.New("can't create the tray window")
	}
	// Windows 11: rounded corners and the dark frame when the system is dark.
	round := uint32(2) // DWMWCP_ROUND
	pDwmSetWindowAttr.Call(p.hwnd, 33, uintptr(unsafe.Pointer(&round)), 4)
	if !lightTheme() {
		dark := uint32(1)
		pDwmSetWindowAttr.Call(p.hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	}
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Kordon", "popover-webview")
	p.web = edge.NewChromium()
	p.web.DataPath = dir
	p.web.MessageCallback = p.message
	return nil
}

func (p *winPop) dispatch(f func()) {
	p.mu.Lock()
	p.queue = append(p.queue, f)
	p.mu.Unlock()
	pPostThreadMessage.Call(p.thread, wmApp, 0, 0)
}

func (p *winPop) visible() bool {
	r, _, _ := pIsWindowVisible.Call(p.hwnd)
	return r != 0
}

func (p *winPop) command(c string) {
	switch c {
	case "toggle":
		if p.visible() {
			p.hide()
			return
		}
		// A click on the tray icon first takes the focus away (which hides the
		// window) and only then arrives here: that click means "close".
		if time.Since(p.hiddenAt) < 350*time.Millisecond {
			return
		}
		p.show()
	case "show":
		p.show()
	case "hide":
		p.hide()
	}
}

func (p *winPop) scale() float64 {
	dpi, _, _ := pGetDpiForWindow.Call(p.hwnd)
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96
}

// place puts the window next to the tray, on the side where the taskbar is.
func (p *winPop) place() (x, y, w, h int32) {
	var c point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	mon, _, _ := pMonitorFromPoint.Call(uintptr(uint32(c.X))|uintptr(uint32(c.Y))<<32, 2) // POINT by value; MONITOR_DEFAULTTONEAREST
	mi := monitorInfo{}
	mi.Size = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	s := p.scale()
	w, h = int32(popW*s), int32(float64(p.height)*s)
	g := int32(gap * s)
	work := mi.Work
	clamp := func(v, lo, hi int32) int32 { return max(lo, min(v, hi)) }
	x = clamp(c.X-w/2, work.Left+g, work.Right-w-g)
	p.fromTop = false
	switch {
	case c.Y < work.Top: // taskbar at the top
		y, p.fromTop = work.Top+g, true
	case c.X < work.Left: // on the left
		x, y = work.Left+g, clamp(c.Y-h/2, work.Top+g, work.Bottom-h-g)
	case c.X > work.Right: // on the right
		x, y = work.Right-w-g, clamp(c.Y-h/2, work.Top+g, work.Bottom-h-g)
	default: // at the bottom, as usual
		y = work.Bottom - h - g
	}
	return
}

func (p *winPop) show() {
	x, y, w, h := p.place()
	pSetWindowPos.Call(p.hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpShowWindow)
	pShowWindow.Call(p.hwnd, swShowNormal)
	pSetForegroundWindow.Call(p.hwnd)
	p.web.Focus()
	p.shownAt = time.Now()
	// Backup for the focus check: Windows may refuse to bring us forward.
	pSetTimer.Call(p.hwnd, 1, 250, 0)
	p.web.Eval("window.kordonShow && kordonShow()")
}

func (p *winPop) hide() {
	if !p.visible() {
		return
	}
	pKillTimer.Call(p.hwnd, 1)
	pShowWindow.Call(p.hwnd, swHide)
	p.hiddenAt = time.Now()
	p.web.Eval("window.kordonHide && kordonHide()")
}

// resize keeps the edge next to the taskbar in place.
func (p *winPop) resize(cssH int) {
	if cssH < 120 || cssH > 900 || cssH == p.height {
		return
	}
	p.height = cssH
	if !p.visible() {
		return
	}
	var r rect
	getWindowRect(p.hwnd, &r)
	s := p.scale()
	h := int32(float64(cssH) * s)
	y := r.Top
	if !p.fromTop {
		y = r.Bottom - h
	}
	pSetWindowPos.Call(p.hwnd, hwndTopmost, uintptr(r.Left), uintptr(y), uintptr(r.Right-r.Left), uintptr(h), swpNoActivate)
}

func (p *winPop) message(s string) {
	var m struct {
		Op     string  `json:"op"`
		Height float64 `json:"height"`
	}
	if json.Unmarshal([]byte(s), &m) != nil {
		return
	}
	switch m.Op {
	case "hide":
		p.hide()
	case "resize":
		p.resize(int(m.Height + 0.5))
	}
}

var pGetWindowRect = user32.NewProc("GetWindowRect")

func getWindowRect(h uintptr, r *rect) { pGetWindowRect.Call(h, uintptr(unsafe.Pointer(r))) }

func wndProc(hwnd, m, wp, lp uintptr) uintptr {
	p := pop
	if p != nil && hwnd == p.hwnd {
		switch m {
		case wmSize:
			if p.web != nil {
				p.web.Resize()
			}
		case wmMove:
			if p.web != nil {
				_ = p.web.NotifyParentWindowPositionChanged()
			}
		case wmActivate:
			if wp&0xFFFF == waInactive {
				p.hide()
			}
		case wmTimer:
			fg, _, _ := pGetForegroundWindow.Call()
			if fg != p.hwnd && time.Since(p.shownAt) > 600*time.Millisecond {
				p.hide()
			}
			return 0
		case wmClose:
			p.hide() // Alt+F4 closes the window, not Kordon
			return 0
		}
	}
	r, _, _ := pDefWindowProc.Call(hwnd, m, wp, lp)
	return r
}

// lightTheme reads the "app mode" switch in Windows personalisation settings.
func lightTheme() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err != nil || v != 0
}
