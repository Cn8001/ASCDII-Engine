// Command hello is the smallest ASCDII Engine program.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/pkg/ascdii"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	e := ascdii.New(ascdii.Options{})
	s := e.Scene()
	s.Camera.Span = 34
	s.Add("ground", ascdii.Ground(40, 40, ascdii.Hex("#46623e")))
	s.Add("crate", ascdii.Box(ascdii.V3(2, 2, 2), ascdii.Hex("#b08050"))).At(-4, 0, 0).Blocking()
	s.Add("pillar", ascdii.Cylinder(1, 5, ascdii.Hex("#9a9890"))).At(5, -3, 0).Blocking()
	s.Add("hero", ascdii.Blob(.8, ascdii.Hex("#e8b04b"))).At(0, 5, 0).With(ascdii.Player(), ascdii.Quit("q"))
	s.Animate("crate", ascdii.Tween("pos.z", 0.0, 1.5, 1.2, ascdii.InOut).Bouncing())
	if len(args) > 0 && args[0] == "--preview" {
		e.Resize(100, 32)
		return e.Preview(ctx, os.Stdout, 1, false)
	}
	return e.Run(ctx, os.Stdin, os.Stdout)
}
