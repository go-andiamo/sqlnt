package sqlnt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (n *namedTemplate) buildArgs() error {
	if err := n.replaceTokens(true); err != nil {
		return err
	}
	var builder strings.Builder
	n.argsCount = 0
	lastPos := 0
	runes := []rune(n.originalStatement)
	rlen := len(runes)
	purge := func(pos int) {
		if pos > lastPos {
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

var tokenRegexp = regexp.MustCompile(`\{\{([^}]*)}}`)

func (n *namedTemplate) replaceTokens(first bool) error {
	errs := make([]string, 0)
	n.originalStatement = tokenRegexp.ReplaceAllStringFunc(n.originalStatement, func(s string) string {
		token := s[2 : len(s)-2]
		for _, tr := range n.tokenOptions {
			if r, ok := tr.Replace(token); ok {
				return r
			}
		}
		errs = append(errs, token)
		return ""
	})
	if len(errs) == 1 {
		return fmt.Errorf("unknown token: %s", errs[0])
	} else if len(errs) > 0 {
		return fmt.Errorf("unknown tokens: %s", strings.Join(errs, ", "))
	}
	if first && strings.Contains(n.originalStatement, "{{") && strings.Contains(n.originalStatement, "}}") {
		return n.replaceTokens(false)
	}
	return nil
}

func isNameRune(r rune) bool {
	return r == '_' || r == '-' || r == '.' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
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
		return arg.tag
	} else {
		tag := n.argTag + strconv.Itoa(n.argsCount+1)
		n.args[name] = &namedArg{
			tag:       tag,
			positions: []int{n.argsCount},
			omissible: omissible,
		}
		n.argsCount++
		return tag
	}
}

func (n *namedTemplate) addNamedArgNonPositional(name string, omissible bool) string {
	if arg, ok := n.args[name]; ok {
		arg.setOmissible(omissible)
		arg.positions = append(arg.positions, n.argsCount)
	} else {
		n.args[name] = &namedArg{
			tag:       n.argTag,
			positions: []int{n.argsCount},
			omissible: omissible,
		}
	}
	n.argsCount++
	return n.argTag
}
