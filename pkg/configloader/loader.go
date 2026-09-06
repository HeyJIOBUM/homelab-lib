package configloader

type LoaderInterface interface {
	Name() string
	ReadValue(name string) (any, error)
	SupportsTag(tag string) bool
}

type Loader struct {
	loaders []LoaderInterface
}

type LoaderBuilder struct {
	loaders []LoaderInterface
}

func NewLoaderBuilder() *LoaderBuilder {
	return &LoaderBuilder{
		loaders: make([]LoaderInterface, 0),
	}
}

func (lb *LoaderBuilder) WithLoader(li LoaderInterface) *LoaderBuilder {
	lb.loaders = append(lb.loaders, li)
	return lb
}

func (lb *LoaderBuilder) Build() *Loader {
	return &Loader{
		loaders: lb.loaders,
	}
}

func (l *Loader) Load(map[string]any) error {
	return ErrInvalidStruct
}
