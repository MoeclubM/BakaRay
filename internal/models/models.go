package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `json:"-" gorm:"size:128;not null"`
        Balance      int64     `json:"balance" gorm:"default:0"` // 单位：字节（剩余可用流量）
	UserGroupID  uint      `json:"user_group_id"`
Role         string    `json:"role" gorm:"size:20;default:'user'"` // admin, user
IsAdmin      bool      `json:"is_admin" gorm:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) AfterFind(_ *gorm.DB) error {
	u.IsAdmin = u.Role == "admin"
	return nil
}

// UserGroup 用户组表
type UserGroup struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	Description string    `json:"description" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
}

// Node 节点表
type Node struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	Host         string    `json:"host" gorm:"size:255;not null"`
	Port         int       `json:"port" gorm:"not null"`
	Secret       string    `json:"-" gorm:"size:128;not null"`
	Status       string    `json:"status" gorm:"size:20;default:'offline'"` // online, offline
	NodeGroupID  uint      `json:"node_group_id"`
        Protocols    StringSlice `json:"protocols" gorm:"type:text"` // JSON数组：["gost","iptables"]
	Multiplier   float64   `json:"multiplier" gorm:"default:1"`
	Region       string    `json:"region" gorm:"size:64"`
	LastSeen     *time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NodeAllowedGroups 节点-用户组关联表
type NodeAllowedGroup struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	NodeID       uint      `json:"node_id" gorm:"index;not null"`
	UserGroupID  uint      `json:"user_group_id" gorm:"index;not null"`
	CreatedAt    time.Time `json:"created_at"`
}

// NodeGroup 节点组表
type NodeGroup struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Type        string    `json:"type" gorm:"size:20;not null"` // entry, target
	Description string    `json:"description" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
}

// ForwardingRule 转发规则表
type ForwardingRule struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	NodeID       uint      `json:"node_id" gorm:"index;not null"`
	UserID       uint      `json:"user_id" gorm:"index;not null"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	Protocol     string    `json:"protocol" gorm:"size:20;not null"` // gost, iptables
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	TrafficUsed  int64     `json:"traffic_used" gorm:"default:0"`   // 单位：字节
	TrafficLimit int64     `json:"traffic_limit" gorm:"default:0"`  // 单位：字节
	SpeedLimit   int64     `json:"speed_limit" gorm:"default:0"`    // 单位：kbps
	Mode         string    `json:"mode" gorm:"size:20;default:'direct'"` // direct, rr, lb
	ListenPort   int       `json:"listen_port" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Target 转发目标表
type Target struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RuleID    uint      `json:"rule_id" gorm:"index;not null"`
	Host      string    `json:"host" gorm:"size:255;not null"`
	Port      int       `json:"port" gorm:"not null"`
	Weight    int       `json:"weight" gorm:"default:1"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
}

// GostRule gost 协议专用配置
type GostRule struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RuleID    uint      `json:"rule_id" gorm:"uniqueIndex;not null"`
	Transport string    `json:"transport" gorm:"size:20;default:'tcp'"` // tcp, udp, quic
	TLS       bool      `json:"tls" gorm:"default:false"`
	Chain     string    `json:"chain" gorm:"size:255"` // 代理链配置
	Timeout   int       `json:"timeout" gorm:"default:0"` // 秒
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IPTablesRule iptables 协议专用配置
type IPTablesRule struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RuleID    uint      `json:"rule_id" gorm:"uniqueIndex;not null"`
	Proto     string    `json:"proto" gorm:"size:10;default:'tcp'"` // tcp, udic
	SNAT      bool      `json:"snat" gorm:"default:false"`
	Iface     string    `json:"iface" gorm:"size:64"` // 网络接口
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Package 套餐表
type Package struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	Traffic      int64     `json:"traffic" gorm:"not null"` // 单位：字节
	Price        int64     `json:"price" gorm:"not null"`   // 单位：分
	UserGroupID  uint      `json:"user_group_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Order 订单表
type Order struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	PackageID uint      `json:"package_id" gorm:"index"`
	Amount    int64     `json:"amount" gorm:"not null"` // 单位：分
	Status    string    `json:"status" gorm:"size:20;default:'pending'"` // pending, success, failed
	TradeNo   string    `json:"trade_no" gorm:"size:64;uniqueIndex;not null"`
	PayType   string    `json:"pay_type" gorm:"size:32"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PaymentConfig 支付配置表
type PaymentConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:128;not null"`
	Provider     string    `json:"provider" gorm:"size:32;not null"` // epay, custom
	MerchantID   string    `json:"merchant_id" gorm:"size:64"`
	MerchantKey  string    `json:"merchant_key" gorm:"size:255"`
	APIURL       string    `json:"api_url" gorm:"size:255"`
	NotifyURL    string    `json:"notify_url" gorm:"size:255"`
	ExtraParams  string    `json:"extra_params" gorm:"type:text"` // JSON
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PaymentProvider 支付提供商扩展表
type PaymentProvider struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"size:64;not null"`
	Code          string    `json:"code" gorm:"size:32;uniqueIndex;not null"`
	Description   string    `json:"description" gorm:"size:255"`
	ConfigSchema  string    `json:"config_schema" gorm:"type:text"` // JSON Schema
	CreatedAt     time.Time `json:"created_at"`
}

// SiteConfig 站点配置表
type SiteConfig struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	SiteName           string    `json:"site_name" gorm:"size:128;not null"`
	SiteDomain         string    `json:"site_domain" gorm:"size:255"`
	NodeSecret         string    `json:"node_secret" gorm:"size:128"`
	NodeReportInterval int       `json:"node_report_interval" gorm:"default:30"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TrafficLog 流量日志表
type TrafficLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RuleID    uint      `json:"rule_id" gorm:"index;not null"`
	NodeID    uint      `json:"node_id" gorm:"index;not null"`
	BytesIn   int64     `json:"bytes_in" gorm:"default:0"`
	BytesOut  int64     `json:"bytes_out" gorm:"default:0"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

// ProbeData 探针数据结构（不存数据库，仅 Redis 缓存）
type ProbeData struct {
	Timestamp int64            `json:"timestamp"`
	CPU       CPUInfo          `json:"cpu"`
	Memory    MemoryInfo       `json:"memory"`
	Network   []NetworkInfo    `json:"network"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsagePercent float64 `json:"usage_percent"`
}

// NetworkInfo 网卡信息
type NetworkInfo struct {
	Name    string `json:"name"`
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	RxSpeed uint64 `json:"rx_speed"`
	TxSpeed uint64 `json:"tx_speed"`
}
