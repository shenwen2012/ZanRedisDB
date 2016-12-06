package node

import (
	"bytes"
	"errors"
	"github.com/absolute8511/ZanRedisDB/rockredis"
	"github.com/tidwall/redcon"
	"strconv"
	"strings"
)

var (
	errInvalidRange = errors.New("Invalid range string")
	errInvalidArgs  = errors.New("Invalid arguments")
)

func getScoreRange(left []byte, right []byte) (int64, int64, error) {
	if len(left) == 0 || len(right) == 0 {
		return 0, 0, errInvalidRange
	}
	var leftRange int64
	var rightRange int64
	var err error
	isLOpen := false
	rangeD := left
	if strings.ToLower(string(rangeD)) == "-inf" {
		leftRange = rockredis.MinScore
	} else {
		if left[0] == '(' {
			isLOpen = true
			rangeD = left[1:]
		}
		leftRange, err = strconv.ParseInt(string(rangeD), 10, 64)
		if err != nil {
			return leftRange, rightRange, err
		}
		if leftRange <= rockredis.MinScore || leftRange >= rockredis.MaxScore {
			return leftRange, rightRange, errInvalidRange
		}
		if isLOpen {
			leftRange++
		}
	}
	rangeD = right
	isROpen := false
	if strings.ToLower(string(rangeD)) == "+inf" {
		rightRange = rockredis.MaxScore
	} else {
		if right[0] == '(' {
			isROpen = true
			rangeD = right[1:]
		}
		rightRange, err = strconv.ParseInt(string(rangeD), 10, 64)
		if err != nil {
			return leftRange, rightRange, err
		}
		if rightRange <= rockredis.MinScore || rightRange >= rockredis.MaxScore {
			return leftRange, rightRange, errInvalidRange
		}
		if isROpen {
			rightRange--
		}

	}
	return leftRange, rightRange, err
}

func getLexRange(left []byte, right []byte) ([]byte, []byte, uint8, error) {
	if len(left) == 0 || len(right) == 0 {
		return nil, nil, 0, errInvalidRange
	}
	var err error
	rangeType := rockredis.RangeClose
	isLOpen := false
	if bytes.Equal(left, []byte("-")) {
		left = nil
	} else {
		if left[0] == '(' {
			isLOpen = true
			left = left[1:]
		} else if left[0] == '[' {
			isLOpen = false
			left = left[1:]
		} else {
			return left, right, rangeType, errInvalidRange
		}
	}
	isROpen := false
	if bytes.Equal(right, []byte("+")) {
		right = nil
	} else {
		if right[0] == '(' {
			isROpen = true
			right = right[1:]
		} else if right[0] == '[' {
			isROpen = false
			right = right[1:]
		} else {
			return left, right, rangeType, errInvalidRange
		}
	}
	if isLOpen && isROpen {
		rangeType = rockredis.RangeOpen
	} else if isROpen {
		rangeType = rockredis.RangeROpen
	} else if isLOpen {
		rangeType = rockredis.RangeLOpen
	}
	return left, right, rangeType, err
}

func (self *KVNode) zscoreCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	val, err := self.store.ZScore(cmd.Args[1], cmd.Args[2])
	if err != nil {
		conn.WriteNull()
	} else {
		conn.WriteBulk([]byte(strconv.FormatInt(val, 10)))
	}
}

func (self *KVNode) zcountCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	min, max, err := getScoreRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	val, err := self.store.ZCount(cmd.Args[1], min, max)
	if err != nil {
		conn.WriteError(err.Error())
	} else {
		conn.WriteInt64(val)
	}
}

func (self *KVNode) zcardCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	n, err := self.store.ZCard(cmd.Args[1])
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	conn.WriteInt64(n)
}

func (self *KVNode) zlexcountCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	start, stop, rt, err := getLexRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}
	n, err := self.store.ZLexCount(cmd.Args[1], start, stop, rt)
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	conn.WriteInt64(n)
}

func (self *KVNode) zrangeFunc(conn redcon.Conn, cmd redcon.Command, reverse bool) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	start, err := strconv.ParseInt(string(cmd.Args[2]), 10, 64)
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}
	end, err := strconv.ParseInt(string(cmd.Args[3]), 10, 64)
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}

	vlist, err := self.store.ZRangeGeneric(cmd.Args[1], int(start), int(end), reverse)
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	conn.WriteArray(len(vlist))
	for _, d := range vlist {
		conn.WriteBulk(d.Member)
	}
}

func (self *KVNode) zrangeCommand(conn redcon.Conn, cmd redcon.Command) {
	self.zrangeFunc(conn, cmd, false)
}

func (self *KVNode) zrevrangeCommand(conn redcon.Conn, cmd redcon.Command) {
	self.zrangeFunc(conn, cmd, true)
}

