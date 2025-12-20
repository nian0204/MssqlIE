// exporter/export.go
package exporter

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mssql_ie/config"
	"github.com/mssql_ie/utils"
)

// TableToCSV 将指定表的数据导出到CSV文件
func TableToCSV(db *sql.DB, cfg config.ExportConfig) error {
	if cfg.Table == "" {
		return fmt.Errorf("表名不能为空")
	}
	if cfg.CSVPath == "" {
		return fmt.Errorf("CSV文件路径不能为空")
	}

	// 安全地转义表名
	escapedTable, err := utils.EscapeQualifiedName(cfg.Table)
	if err != nil {
		return fmt.Errorf("无效的表名格式: %w", err)
	}

	// 构建查询
	var query string
	if cfg.Limit > 0 {
		query = fmt.Sprintf("SELECT * FROM %s", escapedTable)
		// 添加TOP限制
		query = fmt.Sprintf("SELECT TOP %d * FROM %s", cfg.Limit, escapedTable)
	} else {
		query = fmt.Sprintf("SELECT * FROM %s", escapedTable)
	}

	return exportQueryResultToCSV(db, query, cfg)
}

// SQLToCSV 执行自定义SQL并将结果导出到CSV文件
func SQLToCSV(db *sql.DB, cfg config.ExportConfig) error {
	if cfg.SQL == "" {
		return fmt.Errorf("SQL语句不能为空")
	}
	if cfg.CSVPath == "" {
		return fmt.Errorf("CSV文件路径不能为空")
	}

	return exportQueryResultToCSV(db, cfg.SQL, cfg)
}

// exportQueryResultToCSV 通用导出逻辑
func exportQueryResultToCSV(db *sql.DB, query string, cfg config.ExportConfig) error {
	// 添加WITH (NOLOCK) 提示以避免锁定
	if cfg.Table != "" && !strings.Contains(strings.ToUpper(query), "WITH (NOLOCK)") {
		query = strings.TrimSuffix(query, ";")
		query += " WITH (NOLOCK)"
	}

	// 执行查询
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	// 获取列名
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("获取列名失败: %w", err)
	}

	// 创建CSV文件
	file, err := os.Create(cfg.CSVPath)
	if err != nil {
		return fmt.Errorf("创建CSV文件失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = cfg.Delimiter
	defer writer.Flush()

	// 写入列标题
	if cfg.Header {
		if err := writer.Write(cols); err != nil {
			return fmt.Errorf("写入列名失败: %w", err)
		}
	}

	// 准备行数据接收容器
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// 遍历数据行并写入CSV
	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("解析行数据失败(行%d): %w", rowCount+1, err)
		}

		// 转换为字符串
		row := make([]string, len(cols))
		for i, v := range values {
			row[i] = convertValueToString(v)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入CSV行失败(行%d): %w", rowCount+1, err)
		}
		rowCount++

		// 输出进度
		if rowCount%10000 == 0 {
			fmt.Printf("已处理 %d 行...\n", rowCount)
		}

		// 如果设置了限制，检查是否达到限制
		if cfg.Limit > 0 && rowCount >= cfg.Limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历行数据异常: %w", err)
	}

	fmt.Printf("✅ 导出完成，共 %d 行数据，文件路径: %s\n", rowCount, cfg.CSVPath)
	return nil
}

// convertValueToString 将数据库返回值转换为字符串
func convertValueToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int:
		return strconv.Itoa(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05.000")
	default:
		return fmt.Sprintf("%v", val)
	}
}
