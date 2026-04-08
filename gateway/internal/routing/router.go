package routing

type Router struct {
	threads map[string]string
}

func NewRouter() *Router {
	return &Router{threads: map[string]string{}}
}

func (r *Router) TrackThread(threadID string, machineID string) {
	r.threads[threadID] = machineID
}

func (r *Router) ResolveThread(threadID string) (string, bool) {
	machineID, ok := r.threads[threadID]
	return machineID, ok
}
