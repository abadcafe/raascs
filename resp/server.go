package resp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"
)

type CommandHandler func(cmd Command) error

type Server struct {
	listener    net.Listener
	cmdHandlers map[string]CommandHandler
	clientIds   int
	connWg      sync.WaitGroup
	running     bool
}

type cliConn struct {
	net.Conn
	id int
}

const stopCheckInterval = 500 * time.Millisecond

func NewServer(l net.Listener) *Server {
	s := &Server{
		connWg:      sync.WaitGroup{},
		listener:    l,
		cmdHandlers: map[string]CommandHandler{},
	}

	_ = s.RegisterCommand("QUIT", func(cmd Command) error {
		return fmt.Errorf("just quit")
	})
	return s
}

func (s *Server) RegisterCommand(name string, h CommandHandler) error {
	name = strings.ToUpper(name)

	if _, ok := s.cmdHandlers[name]; ok {
		return fmt.Errorf("command %s already existed", name)
	}

	s.cmdHandlers[name] = h
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

	for s.running {
		err := c.SetReadDeadline(time.Now().Add(stopCheckInterval))
		if err != nil {
			log.WithError(err).Error("set connection read deadline failed")
			break
		}

		cmd, err := readCommand(cc)
		if err != nil {
			if err, ok := err.(net.Error); ok {
				if err.Timeout() {
					continue
				}
			}

			log.WithError(err).Error("read command failed")
			break
		} else if cmd == nil {
			continue
		}

		handler, ok := s.cmdHandlers[cmd.Name()]
		if !ok {
			err := cmd.WriteError(fmt.Sprintf("ERR unknown command `%s`", cmd.Name()))
			if err != nil {
				log.WithError(err).Error("write RESP error failed")
				break
			}

			goto flushAndContinue
		}

		err = handler(cmd)
		if err != nil {
			log.WithError(err).Error("handle command", cmd.Name(), "failed")
			break
		}

	flushAndContinue:
		err = cmd.FlushWrites()
		if err != nil {
			log.WithError(err).Error("flush writes failed")
			break
		}

		continue
	}

	log.Info("deal with client", cc.id, "finished")
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
		if err, ok := err.(net.Error); ok {
			if !s.running {
				return nil
			}

			log.WithError(err).Error("accept failed")
			if !err.Temporary() {
				return err
			}
		}

		go s.serveConn(c)
	}

	return nil
}
