package heuristics

// ZeroGuesses guess zero value for every type, which
// is the 2nd element of an array, based on its name.
// It is also given precomputed zeroes slice where
// some elements may already be computed using more
// reliable methods. These should not be replaced.
//
// Return nil slice only if kind "unclear" detected on any type.
func ZeroGuesses(ps [][2]string, zeroes []string) []string {
	var res []string
	if len(zeroes) == 0 {
		zeroes = make([]string, len(ps))
	}

	for i, p := range ps {
		if zeroes[i] != "" {
			continue
		}

		k := guessKind(p[1])
		switch k {
		case unclear:
			return nil
		case errortype:
			if i != len(ps)-1 {
				res = append(res, "nil")
				continue
			}
		}

		res = append(res, kindZero(p[1], k))
	}

	return res
}
