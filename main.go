package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/mssql_ie/config"
	"github.com/mssql_ie/conn"
	"github.com/mssql_ie/exporter"
	"github.com/mssql_ie/importer"
	"github.com/urfave/cli/v2"
)

var (
	version = "1.0.0"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	app := &cli.App{
		Name:     "mssql-ie",
		Version:  fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		Usage:    "SQL Server 数据导入导出工具",
		Suggest:  true,
		HideHelp: false,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "server",
				Aliases: []string{"S"},
				Value:   "localhost",
				Usage:   "SQL Server地址",
				EnvVars: []string{"MSSQL_SERVER", "DB_SERVER"},
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"P"},
				Value:   1433,
				Usage:   "SQL Server端口",
				EnvVars: []string{"MSSQL_PORT", "DB_PORT"},
			},
			&cli.StringFlag{
				Name:    "user",
				Aliases: []string{"U"},
				Value:   "sa",
				Usage:   "数据库用户名",
				EnvVars: []string{"MSSQL_USER", "DB_USER"},
			},
			&cli.StringFlag{
				Name:     "password",
				Aliases:  []string{"W"},
				Usage:    "数据库密码",
				Required: true,
				EnvVars:  []string{"MSSQL_PASSWORD", "DB_PASSWORD"},
			},
			&cli.StringFlag{
				Name:     "db",
				Aliases:  []string{"D"},
				Usage:    "数据库名",
				Required: true,
				EnvVars:  []string{"MSSQL_DBNAME", "DB_NAME"},
			},
			&cli.StringFlag{
				Name:    "encrypt",
				Aliases: []string{"E"},
				Value:   "off",
				Usage:   "是否启用加密连接",
				EnvVars: []string{"MSSQL_ENCRYPT"},
			},
			&cli.StringFlag{
				Name:    "charset",
				Aliases: []string{"C"},
				Usage:   "字符集 (例如: utf8, gbk)",
				Value:   "utf8",
				EnvVars: []string{"MSSQL_CHARSET"},
			},
			&cli.IntFlag{
				Name:    "timeout",
				Aliases: []string{"T"},
				Value:   30,
				Usage:   "连接超时时间(秒)",
				EnvVars: []string{"MSSQL_TIMEOUT"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "export",
				Aliases: []string{"e"},
				Usage:   "导出数据到CSV文件",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "csv",
						Aliases:  []string{"o"},
						Usage:    "CSV输出文件路径",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "table",
						Aliases: []string{"t"},
						Usage:   "要导出的表名 (与 --sql 二选一)",
					},
					&cli.StringFlag{
						Name:    "sql",
						Aliases: []string{"s"},
						Usage:   "自定义SQL查询 (与 --table 二选一)",
					},
					&cli.BoolFlag{
						Name:  "header",
						Usage: "包含列标题",
						Value: true,
					},
					&cli.StringFlag{
						Name:  "delimiter",
						Usage: "CSV分隔符",
						Value: ",",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "限制导出记录数 (0表示无限制)",
						Value:   0,
					},
					&cli.StringFlag{
						Name:    "binary-format",
						Aliases: []string{"bf"},
						Usage:   "二进制数格式 {hex, base64, raw}",
						Value:   "raw",
					},
					&cli.StringFlag{
						Name:    "file-charset",
						Aliases: []string{"fc"},
						Usage:   "文件的字符集 {utf8,gbk,latinl}",
						Value:   "utf8",
					},
				},
				Before: validateExportFlags,
				Action: exportCommand,
			},
			{
				Name:    "import",
				Aliases: []string{"i"},
				Usage:   "从CSV文件导入数据",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "csv",
						Aliases:  []string{"i"},
						Usage:    "CSV输入文件路径",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "目标表名",
						Required: true,
					},
					&cli.IntFlag{
						Name:    "batch",
						Aliases: []string{"b"},
						Usage:   "批量插入大小",
						Value:   1000,
					},
					&cli.BoolFlag{
						Name:  "header",
						Usage: "CSV文件包含列标题",
						Value: true,
					},
					&cli.StringFlag{
						Name:  "delimiter",
						Usage: "CSV分隔符",
						Value: ",",
					},
					&cli.BoolFlag{
						Name:  "truncate",
						Usage: "导入前清空表",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "skip-errors",
						Usage: "跳过错误行继续导入",
						Value: false,
					},
					&cli.StringFlag{
						Name:    "binary-format",
						Aliases: []string{"bf"},
						Usage:   "二进制数格式 {hex, base64, raw}",
						Value:   "raw",
					},
					&cli.StringFlag{
						Name:    "file-charset",
						Aliases: []string{"fc"},
						Usage:   "文件的字符集 {utf8,gbk,latinl}",
						Value:   "utf8",
					},
				},
				Before: validateImportFlags,
				Action: importCommand,
			},
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "测试数据库连接",
				Action:  testConnection,
			},
		},
		Before: func(c *cli.Context) error {
			if c.String("password") == "" {
				return cli.Exit("错误: 密码不能为空，请通过 -password 参数或环境变量设置", 1)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			cli.ShowAppHelp(c)
			return nil
		},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil {
				fmt.Fprintf(c.App.Writer, "❌ 错误: %v\n", err)
			}
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

// 构建数据库配置
func buildDBConfig(c *cli.Context) config.DBConfig {
	return config.DBConfig{
		Server:   c.String("server"),
		Port:     uint64(c.Int("port")),
		User:     c.String("user"),
		Password: c.String("password"),
		DBName:   c.String("db"),
		Encrypt:  c.String("encrypt"),
		Charset:  c.String("charset"),
		Timeout:  uint64(c.Int("timeout")),
	}
}

// 连接数据库
func connectDB(c *cli.Context) (*sql.DB, error) {
	dbCfg := buildDBConfig(c)
	return conn.Connect(dbCfg)
}

// 导出命令
func exportCommand(c *cli.Context) error {
	db, err := connectDB(c)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 解析分隔符
	delimiter := ','
	if delim := c.String("delimiter"); len(delim) > 0 {
		delimiter = []rune(delim)[0]
	}

	cfg := config.ExportConfig{
		Table:        c.String("table"),
		SQL:          c.String("sql"),
		CSVPath:      c.String("csv"),
		Header:       c.Bool("header"),
		Delimiter:    delimiter,
		Limit:        c.Int("limit"),
		BinaryFormat: c.String("binary-format"),
		FileCharset:  c.String("file-charset"),
	}

	if cfg.Table != "" {
		if err := exporter.TableToCSV(db, cfg); err != nil {
			return fmt.Errorf("导出表失败: %w", err)
		}
	} else {
		if err := exporter.SQLToCSV(db, cfg); err != nil {
			return fmt.Errorf("导出SQL结果失败: %w", err)
		}
	}

	fmt.Printf("✅ 导出成功: 数据已保存到 %s\n", cfg.CSVPath)
	return nil
}

// 导入命令
func importCommand(c *cli.Context) error {
	db, err := connectDB(c)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 解析分隔符
	delimiter := ','
	if delim := c.String("delimiter"); len(delim) > 0 {
		delimiter = []rune(delim)[0]
	}

	cfg := config.ImportConfig{
		Table:        c.String("table"),
		CSVPath:      c.String("csv"),
		Batch:        c.Int("batch"),
		Header:       c.Bool("header"),
		Delimiter:    delimiter,
		Truncate:     c.Bool("truncate"),
		SkipErrors:   c.Bool("skip-errors"),
		BinaryFormat: c.String("binary-format"),
		FileCharset:  c.String("file-charset"),
	}

	if err := importer.CSVToTable(db, cfg); err != nil {
		return fmt.Errorf("导入失败: %w", err)
	}

	fmt.Printf("✅ 导入成功: 数据已导入到表 %s\n", cfg.Table)
	return nil
}

// 测试连接命令
func testConnection(c *cli.Context) error {
	db, err := connectDB(c)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 获取数据库信息
	var version, dbName string
	err = db.QueryRowContext(context.Background(), "SELECT @@VERSION, DB_NAME()").Scan(&version, &dbName)
	if err != nil {
		return fmt.Errorf("查询数据库信息失败: %w", err)
	}

	fmt.Println("✅ 数据库连接测试成功!")
	fmt.Printf("   数据库: %s\n", dbName)
	fmt.Printf("   服务器: %s:%d\n", c.String("server"), c.Int("port"))
	fmt.Printf("   版本: %s\n", version)

	return nil
}

// 导出参数验证
func validateExportFlags(c *cli.Context) error {
	table := c.String("table")
	sql := c.String("sql")
	csv := c.String("csv")

	if csv == "" {
		return cli.Exit("错误: 必须指定 --csv 参数", 1)
	}

	if (table == "" && sql == "") || (table != "" && sql != "") {
		return cli.Exit("错误: 必须且只能指定 --table 或 --sql 参数之一", 1)
	}

	// 检查文件是否可以创建
	if _, err := os.Stat(csv); err == nil {
		// 文件已存在，询问是否覆盖
		fmt.Printf("警告: 文件 %s 已存在，是否覆盖? (y/N): ", csv)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return cli.Exit("操作已取消", 0)
		}
	}

	return nil
}

// 导入参数验证
func validateImportFlags(c *cli.Context) error {
	table := c.String("table")
	csv := c.String("csv")
	batch := c.Int("batch")

	if table == "" {
		return cli.Exit("错误: 必须指定 --table 参数", 1)
	}

	if csv == "" {
		return cli.Exit("错误: 必须指定 --csv 参数", 1)
	}

	if batch <= 0 {
		return cli.Exit("错误: --batch 参数必须大于0", 1)
	}

	// 检查文件是否存在
	if _, err := os.Stat(csv); os.IsNotExist(err) {
		return cli.Exit(fmt.Sprintf("错误: CSV文件不存在: %s", csv), 1)
	}

	return nil
}
