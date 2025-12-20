// Package exporter 提供SQL Server数据导出为CSV的功能
package exporter

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mssql_ie/config" // 替换为实际命名空间
)

// TableToCSV 将指定表的数据导出到CSV文件
// 参数: db - 已建立的数据库连接; cfg - 导出配置
// 返回: error - 导出错误（非nil表示失败）
func TableToCSV(db *sql.DB, cfg config.ExportConfig) error {
	if cfg.Table == "" {
		return fmt.Errorf("表名不能为空")
	}
	if cfg.CSVPath == "" {
		return fmt.Errorf("CSV文件路径不能为空")
	}

	// 防SQL注入，表名用[]包裹
	query := fmt.Sprintf("SELECT * FROM [%s]", strings.ReplaceAll(cfg.Table, "]", "]]"))
	return exportQueryResultToCSV(db, query, cfg.CSVPath)
}

// SQLToCSV 执行自定义SQL并将结果导出到CSV文件
// 参数: db - 已建立的数据库连接; cfg - 导出配置
// 返回: error - 导出错误（非nil表示失败）
func SQLToCSV(db *sql.DB, cfg config.ExportConfig) error {
	if cfg.SQL == "" {
		return fmt.Errorf("SQL语句不能为空")
	}
	if cfg.CSVPath == "" {
		return fmt.Errorf("CSV文件路径不能为空")
	}

	return exportQueryResultToCSV(db, cfg.SQL, cfg.CSVPath)
}

// exportQueryResultToCSV 通用导出逻辑：执行查询并写入CSV
func exportQueryResultToCSV(db *sql.DB, query, csvPath string) error {
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
	file, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("创建CSV文件失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入列名
	if err := writer.Write(cols); err != nil {
		return fmt.Errorf("写入列名失败: %w", err)
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

		// 转换为字符串（处理NULL和不同数据类型）
		row := make([]string, len(cols))
		for i, v := range values {
			row[i] = convertValueToString(v)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入CSV行失败(行%d): %w", rowCount+1, err)
		}
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历行数据异常: %w", err)
	}

	fmt.Printf("✅ 导出完成，共 %d 行数据，文件路径: %s\n", rowCount, csvPath)
	return nil
}

// convertValueToString 将数据库返回值转换为字符串（处理NULL和各种数据类型）
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
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", val)
	}
}