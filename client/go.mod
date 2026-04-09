module code-agent-gateway/client

go 1.23

require (
	code-agent-gateway/common v0.0.0
	github.com/coder/websocket v1.8.14
	github.com/pelletier/go-toml/v2 v2.2.4
)

replace code-agent-gateway/common => ../common
