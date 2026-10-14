package stockpop

import (
	"bytes"
	"strings"
)

// waveSpawnsMax bounds the WaveSpawns of one Wave the reordering will look at.
// Valve's largest has 17.
const waveSpawnsMax = 1024

// token is a word, a quoted string or a brace of a population file, with the
// bytes it covers. Comments are skipped.
type token struct {
	text       string
	start, end int
}

func lex(body []byte) []token {
	var tokens []token
	for i := 0; i < len(body); {
		switch c := body[i]; {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < len(body) && body[i+1] == '/':
			for i < len(body) && body[i] != '\n' {
				i++
			}
		case c == '{' || c == '}':
			tokens = append(tokens, token{string(c), i, i + 1})
			i++
		case c == '"':
			end := bytes.IndexByte(body[i+1:], '"')
			if end < 0 {
				end = len(body) - i - 1
			}
			tokens = append(tokens, token{string(body[i+1 : i+1+end]), i, min(i+end+2, len(body))})
			i += end + 2
		default:
			start := i
			for i < len(body) && !endsWord(body, i) {
				i++
			}
			tokens = append(tokens, token{string(body[start:i]), start, i})
		}
	}
	return tokens
}

func endsWord(body []byte, i int) bool {
	switch body[i] {
	case ' ', '\t', '\r', '\n', '{', '}', '"':
		return true
	case '/':
		return i+1 < len(body) && body[i+1] == '/'
	}
	return false
}

// spawn is a WaveSpawn block: the bytes it occupies and what it waits for.
type spawn struct {
	start, end int
	name       string
	waits      []string
}

// waveSpawns returns the WaveSpawn blocks of each Wave, in file order.
func waveSpawns(body []byte, tokens []token) [][]spawn {
	type frame struct {
		key  string
		wave int
	}
	var (
		stack []frame
		waves [][]spawn
	)
	for i := 1; i < len(tokens); i++ {
		switch tokens[i].text {
		case "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		case "{":
		default:
			continue
		}
		key := tokens[i-1]
		if tokens[i-1].text == "{" || tokens[i-1].text == "}" {
			key = token{}
		}
		frame := frame{key: strings.ToLower(key.text), wave: -1}
		if frame.key == "wave" {
			waves = append(waves, nil)
			frame.wave = len(waves) - 1
		}
		if frame.key == "wavespawn" && len(stack) > 0 && stack[len(stack)-1].key == "wave" {
			block, last := readSpawn(body, tokens, key, i)
			wave := stack[len(stack)-1].wave
			waves[wave] = append(waves[wave], block)
			i = last
			continue
		}
		stack = append(stack, frame)
	}
	return waves
}

// readSpawn reads the block whose key is tokens[open-1] and whose brace is
// tokens[open]. It returns the block and the index of its closing brace.
func readSpawn(body []byte, tokens []token, key token, open int) (spawn, int) {
	s := spawn{start: key.start}
	for at := key.start; at > 0 && (body[at-1] == ' ' || body[at-1] == '\t'); at-- {
		s.start = at - 1
	}
	if s.start > 0 && body[s.start-1] != '\n' {
		s.start = key.start
	}

	depth, k := 1, open+1
	for ; k < len(tokens) && depth > 0; k++ {
		switch tokens[k].text {
		case "{":
			depth++
		case "}":
			depth--
		default:
			if depth != 1 || k+1 >= len(tokens) || tokens[k+1].text == "{" || tokens[k+1].text == "}" {
				continue
			}
			switch strings.ToLower(tokens[k].text) {
			case "name":
				s.name = strings.ToLower(tokens[k+1].text)
			case "waitforalldead", "waitforallspawned":
				s.waits = append(s.waits, strings.ToLower(tokens[k+1].text))
			}
			k++
		}
	}
	last := k - 1
	s.end = tokens[last].end
	rest := s.end
	for rest < len(body) && (body[rest] == ' ' || body[rest] == '\t' || body[rest] == '\r') {
		rest++
	}
	if rest < len(body) && body[rest] == '\n' {
		s.end = rest + 1
	}
	return s, last
}

// dependencyOrder returns the indexes of spawns with each one after every
// spawn whose name it waits for, and otherwise in file order. A cycle keeps
// the file's order for what it leaves.
func dependencyOrder(spawns []spawn) []int {
	order := make([]int, 0, len(spawns))
	placed := make([]bool, len(spawns))
	ready := func(i int) bool {
		for _, wait := range spawns[i].waits {
			for j, other := range spawns {
				if j != i && !placed[j] && other.name == wait {
					return false
				}
			}
		}
		return true
	}
	for len(order) < len(spawns) {
		next := -1
		for i := range spawns {
			if !placed[i] && ready(i) {
				next = i
				break
			}
		}
		if next < 0 {
			for i := range spawns {
				if !placed[i] {
					next = i
					break
				}
			}
		}
		placed[next] = true
		order = append(order, next)
	}
	return order
}

// orderWaveSpawns lists every WaveSpawn after the ones it waits for.
//
// The game decides a wave is done in CWave::ActiveWaveUpdate, which walks the
// WaveSpawns in file order and skips the ones still waiting without clearing
// "every WaveSpawn is done". A spawn listed before the one it waits for is
// skipped on the tick that one finishes, and if the rest are done or waiting
// too, the wave ends. Valve lists the tanks of wave07, which wait for wave06,
// ahead of wave06's 64 spies: the wave ends when the last spy dies and neither
// those tanks nor anything that waits for them ever starts.
func orderWaveSpawns(body []byte) []byte {
	var out []byte
	copied := 0
	for _, spawns := range waveSpawns(body, lex(body)) {
		if len(spawns) < 2 || len(spawns) > waveSpawnsMax {
			continue
		}
		order := dependencyOrder(spawns)
		for slot, from := range order {
			if slot == from {
				continue
			}
			out = append(out, body[copied:spawns[slot].start]...)
			block := body[spawns[from].start:spawns[from].end]
			out = append(out, block...)
			copied = spawns[slot].end
			if !bytes.HasSuffix(block, []byte("\n")) && bytes.HasSuffix(body[spawns[slot].start:spawns[slot].end], []byte("\n")) {
				out = append(out, '\n')
			}
		}
	}
	if copied == 0 {
		return body
	}
	return append(out, body[copied:]...)
}
