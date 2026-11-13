package mesh

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

var ErrBadFBX = errors.New("bad fbx")

const fbxMagic = "Kaydara FBX Binary  \x00"

// fbxNode is one record of either FBX encoding. Props hold int64, float64,
// bool, string, []byte, []int64 or []float64.
type fbxNode struct {
	Name  string
	Props []any
	Kids  []*fbxNode
}

func (n *fbxNode) is(name string) bool { return n != nil && n.Name == name }

func (n *fbxNode) kids() []*fbxNode {
	if n == nil {
		return nil
	}
	return n.Kids
}

func (n *fbxNode) kid(name string) *fbxNode {
	if n == nil {
		return nil
	}
	for _, k := range n.Kids {
		if k.Name == name {
			return k
		}
	}
	return nil
}

func (n *fbxNode) str(i int) string {
	if n == nil || i >= len(n.Props) {
		return ""
	}
	s, _ := n.Props[i].(string)
	return s
}

func (n *fbxNode) num(i int) (float64, bool) {
	if n == nil || i >= len(n.Props) {
		return 0, false
	}
	switch v := n.Props[i].(type) {
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (n *fbxNode) i64(i int) int64 {
	if n == nil || i >= len(n.Props) {
		return 0
	}
	switch v := n.Props[i].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	return 0
}

// floats flattens every numeric prop, arrays and scalars alike, so binary
// arrays and ASCII lists read the same.
func (n *fbxNode) floats() []float64 {
	if n == nil {
		return nil
	}
	var out []float64
	for _, p := range n.Props {
		switch v := p.(type) {
		case []float64:
			out = append(out, v...)
		case []int64:
			for _, x := range v {
				out = append(out, float64(x))
			}
		case float64:
			out = append(out, v)
		case int64:
			out = append(out, float64(v))
		}
	}
	return out
}

func (n *fbxNode) ints() []int64 {
	if n == nil {
		return nil
	}
	var out []int64
	for _, p := range n.Props {
		switch v := p.(type) {
		case []int64:
			out = append(out, v...)
		case []float64:
			for _, x := range v {
				out = append(out, int64(x))
			}
		case int64:
			out = append(out, v)
		case float64:
			out = append(out, int64(v))
		}
	}
	return out
}

// readFBX returns the top level nodes and the file version, such as 7400.
func readFBX(b []byte) ([]*fbxNode, int, error) {
	if bytes.HasPrefix(b, []byte(fbxMagic)) {
		return readBinaryFBX(b)
	}
	return readASCIIFBX(b)
}

type fbxReader struct {
	b    []byte
	off  int
	wide bool // 7.5 and later use 64 bit offsets
}

func readBinaryFBX(b []byte) ([]*fbxNode, int, error) {
	if len(b) < 27 {
		return nil, 0, fmt.Errorf("%w: truncated header", ErrBadFBX)
	}
	ver := int(binary.LittleEndian.Uint32(b[23:27]))
	r := &fbxReader{b: b, off: 27, wide: ver >= 7500}
	var roots []*fbxNode
	for {
		n, err := r.node(0)
		if err != nil {
			return nil, 0, err
		}
		if n == nil {
			return roots, ver, nil
		}
		roots = append(roots, n)
	}
}

func (r *fbxReader) take(n int) ([]byte, error) {
	if n < 0 || r.off+n > len(r.b) {
		return nil, fmt.Errorf("%w: truncated at byte %d", ErrBadFBX, r.off)
	}
	s := r.b[r.off : r.off+n]
	r.off += n
	return s, nil
}

func (r *fbxReader) u32() (uint32, error) {
	s, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(s), nil
}

func (r *fbxReader) word() (uint64, error) {
	if !r.wide {
		v, err := r.u32()
		return uint64(v), err
	}
	s, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(s), nil
}

// node reads one record; nil, nil is the null record that closes a list.
func (r *fbxReader) node(depth int) (*fbxNode, error) {
	if depth > 64 {
		return nil, fmt.Errorf("%w: nodes nested too deep", ErrBadFBX)
	}
	if r.off >= len(r.b) {
		return nil, nil // some writers end the top level without a null record
	}
	end, err := r.word()
	if err != nil {
		return nil, err
	}
	count, err := r.word()
	if err != nil {
		return nil, err
	}
	if _, err := r.word(); err != nil {
		return nil, err
	}
	nameLen, err := r.take(1)
	if err != nil {
		return nil, err
	}
	if end == 0 {
		return nil, nil
	}
	if end > uint64(len(r.b)) || int(end) < r.off {
		return nil, fmt.Errorf("%w: node ends outside the file at byte %d", ErrBadFBX, r.off)
	}
	name, err := r.take(int(nameLen[0]))
	if err != nil {
		return nil, err
	}
	n := &fbxNode{Name: string(name)}
	if count > uint64(len(r.b)) {
		return nil, fmt.Errorf("%w: %s claims %d properties", ErrBadFBX, n.Name, count)
	}
	for range count {
		p, err := r.prop()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n.Name, err)
		}
		n.Props = append(n.Props, p)
	}
	for r.off < int(end) {
		k, err := r.node(depth + 1)
		if err != nil {
			return nil, err
		}
		if k == nil {
			break
		}
		n.Kids = append(n.Kids, k)
	}
	r.off = int(end)
	return n, nil
}

