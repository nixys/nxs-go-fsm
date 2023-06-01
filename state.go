package fsm

type State struct {
	NextStates []NextState
}

type StateName string

type NextState struct {
	Name        StateName
	Switch      Switch
	DataHandler func(any, []byte, []byte) ([]byte, error)
}

// index returns the minimal index within the available triggers for
// current state or -1 if any triggers were found or delimiters/escapes
// conditions was false
func (s State) index(buf, prevSrc []byte, prevEscs int, isEOF bool) (int, NextState) {

	type NextStatesMin struct {
		i  int
		ns NextState
	}

	nsMin := NextStatesMin{
		i:  -1,
		ns: NextState{},
	}

	for _, ns := range s.NextStates {

		i := ns.Switch.index(buf, prevSrc, prevEscs, isEOF)
		if i >= 0 {
			if nsMin.i == -1 || (nsMin.i > 0 && i < nsMin.i) {
				nsMin.i = i
				nsMin.ns = ns
			}
		}
	}

	return nsMin.i, nsMin.ns
}

func (s State) skipMaxLen() int {

	ll := 0

	for _, ns := range s.NextStates {

		l := len(ns.Switch.Trigger)

		if len(ns.Switch.Delimiters.L) > 0 {
			l++
		}
		if len(ns.Switch.Delimiters.R) > 0 {
			l++
		}

		if l > ll {
			ll = l
		}
	}

	return ll
}
