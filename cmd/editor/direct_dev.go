//go:build dev || bindings

package main

// Wails runs the editor itself to generate bindings and in wails dev, where
// there is no launcher to come from.
const openDirectly = true
