package utils

func IsValidGUID(guid string) bool {
	// 检查GUID是否符合标准格式
	return len(guid) == 36 &&
		guid[8] == '-' &&
		guid[13] == '-' &&
		guid[18] == '-' &&
		guid[23] == '-'
}
