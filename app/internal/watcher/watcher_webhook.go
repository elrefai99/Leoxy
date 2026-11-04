package watcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func Notify(client *http.Client, target string, report Report) error {
	body, err := json.Marshal(report)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	var notifyErr error
	if response.StatusCode >= http.StatusMultipleChoices {
		notifyErr = fmt.Errorf("webhook returned http %d", response.StatusCode)
	}
	return errors.Join(notifyErr, response.Body.Close())
}
