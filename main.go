// Package main 演示如何使用sqlserver-csv库构建CLI工具
package main

import (
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

	// 解析命令行
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 先解析共享参数
	if err := rootFlags.Parse(os.Args[2:]); err != nil {
		printUsage()
		os.Exit(1)
	}

	// 构建数据库配置
	dbCfg := config.DBConfig{
		Server:   *server,
		Port:     *port,
		User:     *user,
		Password: *password,
		DBName:   *dbName,
		Encrypt:  *encrypt,
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

	// 处理子命令
	switch os.Args[1] {
	case "export":
		handleExport(exportFlags, rootFlags.Args(), db, dbCfg)
	case "import":
		handleImport(importFlags, rootFlags.Args(), db, dbCfg)
	default:
		fmt.Printf("❌ 未知子命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// handleExport 处理导出命令
func handleExport(exportFlags *flag.FlagSet, args []string, db *config.DB, dbCfg config.DBConfig) {
	if err := exportFlags.Parse(args); err != nil {
		printUsage()
		os.Exit(1)
	}

	exportCfg := config.ExportConfig{
		DBConfig: dbCfg,
		Table:    *exportFlags.Lookup("table").Value.(flag.Getter).Get().(string),
		SQL:      *exportFlags.Lookup("sql").Value.(flag.Getter).Get().(string),
		CSVPath:  *exportFlags.Lookup("csv").Value.(flag.Getter).Get().(string),
	}

	// 校验导出参数
	if exportCfg.CSVPath == "" {
		fmt.Println("❌ 错误: export 命令必须指定 -csv 参数")
		printUsage()
		os.Exit(1)
	}
	if (exportCfg.Table == "" && exportCfg.SQL == "") || (exportCfg.Table != "" && exportCfg.SQL != "") {
		fmt.Println("❌ 错误: export 命令必须且只能指定 -table 或 -sql 其中一个参数")
		printUsage()
		os.Exit(1)
	}

	// 执行导出
	if exportCfg.Table != "" {
		if err := exporter.TableToCSV(db, exportCfg); err != nil {
			fmt.Printf("❌ 表导出失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := exporter.SQLToCSV(db, exportCfg); err != nil {
			fmt.Printf("❌ SQL导出失败: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleImport 处理导入命令
func handleImport(importFlags *flag.FlagSet, args []string, db *config.DB, dbCfg config.DBConfig) {
	if err := importFlags.Parse(args); err != nil {
		printUsage()
		os.Exit(1)
	}

	importCfg := config.ImportConfig{
		DBConfig: dbCfg,
		Table:    *importFlags.Lookup("table").Value.(flag.Getter).Get().(string),
		CSVPath:  *importFlags.Lookup("csv").Value.(flag.Getter).Get().(string),
		Batch:    *importFlags.Lookup("batch").Value.(flag.Getter).Get().(int),
	}

	// 校验导入参数
	if importCfg.Table == "" || importCfg.CSVPath == "" {
		fmt.Println("❌ 错误: import 命令必须指定 -table 和 -csv 参数")
		printUsage()
		os.Exit(1)
	}

	// 执行导入
	if err := importer.CSVToTable(db, importCfg); err != nil {
		fmt.Printf("❌ CSV导入失败: %v\n", err)
		os.Exit(1)
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("SQL Server CSV 导入导出工具（基于sqlserver-csv库）")
	fmt.Println("========================================")
	fmt.Println("用法:")
	fmt.Println("  导出: ./sqlserver-csv export [参数]")
	fmt.Println("  导入: ./sqlserver-csv import [参数]")
	fmt.Println("\n共享参数（所有命令必选）:")
	fmt.Println("  -server    SQL Server地址 (默认: localhost)")
	fmt.Println("  -port      SQL Server端口 (默认: 1433)")
	fmt.Println("  -user      数据库用户名 (默认: sa)")
	fmt.Println("  -password  数据库密码 (必填)")
	fmt.Println("  -db        数据库名 (必填)")
	fmt.Println("  -encrypt   是否启用加密连接 (默认: false)")
	fmt.Println("\n导出参数 (export):")
	fmt.Println("  -table     要导出的表名（与-sql二选一）")
	fmt.Println("  -sql       自定义导出SQL（与-table二选一）")
	fmt.Println("  -csv       CSV输出文件路径 (必填)")
	fmt.Println("\n导入参数 (import):")
	fmt.Println("  -table     目标表名 (必填)")
	fmt.Println("  -csv       CSV文件路径 (必填)")
	fmt.Println("  -batch     批量插入大小 (默认: 1000)")
	fmt.Println("\n示例:")
	fmt.Println("  导出表:")
	fmt.Println("    ./sqlserver-csv export -server 127.0.0.1 -port 1433 -user sa -password 123456 -db TestDB -table Users -csv ./users.csv")
	fmt.Println("  导入CSV:")
	fmt.Println("    ./sqlserver-csv import -server 127.0.0.1 -port 1433 -user sa -password 123456 -db TestDB -table Users -csv ./users.csv -batch 500")
}