func (r *fbxReader) prop() (any, error) {
	t, err := r.take(1)
	if err != nil {
		return nil, err
	}
	le := binary.LittleEndian
	switch t[0] {
	case 'C':
		s, err := r.take(1)
		if err != nil {
			return nil, err
		}
		return s[0] != 0, nil
	case 'Y':
		s, err := r.take(2)
		if err != nil {
			return nil, err
		}
		return int64(int16(le.Uint16(s))), nil
	case 'I':
		s, err := r.take(4)
		if err != nil {
			return nil, err
		}
		return int64(int32(le.Uint32(s))), nil
	case 'L':
		s, err := r.take(8)
		if err != nil {
			return nil, err
		}
		return int64(le.Uint64(s)), nil
	case 'F':
		s, err := r.take(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(le.Uint32(s))), nil
	case 'D':
		s, err := r.take(8)
		if err != nil {
			return nil, err
		}
		return math.Float64frombits(le.Uint64(s)), nil
	case 'S', 'R':
		n, err := r.u32()
		if err != nil {
			return nil, err
		}
		s, err := r.take(int(n))
		if err != nil {
			return nil, err
		}
		if t[0] == 'S' {
			return string(s), nil
		}
		return bytes.Clone(s), nil
	case 'f', 'd', 'i', 'l', 'b':
		return r.array(t[0])
	}
	return nil, fmt.Errorf("%w: unknown property type %q", ErrBadFBX, t[0])
}

func (r *fbxReader) array(t byte) (any, error) {
	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	enc, err := r.u32()
	if err != nil {
		return nil, err
	}
	size, err := r.u32()
	if err != nil {
		return nil, err
	}
	raw, err := r.take(int(size))
	if err != nil {
		return nil, err
	}
	elem := 4
	switch t {
	case 'd', 'l':
		elem = 8
	case 'b':
		elem = 1
	}
	want := int(count) * elem
	switch enc {
	case 0:
		if want != len(raw) {
			return nil, fmt.Errorf("%w: array of %d needs %d bytes, has %d", ErrBadFBX, count, want, len(raw))
		}
	case 1:
		if want > len(raw)*1100+64 {
			return nil, fmt.Errorf("%w: array of %d cannot come from %d deflated bytes", ErrBadFBX, count, len(raw)) // past deflate's best ratio
		}
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrBadFBX, err)
		}
		out := make([]byte, want)
		if _, err := io.ReadFull(zr, out); err != nil {
			return nil, fmt.Errorf("%w: inflate array: %w", ErrBadFBX, err)
		}
		raw = out
	default:
		return nil, fmt.Errorf("%w: unknown array encoding %d", ErrBadFBX, enc)
	}
	le := binary.LittleEndian
	switch t {
	case 'f', 'd':
		out := make([]float64, count)
		for i := range out {
			if t == 'f' {
				out[i] = float64(math.Float32frombits(le.Uint32(raw[i*4:])))
			} else {
				out[i] = math.Float64frombits(le.Uint64(raw[i*8:]))
			}
		}
		return out, nil
	}
	out := make([]int64, count)
	for i := range out {
		switch t {
		case 'i':
			out[i] = int64(int32(le.Uint32(raw[i*4:])))
		case 'l':
			out[i] = int64(le.Uint64(raw[i*8:]))
		default:
			out[i] = int64(raw[i])
		}
	}
	return out, nil
}

type fbxLexer struct {
	s    string
	off  int
	line int
}

type fbxToken struct {
	kind byte // k key, s string, n number, w word, { } , and * for an array length
	text string
}

