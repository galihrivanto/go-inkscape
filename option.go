package inkscape

// Option define method to modify config options
type Option func(o *Options)

// Options of configuration package
type Options struct {
	// command name, by default is "inkscape"
	// but it may depends on system setup
	// therefore allow user to override if needed
	commandName string

	// maximum retry attempt
	maxRetry int

	// maximum command queue size
	commandQueueLength int

	// set verbosity
	verbose bool

	// allow to suppress warning
	suppressWarning bool
}

// CommandName customize inkscape executable name
// this may vary based on system setup / configuration
func CommandName(commandName string) Option {
	return func(o *Options) {
		o.commandName = commandName
	}
}

// MaxRetry override maximum retry attempt when running
// inkscape background process
func MaxRetry(retry int) Option {
	return func(o *Options) {
		o.maxRetry = retry
	}
}

// CommandQueueLength override maximum command queue size
func CommandQueueLength(length int) Option {
	return func(o *Options) {
		o.commandQueueLength = length
	}
}

// SuppressWarning override default suppress warning option, that are enabled
func SuppressWarning(suppress bool) Option {
	return func(o *Options) {
		o.suppressWarning = suppress
	}
}

// Verbose override log verbosity
// useful for debugging
func Verbose(verbose bool) Option {
	return func(o *Options) {
		o.verbose = verbose
	}
}

func mergeOptions(dest Options, opts ...Option) Options {
	for _, opt := range opts {
		opt(&dest)
	}

	return dest
}
