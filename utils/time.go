package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// LocalTime 自定义时间类型，JSON 序列化为 "2006-01-02 15:04:05" 格式
type LocalTime time.Time

// MarshalJSON 实现 json.Marshaler
func (t LocalTime) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05"))), nil
}

// UnmarshalJSON 实现 json.Unmarshaler
func (t *LocalTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" || s == `""` {
		*t = LocalTime(time.Time{})
		return nil
	}
	// 去掉引号
	s = s[1 : len(s)-1]
	tt, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*t = LocalTime(tt)
	return nil
}

// Scan 实现 sql.Scanner (GORM 读取)
func (t *LocalTime) Scan(value interface{}) error {
	if value == nil {
		*t = LocalTime(time.Time{})
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = LocalTime(v)
	default:
		return fmt.Errorf("unsupported type for LocalTime: %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer (GORM 写入)
func (t LocalTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// String 返回格式化字符串
func (t LocalTime) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
}

// ToTime 转换为 time.Time
func (t LocalTime) ToTime() time.Time {
	return time.Time(t)
}
