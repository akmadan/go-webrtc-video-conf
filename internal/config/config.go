package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	CORS     CORSConfig
	Security SecurityConfig
	Signaling SignalingConfig
	WebRTC   WebRTCConfig
	Redis    RedisConfig
	Limits   LimitsConfig
	LogLevel string
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
	Host string
}

// CORSConfig holds CORS-related configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// SecurityConfig holds authentication and transport security config.
type SecurityConfig struct {
	SignalingToken string
}

// SignalingConfig holds signaling behavior controls.
type SignalingConfig struct {
	PublishEventsToLog bool
}

// WebRTCConfig holds STUN/TURN settings for clients.
type WebRTCConfig struct {
	STUNURLs     []string
	TURNEnabled  bool
	TURNURLs     []string
	TURNUsername string
	TURNPassword string
}

// ICEServer represents a WebRTC ICE server payload.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// RedisConfig holds Redis bus settings for cross-instance signaling.
type RedisConfig struct {
	Enabled bool
	Addr    string
	Password string
	DB      int
	Channel string
}

// LimitsConfig holds runtime limits to reduce abuse and protect server health.
type LimitsConfig struct {
	MaxPeersPerRoom     int
	MaxHTTPPerMinute    int
	MaxWSMessagesPerSec int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	host := getEnv("HOST", "localhost")
	logLevel := getEnv("LOG_LEVEL", "info")
	signalingToken := getEnv("SIGNALING_TOKEN", "")

	allowedOrigins := getEnvSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000"})
	maxPeersPerRoom := getEnvInt("MAX_PEERS_PER_ROOM", 8)
	maxHTTPPerMinute := getEnvInt("MAX_HTTP_PER_MINUTE", 300)
	maxWSMessagesPerSec := getEnvInt("MAX_WS_MESSAGES_PER_SEC", 30)
	publishEventsToLog := getEnvBool("PUBLISH_SIGNALING_EVENTS_TO_LOG", false)
	stunURLs := getEnvSlice("STUN_URLS", []string{"stun:stun.l.google.com:19302"})
	turnEnabled := getEnvBool("TURN_ENABLED", false)
	turnURLs := getEnvSlice("TURN_URLS", []string{})
	turnUsername := getEnv("TURN_USERNAME", "")
	turnPassword := getEnv("TURN_PASSWORD", "")
	redisEnabled := getEnvBool("REDIS_ENABLED", false)
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	redisChannel := getEnv("REDIS_CHANNEL", "go-webrtc-video-conf:events")

	return &Config{
		Server: ServerConfig{
			Port: port,
			Host: host,
		},
		CORS: CORSConfig{
			AllowedOrigins: allowedOrigins,
		},
		Security: SecurityConfig{
			SignalingToken: signalingToken,
		},
		Signaling: SignalingConfig{
			PublishEventsToLog: publishEventsToLog,
		},
		WebRTC: WebRTCConfig{
			STUNURLs:     stunURLs,
			TURNEnabled:  turnEnabled,
			TURNURLs:     turnURLs,
			TURNUsername: turnUsername,
			TURNPassword: turnPassword,
		},
		Redis: RedisConfig{
			Enabled: redisEnabled,
			Addr: redisAddr,
			Password: redisPassword,
			DB: redisDB,
			Channel: redisChannel,
		},
		Limits: LimitsConfig{
			MaxPeersPerRoom:     maxPeersPerRoom,
			MaxHTTPPerMinute:    maxHTTPPerMinute,
			MaxWSMessagesPerSec: maxWSMessagesPerSec,
		},
		LogLevel: logLevel,
	}, nil
}

// GetICEServers returns ICE servers based on runtime configuration.
func (c *Config) GetICEServers() []ICEServer {
	servers := make([]ICEServer, 0, 2)
	if len(c.WebRTC.STUNURLs) > 0 {
		servers = append(servers, ICEServer{
			URLs: c.WebRTC.STUNURLs,
		})
	}
	if c.WebRTC.TURNEnabled && len(c.WebRTC.TURNURLs) > 0 {
		servers = append(servers, ICEServer{
			URLs:       c.WebRTC.TURNURLs,
			Username:   c.WebRTC.TURNUsername,
			Credential: c.WebRTC.TURNPassword,
		})
	}
	return servers
}

// GetAddress returns the full server address
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvSlice retrieves an environment variable as a slice, splitting by comma
func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Split by comma and trim spaces
	var result []string
	parts := strings.Split(value, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	
	if len(result) == 0 {
		return defaultValue
	}
	
	return result
}

// getEnvInt retrieves an environment variable as an integer or returns a default value
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

