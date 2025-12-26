// utils/escape.go
package utils

import (
	"strings"
	"unicode"
)

// EscapeIdentifier 安全地转义SQL Server标识符
// 处理：移除现有方括号，转义内部右方括号，然后重新包裹
func EscapeIdentifier(identifier string) string {
	// 如果标识符为空，直接返回
	if identifier == "" {
		return ""
	}

	// 去除两端的空白字符
	identifier = strings.TrimSpace(identifier)

	// 如果已经是方括号包裹，先去掉包裹
	if isBracketedIdentifier(identifier) {
		identifier = identifier[1 : len(identifier)-1]
	}

	// 转义内部的双引号和右方括号
	// 在SQL Server中，] 需要转义为 ]]
	escaped := strings.ReplaceAll(identifier, "]", "]]")

	// 重新用方括号包裹
	return "[" + escaped + "]"
}

// EscapeQualifiedName 转义限定名（如 schema.table）
// 支持格式: table, schema.table, [schema].[table]
func EscapeQualifiedName(name string) (string, error) {
	if name == "" {
		return "", nil
	}

	name = strings.TrimSpace(name)

	// 分割为各个部分
	parts, err := parseQualifiedName(name)
	if err != nil {
		return "", err
	}

	// 转义每个部分
	escapedParts := make([]string, len(parts))
	for i, part := range parts {
		escapedParts[i] = EscapeIdentifier(part)
	}

	// 重新组合
	return strings.Join(escapedParts, "."), nil
}

// parseQualifiedName 解析限定名
// 处理: table, schema.table, [schema].[table], [schema.table]
func parseQualifiedName(name string) ([]string, error) {
	if name == "" {
		return nil, nil
	}

	var parts []string
	var current strings.Builder
	inBrackets := false

	for i, r := range name {
		switch r {
		case '[':
			if inBrackets {
				// [[ 视为转义的左方括号
				current.WriteRune('[')
			} else {
				inBrackets = true
			}
		case ']':
			if inBrackets {
				// 检查下一个字符是否是 ]
				if i+1 < len(name) && name[i+1] == ']' {
					// ]] 视为转义的右方括号
					current.WriteRune(']')
					// 跳过下一个字符
					continue
				}
				inBrackets = false
				// 方括号结束，保存当前部分
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				return nil, newEscapeError("非转义右方括号 ']' 出现在非法位置")
			}
		case '.':
			if inBrackets {
				// 在方括号内，点号是标识符的一部分
				current.WriteRune('.')
			} else {
				// 在方括号外，点号是分隔符
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			}
		default:
			if !inBrackets && !isValidIdentifierRune(r, current.Len() == 0) {
				return nil, newEscapeError("标识符包含无效字符")
			}
			current.WriteRune(r)
		}
	}

	// 添加最后一个部分
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	// 检查方括号是否匹配
	if inBrackets {
		return nil, newEscapeError("未匹配的方括号")
	}

	// 验证部分数量
	if len(parts) > 4 {
		return nil, newEscapeError("标识符包含太多部分 (最大支持4部分: server.database.schema.object)")
	}

	return parts, nil
}

// isValidIdentifierRune 检查字符是否可以用作标识符的一部分
func isValidIdentifierRune(r rune, isFirst bool) bool {
	if isFirst {
		// 首字符必须是字母、_、@、# 或 Unicode 字母
		return unicode.IsLetter(r) || r == '_' || r == '@' || r == '#'
	}
	// 后续字符可以是字母、数字、_、@、$ 或 #
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '@' || r == '$' || r == '#'
}

// isBracketedIdentifier 检查标识符是否已经被方括号包裹
func isBracketedIdentifier(s string) bool {
	return len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']'
}

// EscapeError 转义错误
type EscapeError struct {
	Message string
	Input   string
}

func newEscapeError(msg string) *EscapeError {
	return &EscapeError{Message: msg}
}

func (e *EscapeError) Error() string {
	return e.Message
}
