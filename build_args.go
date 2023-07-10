package sqlnt

import (
	"fmt"
	"strconv"
	"strings"
)

func (n *namedTemplate) buildArgs() error {
	var builder strings.Builder
	n.argsCount = 0
	lastPos := 0
	runes := []rune(n.originalStatement)
	rlen := len(runes)
	purge := func(pos int) {
		if lastPos != -1 && pos > lastPos {
			builder.WriteString(string(runes[lastPos:pos]))
		}
	}
	getNamed := func(pos int) (string, int, bool, error) {
		i := pos + 1
		skip := 0
		for ; i < rlen; i++ {
			if !isNameRune(runes[i]) {
				break
			}
			skip++
		}
		if skip == 0 {
			return "", 0, false, fmt.Errorf("named marker ':' without name (at position %d)", pos)
		}
		omissible := false
		if (i+1) <= rlen && runes[i] == '?' {
			omissible = true
			skip++
		}
		return string(runes[pos+1 : i]), skip, omissible, nil
	}
	for pos := 0; pos < rlen; pos++ {
		if runes[pos] == ':' {
			purge(pos)
			if (pos+1) < rlen && runes[pos+1] == ':' {
				// double escaped name marker...
				pos++
				lastPos = pos
			} else {
				name, skip, omissible, err := getNamed(pos)
				if err != nil {
					return err
				}
				pos += skip
				lastPos = pos + 1
				builder.WriteString(n.addNamedArg(name, omissible))
			}
		}
	}
	purge(rlen)
	n.statement = builder.String()
	return nil
}

func isNameRune(r rune) bool {
	return r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func (n *namedTemplate) addNamedArg(name string, omissible bool) string {
	if n.usePositionalTags {
		return n.addNamedArgPositional(name, omissible)
	} else {
		return n.addNamedArgNonPositional(name, omissible)
	}
}

func (n *namedTemplate) addNamedArgPositional(name string, omissible bool) string {
	if arg, ok := n.args[name]; ok {
		arg.setOmissible(omissible)
		return n.argTag + strconv.Itoa(arg.positions[0]+1)
	} else {
		n.args[name] = &namedArg{
			positions: []int{n.argsCount},
			omissible: omissible,
		}
		n.argsCount++
		return n.argTag + strconv.Itoa(n.argsCount)
	}
}

func (n *namedTemplate) addNamedArgNonPositional(name string, omissible bool) string {
	if arg, ok := n.args[name]; ok {
		arg.setOmissible(omissible)
		arg.positions = append(arg.positions, n.argsCount)
	} else {
		n.args[name] = &namedArg{
			positions: []int{n.argsCount},
			omissible: omissible,
		}
	}
	n.argsCount++
	return n.argTag
}

type namedArg struct {
	positions []int
	omissible bool
	defValue  DefaultValueFunc
}

func (a *namedArg) clone() *namedArg {
	return &namedArg{
		positions: a.positions,
		omissible: a.omissible,
		defValue:  a.defValue,
	}
}

func (a *namedArg) setOmissible(omissible bool) {
	if !a.omissible {
		// can only be set when not yet omissible
		a.omissible = omissible
	}
}
