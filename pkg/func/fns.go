package funcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/jinzhu/copier"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
)

// 特殊的标记性错误，用于提示停止迭代
var StopIteration = errors.New("stop iteration")

// 获得一个对象的长度： 对于数组、切片、映射等容器，返回元素个数；
// 对于字符串，返回其长度，对于整数，返回其总位数
func Len(obj interface{}) int {
	if obj == nil {
		return 0
	}
	val := reflect.ValueOf(obj)
	if val.IsZero() {
		return 0
	}
	switch val.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return val.Len()
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return len(fmt.Sprintf("%d", cast.ToInt64(obj)))
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return len(fmt.Sprintf("%d", cast.ToUint64(obj)))
	default:
		return 0
	}
}

// 判断指定的所有对象是否均为空。nil、空字符串、元素个数为零的容器、false以及所有零值被看作是空的
func IsEmpty(objs ...interface{}) bool {
	for _, obj := range objs {
		if !funk.IsEmpty(obj) {
			return false
		}
	}
	return true
}

// 判断两个对象是否相等
func Equal(obj1, obj2 interface{}) bool {
	return funk.Equal(obj1, obj2)
}

// 判断对象obj是否包含元素elem
func Contains(obj, elem interface{}) bool {
	return funk.Contains(obj, elem)
}

// 将一个结构转换为映射
func ToMap(obj interface{}) map[string]interface{} {
	if obj == nil {
		return nil
	}
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return nil
	}
	ret := make(map[string]interface{})
	json.Unmarshal(objBytes, &ret)
	return ret
}

// 将对象from拷贝进to
func Copy(to, from interface{}, deep bool) error {
	return copier.CopyWithOption(to, from, copier.Option{
		IgnoreEmpty: false,
		DeepCopy:    deep,
		Converters:  nil,
	})
}

// 逻辑取反
func Not(value interface{}) bool {
	return !cast.ToBool(value)
}
