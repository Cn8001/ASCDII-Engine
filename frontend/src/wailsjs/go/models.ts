export namespace anim {
	
	export class Key {
	    t: number;
	    v: any;
	    ease?: string;
	
	    static createFrom(source: any = {}) {
	        return new Key(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.t = source["t"];
	        this.v = source["v"];
	        this.ease = source["ease"];
	    }
	}
	export class Track {
	    prop: string;
	    keys: Key[];
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prop = source["prop"];
	        this.keys = this.convertValues(source["keys"], Key);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace animator {
	
	export class Cond {
	    param: string;
	    op?: string;
	    value?: number;
	
	    static createFrom(source: any = {}) {
	        return new Cond(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.param = source["param"];
	        this.op = source["op"];
	        this.value = source["value"];
	    }
	}
	export class Transition {
	    from: string;
	    to: string;
	    when?: Cond[];
	    exit?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Transition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.when = this.convertValues(source["when"], Cond);
	        this.exit = source["exit"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class State {
	    name: string;
	    clip: string;
	    speed?: number;
	    x?: number;
	    y?: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.clip = source["clip"];
	        this.speed = source["speed"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class Param {
	    name: string;
	    type: string;
	    value?: number;
	
	    static createFrom(source: any = {}) {
	        return new Param(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.value = source["value"];
	    }
	}
	export class Controller {
	    parameters: Param[];
	    states: State[];
	    start: string;
	    transitions: Transition[];
	
	    static createFrom(source: any = {}) {
	        return new Controller(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parameters = this.convertValues(source["parameters"], Param);
	        this.states = this.convertValues(source["states"], State);
	        this.start = source["start"];
	        this.transitions = this.convertValues(source["transitions"], Transition);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace asset {
	
	export class Entry {
	    path: string;
	    kind: string;
	    size: number;
	    guid: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.guid = source["guid"];
	    }
	}

}

export namespace pack {
	
	export class Manifest {
	    format: number;
	    name: string;
	    version: string;
	    engine?: string;
	    description?: string;
	    author?: string;
	    modules?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.engine = source["engine"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.modules = source["modules"];
	    }
	}

}

export namespace pipeline {
	
	export class Info {
	    name: string;
	    stage: string;
	    enabled: boolean;
	    micros: number;
	    editable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.stage = source["stage"];
	        this.enabled = source["enabled"];
	        this.micros = source["micros"];
	        this.editable = source["editable"];
	    }
	}

}

export namespace prop {
	
	export class Prop {
	    name: string;
	    type: string;
	    value: any;
	    options?: string[];
	    min?: number;
	    max?: number;
	    step?: number;
	    group?: string;
	    locked?: boolean;
	    overridden?: boolean;
	    display?: string;
	
	    static createFrom(source: any = {}) {
	        return new Prop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.options = source["options"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.step = source["step"];
	        this.group = source["group"];
	        this.locked = source["locked"];
	        this.overridden = source["overridden"];
	        this.display = source["display"];
	    }
	}

}

export namespace session {
	
	export class BuildOptions {
	    single: boolean;
	    version: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.single = source["single"];
	        this.version = source["version"];
	        this.icon = source["icon"];
	    }
	}
	export class BuildResult {
	    files: string[];
	    dir: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.dir = source["dir"];
	    }
	}
	export class Clip {
	    index: number;
	    name: string;
	    target: string;
	    mode: string;
	    speed: number;
	    length: number;
	    tracks: anim.Track[];
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Clip(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.target = source["target"];
	        this.mode = source["mode"];
	        this.speed = source["speed"];
	        this.length = source["length"];
	        this.tracks = this.convertValues(source["tracks"], anim.Track);
	        this.active = source["active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Handle {
	    key: string;
	    name: string;
	    at: number[];
	
	    static createFrom(source: any = {}) {
	        return new Handle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.at = source["at"];
	    }
	}
	export class Gizmo {
	    id: number;
	    corners: number[][];
	    anchor: number[];
	    axes: number[][];
	    handles: Handle[];
	    others: number[][][];
	
	    static createFrom(source: any = {}) {
	        return new Gizmo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.corners = source["corners"];
	        this.anchor = source["anchor"];
	        this.axes = source["axes"];
	        this.handles = this.convertValues(source["handles"], Handle);
	        this.others = source["others"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Frame {
	    cols: number;
	    rows: number;
	    cells?: string;
	    seq: number;
	    time: number;
	    mode: string;
	    millis: number;
	    passes: pipeline.Info[];
	    gizmo?: Gizmo;
	    view: string;
	    unsaved: boolean;
	    selection: number[];
	
	    static createFrom(source: any = {}) {
	        return new Frame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.cells = source["cells"];
	        this.seq = source["seq"];
	        this.time = source["time"];
	        this.mode = source["mode"];
	        this.millis = source["millis"];
	        this.passes = this.convertValues(source["passes"], pipeline.Info);
	        this.gizmo = this.convertValues(source["gizmo"], Gizmo);
	        this.view = source["view"];
	        this.unsaved = source["unsaved"];
	        this.selection = source["selection"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Info {
	    open: boolean;
	    name: string;
	    version: string;
	    engine: string;
	    dir: string;
	    manifest: string;
	    scene: string;
	    scenes: string[];
	    unsaved: boolean;
	    mode: string;
	    view: string;
	    live: boolean;
	    templates: string[];
	    canUndo: boolean;
	    canRedo: boolean;
	    prefab?: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.open = source["open"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.engine = source["engine"];
	        this.dir = source["dir"];
	        this.manifest = source["manifest"];
	        this.scene = source["scene"];
	        this.scenes = source["scenes"];
	        this.unsaved = source["unsaved"];
	        this.mode = source["mode"];
	        this.view = source["view"];
	        this.live = source["live"];
	        this.templates = source["templates"];
	        this.canUndo = source["canUndo"];
	        this.canRedo = source["canRedo"];
	        this.prefab = source["prefab"];
	    }
	}
	export class Kind {
	    kind: string;
	    category: string;
	    label: string;
	    object: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Kind(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.category = source["category"];
	        this.label = source["label"];
	        this.object = source["object"];
	    }
	}
	export class Kinds {
	    components: Kind[];
	    prefabs: string[];
	    models: string[];
	
	    static createFrom(source: any = {}) {
	        return new Kinds(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.components = this.convertValues(source["components"], Kind);
	        this.prefabs = source["prefabs"];
	        this.models = source["models"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LogEntry {
	    time: string;
	    level: string;
	    source: string;
	    text: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.source = source["source"];
	        this.text = source["text"];
	        this.count = source["count"];
	    }
	}
	export class MaterialDrop {
	    id: number;
	    part: string;
	
	    static createFrom(source: any = {}) {
	        return new MaterialDrop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.part = source["part"];
	    }
	}
	export class Node {
	    id: number;
	    name: string;
	    kind: string;
	    visible: boolean;
	    prefab?: string;
	    overrides?: number;
	    inherited?: boolean;
	    components?: string[];
	    children?: Node[];
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.visible = source["visible"];
	        this.prefab = source["prefab"];
	        this.overrides = source["overrides"];
	        this.inherited = source["inherited"];
	        this.components = source["components"];
	        this.children = this.convertValues(source["children"], Node);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Package {
	    name: string;
	    version: string;
	    description: string;
	    author: string;
	    modules: string[];
	    missing: string[];
	    source: string;
	    installed: string;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new Package(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.modules = source["modules"];
	        this.missing = source["missing"];
	        this.source = source["source"];
	        this.installed = source["installed"];
	        this.files = source["files"];
	    }
	}
	export class Packages {
	    installed: Package[];
	    available: Package[];
	
	    static createFrom(source: any = {}) {
	        return new Packages(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = this.convertValues(source["installed"], Package);
	        this.available = this.convertValues(source["available"], Package);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Preview {
	    cols: number;
	    rows: number;
	    cells: string;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.cells = source["cells"];
	    }
	}
	export class ProjectSettings {
	    name: string;
	    version: string;
	    start: string;
	    cols: number;
	    rows: number;
	    fps: number;
	    mods: boolean;
	    modApi: string;
	    icon: string;
	    display: string;
	    scale: number;
	    width: number;
	    height: number;
	    fullscreen: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProjectSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.start = source["start"];
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.fps = source["fps"];
	        this.mods = source["mods"];
	        this.modApi = source["modApi"];
	        this.icon = source["icon"];
	        this.display = source["display"];
	        this.scale = source["scale"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.fullscreen = source["fullscreen"];
	    }
	}

}

