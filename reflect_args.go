package rfrouter

import (
	"errors"
	"reflect"
	"strconv"
	"sync"
)

type argumentValueFn func(string) (reflect.Value, error)

// Parseable implements a Parse(string) method for data structures that can be
// used as arguments.
type Parseable interface {
	Parse(string) error
}

// ManaulParseable implements a ParseContent(string) method. If the library sees
// this for an argument, it will send all of the arguments (including the
// command) into the method. If used, this should be the only argument followed
// after the Message Create event. Any more and the router will ignore.
type ManualParseable interface {
	// $0 will have its prefix trimmed.
	ParseContent([]string) error
}

// nilV, only used to return an error
var nilV = reflect.Value{}

var customParsers = map[reflect.Type]Parseable{}
var customParsersMu sync.Mutex

// RegisterCustomArgument adds Parseable as a possible argument to parse
// arguments to. A plugin should call this function on its init().
func RegisterCustomArgument(p Parseable) {
	customParsersMu.Lock()
	defer customParsersMu.Unlock()

	customParsers[reflect.TypeOf(p)] = p
}

func getArgumentValueFn(t reflect.Type) (argumentValueFn, error) {
	// Custom ones get higher priorities
	customParsersMu.Lock()

	if p, ok := customParsers[t]; ok {
		customParsersMu.Unlock()

		return func(s string) (reflect.Value, error) {
			if err := p.Parse(s); err != nil {
				return nilV, err
			}

			return reflect.ValueOf(p), nil
		}, nil
	}

	customParsersMu.Unlock()

	var fn argumentValueFn

	switch t.Kind() {
	case reflect.String:
		fn = func(s string) (reflect.Value, error) {
			return reflect.ValueOf(s), nil
		}

	case reflect.Int, reflect.Int8,
		reflect.Int16, reflect.Int32, reflect.Int64:

		fn = func(s string) (reflect.Value, error) {
			i, err := strconv.ParseInt(s, 10, 64)
			return quickRet(i, err, t)
		}

	case reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64:

		fn = func(s string) (reflect.Value, error) {
			u, err := strconv.ParseUint(s, 10, 64)
			return quickRet(u, err, t)
		}

	case reflect.Float32, reflect.Float64:
		fn = func(s string) (reflect.Value, error) {
			f, err := strconv.ParseFloat(s, 64)
			return quickRet(f, err, t)
		}

	case reflect.Bool:
		fn = func(s string) (reflect.Value, error) {
			switch s {
			case "true", "yes", "y", "Y", "1":
				return reflect.ValueOf(true), nil
			case "false", "no", "n", "N", "0":
				return reflect.ValueOf(false), nil
			default:
				return nilV, errors.New("invalid bool [true/false]")
			}
		}
	}

	if fn == nil {
		return nil, errors.New("invalid type: " + t.String())
	}

	return fn, nil
}

func quickRet(v interface{}, err error, t reflect.Type) (reflect.Value, error) {
	if err != nil {
		return nilV, err
	}

	rv := reflect.ValueOf(v)

	if t == nil {
		return rv, nil
	}

	return rv.Convert(t), nil
}
