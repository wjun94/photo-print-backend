package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

// Int64Str 用于 JSON 序列化时自动转为字符串，数据库存储仍为 int64
type Int64Str int64

// MarshalJSON 实现 json.Marshaler
func (i Int64Str) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(i), 10))
}

// UnmarshalJSON 实现 json.Unmarshaler
func (i *Int64Str) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*i = Int64Str(val)
	return nil
}

// Scan 实现 sql.Scanner（GORM 从数据库读取）
func (i *Int64Str) Scan(value interface{}) error {
	if value == nil {
		*i = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*i = Int64Str(v)
	case []byte:
		val, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return err
		}
		*i = Int64Str(val)
	default:
		return fmt.Errorf("unsupported type for Int64Str: %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer（GORM 写入数据库）
func (i Int64Str) Value() (driver.Value, error) {
	return int64(i), nil
}

// ToInt64 辅助函数
func (i Int64Str) Int64() int64 {
	return int64(i)
}

// String 返回字符串形式
func (i Int64Str) String() string {
	return strconv.FormatInt(int64(i), 10)
}
