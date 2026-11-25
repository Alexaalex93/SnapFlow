package main

import (
	"errors"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// ── Animated window movement ──────────────────────────────────────────────────
//
// The REAL window is moved and resized a little every frame (ease-out,
// paced by the compositor), so its content re-lays out continuously and the
// window is already at its true final size when the motion ends — no
// stretched snapshot and no jump at the end. (A first version animated a
// DWM thumbnail of the window and swapped in the real window at the end;
// the stretched content followed by that swap looked bad.)
//
// Safeguards:
//   - runs on its own thread, so a slow app never blocks SnapFlow's main
//     thread (SetWindowPos on another process's window waits for it);
//   - adaptive: if the app takes too long to process resizes, the motion is
//     cut short and the window lands immediately instead of stuttering;
//   - hung windows, maximised/minimised windows, Windows' "Animation
//     effects: off", and the AnimateWindows option all skip animation;
//   - a new request for the same window retargets mid-flight; grabbing the
//     window with the mouse cancels the flight.
//
// Bookkeeping (history, corners) is done synchronously by the caller with
// the expected frame; the animator corrects history if an app's minimum
// size changes the final result.

const (
	windowAnimDur = 240 * time.Millisecond

	// If the app needs longer than this per resize (twice), stop animating
	// and land the window: a stuttering animation is worse than none.
	slowFrameLimit = 45 * time.Millisecond

	spiGetClientAreaAnimation = 0x1042
)

var procIsHungAppWindow = user32DLL.NewProc("IsHungAppWindow")

// windowAnim is the process-wide animator (nil if it couldn't start).
var windowAnim *WindowAnimator

type animReq struct {
	hwnd     w32.HWND
	from     w32.RECT // visible frame now
	target   w32.RECT // frame to reach
	expected w32.RECT // what placement will actually produce (flush edges extended)
	cancel   bool
}

type WindowAnimator struct {
	threadID uint32
	reqCh    chan animReq
	done     chan struct{}

	mu      sync.Mutex
	targets map[w32.HWND]w32.RECT // logical frame of windows in flight
}

func NewWindowAnimator() (*WindowAnimator, error) {
	a := &WindowAnimator{
		reqCh:   make(chan animReq, 32),
		done:    make(chan struct{}),
		targets: map[w32.HWND]w32.RECT{},
	}
	ready := make(chan error, 1)
	go a.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return a, nil
}

// LogicalFrame returns where hwnd is heading: its animation target while in
// flight, otherwise its current frame.
func (a *WindowAnimator) LogicalFrame(hwnd w32.HWND) (w32.RECT, bool) {
	if a != nil {
		a.mu.Lock()
		r, ok := a.targets[hwnd]
		a.mu.Unlock()
		if ok {
			return r, true
		}
	}
	return frameRect(hwnd)
}

func (a *WindowAnimator) inFlight(hwnd w32.HWND) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.targets[hwnd]
	return ok
}

// Animate starts (or retargets) an animated move of hwnd's frame to target.
// Returns false when the window can't/shouldn't be animated — the caller
// then places it directly.
func (a *WindowAnimator) Animate(hwnd w32.HWND, from, target, work w32.RECT) bool {
	if a == nil {
		return false
	}
	expected := extendFlushEdges(target, work)
	if a.inFlight(hwnd) {
		a.mu.Lock()
		a.targets[hwnd] = target
		a.mu.Unlock()
		a.send(animReq{hwnd: hwnd, target: target, expected: expected})
		return true
	}
	if rectNear(from, expected, 2) || w32ex.IsIconic(hwnd) || w32ex.IsZoomed(hwnd) {
		return false
	}
	if !systemAnimationsEnabled() || isHung(hwnd) {
		return false
	}
	a.mu.Lock()
	a.targets[hwnd] = target
	a.mu.Unlock()
	a.send(animReq{hwnd: hwnd, from: from, target: target, expected: expected})
	return true
}

// Cancel stops a flight without landing it (the user grabbed the window).
func (a *WindowAnimator) Cancel(hwnd w32.HWND) {
	if a == nil || !a.inFlight(hwnd) {
		return
	}
	a.mu.Lock()
	delete(a.targets, hwnd)
	a.mu.Unlock()
	a.send(animReq{hwnd: hwnd, cancel: true})
}

func (a *WindowAnimator) send(r animReq) {
	a.reqCh <- r // buffered; the animator drains it every frame
	procPostThreadMessageW.Call(uintptr(a.threadID), wmOverlayWake, 0, 0)
}

// Close lands any window still in flight and stops the animator.
func (a *WindowAnimator) Close() {
	if a == nil {
		return
	}
	procPostThreadMessageW.Call(uintptr(a.threadID), wmQuit, 0, 0)
	select {
	case <-a.done:
	case <-time.After(time.Second):
	}
}

func systemAnimationsEnabled() bool {
	var on int32 = 1
	w32ex.SystemParametersInfo(spiGetClientAreaAnimation, 0, unsafe.Pointer(&on), 0)
	return on != 0
}

