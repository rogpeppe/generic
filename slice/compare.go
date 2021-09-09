package slice

func Compare[T ordered](s1, s2 []T) int {
	return CompareFunc(s1, s2, func(x, y T) int {
		switch {
		case x == y:
			return 0
		case x < y:
			return -1
		}
		return 1
	})
}

func Less[T ordered](s1, s2 []T) bool {
	return Compare(s1, s2) < 0
}

func CompareFunc[T any](s1, s2 []T, cmp func(x, y T) int) int {
	for i := 0; i < len(s1) && i < len(s2); i++ {
		if c := cmp(s1[i], s2[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(s1) == len(s2):
		return 0
	case len(s1) < len(s2):
		return -1
	}
	return 1
}

type ordered interface {
	int | uint | string // etc
}
