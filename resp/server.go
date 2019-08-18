package resp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"
)

type CommandHandler func(cmd *CommandRequest) error

type Command struct {
	MinArgCount int
	MaxArgCount int
	Handler     CommandHandler
}

type Server struct {
	listener  net.Listener
	cmds      map[string]*Command
	clientIds int
	connWg    sync.WaitGroup
	running   bool
}

type cliConn struct {
	net.Conn
	id int
}

const stopCheckInterval = 500 * time.Millisecond

func NewServer(l net.Listener) *Server {
	s := &Server{
		connWg:   sync.WaitGroup{},
		listener: l,
		cmds:     map[string]*Command{},
		clientIds: 3, // mimic the real redis.
	}

	_ = s.RegisterCommand("QUIT", &Command{
		MinArgCount: 0,
		MaxArgCount: 0,
		Handler: func(_ *CommandRequest) error {
			return fmt.Errorf("just quit")
		},
	})
	return s
}

func (s *Server) RegisterCommand(name string, cmd *Command) error {
	name = strings.ToUpper(name)

	if _, ok := s.cmds[name]; ok {
		return fmt.Errorf("command %s already existed", name)
	}

	s.cmds[name] = cmd
	return nil
}

func (s *Server) serveConn(c net.Conn) {
	defer func() { _ = c.Close() }()

	s.connWg.Add(1)
	defer func() { s.connWg.Done() }()

	cc := &cliConn{
		Conn: c,
		id:   s.clientIds,
	}
	s.clientIds++
	cliLogger := log.WithField("clientId", cc.id)

	for s.running {
		err := c.SetReadDeadline(time.Now().Add(stopCheckInterval))
		if err != nil {
			cliLogger.WithError(err).Error("set connection read deadline failed")
			break
		}

		req, err := buildCommandRequest(cc)
		if err != nil {
			err, ok := err.(net.Error)
			if ok && err.Timeout() {
				continue
			}

			cliLogger.WithError(err).Error("build command request failed")
			break
		} else if req == nil {
			continue
		}

		cmd, ok := s.cmds[req.Name()]
		if !ok {
			err := req.WriteError(fmt.Sprintf("ERR unknown command `%s`", req.Name()))
			if err != nil {
				cliLogger.WithError(err).Error("write RESP error failed")
				break
			}

			goto flushAndContinue
		}

		if (cmd.MaxArgCount >= 0 && req.ArgCount() > cmd.MaxArgCount) ||
			(cmd.MinArgCount >= 0 && req.ArgCount() < cmd.MinArgCount) {
			err = req.WriteError(fmt.Sprintf("ERR wrong number of arguments for '%s' command", req.Name()))
		} else {
			err = cmd.Handler(req)
		}
		if err != nil {
			cliLogger.WithError(err).Error("handle command ", req.Name(), " failed")
			break
		}

	flushAndContinue:
		err = req.FlushWrites()
		if err != nil {
			cliLogger.WithError(err).Error("flush writes failed")
			break
		}

		continue
	}

	cliLogger.Info("client quit")
}

func (s *Server) GracefulStop() {
	if !s.running {
		return
	}

	s.running = false
	err := s.listener.Close()
	if err != nil {
		log.WithError(err).Fatal("close listener failed")
	}
	s.connWg.Wait()
}

func (s *Server) Serve() error {
	s.running = true

	for s.running {
		c, err := s.listener.Accept()
		if err != nil {
			log.WithError(err).Error("accept failed")

			if !s.running {
				return nil
			}

			err, ok := err.(net.Error)
			if ok && !err.Temporary() {
				return err
			}
		}

		go s.serveConn(c)
	}

	return nil
}
