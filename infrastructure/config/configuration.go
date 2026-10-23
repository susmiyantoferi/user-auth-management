package config

type Configuration struct {
	App      AppConfig      `mapstructure:"app"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Jwt      JwtConfig      `mapstructure:"jwt"`
}

type AppConfig struct {
	Port        string `mapstructure:"port"`
	Environment string `mapstructure:"environment"`
}

type PostgresConfig struct {
	Host        string `mapstructure:"host"`
	Port        string `mapstructure:"port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	DbName      string `mapstructure:"dbname"`
	IdleConnect int    `mapstructure:"idleconnect"`
	MaxConnect  int    `mapstructure:"maxconnect"`
	LifeConnect int    `mapstructure:"lifeconnect"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Db       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
	Username string `mapstructure:"username"`
}

type LoggerConfig struct {
	Level int    `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type JwtConfig struct {
	SecretKey          string `mapstructure:"secretkey"`
	ExpiredTime        int    `mapstructure:"expiredtime"`
	RefreshExpiredTime int    `mapstructure:"refresh_expiredtime"`
}
