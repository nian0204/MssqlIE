package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/mssql_ie/config"
	"github.com/mssql_ie/conn"
	"github.com/mssql_ie/exporter"
	"github.com/mssql_ie/importer"
)

func main() {
	// 根参数（共享数据库配置）
	rootFlags := flag.NewFlagSet("root", flag.ContinueOnError)
	server := rootFlags.String("server", "localhost", "SQL Server地址")
	port := rootFlags.Int("port", 1433, "SQL Server端口")
	user := rootFlags.String("user", "sa", "数据库用户名")
	password := rootFlags.String("password", "", "数据库密码")
	dbName := rootFlags.String("db", "", "数据库名")
	encrypt := rootFlags.Bool("encrypt", false, "是否启用加密连接")
	charset := rootFlags.String("charset", "", "字符集")

	// 导出子命令参数
	exportFlags := flag.NewFlagSet("export", flag.ContinueOnError)
	exportTable := exportFlags.String("table", "", "导出表名")
	exportSQL := exportFlags.String("sql", "", "自定义导出SQL")
	exportCSV := exportFlags.String("csv", "", "CSV输出路径")

	// 导入子命令参数
	importFlags := flag.NewFlagSet("import", flag.ContinueOnError)
	importTable := importFlags.String("table", "", "目标表名")
	importCSV := importFlags.String("csv", "", "CSV文件路径")
	importBatch := importFlags.Int("batch", 1000, "批量插入大小")

	// 校验最少参数（必须包含子命令）
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 解析全局参数（子命令前的参数）
	if err := rootFlags.Parse(os.Args[2:]); err != nil {
		printUsage()
		os.Exit(1)
	}

	// 构建数据库配置（修复类型不匹配问题）
	dbCfg := config.DBConfig{
		Server:   *server,
		Port:     uint64(*port), // 转换为uint64匹配config定义
		User:     *user,
		Password: *password,
		DBName:   *dbName,
		// 转换bool为字符串（匹配conn包的加密配置处理）
		Encrypt: map[bool]string{true: "strict", false: "off"}[*encrypt],
		Charset: *charset,
	}

	// 校验共享参数
	if dbCfg.Password == "" || dbCfg.DBName == "" {
		fmt.Println("❌ 错误: -password 和 -db 参数不能为空")
		printUsage()
		os.Exit(1)
	}

	// 连接数据库
	db, err := conn.Connect(dbCfg)
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 处理子命令（修复参数解析逻辑）
	switch os.Args[1] {
	case "export":
		// 解析导出子命令参数
		if err := exportFlags.Parse(rootFlags.Args()); err != nil {
			printUsage()
			os.Exit(1)
		}
		// 调用导出处理函数
		if err := handleExport(db, *exportTable, *exportSQL, *exportCSV); err != nil {
			fmt.Printf("❌ 导出失败: %v\n", err)
			os.Exit(1)
		}
	case "import":
		// 解析导入子命令参数
		if err := importFlags.Parse(rootFlags.Args()); err != nil {
			printUsage()
			os.Exit(1)
		}
		// 调用导入处理函数
		if err := handleImport(db, *importTable, *importCSV, *importBatch); err != nil {
			fmt.Printf("❌ 导入失败: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("❌ 未知子命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// 处理导出逻辑（新增函数，连接exporter包）
func handleExport(db *sql.DB, table, sql, csvPath string) error {
	cfg := config.ExportConfig{
		Table:   table,
		SQL:     sql,
		CSVPath: csvPath,
	}

	// 校验导出参数（二选一：表名或SQL）
	if (table == "" && sql == "") || (table != "" && sql != "") {
		return fmt.Errorf("必须且只能指定 -table 或 -sql 参数")
	}

	// 调用导出功能
	if table != "" {
		return exporter.TableToCSV(db, cfg)
	}
	return exporter.SQLToCSV(db, cfg)
}

// 处理导入逻辑（新增函数，连接importer包）
func handleImport(db *sql.DB, table, csvPath string, batch int) error {
	cfg := config.ImportConfig{
		Table:   table,
		CSVPath: csvPath,
		Batch:   batch,
	}
	return importer.CSVToTable(db, cfg)
}

// 打印使用说明（新增函数，修复未定义问题）
func printUsage() {
	usage := `用法: mssql-ie [全局参数] 子命令 [子命令参数]

子命令:
  export    将数据从SQL Server导出为CSV
  import    将CSV数据导入到SQL Server

全局参数:
  -server   SQL Server地址 (默认: localhost)
  -port     SQL Server端口 (默认: 1433)
  -user     数据库用户名 (默认: sa)
  -password 数据库密码 (必填)
  -db       数据库名 (必填)
  -encrypt  是否启用加密连接 (默认: false)
  -charset  字符集 (可选)

export子命令参数:
  -table    导出表名（与-sql二选一）
  -sql      自定义导出SQL（与-table二选一）
  -csv      CSV输出路径（必填）

import子命令参数:
  -table    目标表名（必填）
  -csv      CSV文件路径（必填）
  -batch    批量插入大小 (默认: 1000)
`
	fmt.Print(usage)
}
