// What the Features page shows. Images and guides are the tutorial's own
// files in docs/usage; the site only links to them.

export interface Shot {
  src: string;
  label: string;
  alt: string;
}

export interface Feature {
  id: string;
  title: string;
  body: string[];
  guide: string; // a page in docs/usage
  facts?: string[];
  shots?: Shot[];
  code?: string; // highlighted HTML, shown when there is no screenshot
}

export interface Group {
  title: string;
  features: Feature[];
}

const img = (name: string) => `usage/img/${name}`;

const lua = `<span class="c">-- door.lua</span>
<span class="k">fields</span> = {
  hero  = <span class="n">object</span>(),
  reach = <span class="n">range</span>(3, 1, 10),
  lift  = 4,
}

<span class="k">function</span> <span class="n">key</span>(self, k)
  <span class="k">if</span> k == <span class="s">"e"</span> <span class="k">and</span> self.hero <span class="k">and</span> self:distance(self.hero) &lt; self.reach <span class="k">then</span>
    self.open = <span class="k">not</span> self.open
    self:tween(<span class="s">"pos.z"</span>, self.open <span class="k">and</span> self.lift <span class="k">or</span> 0, 0.6)
    <span class="k">return</span> true
  <span class="k">end</span>
<span class="k">end</span>`;

export const goExample = `e := ascdii.<span class="n">New</span>(ascdii.Options{})
s := e.<span class="n">Scene</span>()
s.Add(<span class="s">"camera"</span>, ascdii.<span class="n">CameraComponent</span>(34))
s.Add(<span class="s">"ground"</span>, ascdii.<span class="n">Ground</span>(40, 40, ascdii.Hex(<span class="s">"#46623e"</span>)))
s.Add(<span class="s">"crate"</span>, ascdii.<span class="n">Box</span>(ascdii.V3(2, 2, 2), ascdii.Hex(<span class="s">"#b08050"</span>))).Blocking()
s.Add(<span class="s">"hero"</span>, ascdii.<span class="n">Blob</span>(.8, ascdii.Hex(<span class="s">"#e8b04b"</span>))).At(0, 5, 0).With(ascdii.Player())
s.Animate(<span class="s">"crate"</span>, ascdii.<span class="n">Tween</span>(<span class="s">"pos.z"</span>, 0.0, 1.5, 1.2, ascdii.InOut).Bouncing())
err := e.<span class="n">Run</span>(ctx, os.Stdin, os.Stdout)`;

