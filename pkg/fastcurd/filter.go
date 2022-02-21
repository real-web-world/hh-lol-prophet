package fastcurd

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type (
	Filter     = map[string]FilterItem
	FilterCond string
	FilterItem struct {
		Condition FilterCond
		Val       interface{}
	}
)

const (
	// 筛选条件
	CondUndefined FilterCond = "undefined"
	// 数值
	CondEq           FilterCond = "eq"
	CondLt           FilterCond = "lt"
	CondElt          FilterCond = "elt"
	CondGt           FilterCond = "gt"
	CondEgt          FilterCond = "egt"
	CondNeq          FilterCond = "neq"
	CondBetweenValue FilterCond = "betweenValue"
	// 字符串
	CondEqString  FilterCond = "eqString"
	CondLike      FilterCond = "like"
	CondNotLike   FilterCond = "notLike"
	CondNeqString FilterCond = "neqString"
	// 时间
	CondBefore      FilterCond = "before"
	CondAfter       FilterCond = "after"
	CondBetweenTime FilterCond = "betweenTime"
	// 数组
	CondIn    FilterCond = "in"
	CondNotIn FilterCond = "notIn"
	// 内部使用
	CondRaw FilterCond = "raw"

	// order
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

var (
	// CondMapDbCond 条件映射 数据库条件
	CondMapDbCond = map[FilterCond]string{
		CondEq:           "=",
		CondLt:           "<",
		CondElt:          "<=",
		CondGt:           ">",
		CondEgt:          ">=",
		CondNeq:          "<>",
		CondBetweenValue: "BETWEEN",
		CondEqString:     "=",
		CondLike:         "LIKE",
		CondNotLike:      "NOT LIKE",
		CondNeqString:    "<>",
		CondBefore:       "<",
		CondAfter:        ">",
		CondBetweenTime:  "BETWEEN",
		CondIn:           "IN",
		CondNotIn:        "NOT IN",
	}
)

func FmtCondVal(cond FilterCond, val interface{}) interface{} {
	switch cond {
	case CondLike, CondNotLike:
		if val, ok := val.(string); !ok {
			panic("筛选条件为" + cond + "时,val必须为字符串")
		} else {
			return "%" + val + "%"
		}
	case CondBetweenValue, CondBetweenTime:
		switch val.(type) {
		case []int:
			return []int{val.([]int)[0], val.([]int)[1]}
		case []string:
			return []string{val.([]string)[0], val.([]string)[1]}
		case []time.Time:
			location := time.FixedZone("UTC", 8*3600)
			return []string{val.([]time.Time)[0].In(location).Format("2006-01-02 15:04:05"),
				val.([]time.Time)[1].In(location).Format("2006-01-02 15:04:05")}
		case []interface{}:
			return val
		default:
			panic("筛选条件为" + cond + "时,val必须为数组")
		}
	default:
		return val
	}
}
func FmtValPlaceholder(cond FilterCond) interface{} {
	switch cond {
	case CondIn, CondNotIn:
		return "(?)"
	case CondBetweenTime, CondBetweenValue:
		return "? and ?"
	default:
		return "?"
	}
}
func BuildFilterCond(filterMap map[string]string, db *gorm.DB, filter Filter) *gorm.DB {
	for filterKey, filterItem := range filter {
		if dbField, ok := filterMap[filterKey]; (ok || filterItem.Condition ==
			CondRaw) && filterItem.Condition != CondUndefined && filterItem.Val != nil {
			switch filterItem.Condition {
			case CondLike, CondNotLike:
				dbFieldList := strings.Split(dbField, "|")
				sql := ""
				actValArr := make([]interface{}, 0, 1)
				for _, field := range dbFieldList {
					if !IsValidQueryField(field) {
						continue
					}
					actCondition := CondMapDbCond[filterItem.Condition]
					actVal := FmtCondVal(filterItem.Condition, filterItem.Val)
					valPlaceholder := FmtValPlaceholder(filterItem.Condition)
					if arrVal, ok := actVal.([]string); ok {
						sql += fmt.Sprintf("%s %s %s", field, actCondition, valPlaceholder)
						actValArr = append(actValArr, arrVal[0], arrVal[1])
					} else {
						sql += fmt.Sprintf("%s %s %s", field, actCondition, valPlaceholder)
						actValArr = append(actValArr, actVal)
					}
					sql += " or "
				}
				sql = sql[:len(sql)-4]
				db = db.Where(sql, actValArr...)
			case CondRaw:
				rawSQLData := filterItem.Val.([]interface{})
				db = db.Where(rawSQLData[0].(string), rawSQLData[1].([]interface{})...)
			default:
				if !IsValidQueryField(dbField) {
					continue
				}
				actCondition := CondMapDbCond[filterItem.Condition]
				actVal := FmtCondVal(filterItem.Condition, filterItem.Val)
				valPlaceholder := FmtValPlaceholder(filterItem.Condition)
				if arrVal, ok := actVal.([]string); ok {
					db = db.Where(fmt.Sprintf("%s %s %s", dbField, actCondition, valPlaceholder),
						arrVal[0], arrVal[1])
				} else if arrVal, ok := actVal.([]interface{}); ok && len(arrVal) == 2 {
					db = db.Where(fmt.Sprintf("%s %s %s", dbField, actCondition, valPlaceholder),
						arrVal[0], arrVal[1])
				} else {
					db = db.Where(fmt.Sprintf("%s %s %s", dbField, actCondition, valPlaceholder), actVal)
				}
			}
		}
	}
	return db
}
func BuildOrderCond(orderKeyMap map[string]string, q *gorm.DB, order map[string]string) *gorm.DB {
	for orderKey, orderVal := range order {
		if actKey, ok := orderKeyMap[orderKey]; ok {
			if orderVal == OrderDesc {
				q = q.Order(actKey + " desc")
			} else {
				q = q.Order(actKey + " asc")
			}
		}
	}
	return q
}
