// Package dbconn 提供SQL Server数据库连接功能
package conn

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"

	"github.com/mssql_ie/config"
)

// Connect 建立并返回SQL Server数据库连接
// 参数: cfg - 数据库连接配置
// 返回: *sql.DB - 数据库连接对象; error - 连接错误（非nil表示失败）
func Connect(cfg config.DBConfig) (*sql.DB, error) {
	// 构建连接字符串
	connStr, err := buildConnStr(cfg)
	if err != nil {
		return nil, fmt.Errorf("构建连接字符串失败: %w", err)
	}

	// 打开连接
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, fmt.Errorf("创建连接失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	setConnPool(db)

	return db, nil
}

// buildConnStr 构建SQL Server连接字符串
func buildConnStr(cfg config.DBConfig) (string, error) {
	/*
		EncryptionOff      = 0
		EncryptionRequired = 1
		EncryptionDisabled = 3
		EncryptionStrict   = 4
	*/
	encrypt := 3
	switch strings.ToLower(cfg.Encrypt) {
	case "off":
		encrypt = 0
	case "required":
		encrypt = 1
	case "disabled":
		encrypt = 3
	case "strict":
		encrypt = 4
	default:
		encrypt = 3
	}

	mssqlConfig := msdsn.Config{
		User:       cfg.User,
		Password:   cfg.Password,
		Port:       cfg.Port,
		Database:   cfg.DBName,
		Host:       cfg.Server,
		Encryption: msdsn.Encryption(encrypt),
	}
	params := make(map[string]interface{})
	params["protocol"] = "tcp"
	query := url.Values{}
	if len(cfg.Charset) > 0 {
		query.Add("charset", cfg.Charset)

	}
	mssqlConfig.URL().RawQuery = query.Encode()
	return mssqlConfig.URL().String(), nil
}

// setConnPool 配置连接池参数
func setConnPool(db *sql.DB) {
	db.SetMaxOpenConns(10)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大存活时间
}
