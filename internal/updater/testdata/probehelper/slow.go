//go:build probehelper_slow

package main

import "time"

// init makes the slow variant hang so the probe timeout can be exercised.
func init() {
	time.Sleep(30 * time.Second)
}
