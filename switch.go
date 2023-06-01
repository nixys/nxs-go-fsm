package fsm

import "bytes"

type Switch struct {
	Trigger    []byte
	Delimiters Delimiters
	Escape     bool
}

type Delimiters struct {
	L []byte
	R []byte
}

// index returns the index of the first instance
// of state trigger considering specified delimiters and escapes.
//
// If trigger is not present in buf, or delimiters and escapes conditions are false -1 will be returned.
//
// `prevEscs` it is a count of continuous escape bytes at the end of previous buffer
func (s Switch) index(buf, prevSrc []byte, prevEscs int, isEOF bool) int {

	for i := 0; i < len(buf); i++ {

		k := bytes.Index(buf[i:], s.Trigger)
		if k < 0 {
			return -1
		}
		i += k

		if len(s.Delimiters.L) > 0 {

			if b := func() bool {

				var c byte

				if i == 0 {

					if len(prevSrc) == 0 {
						return true
					}

					c = prevSrc[len(prevSrc)-1]
				} else {
					c = buf[i-1]
				}

				for _, d := range s.Delimiters.L {
					if c == d {
						return true
					}
				}

				return false
			}(); b == false {
				continue
			}
		}

		if len(s.Delimiters.R) > 0 {

			if b := func() bool {

				if i+len(s.Trigger) == len(buf) {
					if isEOF == true {
						return true
					} else {
						return false
					}
				}

				c := buf[i+len(s.Trigger)]
				for _, d := range s.Delimiters.R {
					if c == d {
						return true
					}
				}
				return false
			}(); b == false {
				continue
			}
		}

		if s.Escape == true {

			// Check if found sequence is escaped (has a leading symbol from escape set)
			// True when it has
			if b := func() bool {

				ec := escapesCount(buf[:i])
				if len(buf[:i])-ec == 0 {
					// Escapes is from previous position till beginning of buffer
					ec += prevEscs
				}

				if ec%2 == 0 {
					return false
				}
				return true
			}(); b == true {
				continue
			}
		}

		return i
	}

	return -1
}
