// Package importer 提供从CSV文件导入数据到SQL Server表的功能
package importer

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mssql_ie/config" // 替换为实际命名空间
)

// CSVToTable 从CSV文件导入数据到指定表
// 参数: db - 已建立的数据库连接; cfg - 导入配置
// 返回: error - 导入错误（非nil表示失败）
func CSVToTable(db *sql.DB, cfg config.ImportConfig) error {
	// 参数校验
	if err := validateImportConfig(cfg); err != nil {
		return fmt.Errorf("配置校验失败: %w", err)
	}

	// 打开CSV文件
	file, err := os.Open(cfg.CSVPath)
	if err != nil {
		return fmt.Errorf("打开CSV文件失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// 读取列名（第一行）
	cols, err := reader.Read()
	if err != nil {
		return fmt.Errorf("读取CSV列名失败: %w", err)
	}

	// 构建插入SQL
	insertSQL, err := buildInsertSQL(cfg.Table, cols)
	if err != nil {
		return fmt.Errorf("构建插入SQL失败: %w", err)
	}

	// 开始事务批量插入
	return batchInsert(db, insertSQL, reader, cols, cfg.Batch)
}

// validateImportConfig 校验导入配置
func validateImportConfig(cfg config.ImportConfig) error {
	if cfg.Table == "" {
		return fmt.Errorf("目标表名不能为空")
	}
	if cfg.CSVPath == "" {
		return fmt.Errorf("CSV文件路径不能为空")
	}
	if cfg.Batch <= 0 {
		return fmt.Errorf("批量大小必须大于0（建议500-2000）")
	}
	return nil
}

// buildInsertSQL 构建参数化插入SQL
func buildInsertSQL(table string, cols []string) (string, error) {
	if len(cols) == 0 {
		return "", fmt.Errorf("列名不能为空")
	}

	// 防SQL注入，表名/列名用[]包裹
	safeTable := fmt.Sprintf("[%s]", strings.ReplaceAll(table, "]", "]]"))
	safeCols := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, col := range cols {
		safeCols[i] = fmt.Sprintf("[%s]", strings.ReplaceAll(col, "]", "]]"))
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		safeTable,
		strings.Join(safeCols, ","),
		strings.Join(placeholders, ","),
	), nil
}

// batchInsert 批量插入数据（事务+预处理）
func batchInsert(db *sql.DB, insertSQL string, reader *csv.Reader, cols []string, batchSize int) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	// 异常回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("❌ 导入异常，已回滚事务: %v\n", r)
		}
	}()

	// 预处理插入语句
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("预处理插入语句失败: %w", err)
	}
	defer stmt.Close()

	batchCount := 0
	totalCount := 0

	// 循环读取CSV行并插入
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			tx.Rollback()
			return fmt.Errorf("读取CSV行失败(行%d): %w", totalCount+2, err) // +2 因为第一行是列名
		}

		// 列数校验
		if len(row) != len(cols) {
			tx.Rollback()
			return fmt.Errorf("行%d数据列数不匹配（期望%d列，实际%d列）", totalCount+2, len(cols), len(row))
		}

		// 转换参数（空字符串转NULL）
		args := make([]interface{}, len(row))
		for i, v := range row {
			if v == "" {
				args[i] = nil
			} else {
				args[i] = v
			}
		}

		// 执行插入
		if _, err := stmt.Exec(args...); err != nil {
			tx.Rollback()
			return fmt.Errorf("插入行失败(行%d): %w", totalCount+2, err)
		}

		batchCount++
		totalCount++

		// 达到批量大小提交事务
		if batchCount >= batchSize {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("提交批量事务失败(累计%d行): %w", totalCount, err)
			}
			// 重新开启事务
			tx, err = db.Begin()
			if err != nil {
				return fmt.Errorf("重新开启事务失败: %w", err)
			}
			// 重新预处理语句
			stmt, err = tx.Prepare(insertSQL)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("重新预处理语句失败: %w", err)
			}
			batchCount = 0
		}
	}

	// 提交剩余数据
	if batchCount > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交剩余数据失败: %w", err)
		}
	}

	fmt.Printf("✅ CSV导入完成，共插入 %d 行数据到表 [%s]\n", totalCount, strings.Split(insertSQL, " ")[2])
	return nil
}
