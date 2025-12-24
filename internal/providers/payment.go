package providers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// PaymentProvider 支付提供商接口
type PaymentProvider interface {
	Name() string
	CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error)
	VerifyCallback(params map[string]string) (*VerifyResult, error)
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	TradeNo   string
	Amount    int64
	Subject   string
	NotifyURL string
	ReturnURL string
	Extra     map[string]string
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	PayURL    string
	TradeNo   string
	Amount    int64
	ExpiresAt int64
}

// VerifyResult 验证回调结果
type VerifyResult struct {
	TradeNo   string
	Amount    int64
	Status    string
	RawParams map[string]string
}

// EpayProvider 彩虹易支付提供商
type EpayProvider struct {
	MerchantID  string
	MerchantKey string
	APIURL      string
	NotifyURL   string
}

// NewEpayProvider 创建Epay提供商
func NewEpayProvider(merchantID, merchantKey, apiURL, notifyURL string) *EpayProvider {
	return &EpayProvider{
		MerchantID:  merchantID,
		MerchantKey: merchantKey,
		APIURL:      apiURL,
		NotifyURL:   notifyURL,
	}
}

func (p *EpayProvider) Name() string {
	return "epay"
}

// CreateOrder 创建支付订单
func (p *EpayProvider) CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error) {
	params := map[string]string{
		"pid":          p.MerchantID,
		"type":         "alipay", // 默认支付宝，可扩展
		"out_trade_no": req.TradeNo,
		"notify_url":   p.NotifyURL,
		"return_url":   req.ReturnURL,
		"amount":       fmt.Sprintf("%.2f", float64(req.Amount)/100),
		"subject":      req.Subject,
	}

	// 生成签名
	sign := p.generateSign(params)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// 构建支付URL
	payURL := p.APIURL + "/submit.php?" + p.buildQueryString(params)

	return &CreateOrderResponse{
		PayURL:    payURL,
		TradeNo:   req.TradeNo,
		Amount:    req.Amount,
		ExpiresAt: 0, // 永久有效
	}, nil
}

// VerifyCallback 验证支付回调
func (p *EpayProvider) VerifyCallback(params map[string]string) (*VerifyResult, error) {
	// 提取签名
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("缺少签名")
	}

	// 验证签名
	if !p.verifySign(params, sign) {
		return nil, fmt.Errorf("签名验证失败")
	}

	// 提取订单信息
	tradeNo := params["out_trade_no"]
	amountStr := params["amount"]
	status := params["trade_status"]

	// 解析金额（分）
	f, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("金额格式错误")
	}
	amount := int64(f*100 + 0.5) // 转为分（四舍五入）

	return &VerifyResult{
		TradeNo:   tradeNo,
		Amount:    amount,
		Status:    status,
		RawParams: params,
	}, nil
}

// generateSign 生成签名
func (p *EpayProvider) generateSign(params map[string]string) string {
	// 排序并拼接参数
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && k != "sign_type" && params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var signStr strings.Builder
	for _, k := range keys {
		signStr.WriteString(k + "=" + params[k] + "&")
	}
	signStr.WriteString("key=" + p.MerchantKey)

	// MD5签名
	hash := md5.Sum([]byte(signStr.String()))
	return hex.EncodeToString(hash[:])
}

// verifySign 验证签名
func (p *EpayProvider) verifySign(params map[string]string, expectedSign string) bool {
	// 保存原签名
	originalSign := params["sign"]
	originalSignType := params["sign_type"]

	// 临时移除sign字段进行验证
	delete(params, "sign")
	delete(params, "sign_type")

	// 生成签名
	computedSign := p.generateSign(params)

	// 恢复原签名
	params["sign"] = originalSign
	if originalSignType != "" {
		params["sign_type"] = originalSignType
	}

	return computedSign == expectedSign
}

// buildQueryString 构建查询字符串
func (p *EpayProvider) buildQueryString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(params[k])))
	}
	return strings.Join(parts, "&")
}
