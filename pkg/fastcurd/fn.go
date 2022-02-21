package fastcurd

// 判断查询字段是否为合规的字符串 防注入
func IsValidQueryField(field string) bool {
	for _, c := range field {
		if (c < 'a' || c > 'z') && c != '_' {
			return false
		}
	}
	return true
}
