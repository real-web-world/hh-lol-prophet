package bdk

import (
	"bytes"
	"fmt"
	"time"
)

type (
	DayTime  time.Time // 2020-01-02
	DateTime time.Time // 2020-01-02 15:04:05
)

const (
	DayTimeFmt  = "2006-01-02"
	DateTimeFmt = "2006-01-02 15:04:05"
)

func (t *DayTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Time(*t).Format(DayTimeFmt))), nil
}
func (t *DayTime) String() string {
	return time.Time(*t).Format(DayTimeFmt)
}
func (t *DayTime) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, "\"")
	ext, err := time.ParseInLocation(DayTimeFmt, string(b), time.Local)
	if err != nil {
		return err
	}
	*t = DayTime(ext)
	return nil
}
func (t *DateTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Time(*t).Format(DateTimeFmt))), nil
}
func (t *DateTime) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, "\"")
	ext, err := time.ParseInLocation(DateTimeFmt, string(b), time.Local)
	if err != nil {
		return err
	}
	*t = DateTime(ext)
	return nil
}
func (t *DateTime) String() string {
	return time.Time(*t).Format(DateTimeFmt)
}