export const groups: Group[] = [
  {
    title: "Build worlds",
    features: [
      {
        id: "editor",
        title: "The editor",
        guide: "editor.html",
        body: [
          "A desktop editor with a Hierarchy, a live 2.5D Scene view with move, rotate and scale tools, an Inspector with component cards, a Project window, the Console and builds.",
          "Press Space to play the scene right inside the editor, and Esc to stop. Ctrl+Z and Ctrl+Y undo and redo everything.",
        ],
        shots: [
          { src: img("editor-main.png"), label: "Scene", alt: "The editor with the Dragonfall scene open: Hierarchy, Scene view, Inspector and Project window" },
          { src: img("editor-play.png"), label: "Play mode", alt: "The editor playing the scene in the Game view" },
          { src: img("editor-start.png"), label: "Start screen", alt: "The start screen with templates on the left and recent projects on the right" },
        ],
      },
      {
        id: "objects",
        title: "GameObjects and prefabs",
        guide: "gameobjects.html",
        body: [
          "GameObjects carry any number of components in parent and child hierarchies. Drag to reparent, right click to create or duplicate, and save any tree as a prefab with overrides.",
          "Reference slots link objects and assets to each other, so a script or component can point at the hero or a sound by dragging it in.",
        ],
        shots: [
          { src: img("editor-components.png"), label: "Components", alt: "The Inspector showing component cards on a GameObject" },
          { src: img("editor-gameobject-menu.png"), label: "Create menu", alt: "The GameObject menu with shapes, lights, cameras and effects" },
          { src: img("editor-references.png"), label: "References", alt: "Reference slots in the Inspector linking objects and assets" },
        ],
      },
      {
        id: "assets",
        title: "Textures, materials and models",
        guide: "textures.html",
        body: [
          "Import textures, materials, OBJ and FBX models, and sprites drawn as glyph art in palette JSON or REXPaint. Folders are your own; only the project manifest has a fixed place.",
          "Every asset keeps a UUIDv7 in its .meta file, so references stay valid when assets are renamed or moved.",
        ],
        facts: [".obj", ".fbx", ".png", ".xp", ".mtl"],
        shots: [
          { src: img("editor-material.png"), label: "Material", alt: "A material in the Inspector with its colour and texture settings" },
          { src: img("editor-texture.png"), label: "Texture", alt: "Texture import settings in the Inspector" },
          { src: img("editor-model.png"), label: "Model", alt: "An imported model in the Scene view" },
          { src: img("editor-project.png"), label: "Project", alt: "The Project window with folders and assets" },
        ],
      },
      {
        id: "physics",
        title: "Physics",
        guide: "physics.html",
        body: [
          "Box, circle and mesh colliders, triggers and rigidbodies, edited directly in the Scene view.",
          "Collision reads the same bounds that drawing uses, so what you see is what you bump into.",
        ],
        shots: [{ src: img("editor-colliders.png"), label: "Colliders", alt: "Colliders drawn over objects in the Scene view" }],
      },
    ],
  },
  {
    title: "Bring them to life",
    features: [
      {
        id: "scripting",
        title: "Lua scripting",
        guide: "scripting.html",
        body: [
          "Lua scripts are components. They declare fields the Inspector edits and reach every object, component, the camera, the environment and the render pipeline.",
          "Scripts reload when you save them, so you can change a game while it runs.",
        ],
        code: lua,
      },
      {
        id: "animation",
        title: "Animation",
        guide: "animation.html",
        body: [
          "Clips with keys and easing on a timeline, and animator controllers with states, parameters and transitions.",
          "Anything with a value can be animated, from a position to a light's colour.",
        ],
        shots: [{ src: img("editor-animation.png"), label: "Timeline", alt: "The Animation timeline with keys on several properties" }],
      },
      {
        id: "particles",
        title: "Particles",
        guide: "particles.html",
        body: [
          "A particle system whose particles move, fall, change colour and size and bounce off the ground, with a Particle Editor that shows every change while the effect runs.",
          "Start from one of nine presets: embers, fire, smoke, sparks, rain, snow, magic, fountain or explosion.",
        ],
        shots: [{ src: img("editor-particles.png"), label: "Particle Editor", alt: "The Particle Editor with a fire effect and its settings" }],
      },
      {
        id: "rendering",
        title: "Rendering",
        guide: "rendering.html",
        body: [
          "A deferred renderer with shadows, point, spot and directional lights, fog, water and colour grading, all resolved per Braille dot. A day and night cycle lights the world by the hour.",
          "Every terminal cell holds 2 by 4 lit, depth tested dots reduced to one foreground and one background colour. Add your own render passes from Go or Lua.",
        ],
        shots: [
          { src: img("game-day.png"), label: "Day", alt: "Dragonfall in the afternoon: a castle gate, torches, a road, a river and a bridge" },
          { src: img("game-night.png"), label: "Night", alt: "Dragonfall at night, lit by torches, with a dragon breathing fire" },
          { src: img("game-window.png"), label: "Window", alt: "Dragonfall in a desktop window, where every dot keeps its own colour" },
          { src: img("game-blocks.png"), label: "Blocks", alt: "Dragonfall drawn with half blocks instead of Braille" },
        ],
      },
      {
        id: "audio",
        title: "Audio",
        guide: "audio.html",
        body: [
          "WAV and MP3 sounds played by audio sources on GameObjects or straight from scripts.",
        ],
        facts: [".wav", ".mp3"],
        code: `<span class="c">-- play a sound from a script</span>
<span class="k">function</span> <span class="n">key</span>(self, k)
  <span class="k">if</span> k == <span class="s">"space"</span> <span class="k">then</span>
    <span class="n">play_sound</span>(<span class="s">"sounds/jump.wav"</span>, 0.8)
  <span class="k">end</span>
<span class="k">end</span>`,
      },
    ],
  },
  {
    title: "Ship it",
    features: [
      {
        id: "terminal",
        title: "Runs in the terminal",
        guide: "install.html",
        body: [
          "Games run in any terminal with 24 bit colour and a font with Braille characters: Windows Terminal, WezTerm, Kitty, Alacritty, GNOME Terminal, iTerm2 and the VS Code terminal.",
          "If a font draws Braille poorly, block mode uses half blocks instead. A game can also open in its own desktop window.",
        ],
        shots: [{ src: img("terminal.png"), label: "Terminal", alt: "Dragonfall running in a terminal emulator" }],
      },
      {
        id: "building",
        title: "Builds",
        guide: "building.html",
        body: [
          "A build is a folder per platform with the player, the whole project compiled into one encrypted data file, and checksums. The build is the release.",
          "Builds target Windows and Linux, as a folder or as a single executable, and a game can open in the terminal or in its own window.",
        ],
        shots: [
          { src: img("editor-build.png"), label: "Build", alt: "The build window with platforms and output settings" },
          { src: img("project-settings.png"), label: "Project settings", alt: "Project settings with the display, start scene and modules" },
        ],
      },
      {
        id: "packages",
        title: "Packages and mods",
        guide: "packages.html",
        body: [
          "Share assets as one .ascdiipack file and install it into any project from the Package Manager. Installed packages are read only and removed as a whole.",
          "The Dragonfall sample game is a package itself, Go components included.",
        ],
        shots: [
          { src: img("editor-packages.png"), label: "Package Manager", alt: "The Package Manager listing installed and built in packages" },
          { src: img("editor-package-tree.png"), label: "Contents", alt: "An installed package's files in the Project window" },
        ],
      },
    ],
  },
];

export const allFeatures = groups.flatMap((g) => g.features);

export const repo = "https://github.com/Cn8001/ASCDII-Engine";
export const version = __ENGINE_VERSION__;
