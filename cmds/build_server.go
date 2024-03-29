package cmds

import (
	"github.com/abadcafe/raascs/resp"
	"net"
)

var cmds = make(map[string]*resp.Command)

func registerCommand(name string, cmd *resp.Command) {
	cmds[name] = cmd
}

func BuildRespServer(addr string) (*resp.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := resp.NewServer(listener)

	for name, handler := range cmds {
		err := server.RegisterCommand(name, handler)
		if err != nil {
			return nil, err
		}
	}

	return server, nil
}
