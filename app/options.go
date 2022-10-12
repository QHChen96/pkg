package app

type CliOptions interface {
	Flags()
	Validate() []error
}

type ConfigurableOptions interface {
	ApplyFlags() []error
}

type CompletableOptions interface {
	Complete() error
}

type PrintableOptions interface {
	String() string
}