func (l *fbxLexer) next() fbxToken {
	for l.off < len(l.s) {
		c := l.s[l.off]
		switch {
		case c == '\n':
			l.line++
			l.off++
		case c == ' ' || c == '\t' || c == '\r':
			l.off++
		case c == ';':
			for l.off < len(l.s) && l.s[l.off] != '\n' {
				l.off++
			}
		case c == '"':
			end := strings.IndexByte(l.s[l.off+1:], '"')
			if end < 0 {
				end = len(l.s) - l.off - 1
			}
			t := l.s[l.off+1 : l.off+1+end]
			l.off += end + 2
			return fbxToken{kind: 's', text: strings.ReplaceAll(t, "&quot;", `"`)}
		case c == '{' || c == '}' || c == ',':
			l.off++
			return fbxToken{kind: c}
		default:
			start := l.off
			for l.off < len(l.s) && !strings.ContainsRune(" \t\r\n{},:\";", rune(l.s[l.off])) {
				l.off++
			}
			t := l.s[start:l.off]
			if l.off < len(l.s) && l.s[l.off] == ':' {
				l.off++
				return fbxToken{kind: 'k', text: t}
			}
			if t == "" {
				l.off++ // a stray colon
				continue
			}
			if strings.HasPrefix(t, "*") {
				return fbxToken{kind: '*', text: t}
			}
			if _, err := strconv.ParseFloat(t, 64); err == nil {
				return fbxToken{kind: 'n', text: t}
			}
			return fbxToken{kind: 'w', text: t}
		}
	}
	return fbxToken{}
}

type fbxParser struct {
	lex  fbxLexer
	tok  fbxToken
	peek bool
}

func (p *fbxParser) look() fbxToken {
	if !p.peek {
		p.tok, p.peek = p.lex.next(), true
	}
	return p.tok
}

func (p *fbxParser) eat() fbxToken {
	t := p.look()
	p.peek = false
	return t
}

func readASCIIFBX(b []byte) ([]*fbxNode, int, error) {
	if !bytes.Contains(b[:min(len(b), 4096)], []byte("FBXHeaderExtension")) {
		return nil, 0, fmt.Errorf("%w: neither binary nor ASCII FBX", ErrBadFBX)
	}
	p := &fbxParser{lex: fbxLexer{s: string(b), line: 1}}
	roots, err := p.nodes(0)
	if err != nil {
		return nil, 0, err
	}
	if t := p.look(); t.kind != 0 {
		return nil, 0, fmt.Errorf("%w: line %d: unexpected %q", ErrBadFBX, p.lex.line, t.kind)
	}
	ver := 0
	for _, r := range roots {
		if r.Name == "FBXHeaderExtension" {
			if v, ok := r.kid("FBXVersion").num(0); ok {
				ver = int(v)
			}
		}
	}
	return roots, ver, nil
}

func (p *fbxParser) nodes(depth int) ([]*fbxNode, error) {
	if depth > 64 {
		return nil, fmt.Errorf("%w: line %d: nodes nested too deep", ErrBadFBX, p.lex.line)
	}
	var out []*fbxNode
	for p.look().kind == 'k' {
		n := &fbxNode{Name: p.eat().text}
		for {
			t := p.look()
			if t.kind == 's' || t.kind == 'n' || t.kind == 'w' {
				n.Props = append(n.Props, asciiValue(p.eat()))
			} else if t.kind == '*' {
				p.eat() // the array length; the values follow in an "a" node
				continue
			}
			if p.look().kind != ',' {
				break
			}
			p.eat()
		}
		if p.look().kind == '{' {
			p.eat()
			kids, err := p.nodes(depth + 1)
			if err != nil {
				return nil, err
			}
			if t := p.eat(); t.kind != '}' {
				return nil, fmt.Errorf("%w: line %d: %s is not closed", ErrBadFBX, p.lex.line, n.Name)
			}
			n.Kids = kids
		}
		if len(n.Kids) == 1 && n.Kids[0].Name == "a" && len(n.Props) == 0 {
			n.Props, n.Kids = []any{asciiArray(n.Kids[0].Props)}, nil
		}
		out = append(out, n)
	}
	return out, nil
}

func asciiValue(t fbxToken) any {
	switch t.kind {
	case 'n':
		if i, err := strconv.ParseInt(t.text, 10, 64); err == nil {
			return i
		}
		f, _ := strconv.ParseFloat(t.text, 64)
		return f
	}
	return t.text
}

// asciiArray turns the values of an "a" node into the array a binary file
// would hold, floats when any value has a fraction.
func asciiArray(vals []any) any {
	ints := make([]int64, 0, len(vals))
	for _, v := range vals {
		i, ok := v.(int64)
		if !ok {
			break
		}
		ints = append(ints, i)
	}
	if len(ints) == len(vals) {
		return ints
	}
	out := make([]float64, 0, len(vals))
	for _, v := range vals {
		switch x := v.(type) {
		case int64:
			out = append(out, float64(x))
		case float64:
			out = append(out, x)
		}
	}
	return out
}
