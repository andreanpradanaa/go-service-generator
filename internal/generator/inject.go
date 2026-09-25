package generator

import (
	"fmt"
	"go/format"
	"os"
	"strings"
)

// The functions below register generated code in an existing service by
// anchoring on its own code structure (e.g. the `var Module = fx.Provide(`
// call), so generated services carry no generator-specific markers.

// appendToBlock inserts line as the last entry of the parenthesised or braced
// block that opener starts, e.g. `var routerModule = fx.Provide(`. Nothing is
// written if the block already contains line. The file is gofmt-ed.
func appendToBlock(file, opener, line string) error {
	return insertInBlock(file, opener, "", line)
}

// insertInBlock is appendToBlock, but places line just above the entry
// anchor when the block has one.
func insertInBlock(file, opener, anchor, line string) error {
	return editGo(file, func(content string) (string, error) {
		start, end, err := findBlock(content, opener)
		if err != nil {
			return "", fmt.Errorf("%s: %w", file, err)
		}
		block := content[start:end]
		if blockHasLine(block, line) {
			return content, nil
		}

		if anchor != "" {
			if i := strings.Index(block, anchor); i >= 0 {
				at := start + strings.LastIndex(block[:i], "\n") + 1
				return content[:at] + "\t" + line + "\n" + content[at:], nil
			}
		}

		before := strings.TrimRight(content[:end], " \t\n")
		return before + "\n\t" + line + "\n" + content[end:], nil
	})
}

// addImport adds importPath to the import block of file, next to the other
// imports of the service's own module so gofmt keeps it in that group.
func addImport(file, module, importPath string) error {
	line := fmt.Sprintf("%q", importPath)
	return editGo(file, func(content string) (string, error) {
		start, end, err := findBlock(content, "import (")
		if err != nil {
			return "", fmt.Errorf("%s: %w", file, err)
		}
		block := content[start:end]
		if blockHasLine(block, line) {
			return content, nil
		}

		// Insert after the last import of the service's own module. When
		// there is none yet, start a new group at the top of the block so
		// gofmt does not sort it in with third-party imports.
		if i := strings.LastIndex(block, `"`+module+"/"); i >= 0 {
			at := start + i + strings.Index(block[i:], "\n") + 1
			return content[:at] + "\t" + line + "\n" + content[at:], nil
		}
		rest := strings.TrimLeft(block, "\n")
		return content[:start] + "\n\t" + line + "\n\n" + rest + content[end:], nil
	})
}

func editGo(file string, edit func(string) (string, error)) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	content, err := edit(string(raw))
	if err != nil {
		return err
	}
	if content == string(raw) {
		return nil
	}
	out, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("gofmt %s: %w", file, err)
	}
	return os.WriteFile(file, out, 0o644)
}

// findBlock returns the span between the bracket that ends opener and its
// matching closing bracket (exclusive), skipping strings and comments.
func findBlock(content, opener string) (start, end int, err error) {
	idx := strings.Index(content, opener)
	if idx < 0 {
		return 0, 0, fmt.Errorf("%q not found — was it renamed?", opener)
	}
	start = idx + len(opener)

	depth := 1
	for i := start; i < len(content); i++ {
		switch c := content[i]; c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				return start, i, nil
			}
		case '"', '`', '\'':
			j := i + 1
			for j < len(content) && content[j] != c {
				if content[j] == '\\' && c != '`' {
					j++
				}
				j++
			}
			i = j
		case '/':
			if i+1 < len(content) && content[i+1] == '/' {
				for i < len(content) && content[i] != '\n' {
					i++
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("no closing bracket for %q", opener)
}

func blockHasLine(block, line string) bool {
	for _, l := range strings.Split(block, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

// addExternalConfig adds a partner section under app.external in
// config.yml, creating the external key at the end of the app section when
// the service has none yet. lines are relative to the partner key.
func addExternalConfig(file, key string, lines []string) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	content := strings.TrimRight(string(raw), "\n")
	all := strings.Split(content, "\n")

	section := make([]string, 0, len(lines)+1)
	section = append(section, "    "+key+":")
	for _, l := range lines {
		section = append(section, "      "+l)
	}

	ext := -1
	for i, l := range all {
		if l == "  external:" {
			ext = i
			break
		}
	}

	var out []string
	if ext < 0 {
		// app is the only top-level key, so its section runs to the end.
		out = append(all, "  external:")
		out = append(out, section...)
	} else {
		// The external block ends at the first line indented less than its
		// children.
		end := ext + 1
		for end < len(all) && (strings.HasPrefix(all[end], "    ") || strings.TrimSpace(all[end]) == "") {
			if strings.TrimSpace(all[end]) == key+":" {
				return nil
			}
			end++
		}
		out = append(out, all[:end]...)
		out = append(out, section...)
		out = append(out, all[end:]...)
	}
	return os.WriteFile(file, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}
