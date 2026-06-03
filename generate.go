package main

// Embed the app icon and version metadata into the Windows PE binary.
// Install goversioninfo once with:
//   go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
//
//go:generate goversioninfo -64=true -o resource.syso
