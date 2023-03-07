package structures

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](ts ...T) Set[T] {
	s := make(Set[T], len(ts))
	for _, t := range ts {
		s[t] = struct{}{}
	}
	return s
}

func (s Set[T]) Add(t T) { s[t] = struct{}{} }

func (s Set[T]) Remove(t T) { delete(s, t) }
