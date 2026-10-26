"""Rewrites the shared frame of every page in docs/usage: head, sidebar and
pager, from the nav list below. Page content between <main> and the pager is
kept. Run it after adding, renaming or reordering a page."""
import re, os, sys
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'usage')
nav = [
  ('Start', [('index.html','Welcome'),('install.html','Install'),('first-game.html','Your first game')]),
  ('Guides', [('editor.html','The editor'),('gameobjects.html','GameObjects and components'),('scenes.html','Scenes and prefabs'),
              ('assets.html','Assets and import'),('packages.html','Packages'),('textures.html','Textures, materials, sprites'),('models.html','Models'),
              ('animation.html','Animation'),('particles.html','Particles'),('audio.html','Audio'),('physics.html','Physics'),('scripting.html','Lua scripting'),('rendering.html','Rendering'),('building.html','Build'),('modding.html','Mods')]),
  ('Reference', [('components.html','Component reference'),('dragonfall.html','Dragonfall module'),('lua-api.html','Lua API'),('cli.html','CLI reference'),('go-api.html','Go API'),('architecture.html','Architecture')]),
]
order = [p for _, ps in nav for p in ps]
section = {f: s for s, ps in nav for f, _ in ps}
titles = dict(order)

def aside(cur):
    out = ['<aside class="side">','<a class="brand" href="index.html"><img src="img/logo.svg" alt=""><span>ASCDII Engine<small>0.0.2 docs</small></span></a>']
    for s, ps in nav:
        out.append(f'<h4>{s}</h4>')
        for f, t in ps:
            on = ' class="on"' if f == cur else ''
            out.append(f'<a href="{f}"{on}>{t}</a>')
    out.append('</aside>')
    return '\n'.join(out)

def pager(cur):
    i = [f for f, _ in order].index(cur)
    prev = f'<a class="prev" href="{order[i-1][0]}"><small>Previous</small>{order[i-1][1]}</a>' if i > 0 else '<span></span>'
    nxt = f'<a class="next" href="{order[i+1][0]}"><small>Next</small>{order[i+1][1]}</a>' if i < len(order)-1 else '<span></span>'
    return f'<nav class="pager">{prev}{nxt}</nav>'

def page(f, body):
    t = titles[f]
    head = 'ASCDII Engine' if f == 'index.html' else f'{t} | ASCDII Engine'
    return f'''<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{head}</title>
<link rel="icon" href="img/logo.svg" type="image/svg+xml">
<link rel="stylesheet" href="style.css">
<script src="zoom.js" defer></script>
</head>
<body>
<div class="layout">
{aside(f)}
<main>
{body.strip()}
{pager(f)}
</main>
</div>
</body>
</html>
'''

for f, _ in order:
    src = open(os.path.join(D, f)).read()
    body = src[src.index('<main>')+6:src.index('</main>')]
    body = re.sub(r'<nav class="pager">.*?</nav>', '', body, flags=re.S)
    html = page(f, body)
    if '\u2014' in html:
        sys.exit(f'em dash in {f}')
    open(os.path.join(D, f), 'w').write(html)
print('ok', len(order))
