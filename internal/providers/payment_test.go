package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEpayProviderGenerateSignV1MD5(t *testing.T) {
	provider := NewEpayProvider("1001", "testkey123", "https://pay.example.com", "")

	sign := provider.generateSign(map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "202604120001",
		"notify_url":   "https://panel.example.com/api/deposit/callback",
		"return_url":   "https://panel.example.com/deposit/callback",
		"money":        "12.34",
		"name":         "BakaRay 账户充值",
		"sign_type":    "MD5",
	})

	require.Equal(t, "18880370e658cfa2cef0e872406385a5", sign)
}

func TestEpayProviderCreateOrderUsesV1MD5Sign(t *testing.T) {
	provider := NewEpayProvider("1001", "testkey123", "https://pay.example.com", "")

	resp, err := provider.CreateOrder(&CreateOrderRequest{
		TradeNo:   "202604120001",
		Amount:    1234,
		Subject:   "BakaRay 账户充值",
		NotifyURL: "https://panel.example.com/api/deposit/callback",
		ReturnURL: "https://panel.example.com/deposit/callback",
		Extra: map[string]string{
			"type": "alipay",
		},
	})

	require.NoError(t, err)
	require.Contains(t, resp.PayURL, "sign=18880370e658cfa2cef0e872406385a5")
	require.Contains(t, resp.PayURL, "sign_type=MD5")
}

func TestEpayProviderVerifyCallbackAcceptsV1MD5(t *testing.T) {
	provider := NewEpayProvider("1001", "testkey123", "https://pay.example.com", "")

	params := map[string]string{
		"pid":          "1001",
		"trade_no":     "EPAY202604120001",
		"out_trade_no": "202604120001",
		"type":         "alipay",
		"name":         "BakaRay 账户充值",
		"money":        "12.34",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    "MD5",
	}
	params["sign"] = provider.generateSign(params)

	result, err := provider.VerifyCallback(params)
	require.NoError(t, err)
	require.Equal(t, "202604120001", result.TradeNo)
	require.Equal(t, int64(1234), result.Amount)
	require.Equal(t, "TRADE_SUCCESS", result.Status)
}
