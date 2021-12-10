package manager

type Manager interface {
	Name() string
	Run(stop <-chan struct{})
}
