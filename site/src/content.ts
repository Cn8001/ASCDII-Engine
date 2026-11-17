// The site only introduces the engine; everything it explains in detail
// lives in docs/usage, the one place to keep up to date.

export const goExample = `e := ascdii.<span class="n">New</span>(ascdii.Options{})
s := e.<span class="n">Scene</span>()
s.Add(<span class="s">"camera"</span>, ascdii.<span class="n">CameraComponent</span>(34))
s.Add(<span class="s">"ground"</span>, ascdii.<span class="n">Ground</span>(40, 40, ascdii.Hex(<span class="s">"#46623e"</span>)))
s.Add(<span class="s">"crate"</span>, ascdii.<span class="n">Box</span>(ascdii.V3(2, 2, 2), ascdii.Hex(<span class="s">"#b08050"</span>))).Blocking()
s.Add(<span class="s">"hero"</span>, ascdii.<span class="n">Blob</span>(.8, ascdii.Hex(<span class="s">"#e8b04b"</span>))).At(0, 5, 0).With(ascdii.Player())
s.Animate(<span class="s">"crate"</span>, ascdii.<span class="n">Tween</span>(<span class="s">"pos.z"</span>, 0.0, 1.5, 1.2, ascdii.InOut).Bouncing())
err := e.<span class="n">Run</span>(ctx, os.Stdin, os.Stdout)`;

export const repo = "https://github.com/Cn8001/ASCDII-Engine";
export const version = __ENGINE_VERSION__;
