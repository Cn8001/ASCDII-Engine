package main

// note documents one component kind for the reference page; types, defaults
// and ranges come from the registry itself.
type note struct {
	summary string
	fields  map[string]string
}

func notes() map[string]note {
	return map[string]note{
		"box": {"A box standing on its position, with separate colors for the top and the sides, or a material with a texture per face.", map[string]string{
			"size": "Width, depth and height in world units.", "top": "Color of the top face.", "side": "Color of the side faces.",
			"emissive": "Glow independent of light, 0 to 1.", "material": "A material asset; replaces both colors when set.",
		}},
		"blob": {"A sphere shaded per dot, for canopies, bushes and rocks.", map[string]string{
			"radius": "Radius in world units.", "color": "Surface color.", "glow": "Glow independent of light, 0 to 1.",
		}},
		"cylinder": {"An upright faceted cylinder standing on its position.", map[string]string{
			"radius": "Radius in world units.", "height": "Height in world units.", "color": "Surface color.", "segments": "Number of side faces.",
		}},
		"cone": {"An upright cone standing on its position; its tip can sway in the wind.", map[string]string{
			"radius": "Base radius in world units.", "height": "Height in world units.", "color": "Surface color.", "sway": "How far the tip moves in the wind, in world units.",
		}},
		"plane": {"A flat rectangle on the ground.", map[string]string{
			"width": "Size along X.", "depth": "Size along Y.", "color": "Surface color.", "material": "A material asset; replaces the color when set.",
		}},
		"quad": {"An upright rectangle facing the camera, for signs and billboards.", map[string]string{
			"width": "Width in world units.", "height": "Height in world units.", "color": "Surface color.", "material": "A material asset; replaces the color when set.", "twoSided": "Draws the back face as well.",
		}},
		"mesh": {"An imported OBJ model, see Models.", map[string]string{
			"model": "The model asset.", "material": "A material asset; replaces the model's own materials when set.", "tint": "Color multiplied into the model's colors.",
			"emissive": "Glow independent of light, 0 to 1.", "twoSided": "Draws back faces, for models with open surfaces.", "shadow": "Casts a shadow on the ground.",
		}},
		"ground": {"A flat terrain patch with a grain pattern.", map[string]string{
			"width": "Size along X.", "depth": "Size along Y.", "color": "Base color.", "grain": "Strength of the color variation, 0 to 1.", "seed": "Varies the grain pattern.",
		}},
		"water": {"A water surface with animated glints and reflections of the sky.", map[string]string{
			"width": "Size along X.", "depth": "Size along Y.", "color": "Base color.",
		}},
		"camera": {"Makes its object the game camera. The object's position is the point the camera looks at and its rotation around Z turns the view; as a child of another object the camera moves with it. The first enabled camera on a visible object is used; a scene without one shows No camera.", map[string]string{
			"span": "World units across the width of the view; smaller values zoom in.", "pitch": "Angle above the ground in degrees; 90 looks straight down.",
			"anchorX": "Horizontal screen position of the target, 0 left to 1 right.", "anchorY": "Vertical screen position of the target, 0 top to 1 bottom.",
		}},
		"light": {"A light source. Point lights light everything within their radius; spot lights a cone aimed by the object's rotation; directional lights replace the environment's sun.", map[string]string{
			"type": "point, spot or directional.", "radius": "Reach in world units, for point and spot lights.", "color": "Light color.",
			"intensity": "Brightness multiplier.", "flicker": "Random variation of the brightness over time, 0 to 1.", "angle": "Width of a spot light's cone in degrees.",
		}},
		"emitter": {"A particle system, see Particles.", map[string]string{
			"active": "Emits new particles; when off, live particles finish their life.", "glow": "Radius of a light that follows the effect, 0 for none.",
			"duration": "Length of one cycle in seconds.", "loop": "Repeats the cycle; without it the system stops after one.", "prewarm": "Starts as if it had been running for one particle lifetime.",
			"rate": "Particles per second.", "burst": "Particles emitted at once at the start of each cycle.", "max": "Most particles alive at once.",
			"shape": "Where particles are born: a point, inside a sphere, inside a box or on a flat ring.", "radius": "Radius of the sphere and ring shapes.", "box": "Size of the box shape.",
			"life": "Lifetime in seconds; each particle varies by 30 percent.", "speed": "Initial speed.", "dir": "Main direction of motion.", "spread": "Widens the direction from a beam (0) to all directions (1).",
			"size": "Size at birth.", "sizeEnd": "Size at the end of the life.", "start": "Color at birth.", "end": "Color at the end of the life.", "emissive": "Glow independent of light, 0 to 1.",
			"gravity": "Downward acceleration; negative values make particles rise.", "drag": "Slows particles over time.", "wind": "Constant acceleration as a vector.",
			"turbulence": "Strength of the swirling motion.", "collide": "Stops particles at the ground.", "bounce": "Share of the speed kept when bouncing off the ground; 0 removes the particle.",
			"space": "world keeps particles where they were born when the object moves; local moves them with the object.", "render": "dot draws a single dot; streak draws a trail along the motion.", "seed": "Varies the random pattern.",
		}},
		"label": {"Text drawn as whole terminal cells, in the world or anchored to the screen.", map[string]string{
			"text": "The text; \\n starts a new line.", "color": "Text color.", "back": "Background color, used with solid.",
			"anchor": "world places the text at the object; the other values pin it to a corner, an edge or the center of the screen.", "solid": "Fills the background behind the text.",
		}},
		"sprite": {"A sprite asset drawn as whole terminal cells at the object's position, hidden behind nearer geometry.", map[string]string{
			"asset": "The sprite asset, a .sprite.json or .xp file.",
		}},
		"audio_source": {"Plays a sound, see Audio.", map[string]string{
			"clip": "The sound asset, WAV or MP3.", "volume": "Volume, 0 to 1.", "loop": "Repeats the sound.", "playOnStart": "Starts playing when the game starts.",
		}},
		"box_collider": {"A box shaped collision area turned with the object around Z, see Physics.", map[string]string{
			"size": "Width, depth and height of the box.", "offset": "Position of the box's base relative to the object.", "trigger": "Reports overlaps to scripts without blocking.",
		}},
		"circle_collider": {"A circular collision area, see Physics.", map[string]string{
			"radius": "Radius in world units.", "height": "Height of the area.", "offset": "Position of the base relative to the object.", "trigger": "Reports overlaps to scripts without blocking.",
		}},
		"mesh_collider": {"A collision area shaped like the outline of the object's mesh seen from above, see Physics.", map[string]string{
			"trigger": "Reports overlaps to scripts without blocking.",
		}},
		"rigidbody": {"Moves the object under gravity and collisions, see Physics.", map[string]string{
			"mass": "Mass; heavier bodies push lighter ones.", "gravity": "Falls under gravity.", "drag": "Slows the body over time.",
			"bounce": "Share of the speed kept when hitting something, 0 to 1.", "kinematic": "Moved by scripts only; other bodies still collide with it.", "velocity": "Current velocity in world units per second.",
		}},
		"paint": {"Strokes made with the Braille brush in the editor, stored in the object's local space.", map[string]string{
			"strokes": "The stroke data; edited with the brush.",
		}},
		"animator": {"Plays the object's animation clips as the states of a controller, see Animation.", map[string]string{
			"controller": "The animator controller asset.", "speed": "Playback speed multiplier.", "state": "The active state; shown while the game runs.",
		}},
		"player": {"Moves its object with the keyboard: walking, jumping, sliding along walls, and optionally the camera following it.", map[string]string{
			"speed": "Walking speed in world units per second.", "jumpSpeed": "Initial upward speed of a jump.", "gravity": "Downward acceleration while in the air.",
			"radius": "Size of the body for collisions.", "follow": "Moves the game camera along. A camera object that is a child of the player moves with it instead.",
			"followRate": "How quickly the camera catches up.", "up": "Key for moving up.", "down": "Key for moving down.", "left": "Key for moving left.", "right": "Key for moving right.", "jump": "Key for jumping.",
		}},
		"toggle": {"Switches a true or false property when a key is pressed.", map[string]string{
			"key": "The key.", "target": "The object whose property changes; empty for this object.", "prop": "The property name, such as visible or light.enabled.",
		}},
		"daynight": {"Moves the environment's time of day to the other phase when a key is pressed.", map[string]string{
			"key": "The key.", "day": "Hour of the day phase.", "night": "Hour of the night phase.", "speed": "Hours per second during the change.",
		}},
		"quit": {"Ends the game when a key is pressed.", map[string]string{
			"key": "The key.",
		}},
		"script": {"Runs a Lua script, see Lua scripting. A script's own fields appear below file.", map[string]string{
			"file": "The Lua file.",
		}},
		"df.terrain": {"The Dragonfall valley: hills, cliffs and a river, generated from a seed.", map[string]string{
			"seed": "Varies the landscape.", "water": "Height of the water level.",
		}},
		"df.forest": {"Trees placed over the Dragonfall terrain.", map[string]string{
			"seed": "Varies the placement.", "density": "Share of possible spots that get a tree.", "sway": "How far the trees move in the wind.",
		}},
		"df.grass": {"Grass blades over the Dragonfall terrain.", map[string]string{
			"seed": "Varies the placement.", "density": "Share of possible spots that get grass.", "detail": "Draws more blades near the camera.",
		}},
		"df.castle": {"The Dragonfall keep with towers, walls and a gate.", map[string]string{
			"stone": "Wall color.", "roof": "Roof color.", "banner": "Banner color.", "torches": "Lights the torches.",
		}},
		"df.bridge": {"A wooden bridge.", map[string]string{
			"length": "Length in world units.", "width": "Width in world units.", "wood": "Wood color.",
		}},
		"df.dragon": {"The Dragonfall dragon.", map[string]string{
			"awake": "Wakes the dragon: it flaps and looks around.", "fire": "Breathes fire while awake.",
		}},
		"df.hero": {"The Dragonfall knight.", map[string]string{
			"walking": "Plays the walk cycle; set by the player component.", "speed": "Walk cycle speed multiplier.",
		}},
		"df.hud": {"The Dragonfall heads-up display: title, area name, time and a map.", map[string]string{
			"title": "Title in the top left corner.", "showMap": "Shows the map.", "showHelp": "Shows the key help.", "seed": "Terrain seed the map is drawn from.",
		}},
	}
}
