package sqlnt

// ArgInfo is the info about a named arg returned from NamedTemplate.GetArgsInfo
type ArgInfo struct {
	// Tag is the final arg tag used for the named arg
	Tag string
	// Positions is the final arg positions for the named arg
	Positions []int
	// Omissible denotes whether the named arg is omissible
	Omissible bool
	// DefaultValue is the DefaultValueFunc provider function
	DefaultValue DefaultValueFunc
	// NullableString denotes whether the named arg is a nullable string
	// (i.e. if the supplied value is an empty, then nil is used)
	NullableString bool
}

type namedArg struct {
	tag            string
	positions      []int
	omissible      bool
	defValue       DefaultValueFunc
	nullableString bool
}

func (a *namedArg) toInfo() ArgInfo {
	return ArgInfo{
		Tag:            a.tag,
		Positions:      a.positions,
		Omissible:      a.omissible,
		DefaultValue:   a.defValue,
		NullableString: a.nullableString,
	}
}

func (a *namedArg) copyOptionsTo(r *namedArg) {
	r.omissible = a.omissible
	r.defValue = a.defValue
	r.nullableString = a.nullableString
}

func (a *namedArg) clone() *namedArg {
	return &namedArg{
		tag:            a.tag,
		positions:      a.positions,
		omissible:      a.omissible,
		defValue:       a.defValue,
		nullableString: a.nullableString,
	}
}

func (a *namedArg) setOmissible(omissible bool) {
	if !a.omissible {
		// can only be set when not yet omissible
		a.omissible = omissible
	}
}

func (a *namedArg) defaultedValue(name string) any {
	return a.value(a.defValue(name))
}

func (a *namedArg) value(v any) any {
	if a.nullableString {
		switch vt := v.(type) {
		case string:
			if vt == "" {
				return nil
			}
		case *string:
			if *vt == "" {
				return nil
			}
		}
	}
	return v
}
