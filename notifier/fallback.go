package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
	gresty "github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

var errBlockChainHTTPError = errors.New("block chain http error")

type NotifyClient struct {
	client *gresty.Client
}

func NewNotifyClient(baseUrl string) (*NotifyClient, error) {
	if baseUrl == "" {
		return nil, fmt.Errorf("blockchain URl connot be empty")
	}

	client := gresty.New()
	client.SetBaseURL(baseUrl)
	client.OnAfterResponse(func(client *gresty.Client, response *gresty.Response) error {
		statusCode := response.StatusCode()
		if statusCode >= http.StatusBadRequest {
			method := response.Request.Method
			url := response.Request.URL
			return fmt.Errorf("%d cannot %s %s: %w", statusCode, method, url, errBlockChainHTTPError)
		}
		return nil
	})

	return &NotifyClient{client: client}, nil
}

func (nc *NotifyClient) BusinessNotify(notifyData *NotifyRequest) (bool, error) {
	body, err := json.Marshal(notifyData)
	if err != nil {
		log.Error("failed to marshal notify data", "err", err)
		return false, err
	}

	res, err := nc.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		SetResult(&NotifyResponse{}).Post("/dapplink/notify")
	if err != nil {
		log.Error("notify http request failed ", "err", err)
		return false, err
	}
	spt, ok := res.Result().(*NotifyResponse)
	if !ok {
		return false, fmt.Errorf("notify response is not of type *NotifyResponse")
	}
	return spt.Success, nil
}
