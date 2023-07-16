package sqlnt

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
)

func (n *namedTemplate) copy() *namedTemplate {
	r := newNamedTemplate(n.originalStatement, n.usePositionalTags, n.argTag, n.tokenOptions)
	r.statement = n.statement
	r.argsCount = n.argsCount
	r.usePositionalTags = n.usePositionalTags
	r.argTag = n.argTag
	for name, arg := range n.args {
		r.args[name] = arg.clone()
	}
	return r
}

func getOptions(options ...any) (Option, []TokenOption, error) {
	opt := DefaultsOption
	tokenOptions := make([]TokenOption, 0)
	for _, o := range options {
		if o != nil {
			o1, ok1 := o.(Option)
			o2, ok2 := o.(TokenOption)
			if !ok1 && !ok2 {
				return nil, nil, errors.New("invalid option")
			}
			if ok1 {
				opt = o1
			}
			if ok2 {
				tokenOptions = append(tokenOptions, o2)
			}
		}
	}
	return opt, tokenOptions, nil
}

func mappedArgs(args ...any) (map[string]any, error) {
	result := map[string]any{}
	for _, arg := range args {
		if arg != nil {
			switch targ := arg.(type) {
			case map[string]any:
				for k, v := range targ {
					result[k] = v
				}
			case *sql.NamedArg:
				result[targ.Name] = targ.Value
			case sql.NamedArg:
				result[targ.Name] = targ.Value
			default:
				if vo := reflect.ValueOf(arg); vo.Kind() == reflect.Map {
					// it's a map, but not a map[string]any...
					iter := vo.MapRange()
					for iter.Next() {
						if k, ok := iter.Key().Interface().(string); ok {
							result[k] = iter.Value().Interface()
						} else {
							return nil, errors.New("invalid map - keys must be string")
						}
					}
				} else {
					// not a type aware of - try marshaling and then unmarshalling to a map...
					if data, err := json.Marshal(arg); err == nil {
						var jm map[string]any
						if err := json.Unmarshal(data, &jm); err == nil {
							for k, v := range jm {
								result[k] = v
							}
						} else {
							return nil, err
						}
					} else {
						return nil, err
					}
				}
			}
		}
	}
	return result, nil
}
