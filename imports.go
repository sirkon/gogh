package gogh

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/chonla/roman-number-go"
)

// NewImports constructs new imports collector
func NewImports(weighter Weighter) *Imports {
	return &Imports{
		pkgs:     map[string]string{},
		weighter: weighter,
	}
}

// Imports is a collector to gather packages imports in of a single Renderer source file
type Imports struct {
	pkgs     map[string]string
	weighter Weighter
}

// Add adds new package into the list of packages
func (i *Imports) Add(alias string, path string) {
	var err error
	defer func() {
		if err == nil {
			return
		}
		switch alias {
		case "":
			panic(fmt.Errorf(`add import of "%s"': %w`, path, err))
		default:
			panic(fmt.Errorf(`add import of "%s" as %s: %w`, path, alias, err))
		}
	}()

	switch path {
	case "":
		err = fmt.Errorf("import path cannot be empty")
		return
	case "C":
		if alias != "" {
			err = fmt.Errorf(`"C" import path must be without an alias`)
			return
		}
	}

	if prev, ok := i.pkgs[path]; ok {
		if prev != alias {
			switch {
			case alias == "":
				err = fmt.Errorf(`already added with custom alias %s`, prev)
				return
			case prev == "":
				err = errors.New(`already added without an alias`)
				return
			default:
				err = fmt.Errorf(`already added with different alias %s`, prev)
				return
			}
		}
		return
	}

	i.pkgs[path] = alias
}

// Import representation of import path
type Import struct {
	Alias string
	Path  string
}

func (i Import) String() string {
	switch i.Alias {
	case "":
		return fmt.Sprintf(`"%s"`, i.Path)
	default:
		return fmt.Sprintf(`%s "%s"`, i.Alias, i.Path)
	}
}

var _ sort.Interface = ImportsGroup{}

// ImportsGroup representation of import groups. Has special heuristic sorting
type ImportsGroup []Import

// Len for sort.Interface implementation
func (g ImportsGroup) Len() int {
	return len(g)
}

// Less  for sort.Interface implementation
func (g ImportsGroup) Less(i, j int) bool {
	path1 := g[i].Path
	path2 := g[j].Path

	return heuristicCmp(path1, path2)
}

func heuristicCmp(a, b string) bool {
	if a == b {
		return false
	}
	if strings.HasPrefix(b, a) {
		return true
	}
	if strings.HasPrefix(a, b) {
		return false
	}

	// cut common prefix of paths. The rest will not be empty on both.
	for k := 0; k < len(a); k++ {
		if a[k] == b[k] {
			continue
		}
		a = a[k:]
		b = b[k:]
		break
	}
	isDigit1 := unicode.IsDigit([]rune(a)[0])
	isDigit2 := unicode.IsDigit([]rune(b)[0])
	switch {
	case !isDigit1 && isDigit2:
		return true
	case isDigit1 && !isDigit2:
		return false
	case isDigit1 && isDigit2:
		// нужно определить, у кого числа больше
		return headNumber(a) < headNumber(b)
	default:
		ar := tryRoman(a)
		br := tryRoman(b)
		if ar > 0 && br > 0 {
			return ar < br
		}
		return a < b
	}
}

func tryRoman(rest string) int {
	r := roman.NewRoman()
	return r.ToNumber(rest)
}

func headNumber(rest string) uint64 {
	var buf bytes.Buffer
	for _, r := range []rune(rest) {
		if unicode.IsDigit(r) {
			buf.WriteRune(r)
		}
		break
	}
	res, _ := strconv.ParseUint(buf.String(), 10, 64)
	return res
}

// Swap  for sort.Interface implementation
func (g ImportsGroup) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

// Result groups imports with their weight and return as a slice of groups
func (i *Imports) Result() []ImportsGroup {
	groups := map[int]ImportsGroup{}

	var weights []int
	for path, alias := range i.pkgs {
		weight := i.weighter.Weight(path)
		if _, ok := groups[weight]; !ok {
			weights = append(weights, weight)
		}
		groups[weight] = append(groups[weight], Import{
			Alias: alias,
			Path:  path,
		})
	}

	sort.Ints(weights)
	var res []ImportsGroup
	for _, weight := range weights {
		sort.Sort(groups[weight])
		res = append(res, groups[weight])
	}

	return res
}
