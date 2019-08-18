package cmds

import (
	"github.com/abadcafe/raascs/resp"
	"strconv"
	"time"
)

func init() {
	registerCommand("SET", cmdSet)
	registerCommand("GET", cmdGet)
}

func cmdSet(cmd *resp.Command) error {
	if cmd.ArgCount() < 2 {
		return cmd.WriteError("ERR wrong number of arguments for 'set' command")
	}

	args, err := cmd.ReadArg(2)
	if err != nil {
		return err
	}

	name := args[0]
	value := args[1]

	nxxxOccurred := false
	expxOccurred := false
	var nx bool
	var xx bool
	var ttl time.Duration
	var flags map[string]*resp.CommandFlag

	if cmd.ArgCount() <= 0 {
		goto exit
	}

	flags = map[string]*resp.CommandFlag{
		"NX": {
			ExclusiveFlag: &nxxxOccurred,
			Receiver: func(s string) error {
				nx = true
				return nil
			},
		},
		"XX": {
			ExclusiveFlag: &nxxxOccurred,
			Receiver: func(s string) error {
				xx = true
				return nil
			},
		},
		"EX": {
			NeedValue: true,
			ExclusiveFlag: &expxOccurred,
			Receiver: func(s string) error {
				seconds, err := strconv.Atoi(s)
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
			Receiver: func(s string) error {
				ms, err := strconv.Atoi(s)
				if err != nil {
					return err
				}

				ttl = time.Duration(ms) * time.Millisecond
				return nil
			},
		},
	}

	err = cmd.ParseArgs(flags)
	if err != nil {
		return cmd.WriteError("ERR syntax error")
	}

	if nx {
		_, exist := globalMap.Load(name)
		if exist {
			return cmd.WriteNullBulkString()
		}
	}

	if xx {
		_, exist := globalMap.Load(name)
		if !exist {
			return cmd.WriteNullBulkString()
		}
	}

exit:
	if expxOccurred {
		globalMap.StoreWithTtl(name, value, ttl)
	} else {
		globalMap.Store(name, value)
	}
	return cmd.WriteSimpleString("OK")
}

func cmdGet(cmd *resp.Command) error {
	if cmd.ArgCount() > 1 {
		return cmd.WriteError("ERR wrong number of arguments for 'get' command")
	}

	args, err := cmd.ReadArg(1)
	if err != nil {
		return err
	}

	value, existed := globalMap.Load(args[0])
	if !existed {
		return cmd.WriteNullBulkString()
	}

	return cmd.WriteBulkString(value.(string))
}
