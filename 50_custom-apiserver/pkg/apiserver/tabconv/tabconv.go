package tabconv

import (
	"50_custom-apiserver/pkg/utils"
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/util/jsonpath"
	"reflect"
	"strings"
)

const (
	defaultItemsField = "Items"

	TypeInteger  = "integer"
	TypeLong     = "long"
	TypeFloat    = "float"
	TypeDouble   = "double"
	TypeString   = "string"
	TypeByte     = "byte"
	TypeBinary   = "binary"
	TypeBoolean  = "boolean"
	TypeDate     = "date"
	TypeDateTime = "dateTime"
	TypePassword = "password"
)

type TableConvertorBuilder func(gr schema.GroupResource) (rest.TableConvertor, error)

type TableConvertorFunc func(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error)

func (t TableConvertorFunc) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return t(ctx, object, tableOptions)
}

func NewFromTableConvertorFunc(t TableConvertorFunc) TableConvertorBuilder {
	return func(gr schema.GroupResource) (rest.TableConvertor, error) {
		return t, nil
	}
}

type TableConfig struct {
	ItemsField string
	Columns    []*TableColumnConfig
}

type TableColumnConfig struct {
	Name     string
	Type     string
	Format   string
	JsonPath string
	jsonpath *jsonpath.JSONPath
}

func NewFromColumnsConfig(cs []*TableColumnConfig) TableConvertorBuilder {
	return NewFromTableConfig(TableConfig{
		ItemsField: defaultItemsField,
		Columns:    cs,
	})
}

func NewFromTableConfig(t TableConfig) TableConvertorBuilder {
	if t.ItemsField == "" {
		t.ItemsField = defaultItemsField
	}
	for _, col := range t.Columns {
		path := jsonpath.New(col.Name)
		path.Parse(fmt.Sprintf("{ %s }", col.JsonPath))
		col.jsonpath = path
	}
	return NewFromTableConvertorFunc(func(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
		var objects []runtime.Object
		_, islist := object.(metav1.ListInterface)
		if islist {
			itemField := reflect.ValueOf(object).Elem().FieldByName(t.ItemsField)
			items := itemField.Interface()
			funk.ForEach(items, func(item interface{}) {
				var obj runtime.Object
				val := reflect.ValueOf(item)
				if val.Kind() == reflect.Ptr {
					obj = val.Interface().(runtime.Object)
				} else {
					obj = reflect.New(val.Type()).Interface().(runtime.Object)
					copier.Copy(obj, val.Interface())
				}
				objects = append(objects, obj)
			})
		} else {
			objects = append(objects, object.(runtime.Object))
		}
		table := &metav1.Table{}
		for _, col := range t.Columns {
			table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{
				Name:     col.Name,
				Type:     col.Type,
				Format:   col.Format,
				Priority: 0,
			})
		}
		for _, obj := range objects {
			var cells []interface{}
			for _, col := range t.Columns {
				var val interface{}
				if col.jsonpath != nil {
					results, err := col.jsonpath.FindResults(obj)
					if err == nil && len(results) > 0 {
						vals := results[0]
						if len(vals) > 0 {
							val = vals[0].Interface()
						}
					}
				}
				cells = append(cells, val)
			}
			table.Rows = append(table.Rows, metav1.TableRow{
				Cells: cells,
				Object: runtime.RawExtension{
					Raw:    nil,
					Object: obj,
				},
			})
		}
		var hanCounts [][]int
		for _, row := range table.Rows {
			var hanCount []int
			for _, cell := range row.Cells {
				hc := 0
				str, ok := cell.(string)
				if ok {
					hc = utils.HanCount(str)
				}
				hanCount = append(hanCount, hc)
			}
			hanCounts = append(hanCounts, hanCount)
		}
		if len(hanCounts) == 0 {
			return table, nil
		}
		cols := len(hanCounts[0])
		rows := len(hanCounts)
		var colMaxHanCounts []int
		for c := 0; c < cols; c++ {
			colMaxHanCount := 0
			headHanCount := utils.HanCount(table.ColumnDefinitions[c].Name)
			if headHanCount > colMaxHanCount {
				colMaxHanCount = headHanCount
			}
			for r := 0; r < rows; r++ {
				str, ok := table.Rows[r].Cells[c].(string)
				if ok {
					hanCount := utils.HanCount(str)
					if hanCount > colMaxHanCount {
						colMaxHanCount = hanCount
					}
				}
			}
			colMaxHanCounts = append(colMaxHanCounts, colMaxHanCount)
		}
		for c := 0; c < cols; c++ {
			colMaxHanCount := colMaxHanCounts[c]
			colHeader := table.ColumnDefinitions[c].Name
			headHanCount := utils.HanCount(colHeader)
			padding := colMaxHanCount - headHanCount
			const hanPadding = "　"
			if padding > 0 {
				table.ColumnDefinitions[c].Name = fmt.Sprintf("%s%s", colHeader, strings.Repeat(hanPadding, padding))
			}
			for r := 0; r < rows; r++ {
				str, ok := table.Rows[r].Cells[c].(string)
				if ok {
					hanCount := utils.HanCount(str)
					padding := colMaxHanCount - hanCount
					if padding > 0 {
						table.Rows[r].Cells[c] = fmt.Sprintf("%s%s", str, strings.Repeat(hanPadding, padding))
					}
				}
			}
			colMaxHanCounts = append(colMaxHanCounts, colMaxHanCount)
		}
		return table, nil
	})
}
