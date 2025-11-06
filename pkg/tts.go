package pkg

import (
	"encoding/base64"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
	"os"
	"time"
)

func TextToVoice(text string) (err error) {
	// 密钥信息
	credential := common.NewCredential(SecretId, SecretKey)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tts.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := tts.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := tts.NewTextToVoiceRequest()
	request.VoiceType = common.Int64Ptr(601005)
	request.Text = common.StringPtr(text)
	request.SessionId = common.StringPtr(string(time.Now().UnixNano()))

	// 返回的resp是一个TextToVoiceResponse的实例，与请求对象对应
	response, err := client.TextToVoice(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		err = fmt.Errorf("an API error has returned: %s", err)
		return
	}
	if err != nil {
		panic(err)
	}
	// 解析响应：base64编码的音频数据 → 解码 → 写入文件
	if response.Response == nil || response.Response.Audio == nil {
		return fmt.Errorf("未获取到合成音频数据")
	}
	// 解码base64音频数据
	audioBase64 := *response.Response.Audio
	audioData, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return fmt.Errorf("base64解码音频失败：%w", err)
	}
	// 写入output.wav文件（权限0644：所有者读写，其他只读）
	if err := os.WriteFile("data/output.wav", audioData, 0644); err != nil {
		return fmt.Errorf("写入音频文件失败：%w", err)
	}

	return
}