func (self *KVNode) zrangebylexCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 && len(cmd.Args) != 7 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	start, stop, rt, err := getLexRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}
	offset := 0
	count := -1
	if len(cmd.Args) == 7 {
		if strings.ToLower(string(cmd.Args[4])) != "limit" {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
		if offset, err = strconv.Atoi(string(cmd.Args[5])); err != nil {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
		if count, err = strconv.Atoi(string(cmd.Args[6])); err != nil {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
	}

	vlist, err := self.store.ZRangeByLex(cmd.Args[1], start, stop, rt, offset, count)
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	conn.WriteArray(len(vlist))
	for _, d := range vlist {
		conn.WriteBulk(d)
	}
}

func (self *KVNode) zrangebyscoreFunc(conn redcon.Conn, cmd redcon.Command, reverse bool) {
	if len(cmd.Args) < 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	var min int64
	var max int64
	var err error
	if !reverse {
		min, max, err = getScoreRange(cmd.Args[2], cmd.Args[3])
	} else {
		min, max, err = getScoreRange(cmd.Args[3], cmd.Args[2])
	}
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	offset := 0
	count := -1
	if len(cmd.Args) == 7 {
		if strings.ToLower(string(cmd.Args[4])) != "limit" {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
		if offset, err = strconv.Atoi(string(cmd.Args[5])); err != nil {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
		if count, err = strconv.Atoi(string(cmd.Args[6])); err != nil {
			conn.WriteError(errInvalidArgs.Error())
			return
		}
	}

	vlist, err := self.store.ZRangeByScoreGeneric(cmd.Args[1], min, max, offset, count, reverse)
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	conn.WriteArray(len(vlist))
	for _, d := range vlist {
		conn.WriteBulk(d.Member)
	}
}

func (self *KVNode) zrangebyscoreCommand(conn redcon.Conn, cmd redcon.Command) {
	self.zrangebyscoreFunc(conn, cmd, false)
}

func (self *KVNode) zrevrangebyscoreCommand(conn redcon.Conn, cmd redcon.Command) {
	self.zrangebyscoreFunc(conn, cmd, true)
}

func (self *KVNode) zrankCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	v, err := self.store.ZRank(cmd.Args[1], cmd.Args[2])
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	if v < 0 {
		conn.WriteNull()
	} else {
		conn.WriteInt64(v)
	}
}

func (self *KVNode) zrevrankCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	v, err := self.store.ZRevRank(cmd.Args[1], cmd.Args[2])
	if err != nil {
		conn.WriteError("Err: " + err.Error())
		return
	}
	if v < 0 {
		conn.WriteNull()
	} else {
		conn.WriteInt64(v)
	}
}

func (self *KVNode) zaddCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) < 4 || len(cmd.Args)%2 != 0 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	_, err := getScorePairs(cmd.Args[2:])
	if err != nil {
		conn.WriteError(err.Error())
		return
	}

	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func (self *KVNode) zincrbyCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	_, err := strconv.ParseInt(string(cmd.Args[2]), 10, 64)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}

	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteBulkString(strconv.FormatInt(rsp, 10))
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func (self *KVNode) zremCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) < 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func (self *KVNode) zremrangebyrankCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	_, err := strconv.ParseInt(string(cmd.Args[2]), 10, 64)
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}
	_, err = strconv.ParseInt(string(cmd.Args[3]), 10, 64)
	if err != nil {
		conn.WriteError("Invalid index: " + err.Error())
		return
	}

	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func (self *KVNode) zremrangebyscoreCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	_, _, err := getScoreRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}

}

func (self *KVNode) zremrangebylexCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 4 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	_, _, _, err := getLexRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func (self *KVNode) zclearCommand(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 2 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}
	v, err := self.Propose(cmd.Raw)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}
	rsp, ok := v.(int64)
	if ok {
		conn.WriteInt64(rsp)
	} else {
		conn.WriteError(errInvalidResponse.Error())
	}
}

func getScorePairs(args [][]byte) ([]rockredis.ScorePair, error) {
	mlist := make([]rockredis.ScorePair, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		s, err := strconv.ParseInt(string(args[i]), 10, 64)
		if err != nil {
			return nil, err
		}
		mlist = append(mlist, rockredis.ScorePair{Score: s, Member: args[i+1]})
	}

	return mlist, nil
}

func (self *KVNode) localZaddCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) < 4 || len(cmd.Args)%2 != 0 {
		return nil, errInvalidArgs
	}

	mlist, err := getScorePairs(cmd.Args[2:])
	if err != nil {
		return nil, err
	}
	v, err := self.store.ZAdd(cmd.Args[1], mlist...)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (self *KVNode) localZincrbyCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) != 4 {
		return nil, errInvalidArgs
	}

	delta, err := strconv.ParseInt(string(cmd.Args[2]), 10, 64)
	if err != nil {
		return nil, err
	}
	return self.store.ZIncrBy(cmd.Args[1], delta, cmd.Args[3])
}

func (self *KVNode) localZremCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) < 3 {
		return nil, errInvalidArgs
	}

	return self.store.ZRem(cmd.Args[1], cmd.Args[2:]...)
}

func (self *KVNode) localZremrangebyrankCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) != 4 {
		return nil, errInvalidArgs
	}
	start, err := strconv.ParseInt(string(cmd.Args[2]), 10, 64)
	if err != nil {
		return nil, err
	}
	stop, err := strconv.ParseInt(string(cmd.Args[3]), 10, 64)
	if err != nil {
		return nil, err
	}

	return self.store.ZRemRangeByRank(cmd.Args[1], int(start), int(stop))
}

func (self *KVNode) localZremrangebyscoreCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) != 4 {
		return nil, errInvalidArgs
	}

	min, max, err := getScoreRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		return nil, err
	}
	return self.store.ZRemRangeByScore(cmd.Args[1], min, max)
}

func (self *KVNode) localZremrangebylexCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) != 4 {
		return nil, errInvalidArgs
	}

	min, max, rt, err := getLexRange(cmd.Args[2], cmd.Args[3])
	if err != nil {
		return nil, err
	}
	return self.store.ZRemRangeByLex(cmd.Args[1], min, max, rt)
}

func (self *KVNode) localZclearCommand(cmd redcon.Command) (interface{}, error) {
	if len(cmd.Args) != 2 {
		return nil, errInvalidArgs
	}
	return self.store.ZClear(cmd.Args[1])
}