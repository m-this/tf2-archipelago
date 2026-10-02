package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// manifestBytesMax bounds what is read of an app manifest. A real one is a
// few kilobytes.
const manifestBytesMax = 1 << 20

func manifestPath(gameDir string) string {
	return filepath.Join(gameDir, "steamapps", "appmanifest_"+AppID+".acf")
}

// InstalledBuild is the Steam build of the installed TF2 server, read from its
// app manifest. A missing manifest is an error wrapping fs.ErrNotExist.
func InstalledBuild(installRoot string) (string, error) {
	file, err := os.Open(manifestPath(gamePath(installRoot)))
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	text, err := io.ReadAll(io.LimitReader(file, manifestBytesMax))
	if err != nil {
		return "", err
	}
	build, err := manifestBuildID(string(text))
	if err != nil {
		return "", fmt.Errorf("%s: %w", file.Name(), err)
	}
	return build, nil
}

// manifestBuildID reads AppState's own buildid out of Valve's KeyValues text.
// Each depot carries a buildid of its own, one block deeper, and those are not
// the app's.
func manifestBuildID(text string) (string, error) {
	tokens, err := keyValueTokens(text)
	if err != nil {
		return "", err
	}
	if len(tokens) < 2 || tokens[0] != `"AppState"` || tokens[1] != "{" {
		return "", errors.New("not an AppState manifest")
	}
	depth, build := 0, ""
	for i := 1; i < len(tokens); i++ {
		switch tokens[i] {
		case "{":
			depth++
			continue
		case "}":
			depth--
			if depth < 0 || depth == 0 && i != len(tokens)-1 {
				return "", errors.New("unbalanced braces")
			}
			continue
		}
		if i+1 >= len(tokens) {
			return "", errors.New("a key without a value")
		}
		key, value := unquote(tokens[i]), tokens[i+1]
		if value != "{" {
			i++
		}
		if depth == 1 && strings.EqualFold(key, "buildid") {
			build = unquote(value)
			if value == "{" || !allDigits(build) {
				return "", fmt.Errorf("buildid %s is not a number", value)
			}
		}
	}
	if depth != 0 {
		return "", errors.New("unbalanced braces")
	}
	if build == "" {
		return "", errors.New("no buildid")
	}
	return build, nil
}

// keyValueTokens splits KeyValues text into quoted strings, kept with their
// quotes, and braces. Manifests never escape a quote, so neither does this.
func keyValueTokens(text string) ([]string, error) {
	var tokens []string
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '{', '}':
			tokens = append(tokens, string(c))
		case '"':
			end := strings.IndexAny(text[i+1:], "\"\n")
			if end < 0 || text[i+1+end] != '"' {
				return nil, errors.New("an unclosed quote")
			}
			tokens = append(tokens, text[i:i+end+2])
			i += end + 1
		case ' ', '\t', '\r', '\n':
		default:
			return nil, fmt.Errorf("unexpected %q", c)
		}
	}
	return tokens, nil
}

func unquote(token string) string { return strings.Trim(token, `"`) }

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
