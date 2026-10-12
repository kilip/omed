package core

type Module interface {
	Boot(state State) error
}

type HasPreBoot interface {
	PreBoot(state State) error
}
