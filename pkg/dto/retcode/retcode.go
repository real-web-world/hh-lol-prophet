package retcode

const (
	Ok = iota
	DefaultError
	NoAuth
	BadReq
	ValidError
	NoLogin
	ServerError
	RateLimitError
)
