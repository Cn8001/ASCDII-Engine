//go:build !windows

package winbrand

import "context"

// Supported is false here so the resource writer stays out of Linux
// programs; Windows games built on Linux keep the player's icon and name.
const Supported = false

func Brand(ctx context.Context, player, dir string, b Branding) (string, error) {
	return player, ctx.Err()
}
