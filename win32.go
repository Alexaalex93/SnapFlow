package main

import (
	"syscall"
)

// Raw Win32 entry points shared by the overlay and the tray window.
var (
	user32DLL   = syscall.NewLazyDLL("user32.dll")
	gdi32DLL    = syscall.NewLazyDLL("gdi32.dll")
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW     = user32DLL.NewProc("CreateWindowExW")
	procDestroyWindow       = user32DLL.NewProc("DestroyWindow")
	procRegisterClassExW    = user32DLL.NewProc("RegisterClassExW")
	procDefWindowProcW      = user32DLL.NewProc("DefWindowProcW")
	procUpdateLayeredWindow = user32DLL.NewProc("UpdateLayeredWindow")
	procShowWindowAsync     = user32DLL.NewProc("ShowWindowAsync")
	procSetWindowPos        = user32DLL.NewProc("SetWindowPos")
	procPeekMessageW        = user32DLL.NewProc("PeekMessageW")
	procPostThreadMessageW  = user32DLL.NewProc("PostThreadMessageW")
	procDeleteObject        = gdi32DLL.NewProc("DeleteObject")
	procGetModuleHandleW    = kernel32DLL.NewProc("GetModuleHandleW")
)

// wndClassExW mirrors the Win32 WNDCLASSEXW struct.
type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}
