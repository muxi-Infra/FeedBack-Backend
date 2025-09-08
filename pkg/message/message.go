package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	TestUrl = "http://127.0.0.1:3000/send_group_msg" // 测试用的 URL
	ProdURL = "http://127.0.0.1:3000/send_group_msg" // 生产用的 URL
	GroupID = int64(740450762)
)

// SendGroupMessage 发送群消息的封装函数
func SendGroupMessage(url string, groupID int64, message []MessageItem) (interface{}, error) {
	// 创建请求体
	requestBody := GroupMessageRequest{
		GroupID: groupID,
		Message: message,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	var result map[string]interface{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应JSON失败: %v", err)
	}
	return result, nil
}
