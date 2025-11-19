package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/maplerime/cl-query/pkg/common"
	"io"
	"net/http"
	"sync"
	"time"
)

type WdsAdmin struct{}

const (
	WdsTokenExpireTime = 10 * time.Minute
	WdsLoginURL        = "/api/v1/login"
	WdsServerListURL   = "/manager/common/v3/server/list"
	WdsVolumeListURL   = "/manager/block/v3/volume/list"
	WdsPoolListURL     = "/manager/osd/v3/pool/list"
)

var (
	wdsClient     *WdsClient
	wdsClientOnce = sync.Once{}
)

type WdsClient struct {
	httpClient    *http.Client
	loginLock     *sync.Mutex
	token         string
	tokenCreateAt time.Time
}

type WdsServerListResponse struct {
	Data struct {
		TotalCount int64 `json:"totalCount"`
		List       []struct {
			ID string `json:"id"`
		} `json:"list"`
	} `json:"data"`
}

type WdsPoolListResponse struct {
	Data struct {
		TotalCount int64 `json:"totalCount"`
		List       []struct {
			ClusterName      string `json:"cluster_name"`
			PhySize          uint64 `json:"phy_size"`
			PhyUsedSize      uint64 `json:"phy_used_size"`
			ReplicateSize    uint64 `json:"replicate_size"`
			VolumeSizeSum    uint64 `json:"volume_size_sum"`
			ThinProvisioning uint64 `json:"thin_provisioning"`
		}
		RawSize      uint64 `json:"raw_size"`
		RealDataSize uint64 `json:"real_data_size"`
	} `json:"data"`
}

func (c *WdsClient) init() (err error) {
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	c.loginLock = &sync.Mutex{}
	return
}

func (c *WdsClient) Login(originReq *http.Request) (err error) {
	c.loginLock.Lock()
	defer c.loginLock.Unlock()

	// check token
	if c.token != "" && time.Now().Sub(c.tokenCreateAt) < WdsTokenExpireTime {
		logger.Debugf("Use cached token: %s", c.token)
		originReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
		return
	}

	// get token
	logger.Debugf("last token is empty or expired, get token")
	c.tokenCreateAt = time.Now()
	payload := map[string]string{
		"name":     common.Config.Wds.Username,
		"password": common.Config.Wds.Password,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("Marshal login payload failed: %+v", err)
		err = fmt.Errorf("marshal login payload failed: %w", err)
		return
	}
	urlStr := fmt.Sprintf("%s%s", common.Config.Wds.Endpoint, WdsLoginURL)
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Errorf("New http request failed: %+v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Read http response body failed: %+v", err)
		return
	}
	logger.Debugf("Login response: %s", string(body))

	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err = json.Unmarshal(body, &loginResp); err != nil {
		logger.Errorf("Unmarshal http response body failed: %+v", err)
		return
	}

	c.token = loginResp.AccessToken
	originReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	return
}

func (c *WdsClient) DoRequest(method, path string, payload any) (resp *http.Response, err error) {
	urlStr := fmt.Sprintf("%s%s", common.Config.Wds.Endpoint, path)
	var payloadBytes []byte
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			logger.Errorf("Marshal payload failed: %+v", err)
			return
		}
	}
	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Errorf("New http request failed: %+v", err)
		return
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// login
	err = c.Login(req)
	if err != nil {
		logger.Errorf("Login failed: %+v", err)
		return
	}
	logger.Debugf("Do request: %s %s", req.Method, urlStr)

	req.Header.Set("Region", "default")
	req.Header.Set("Az", "default")

	resp, err = c.httpClient.Do(req)

	if err != nil {
		logger.Errorf("Do request failed: %+v", err)
		return
	}

	return
}

func (a *WdsAdmin) Request(method, path string, payload any) (body []byte, err error) {
	wdsClientOnce.Do(func() {
		wdsClient = &WdsClient{}
		if initErr := wdsClient.init(); initErr != nil {
			err = initErr
		}
	})
	if err != nil {
		logger.Debugf("Init wds client failed: %+v", err)
		return
	}
	response, err := wdsClient.DoRequest(method, path, payload)
	if err != nil {
		logger.Errorf("Do request failed: %+v", err)
		return
	}
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("Read response body failed: %+v", err)
		return
	}
	logger.Debugf("wds response body: %s", string(body))
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		logger.Errorf("Request Status: %s", response.Status)
		var wdsError struct {
			Message string `json:"msg"`
			RetCode string `json:"code"`
		}
		err = json.Unmarshal(body, &wdsError)
		if err != nil {
			logger.Errorf("Unmarshal error response failed: %+v", err)
			return
		}
		err = fmt.Errorf("%s", wdsError.Message)
		return
	}
	return
}

func (a *WdsAdmin) GetServers(serverType string) (resp *WdsServerListResponse, err error) {
	path := fmt.Sprintf("%s?page_size=1", WdsServerListURL)
	if serverType != "" {
		path += fmt.Sprintf("&server_type=%s", serverType)
	}
	body, err := a.Request("GET", path, nil)
	if err != nil {
		logger.Errorf("Request wds server list failed: %+v", err)
		return
	}
	resp = &WdsServerListResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		logger.Errorf("Unmarshal wds server list response failed: %+v", err)
		return
	}
	return
}

func (a *WdsAdmin) GetVolumeCount() (count int64, err error) {
	path := fmt.Sprintf("%s?page_size=1", WdsVolumeListURL)
	body, err := a.Request("GET", path, nil)
	if err != nil {
		logger.Errorf("Request wds volume list failed: %+v", err)
		return
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logger.Errorf("Unmarshal wds volume list response failed: %+v", err)
		return
	}
	count = int64(resp.Data["totalCount"].(float64))
	return
}

func (a *WdsAdmin) GetPools() (resp *WdsPoolListResponse, err error) {
	path := fmt.Sprintf("%s?page_size=100&need_size=true", WdsPoolListURL)
	body, err := a.Request("GET", path, nil)
	if err != nil {
		logger.Errorf("Request wds server list failed: %+v", err)
		return
	}
	resp = &WdsPoolListResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		logger.Errorf("Unmarshal wds pool list response failed: %+v", err)
		return
	}
	return
}
