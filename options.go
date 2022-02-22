package hh_lol_prophet

type ApplyOption func(o *options)

func WithEnablePprof(enablePprof bool) ApplyOption {
	return func(o *options) {
		o.enablePprof = enablePprof
	}
}
func WithHttpAddr(httpAddr string) ApplyOption {
	return func(o *options) {
		o.httpAddr = httpAddr
	}
}
func WithDebug() ApplyOption {
	return func(o *options) {
		o.debug = true
	}
}
func WithProd() ApplyOption {
	return func(o *options) {
		o.debug = false
	}
}
