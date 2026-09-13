package file

import "os"

const defaultConfig = `server:
  port: "8080"
  security:
    allowed_methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS]

upstream:
  - name: "server_runner"
    path: /
    server_url: "http://localhost:3000"
`

func CreateConfig() error {
	if err := os.MkdirAll("leoxy", 0755); err != nil {
		return err
	}

	configPath := "leoxy/config.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}
