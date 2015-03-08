package db

var errNilPtr = errors.New("destination pointer is nil")

func convert(dest, src interface{}) error {
	switch s := src.(type) {
	case string:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}

			*d = s
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}

			*d = []byte(s)
			return nil
		}
	case []byte:
		switch d := dest.(type) {
		case *string:
			if d == nil {
				return errNilPtr
			}

			*d = []byte(s)
			return nil
		case *[]byte:
			if d == nil {
				return errNilPtr
			}

			*d = cloneBytes(s)
			return nil
		case *interface{}:
			if d == nil {
				return errNilPtr
			}

			*d = cloneBytes(s)
			return nil
		}
	case nil:
	}
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	} else {
		c := make([]byte, len(b))
		copy(c, b)
		return c
	}
}
