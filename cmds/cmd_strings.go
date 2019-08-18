package cmds

import (
	"github.com/abadcafe/raascs/resp"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func init() {
	registerCommand("SET", &resp.Command{
		MaxArgCount: -1,
		MinArgCount: 2,
		Handler: cmdSet,
	})
	registerCommand("GET", &resp.Command{
		MinArgCount: 1,
		MaxArgCount: 1,
		Handler:     cmdGet,
	})
}

func cmdSet(req *resp.CommandRequest) error {
	args, err := req.ReadArg(2)
	if err != nil {
		return err
	}

	name := string(args[0])
	value := args[1]

	var (
		nxxxOccurred bool
		expxOccurred bool
		nx    bool
		xx    bool
		ttl   time.Duration
		flags map[string]*resp.CommandFlag
	)

	if req.ArgCount() <= 0 {
		goto exit
	}

	flags = map[string]*resp.CommandFlag{
		"NX": {
			ExclusiveFlag: &nxxxOccurred,
			ValueReceiver: func(s []byte) error {
				nx = true
				return nil
			},
		},
		"XX": {
			ExclusiveFlag: &nxxxOccurred,
			ValueReceiver: func(s []byte) error {
				xx = true
				return nil
			},
		},
		"EX": {
			NeedValue: true,
			ExclusiveFlag: &expxOccurred,
			ValueReceiver: func(s []byte) error {
				seconds, err := strconv.Atoi(string(s))
				if err != nil {
					return err
				}

				ttl = time.Duration(seconds) * time.Second
				return nil
			},
		},
		"PX": {
			NeedValue: true,
			ExclusiveFlag: &expxOccurred,
			ValueReceiver: func(s []byte) error {
				ms, err := strconv.Atoi(string(s))
				if err != nil {
					return err
				}

				ttl = time.Duration(ms) * time.Millisecond
				return nil
			},
		},
	}

	err = req.ParseFlags(flags)
	if err != nil {
		log.WithError(err).WithField("command", req.Name()).Info("parse flags failed")
		return req.WriteError("ERR syntax error")
	}

	if nx {
		_, exist := globalMap.Load(name)
		if exist {
			return req.WriteNullBulkString()
		}
	}

	if xx {
		_, exist := globalMap.Load(name)
		if !exist {
			return req.WriteNullBulkString()
		}
	}

exit:
	if expxOccurred {
		globalMap.StoreWithTtl(name, value, ttl)
	} else {
		globalMap.Store(name, value)
	}

	return req.WriteSimpleString("OK")
}

func cmdGet(req *resp.CommandRequest) error {
	args, err := req.ReadArg(1)
	if err != nil {
		return err
	}

	value, existed := globalMap.Load(string(args[0]))
	if !existed {
		return req.WriteNullBulkString()
	}

	return req.WriteBulkString(value.([]byte))
}
