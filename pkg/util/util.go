package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// GenerateID generates a random hexadecimal ID of the specified length
func GenerateID(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	id := hex.EncodeToString(bytes)
	if len(id) > length {
		id = id[:length]
	}

	return id, nil
}

// GenerateShortID generates a short random ID (8 characters)
func GenerateShortID() string {
	id, _ := GenerateID(8)
	return id
}

// GenerateLongID generates a long random ID (32 characters)
func GenerateLongID() string {
	id, _ := GenerateID(32)
	return id
}

// ValidateEmail validates an email address using a simple regex
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// SanitizeString removes potentially dangerous characters from a string
func SanitizeString(input string) string {
	// Remove non-printable characters
	result := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, input)

	// Trim whitespace
	result = strings.TrimSpace(result)

	return result
}

// TruncateString truncates a string to the specified length and adds ellipsis if needed
func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}

	if maxLength <= 3 {
		return strings.Repeat(".", maxLength)
	}

	return input[:maxLength-3] + "..."
}

// Contains checks if a slice contains a specific element
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate elements from a slice
func RemoveDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ChunkSlice splits a slice into chunks of the specified size
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		return nil
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// MapKeys returns the keys of a map as a slice
func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// MapValues returns the values of a map as a slice
func MapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// Retry executes a function with retry logic
func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", attempts, err)
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(attempts int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	var err error
	delay := initialDelay

	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	return fmt.Errorf("failed after %d attempts with backoff: %w", attempts, err)
}

// CoalesceString returns the first non-empty string from the arguments
func CoalesceString(strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}
	return ""
}

// CoalesceInt returns the first non-zero integer from the arguments
func CoalesceInt(ints ...int) int {
	for _, i := range ints {
		if i != 0 {
			return i
		}
	}
	return 0
}

// ParseSize parses a human-readable size string (e.g., "1KB", "2MB", "3GB")
func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	units := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	// Check for unit suffixes in order of preference (longest first)
	unitOrder := []string{"TB", "GB", "MB", "KB", "B"}
	for _, unit := range unitOrder {
		if strings.HasSuffix(sizeStr, unit) {
			numberStr := strings.TrimSuffix(sizeStr, unit)
			if numberStr == "" {
				return 0, fmt.Errorf("invalid size format: %s", sizeStr)
			}

			number := big.NewInt(0)
			if _, ok := number.SetString(numberStr, 10); !ok {
				return 0, fmt.Errorf("invalid number in size: %s", numberStr)
			}

			if number.Sign() < 0 {
				return 0, fmt.Errorf("negative size not allowed: %s", sizeStr)
			}

			multiplier := units[unit]
			result := new(big.Int).Mul(number, big.NewInt(multiplier))
			if !result.IsInt64() {
				return 0, fmt.Errorf("size too large: %s", sizeStr)
			}

			return result.Int64(), nil
		}
	}

	// If no unit is specified, assume bytes
	number := big.NewInt(0)
	if _, ok := number.SetString(sizeStr, 10); !ok {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	if !number.IsInt64() {
		return 0, fmt.Errorf("size too large: %s", sizeStr)
	}

	return number.Int64(), nil
}

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "0B"
	}

	units := []struct {
		suffix string
		size   int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, unit := range units {
		if bytes >= unit.size {
			value := float64(bytes) / float64(unit.size)
			if unit.suffix == "B" || value >= 10 {
				return fmt.Sprintf("%.0f%s", value, unit.suffix)
			}
			return fmt.Sprintf("%.1f%s", value, unit.suffix)
		}
	}

	return "0B"
}

// IsValidPort checks if a port number is valid
func IsValidPort(port int) bool {
	return port >= 1 && port <= 65535
}

// IsValidIPAddress checks if a string is a valid IP address (IPv4 or IPv6)
func IsValidIPAddress(ip string) bool {
	if ip == "" {
		return false
	}

	// Simple validation - in production, use net.ParseIP
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if ipv4Regex.MatchString(ip) {
		parts := strings.Split(ip, ".")
		for _, part := range parts {
			num := big.NewInt(0)
			if _, ok := num.SetString(part, 10); !ok {
				return false
			}
			if num.Cmp(big.NewInt(255)) > 0 {
				return false
			}
		}
		return true
	}

	// Basic IPv6 check (simplified)
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}$`)
	return ipv6Regex.MatchString(ip)
}

// MergeStringMaps merges multiple string maps, with later maps overriding earlier ones
func MergeStringMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// DeepCopy creates a deep copy of a value using reflection
func DeepCopy[T any](src T) (T, error) {
	var dst T

	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(&dst).Elem()

	if err := deepCopyRecursive(srcValue, dstValue); err != nil {
		return dst, fmt.Errorf("deep copy failed: %w", err)
	}

	return dst, nil
}

func deepCopyRecursive(src, dst reflect.Value) error {
	if !src.IsValid() {
		return nil
	}

	switch src.Kind() {
	case reflect.Ptr:
		if src.IsNil() {
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(src.Type().Elem()))
		}
		return deepCopyRecursive(src.Elem(), dst.Elem())

	case reflect.Interface:
		if src.IsNil() {
			return nil
		}
		originalValue := src.Elem()
		copyValue := reflect.New(originalValue.Type()).Elem()
		if err := deepCopyRecursive(originalValue, copyValue); err != nil {
			return err
		}
		dst.Set(copyValue)

	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if err := deepCopyRecursive(src.Field(i), dst.Field(i)); err != nil {
				return err
			}
		}

	case reflect.Slice:
		if src.IsNil() {
			return nil
		}
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Cap()))
		for i := 0; i < src.Len(); i++ {
			if err := deepCopyRecursive(src.Index(i), dst.Index(i)); err != nil {
				return err
			}
		}

	case reflect.Map:
		if src.IsNil() {
			return nil
		}
		dst.Set(reflect.MakeMap(src.Type()))
		for _, key := range src.MapKeys() {
			originalValue := src.MapIndex(key)
			copyKey := reflect.New(key.Type()).Elem()
			copyValue := reflect.New(originalValue.Type()).Elem()
			if err := deepCopyRecursive(key, copyKey); err != nil {
				return err
			}
			if err := deepCopyRecursive(originalValue, copyValue); err != nil {
				return err
			}
			dst.SetMapIndex(copyKey, copyValue)
		}

	default:
		dst.Set(src)
	}

	return nil
}

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the maximum of two integers
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ClampInt clamps an integer value between min and max
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
