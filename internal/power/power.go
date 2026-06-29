package power

type Manager interface {
	Acquire() error
	Release() error
}
