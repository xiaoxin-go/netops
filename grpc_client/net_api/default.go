package net_api

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"netops/grpc_client/protobuf/net_api"
	"time"
)

func NewClient(apiServer string) *Client {
	client := &Client{ApiServer: apiServer, timeout: time.Minute * 15, recvSize: 1024 * 1024 * 100}
	client.init()
	return client
}

type Client struct {
	ApiServer string
	client    net_api.DeviceClient
	conn      *grpc.ClientConn
	timeout   time.Duration
	recvSize  int
	Err       error
}

func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) init() {
	conn, err := grpc.Dial(
		c.ApiServer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(c.recvSize)),
	)
	if err != nil {
		c.Err = fmt.Errorf("连接后端设备服务器异常: %s", err.Error())
		return
	}
	//defer conn.Close()
	c.conn = conn
	c.client = net_api.NewDeviceClient(conn)
}

func (c *Client) Show(requestData *net_api.ConfigRequest) ([]*net_api.Command, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	resp, err := c.client.Show(ctx, requestData)
	if err != nil {
		return nil, fmt.Errorf("获取网络配置失败, err: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("获取网络配置失败, msg: %s", resp.Message)
	}
	if len(resp.Results) == 0 {
		return nil, errors.New("获取网络配置为空")
	}
	return resp.Results, nil
}
func (c *Client) Config(request *net_api.ConfigRequest) ([]*net_api.Command, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	l := zap.L().With(zap.String("func", "Config"), zap.String("apiServer", c.ApiServer))
	l.Info("send grpc config--->", zap.Any("commands", request.Commands), zap.String("host", request.Host))
	zap.L().Debug(fmt.Sprintf("apiServer: <%s>", c.ApiServer))
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	res, err := c.client.Config(ctx, request)
	if err != nil {
		l.Error("send failed", zap.Error(err))
		return nil, fmt.Errorf("配置失败, err: %w", err)
	}
	if res.Code != 0 {
		l.Error("send failed", zap.String("message", res.Message))
		return nil, fmt.Errorf("调用GRPC Config失败: <%s>", res.Message)
	}
	return res.Results, nil
}
func (c *Client) Http(request *net_api.HttpRequest) (string, error) {
	if c.Err != nil {
		return "", c.Err
	}
	l := zap.L().With(zap.String("func", "Http"), zap.String("apiServer", c.ApiServer))
	l.Info("send grpc http--->", zap.Any("params", request.Params), zap.String("url", request.Url))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := c.client.Http(ctx, request)
	if err != nil {
		l.Error("send grpc http failed", zap.Error(err))
		return "", fmt.Errorf("send grpc http failed, err: %w", err)
	}
	if res.Code != 0 {
		l.Error("send grpc http failed", zap.String("message", res.Message))
		return "", fmt.Errorf("send grpc http failed, msg: %s", res.Message)
	}
	return res.Message, nil
}
