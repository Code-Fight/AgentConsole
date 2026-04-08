package codex

import "testing"

type fakeRunner struct{}

func (fakeRunner) Call(method string, payload any, out any) error {
	if method == "thread/list" {
		threads := out.(*[]ThreadRecord)
		*threads = []ThreadRecord{{ID: "thread-1", Title: "Investigate flaky test"}}
	}
	return nil
}

func TestClientListsThreadsThroughRunner(t *testing.T) {
	client := NewAppServerClient(fakeRunner{})
	threads, err := client.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ID != "thread-1" {
		t.Fatalf("unexpected threads: %+v", threads)
	}
}
