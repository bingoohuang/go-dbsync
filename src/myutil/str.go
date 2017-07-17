package myutil

import (
	"strconv"
	"strings"
)

func RowToString(cols []string, m map[string]string) string {
	mystr := MyStr{}
	mystr.PS("{")
	for _, col := range cols {
		val, ok := m[col]
		if ok {
			mystr.PKV(col, val)
			delete(m, col)
		}
	}

	return mystr.PM(m).PS("}").Str()
}

type LogItem struct {
	LogName string
	LogFile string
}

func ParseLogItems(log string) []LogItem {
	logItems := SplitTrim(log, ",")

	result := make([]LogItem, 0)
	for i, logItem := range logItems {
		kvs := SplitTrim(logItem, ":")

		logName := "log" + strconv.Itoa(i)
		logFile := kvs[0]

		if len(kvs) >= 2 {
			logName = kvs[0]
			logFile = kvs[1]
		}

		item := LogItem{
			logName,
			logFile,
		}

		result = append(result, item)
	}

	return result
}

func FindLogItem(logItems []LogItem, logName string) *LogItem {
	for _, v := range logItems {
		if v.LogName == logName {
			return &v
		}
	}

	return nil
}

func SplitTrim(str, sep string) []string {
	subs := strings.Split(str, sep)
	ret := make([]string, 0)
	for i, v := range subs {
		v := strings.TrimSpace(v)
		if len(subs[i]) > 0 {
			ret = append(ret, v)
		}
	}

	return ret
}

func HexString(val int64) string {
	return strconv.FormatInt(val, 16)
}

func ParseHex(val string) (int64, error) {
	return strconv.ParseInt(val, 16, 64)
}

func ContainsAny(str string, sub []string) bool {
	if len(sub) == 0 {
		return true
	}

	for _, v := range sub {
		if strings.Contains(str, v) {
			return true
		}
	}

	return false
}

func StartWithBlank(str string) bool {
	if str != "" {
		return false
	}

	ch := str[0]
	return ch == ' ' || ch == '\t'
}
