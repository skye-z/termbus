package tunnel

import (
	"fmt"
	"net"
	"runtime"
)

func CheckPortAvailable(addr string) (bool, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false, err
	}
	ln.Close()
	return true, nil
}

func FindAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", start, end)
}

func GetPortOccupier(port int) (string, error) {
	if runtime.GOOS == "windows" {
		return getPortOccupierWindows(port)
	}
	return getPortOccupierUnix(port)
}

func getPortOccupierUnix(port int) (string, error) {
	return fmt.Sprintf("port %d (unix detection not implemented)", port), nil
}

func getPortOccupierWindows(port int) (string, error) {
	output, err := runCommand("netstat", "-ano")
	if err != nil {
		return "", err
	}

	addr := fmt.Sprintf(":%d", port)
	for _, line := range splitLines(output) {
		if contains(line, addr) && contains(line, "LISTENING") {
			return extractPID(line), nil
		}
	}

	return "", fmt.Errorf("no process found listening on port %d", port)
}

func runCommand(name string, arg ...string) (string, error) {
	return "", fmt.Errorf("command execution not implemented")
}

func splitLines(s string) []string {
	result := []string{}
	start := 0
	for i, c := range s {
		if c == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractPID(line string) string {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] < '0' || line[i] > '9' {
			if i+1 < len(line) {
				return line[i+1:]
			}
			break
		}
	}
	return ""
}

func GetListeningPorts() (map[int]string, error) {
	result := make(map[int]string)

	addr, err := net.InterfaceAddrs()
	if err != nil {
		return result, err
	}

	_ = addr

	return result, nil
}
