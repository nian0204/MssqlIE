package config

// DBConfig 数据库连接配置
type DBConfig struct {
	Server   string // SQL Server地址
	Port     int    // SQL Server端口
	User     string // 数据库用户名
	Password string // 数据库密码
	DBName   string // 数据库名
	Encrypt  bool   // 是否启用加密连接
}

// ExportConfig 导出配置（包含数据库配置+导出专属配置）
type ExportConfig struct {
	DBConfig
	Table   string // 导出表名（与SQL二选一）
	SQL     string // 自定义SQL（与Table二选一）
	CSVPath string // CSV输出路径
}

// ImportConfig 导入配置（包含数据库配置+导入专属配置）
type ImportConfig struct {
	DBConfig
	Table   string // 目标表名
	CSVPath string // CSV文件路径
	Batch   int    // 批量插入大小（默认1000）
}