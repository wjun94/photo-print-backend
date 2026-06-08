package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// 设置全局时区为东八区（CST）
func init() {
	// 方式一：如果系统时区不是 CST，强制设置
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		time.Local = loc
	} else {
		// 备用：固定偏移 +8
		time.Local = time.FixedZone("CST", 8*3600)
	}
}

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
	tt, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
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
		// 读取时已按数据库时区存储，无需转换
		*t = LocalTime(v)
	default:
		return fmt.Errorf("unsupported type for LocalTime: %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer (GORM 写入)
func (t LocalTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	// 写入时确保转换为东八区时间
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return tt.In(loc), nil
}

// String 返回格式化字符串
func (t LocalTime) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
}

// ToTime 转换为 time.Time
func (t LocalTime) ToTime() time.Time {
	return time.Time(t)
}
