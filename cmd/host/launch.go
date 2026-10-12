// 2026-10-12 Windows 本机演示启动：找配置、打开浏览器、缺省 JWT 占位符。
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cznUSER/B-HIS/internal/config"
)

func discoverConfig(explicit string) string {
	if p := strings.TrimSpace(explicit); p != "" {
		return p
	}
	if p := strings.TrimSpace(os.Getenv("BHIS_CONFIG")); p != "" {
		return p
	}
	dirs := []string{"."}
	if d := executableDir(); d != "" {
		dirs = append(dirs, d)
	}
	return searchConfig(dirs)
}

func searchConfig(dirs []string) string {
	for _, name := range []string{"config.local.yaml", "config.yaml"} {
		if p := firstFile(dirs, name); p != "" {
			return p
		}
	}
	return firstFile(dirs, "config.example.yaml")
}

func firstFile(dirs []string, name string) string {
	for _, dir := range dirs {
		p := filepath.Join(dir, name)
		if isFile(p) {
			return p
		}
	}
	return ""
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func executableDir() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return filepath.Dir(p)
}

// loadDemo loads config. With no JWT secret it uses the documented demo placeholder
// so a double-clicked exe can start. It does not invent a DSN.
func loadDemo(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if !config.IsMissingJWT(err) || strings.TrimSpace(os.Getenv("BHIS_JWT_SECRET")) != "" {
		return config.Config{}, err
	}
	if err := os.Setenv("BHIS_JWT_SECRET", config.DemoJWTPlaceholder); err != nil {
		return config.Config{}, err
	}
	return config.Load(path)
}

func loginURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return "http://" + addr + "/login"
}

func browserEnabled(noBrowser bool) bool {
	if noBrowser {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BHIS_OPEN_BROWSER"))) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
