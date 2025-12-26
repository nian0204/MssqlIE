// importer/import.go
package importer

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mssql_ie/config"
	"github.com/mssql_ie/utils"
)

// CSVToTable 从CSV文件导入数据到指定表
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

	// 应用字符集转换
	reader := csv.NewReader(utils.GetTransformersRead(file, cfg.FileCharset))
	reader.Comma = cfg.Delimiter

	// 读取列名
	var columnInfos []ColumnInfo
	// 如果没有标题行，尝试从数据库获取列名
	columnInfos, err = getTableColumns(db, cfg.Table)
	if err != nil {
		return fmt.Errorf("获取表列名失败: %w", err)
	}
	var headerRow []string
	if cfg.Header {
		headerRow, err = reader.Read()
		if err != nil {
			return fmt.Errorf("读取CSV列名失败: %w", err)
		}
		// 检查CSV列名是否与数据库列名匹配
		if len(headerRow) != len(columnInfos) {
			return fmt.Errorf("CSV列数 %d 与数据库列数 %d 不匹配", len(headerRow), len(columnInfos))
		}
		for _, col := range headerRow {
			found := false
			for _, dbCol := range columnInfos {
				if strings.EqualFold(col, dbCol.Name) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("CSV列 %s 与数据库列名不匹配", col)
			}
		}
	} else {
		headerRow = make([]string, len(columnInfos))
		for i, col := range columnInfos {
			headerRow[i] = col.Name
		}
	}

	// 安全地转义列名
	safeCols := make([]string, len(headerRow))
	for i, col := range headerRow {
		safeCols[i] = utils.EscapeIdentifier(col)
	}

	// 构建插入SQL
	insertSQL, err := buildInsertSQL(cfg.Table, safeCols)
	if err != nil {
		return fmt.Errorf("构建插入SQL失败: %w", err)
	}

	// 如果需要，先清空表
	if cfg.Truncate {
		if err := truncateTable(db, cfg.Table); err != nil {
			return fmt.Errorf("清空表失败: %w", err)
		}
	}

	// 开始事务批量插入
	return batchInsert(db, insertSQL, reader, columnInfos, cfg.Batch, cfg.SkipErrors, !cfg.Header, cfg.BinaryFormat)
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

type ColumnInfo struct {
	Name     string
	DataType string
	Nullable bool
}

// getTableColumns 从数据库获取表的列名
func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	// 转义表名
	escapedTable, err := utils.EscapeQualifiedName(tableName)
	if err != nil {
		return nil, fmt.Errorf("转义表名失败: %w", err)
	}

	query := fmt.Sprintf(`
		/* mssql_ie tool query for check column*/
		SELECT COLUMN_NAME ,DATA_TYPE,IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = COALESCE(PARSENAME('%s', 2), 'dbo')
			AND TABLE_NAME = PARSENAME('%s', 1)
		ORDER BY ORDINAL_POSITION
	`, escapedTable, escapedTable)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询表结构失败: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var nullableStr string
		if err := rows.Scan(&col.Name, &col.DataType, &nullableStr); err != nil {
			return nil, err
		}
		col.Nullable = nullableStr == "YES"
		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("表 %s 不存在或没有列", tableName)
	}

	return columns, nil
}

// truncateTable 清空表
func truncateTable(db *sql.DB, tableName string) error {
	escapedTable, err := utils.EscapeQualifiedName(tableName)
	if err != nil {
		return fmt.Errorf("转义表名失败: %w", err)
	}

	// 使用TRUNCATE TABLE
	_, err = db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", escapedTable))
	return err
}

// buildInsertSQL 构建参数化插入SQL
func buildInsertSQL(table string, safeCols []string) (string, error) {
	if len(safeCols) == 0 {
		return "", fmt.Errorf("列名不能为空")
	}

	// 转义表名
	safeTable, err := utils.EscapeQualifiedName(table)
	if err != nil {
		return "", fmt.Errorf("转义表名失败: %w", err)
	}

	// 构建占位符
	placeholders := make([]string, len(safeCols))
	for i := range safeCols {
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		safeTable,
		strings.Join(safeCols, ","),
		strings.Join(placeholders, ","),
	), nil
}

