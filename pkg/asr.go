package pkg

import (
	"encoding/base64"
	"fmt"
	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"os"
)

var (
	EngineModelType        = "16k_zh" // 中文普通话通用引擎
	ChannelNum      uint64 = 1        // 单声道(16k音频仅支持单声道)
	ResTextFormat   uint64 = 0        // 0：基础识别结,仅包含有效人声时间戳
	SourceType      uint64 = 1        // 音频数据来源  0：音频URL； 1：音频数据（post body）
)

// 音频数据base64编码, 数据未进行base64编码时的长度
func getDataFromPcm() (data *string, l *uint64, err error) {
	// 读取PCM文件原始字节流（PCM为未压缩音频格式，直接读取原始数据）
	pcmFilePath := "data/input.wav"
	pcmData, err := os.ReadFile(pcmFilePath)
	for len(pcmData) == 0 {
		pcmData, err = os.ReadFile(pcmFilePath)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read PCM file: %w", err)
	}

	// 初始化返回值（避免nil指针解引用）
	var dataStr string
	var length uint64

	// 计算原始数据长度（字节）
	length = uint64(len(pcmData))

	// 对原始字节进行base64编码（API通常要求音频数据以base64格式传输）
	dataStr = base64.StdEncoding.EncodeToString(pcmData)

	// 返回指针（确保指针非nil）
	return &dataStr, &length, nil
}

func arsInit() *asr.Client {
	// 密钥信息
	credential := common.NewCredential(SecretId, SecretKey)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "asr.tencentcloudapi.com"

	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := asr.NewClient(credential, "", cpf)

	return client
}

// 录音文件识别请求
func CreateRecTask() (taskId *uint64, err error) {

	client := arsInit()

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := asr.NewCreateRecTaskRequest()
	request.EngineModelType = &EngineModelType
	request.ChannelNum = &ChannelNum
	request.ResTextFormat = &ResTextFormat
	request.SourceType = &SourceType
	request.Data, request.DataLen, err = getDataFromPcm()
	if err != nil {
		return
	}

	// 返回的resp是一个CreateRecTaskResponse的实例，与请求对象对应
	response, err := client.CreateRecTask(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		err = fmt.Errorf("an API error has returned: %s", err)
		return
	}
	if err != nil {
		return
	}

	// 返回taskId
	taskId = response.Response.Data.TaskId
	return
}

// 录音文件识别结果查询
func DescribeTaskStatus(taskId *uint64) (status *int64, data *string, err error) {
	// 实例化要请求产品的client对象
	client := arsInit()

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := asr.NewDescribeTaskStatusRequest()
	request.TaskId = taskId

	// 返回的resp是一个DescribeTaskStatusResponse的实例，与请求对象对应
	response, err := client.DescribeTaskStatus(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		err = fmt.Errorf("an API error has returned: %s", err)
		return
	}
	if err != nil {
		return
	}
	// 返回结果
	status = response.Response.Data.Status
	data = response.Response.Data.Result
	if *status == 3 {
		err = fmt.Errorf("an response error has returned: %s", *response.Response.Data.ErrorMsg)
	}
	return
}
