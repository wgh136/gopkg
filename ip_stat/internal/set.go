package internal

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		m: make(map[T]struct{}),
	}
}

func (s *Set[T]) Add(v T) {
	s.m[v] = struct{}{}
}

func (s *Set[T]) Remove(v T) {
	delete(s.m, v)
}

func (s *Set[T]) Contains(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) Size() int {
	return len(s.m)
}

func (s *Set[T]) Clear() {
	s.m = make(map[T]struct{})
}

func (s *Set[T]) Values() []T {
	values := make([]T, 0, len(s.m))
	for v := range s.m {
		values = append(values, v)
	}
	return values
}

func (s *Set[T]) IsEmpty() bool {
	return len(s.m) == 0
}

func (s *Set[T]) Range(f func(T) bool) {
	for v := range s.m {
		if !f(v) {
			break
		}
	}
}