// batchInsert 批量插入数据
func batchInsert(db *sql.DB, insertSQL string, reader *csv.Reader, safeCols []ColumnInfo, batchSize int, skipErrors, skipFirstRow bool, binaryFormat string) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// 异常回滚处理
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // 重新抛出panic
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
	rowNum := 0
	errorRows := []int{}

	// 循环读取CSV行
	for {
		row, err := reader.Read()
		rowNum++

		// 跳过标题行
		if skipFirstRow && rowNum == 1 {
			continue
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			if skipErrors {
				errorRows = append(errorRows, rowNum)
				continue
			}
			tx.Rollback()
			return fmt.Errorf("读取CSV行失败(行%d): %w", rowNum, err)
		}

		// 列数校验
		if len(row) != len(safeCols) {
			if skipErrors {
				errorRows = append(errorRows, rowNum)
				continue
			}
			tx.Rollback()
			return fmt.Errorf("行%d数据列数不匹配（期望%d列，实际%d列）", rowNum, len(safeCols), len(row))
		}

		// 准备参数
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
			if skipErrors {
				errorRows = append(errorRows, rowNum)
				continue
			}
			tx.Rollback()
			return fmt.Errorf("插入行失败(行%d): %w", rowNum, err)
		}

		batchCount++
		totalCount++

		// 达到批量大小提交事务
		if batchCount >= batchSize {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("提交批量事务失败(累计%d行): %w", totalCount, err)
			}

			// 开始新事务
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

			fmt.Printf("已导入 %d 行...\n", totalCount)
		}
	}

	// 提交剩余数据
	if batchCount > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交剩余数据失败: %w", err)
		}
	}

	// 输出结果
	fmt.Printf("✅ CSV导入完成，共插入 %d 行数据\n", totalCount)
	if len(errorRows) > 0 {
		fmt.Printf("⚠️  跳过 %d 行错误数据: %v\n", len(errorRows), errorRows)
	}

	return nil
}

func convertValue(value string, col ColumnInfo, binaryFormat string) (interface{}, error) {
	if value == "" {
		if col.Nullable {
			return nil, nil
		}
		return getDefaultValue(col.DataType), nil
	}
	switch strings.ToLower(col.DataType) {
	case "bit":
		value = strings.ToLower(value)
		if value == "true" || value == "1" || value == "y" || value == "yes" || value == "t" {
			return true, nil
		}
		if value == "false" || value == "0" || value == "n" || value == "no" || value == "f" {
			return false, nil
		}
		return nil, fmt.Errorf("无效的位值: %s", value)
	case "binary", "varbinary", "image":
		return convertBinary(value, binaryFormat)
	case "geometry", "geography":
		return convertGeo(value, binaryFormat)
	case "hierarchyid":
		return convertHierarchyID(value, binaryFormat)
	default: // 其他类型保持字符串
		return value, nil
	}
}
func convertHierarchyID(value, binaryFormat string) ([]byte, error) {
	return convertBinary(value, binaryFormat)
}
func convertGeo(value, binaryFormat string) (interface{}, error) {
	if isValidWKT(value) {
		return value, nil
	}
	return convertBinary(value, binaryFormat)
}
func isValidWKT(wkt string) bool {
	return (strings.HasPrefix(strings.ToUpper(wkt), "POINT(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "LINESTRING(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "POLYGON(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "MULTIPOINT(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "MULTILINESTRING(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "MULTIPOLYGON(") ||
		strings.HasPrefix(strings.ToUpper(wkt), "GEOMETRYCOLLECTION(")) && (strings.HasSuffix(strings.ToUpper(wkt), ")") ||
		strings.HasSuffix(strings.ToUpper(wkt), "ZM)") ||
		strings.HasSuffix(strings.ToUpper(wkt), "M)"))
}
func convertBinary(value string, binaryFormat string) ([]byte, error) {
	switch strings.ToLower(binaryFormat) {
	case "hex":
		return utils.HexToBytes(value)
	case "base64":
		return utils.Base64ToBytes(value)
	default: // raw
		return []byte(value), nil
	}
}
func getDefaultValue(dataType string) interface{} {
	switch strings.ToLower(dataType) {
	case "int", "smallint", "tinyint", "bigint", "numeric", "decimal", "real", "float":
		return 0
	case "bit":
		return false
	case "binary", "varbinary", "image":
		return []byte{}
	case "geometry", "geography":
		return []byte{}
	case "hierarchyid":
		return []byte{}
	default: // 其他类型返回空字符串
		return ""
	}
}
