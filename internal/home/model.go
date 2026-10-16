package home

import "time"

type Options struct {
	Name         string
	Version      string
	Env          string
	StartedAt    time.Time
	SwaggerUI    bool
	SwaggerUIDir string
}

type infoResponse struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Env           string            `json:"env"`
	Status        string            `json:"status"`
	Checks        map[string]string `json:"checks"`
	StartedAt     string            `json:"started_at"`
	Uptime        string            `json:"uptime"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	Time          string            `json:"time"`
	Links         map[string]string `json:"links"`
}