func isHung(hwnd w32.HWND) bool {
	r, _, _ := procIsHungAppWindow.Call(uintptr(hwnd))
	return r != 0
}

// ── Animator thread ───────────────────────────────────────────────────────────

type flight struct {
	animReq
	start     time.Time
	cur       w32.RECT // last frame applied
	slowCount int
}

func (a *WindowAnimator) run(ready chan<- error) {
	runtime.LockOSThread()
	defer close(a.done)
	a.threadID = w32ex.GetCurrentThreadID()
	if a.threadID == 0 {
		ready <- errors.New("no thread id")
		return
	}
	// Make sure this thread has a message queue before anyone posts to it.
	var m w32.MSG
	peekMessage(&m)
	ready <- nil

	var f *flight
	quit := false
	for !quit {
		if f == nil {
			r := w32.GetMessage(&m, 0, 0, 0)
			if r == 0 || r == -1 {
				return
			}
		} else {
			for peekMessage(&m) {
				if m.Message == wmQuit {
					quit = true
				}
			}
		}

	drain:
		for {
			select {
			case r := <-a.reqCh:
				switch {
				case r.cancel:
					if f != nil && f.hwnd == r.hwnd {
						f = nil
					}
				case f != nil && r.hwnd == f.hwnd:
					f.from, f.target, f.expected = f.cur, r.target, r.expected
					f.start = time.Now()
				default:
					if f != nil {
						a.finish(f)
						f = nil
					}
					if r.from.Width() > 0 {
						f = &flight{animReq: r, cur: r.from, start: time.Now()}
					} else {
						a.land(r.hwnd, r.target, r.expected) // retarget after landing
					}
				}
			default:
				break drain
			}
		}

		if f != nil {
			if quit || a.step(f) {
				a.finish(f)
				f = nil
			} else if w32.DwmFlush() != 0 {
				time.Sleep(8 * time.Millisecond)
			}
		}
	}
}

// step moves/resizes the real window one frame; true when the flight should
// end (reached the target, window gone, app too slow, or user grabbed it).
func (a *WindowAnimator) step(f *flight) bool {
	if !w32.IsWindow(f.hwnd) {
		return true
	}
	if dragMgr != nil && dragMgr.IsDragging(f.hwnd) {
		return true
	}
	t := progress(f.start, windowAnimDur, time.Now())
	if t >= 1 {
		return true
	}
	r := lerpRect(toFrect(f.from), toFrect(f.expected), easeOutCubic(t)).round()
	if r == f.cur {
		return false
	}
	in, ok := windowInsets(f.hwnd) // re-read: insets change if the DPI changes mid-way
	if !ok {
		return true
	}
	t0 := time.Now()
	if !setWindowFrame(f.hwnd, r, in, 0) {
		return true
	}
	f.cur = r
	if time.Since(t0) > slowFrameLimit {
		f.slowCount++
		if f.slowCount >= 2 {
			return true // heavy app: land now instead of stuttering
		}
	}
	return false
}

// finish lands the window exactly (unless the user took it over).
func (a *WindowAnimator) finish(f *flight) {
	if dragMgr != nil && dragMgr.IsDragging(f.hwnd) {
		a.forget(f.hwnd, f.target)
		return
	}
	a.land(f.hwnd, f.target, f.expected)
}

func (a *WindowAnimator) land(hwnd w32.HWND, target, expected w32.RECT) {
	if w32.IsWindow(hwnd) {
		got, err := resizeToRect(hwnd, target)
		if err == nil && got.Width() > 0 && !rectNear(got, expected, 0) {
			windowHistory.UpdateSnapped(hwnd, expected, got)
		}
	}
	a.forget(hwnd, target)
}

func (a *WindowAnimator) forget(hwnd w32.HWND, target w32.RECT) {
	a.mu.Lock()
	if t, ok := a.targets[hwnd]; ok && t == target {
		delete(a.targets, hwnd)
	}
	a.mu.Unlock()
}

// ── Entry points used by actions and drags ────────────────────────────────────

// placeWindow moves hwnd's visible frame from `from` to `target`, animated
// when enabled and possible. It returns the frame the window ends up in
// (for animations: the expected frame — the animator corrects history if
// an app's minimum size changes it). Main thread only (reads appCfg).
func placeWindow(hwnd w32.HWND, from, target w32.RECT) (w32.RECT, error) {
	if appCfg.AnimateWindows && windowAnim != nil {
		if work, ok := monitorWork(w32.MonitorFromRect(&target, w32.MONITOR_DEFAULTTONEAREST)); ok &&
			isZonableWindow(hwnd) && windowAnim.Animate(hwnd, from, target, work) {
			return extendFlushEdges(target, work), nil
		}
	}
	return resizeToRect(hwnd, target)
}

// currentFrame is frameRect, but returns the destination of a window that
// is still animating (so repeated shortcuts build on where it is going).
func currentFrame(hwnd w32.HWND) (w32.RECT, bool) {
	if windowAnim != nil {
		return windowAnim.LogicalFrame(hwnd)
	}
	return frameRect(hwnd)
}
