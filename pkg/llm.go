package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"os"
	"time"
)

const (
	HistoryPath = "data/history.txt"
	SetPath     = "data/set.txt"
)

var (
	Model = "hunyuan-role" // 模型名称
)

// ChatRecord 单条对话记录（用于JSON序列化存储）
type ChatRecord struct {
	Role      string `json:"role"`      // user/assistant/system
	Content   string `json:"content"`   // 对话内容
	Timestamp string `json:"timestamp"` // 时间戳（便于追溯）
}

// 从文件读取历史对话（文件不存在则返回空列表）
func LoadChatHistory() ([]*hunyuan.Message, error) {
	// 检查文件是否存在
	if _, err := os.Stat(HistoryPath); os.IsNotExist(err) {
		// 读取设定文件内容
		set, err := os.ReadFile(SetPath)
		if err != nil {
			return nil, fmt.Errorf("读取设定文件失败：%w", err)
		}
		message := &hunyuan.Message{
			Role:    common.StringPtr("system"),
			Content: common.StringPtr(string(set)),
		}
		err = SaveChatHistory("system", string(set))
		return []*hunyuan.Message{message}, err // 首次使用，返回设定
	}

	// 读取文件内容
	data, err := os.ReadFile(HistoryPath)
	if err != nil {
		return nil, fmt.Errorf("读取历史对话失败：%w", err)
	}

	// 反序列化JSON到ChatRecord列表
	var records []ChatRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("解析历史对话JSON失败：%w", err)
	}

	// 转换为混元API要求的Message格式
	messages := make([]*hunyuan.Message, 0, len(records))
	for _, record := range records {
		messages = append(messages, &hunyuan.Message{
			Role:    common.StringPtr(record.Role),
			Content: common.StringPtr(record.Content),
		})
	}

	return messages, nil
}

// 追加新对话到历史文件（覆盖保存完整历史）
func SaveChatHistory(role, content string) error {
	// 读取现有历史
	records, err := loadChatRecords()
	if err != nil {
		return err
	}

	// 追加新记录（带时间戳）
	newRecord := ChatRecord{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	records = append(records, newRecord)

	// 序列化并写入文件
	data, err := json.MarshalIndent(records, "", "  ") // 格式化JSON，可读性强
	if err != nil {
		return fmt.Errorf("序列化对话记录失败：%w", err)
	}

	if err := os.WriteFile(HistoryPath, data, 0644); err != nil {
		return fmt.Errorf("写入历史对话文件失败：%w", err)
	}

	return nil
}

// loadChatRecords 辅助函数：读取ChatRecord列表（内部使用）
func loadChatRecords() ([]ChatRecord, error) {
	if _, err := os.Stat(HistoryPath); os.IsNotExist(err) {
		return []ChatRecord{}, nil
	}

	data, err := os.ReadFile(HistoryPath)
	if err != nil {
		return nil, err
	}

	var records []ChatRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// 带历史上下文的混元对话（传入当前用户提问，自动关联历史）
func Chat(userQuestion string) (assistantReply string, err error) {
	// 1. 读取密钥（从环境变量，安全无硬编码）
	credential := common.NewCredential(SecretId, SecretKey)

	// 2. 初始化客户端
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "hunyuan.tencentcloudapi.com"
	client, _ := hunyuan.NewClient(credential, "", cpf)

	// 3. 构造请求消息（历史对话 + 当前提问）
	// 3.1 读取历史对话
	historyMessages, err := LoadChatHistory()
	if err != nil {
		return "", fmt.Errorf("加载历史对话失败：%w", err)
	}
	// 3.2 追加当前用户提问
	currentMsg := &hunyuan.Message{
		Role:    common.StringPtr("user"),
		Content: common.StringPtr(userQuestion),
	}
	allMessages := append(historyMessages, currentMsg)

	// 4. 构建请求
	request := hunyuan.NewChatCompletionsRequest()
	request.Model = &Model
	request.Messages = allMessages

	// 5. 调用API并获取响应
	response, err := client.ChatCompletions(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		return "", fmt.Errorf("混元API调用错误：%s", err)
	}
	if err != nil {
		return "", fmt.Errorf("请求失败：%w", err)
	}

	// 6. 解析助手回复
	if response.Response == nil || response.Response.Choices == nil || len(response.Response.Choices) == 0 {
		return "", fmt.Errorf("未获取到助手回复")
	}
	assistantReply = *response.Response.Choices[0].Message.Content

	// 7. 保存对话到历史文件（用户提问 + 助手回复）
	if err := SaveChatHistory("user", userQuestion); err != nil {
		return "", fmt.Errorf("保存用户提问失败：%w", err)
	}
	if err := SaveChatHistory("assistant", assistantReply); err != nil {
		return "", fmt.Errorf("保存助手回复失败：%w", err)
	}

	return assistantReply, nil
}
