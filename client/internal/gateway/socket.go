package gateway

import (
	"context"

	cws "github.com/coder/websocket"
)

func Dial(ctx context.Context, url string) (*cws.Conn, error) {
	conn, _, err := cws.Dial(ctx, url, nil)
	return conn, err
}
