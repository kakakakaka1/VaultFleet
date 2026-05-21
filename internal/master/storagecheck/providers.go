package storagecheck

import (
	"encoding/json"
	"os/exec"
	"sync"
)

type S3Provider struct {
	Value string `json:"value"`
	Help  string `json:"help"`
}

func ParseS3Providers(data []byte) ([]S3Provider, error) {
	var result struct {
		Providers []struct {
			Name    string `json:"Name"`
			Options []struct {
				Name     string `json:"Name"`
				Examples []struct {
					Value string `json:"Value"`
					Help  string `json:"Help"`
				} `json:"Examples"`
			} `json:"Options"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	for _, backend := range result.Providers {
		if backend.Name != "s3" {
			continue
		}
		for _, opt := range backend.Options {
			if opt.Name != "provider" {
				continue
			}
			providers := make([]S3Provider, len(opt.Examples))
			for i, ex := range opt.Examples {
				providers[i] = S3Provider{Value: ex.Value, Help: ex.Help}
			}
			return providers, nil
		}
	}
	return nil, nil
}

type ProviderLoader struct {
	RunFunc func() ([]byte, error)

	once      sync.Once
	cached    []S3Provider
	cachedErr error
}

func NewProviderLoader() *ProviderLoader {
	return &ProviderLoader{
		RunFunc: defaultRcloneProviders,
	}
}

func (l *ProviderLoader) Load() ([]S3Provider, error) {
	l.once.Do(func() {
		data, err := l.RunFunc()
		if err != nil {
			l.cachedErr = err
			return
		}
		l.cached, l.cachedErr = ParseS3Providers(data)
	})
	return l.cached, l.cachedErr
}

func defaultRcloneProviders() ([]byte, error) {
	return exec.Command("rclone", "config", "providers").Output()
}
