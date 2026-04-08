package codex

type Runner interface {
	Call(method string, payload any, out any) error
}

type ThreadRecord struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type AppServerClient struct {
	runner Runner
}

func NewAppServerClient(runner Runner) *AppServerClient {
	return &AppServerClient{runner: runner}
}

func (c *AppServerClient) ListThreads() ([]ThreadRecord, error) {
	var threads []ThreadRecord
	if err := c.runner.Call("thread/list", map[string]any{}, &threads); err != nil {
		return nil, err
	}
	return threads, nil
}
