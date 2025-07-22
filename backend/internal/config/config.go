package config

import (
	"fmt"
	"time"
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat"`
	Log       LogConfig       `mapstructure:"log"`
	Alert     AlertConfig     `mapstructure:"alert"`
	JWT       JWTConfig       `mapstructure:"jwt"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	Charset         string        `mapstructure:"charset"`
	ParseTime       bool          `mapstructure:"parse_time"`
	Loc             string        `mapstructure:"loc"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	PoolSize   int    `mapstructure:"pool_size"`
	MaxRetries int    `mapstructure:"max_retries"`
}

type HeartbeatConfig struct {
	Timeout        time.Duration `mapstructure:"timeout"`
	CheckInterval  time.Duration `mapstructure:"check_interval"`
	MaxMissedBeats int           `mapstructure:"max_missed_beats"`
	ProbeTimeout   time.Duration `mapstructure:"probe_timeout"`
}

type LogConfig struct {
	Level          string `mapstructure:"level"`
	Format         string `mapstructure:"format"`
	Output         string `mapstructure:"output"`
	RetentionDays  int    `mapstructure:"retention_days"`
	CleanupTime    string `mapstructure:"cleanup_time"`
	DiskThreshold  int    `mapstructure:"disk_threshold"`
}

type AlertConfig struct {
	Channels ChannelConfig `mapstructure:"channels"`
}

type ChannelConfig struct {
	SMS   SMSConfig   `mapstructure:"sms"`
	Email EmailConfig `mapstructure:"email"`
	Slack SlackConfig `mapstructure:"slack"`
}

type SMSConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	RateLimit int  `mapstructure:"rate_limit"`
}

type EmailConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	RateLimit int    `mapstructure:"rate_limit"`
	SMTPHost  string `mapstructure:"smtp_host"`
	SMTPPort  int    `mapstructure:"smtp_port"`
}

type SlackConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	RateLimit  int    `mapstructure:"rate_limit"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type JWTConfig struct {
	Secret    string        `mapstructure:"secret"`
	ExpiresIn time.Duration `mapstructure:"expires_in"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.Charset,
		c.Database.ParseTime,
		c.Database.Loc,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}