package modules

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

var (
	DatasourceTypeAPI       = "api"
	DatasourceTypeSQLServer = "sqlserver"
	DatasourceTypeMySQL     = "mysql"
	DatasourceTypeSQLite    = "sqlite"
	DatasourceTypeOracle    = "oracle"
)

type Datasource struct {
	Code     string  `yaml:"Code"`              // 编号
	Type     string  `yaml:"Type"`              // 类型
	Url      string  `yaml:"Url" json:"-"`      // 请求地址
	DSN      string  `yaml:"DSN" json:"-"`      // 连接串
	Server   string  `yaml:"Server" json:"-"`   // 服务
	Port     int     `yaml:"Port" json:"-"`     // 端口
	Username string  `yaml:"Username" json:"-"` // 用户名
	Password string  `yaml:"Password" json:"-"` // 密码
	DB       *sql.DB `yaml:"-" json:"-"`        // 连接池
}

func (datasource *Datasource) GetDSN() string {
	if datasource.DSN != "" {
		return datasource.DSN
	}
	return fmt.Sprintf("chartset=utf8mb4;server=%s;port=%d;user id=%s;password=%s;parseTime=true;loc=Local;", datasource.Server, datasource.Port, datasource.Username, datasource.Password)
}

func (datasource *Datasource) Connect() (*sql.DB, error) {
	driverName := ""
	dsn := datasource.GetDSN()
	switch datasource.Type {
	case "":
		driverName = "sqlserver"
	case DatasourceTypeSQLServer:
		driverName = "sqlserver"
	case DatasourceTypeMySQL:
		driverName = "mysql"
	case DatasourceTypeSQLite:
		driverName = "sqlite"
	case DataConfigTypeOracle:
		driverName = "oracle"
	}
	if driverName == "" {
		return nil, fmt.Errorf("database %s not support", datasource.Type)
	}
	return sql.Open(driverName, dsn)
}
