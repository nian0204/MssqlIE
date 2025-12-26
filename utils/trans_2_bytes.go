package utils

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func HexToBytes(hex string) ([]byte, error) {
	if len(hex) == 0 {
		return nil, fmt.Errorf("空字符串")
	}
	if strings.Contains(hex, "0x") {
		hex = hex[2:]
	}
	if len(hex)%2 != 0 {
		return nil, fmt.Errorf("奇数长度的十六进制字符串")
	}
	bytes := make([]byte, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		hexByte := hex[i : i+2]
		var b byte
		n, err := fmt.Sscanf(hexByte, "%02x", &b)
		if err != nil {
			return nil, fmt.Errorf("无效的十六进制字符: %s", hex[i:i+2])
		}
		if n != 1 {
			return nil, fmt.Errorf("无效的十六进制字符: %s", hex[i:i+2])
		}
		bytes[i/2] = b
	}
	return bytes, nil
}
func Base64ToBytes(base64Str string) ([]byte, error) {
	if len(base64Str) == 0 {
		return nil, fmt.Errorf("空字符串")
	}
	base64Str = strings.TrimSpace(base64Str)
	bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		bytes, err = base64.URLEncoding.DecodeString(base64Str)
		if err != nil {
			return nil, fmt.Errorf("无效的Base64字符串: %w", err)
		}
	}
	return bytes, nil
}
