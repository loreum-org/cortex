package util

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantErr    bool
		wantLength int
	}{
		{"valid length 8", 8, false, 8},
		{"valid length 16", 16, false, 16},
		{"valid length 32", 32, false, 32},
		{"zero length", 0, true, 0},
		{"negative length", -1, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenerateID(tt.length)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateID() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateID() unexpected error: %v", err)
				return
			}

			if len(id) != tt.wantLength {
				t.Errorf("GenerateID() length = %d, want %d", len(id), tt.wantLength)
			}

			// Check if it's valid hex
			for _, char := range id {
				if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
					t.Errorf("GenerateID() contains invalid hex character: %c", char)
				}
			}
		})
	}
}

func TestGenerateShortID(t *testing.T) {
	id := GenerateShortID()
	if len(id) != 8 {
		t.Errorf("GenerateShortID() length = %d, want 8", len(id))
	}
}

func TestGenerateLongID(t *testing.T) {
	id := GenerateLongID()
	if len(id) != 32 {
		t.Errorf("GenerateLongID() length = %d, want 32", len(id))
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"", false},
		{"invalid-email", false},
		{"@example.com", false},
		{"test@", false},
		{"test.example.com", false},
		{"test@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			if result != tt.valid {
				t.Errorf("ValidateEmail(%s) = %v, want %v", tt.email, result, tt.valid)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal string", "normal string"},
		{"  trimmed  ", "trimmed"},
		{"string\nwith\tspecial", "stringwithspecial"},
		{"string\x00with\x01control", "stringwithcontrol"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input     string
		maxLength int
		expected  string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a long string", 10, "this is..."},
		{"test", 5, "test"},
		{"test", 2, ".."},
		{"test", 1, "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLength, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		if !Contains(slice, "b") {
			t.Errorf("Contains() should find 'b' in slice")
		}
		if Contains(slice, "d") {
			t.Errorf("Contains() should not find 'd' in slice")
		}
	})

	t.Run("int slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		if !Contains(slice, 2) {
			t.Errorf("Contains() should find 2 in slice")
		}
		if Contains(slice, 4) {
			t.Errorf("Contains() should not find 4 in slice")
		}
	})
}

func TestRemoveDuplicates(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b"}
		expected := []string{"a", "b", "c"}
		result := RemoveDuplicates(input)

		if len(result) != len(expected) {
			t.Errorf("RemoveDuplicates() length = %d, want %d", len(result), len(expected))
		}

		for _, item := range expected {
			if !Contains(result, item) {
				t.Errorf("RemoveDuplicates() missing expected item: %s", item)
			}
		}
	})

	t.Run("int slice", func(t *testing.T) {
		input := []int{1, 2, 1, 3, 2}
		result := RemoveDuplicates(input)
		expected := 3

		if len(result) != expected {
			t.Errorf("RemoveDuplicates() length = %d, want %d", len(result), expected)
		}
	})
}

func TestChunkSlice(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		chunkSize int
		expected  [][]int
	}{
		{
			"normal chunking",
			[]int{1, 2, 3, 4, 5, 6},
			2,
			[][]int{{1, 2}, {3, 4}, {5, 6}},
		},
		{
			"uneven chunking",
			[]int{1, 2, 3, 4, 5},
			2,
			[][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			"chunk size larger than slice",
			[]int{1, 2},
			5,
			[][]int{{1, 2}},
		},
		{
			"empty slice",
			[]int{},
			2,
			nil,
		},
		{
			"zero chunk size",
			[]int{1, 2, 3},
			0,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChunkSlice(tt.slice, tt.chunkSize)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ChunkSlice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMapKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := MapKeys(m)

	if len(keys) != 3 {
		t.Errorf("MapKeys() length = %d, want 3", len(keys))
	}

	for _, key := range []string{"a", "b", "c"} {
		if !Contains(keys, key) {
			t.Errorf("MapKeys() missing key: %s", key)
		}
	}
}

func TestMapValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	values := MapValues(m)

	if len(values) != 3 {
		t.Errorf("MapValues() length = %d, want 3", len(values))
	}

	for _, value := range []int{1, 2, 3} {
		if !Contains(values, value) {
			t.Errorf("MapValues() missing value: %d", value)
		}
	}
}

func TestRetry(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		err := Retry(3, time.Millisecond, func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("Retry() unexpected error: %v", err)
		}

		if attempts != 1 {
			t.Errorf("Retry() attempts = %d, want 1", attempts)
		}
	})

	t.Run("success on third attempt", func(t *testing.T) {
		attempts := 0
		err := Retry(3, time.Millisecond, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Retry() unexpected error: %v", err)
		}

		if attempts != 3 {
			t.Errorf("Retry() attempts = %d, want 3", attempts)
		}
	})

	t.Run("failure after max attempts", func(t *testing.T) {
		attempts := 0
		err := Retry(2, time.Millisecond, func() error {
			attempts++
			return errors.New("persistent error")
		})

		if err == nil {
			t.Errorf("Retry() expected error but got none")
		}

		if attempts != 2 {
			t.Errorf("Retry() attempts = %d, want 2", attempts)
		}
	})
}

func TestCoalesceString(t *testing.T) {
	tests := []struct {
		inputs   []string
		expected string
	}{
		{[]string{"first", "second"}, "first"},
		{[]string{"", "second"}, "second"},
		{[]string{"", "", "third"}, "third"},
		{[]string{"", ""}, ""},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		result := CoalesceString(tt.inputs...)
		if result != tt.expected {
			t.Errorf("CoalesceString(%v) = %q, want %q", tt.inputs, result, tt.expected)
		}
	}
}

func TestCoalesceInt(t *testing.T) {
	tests := []struct {
		inputs   []int
		expected int
	}{
		{[]int{1, 2}, 1},
		{[]int{0, 2}, 2},
		{[]int{0, 0, 3}, 3},
		{[]int{0, 0}, 0},
		{[]int{}, 0},
	}

	for _, tt := range tests {
		result := CoalesceInt(tt.inputs...)
		if result != tt.expected {
			t.Errorf("CoalesceInt(%v) = %d, want %d", tt.inputs, result, tt.expected)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"100", 100, false},
		{"1B", 1, false},
		{"1KB", 1024, false},
		{"2MB", 2 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"KB", 0, true},
		{"-1KB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSize(%q) expected error but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseSize(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0B"},
		{1, "1B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{10240, "10KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
		{-1, "0B"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.input)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		port  int
		valid bool
	}{
		{1, true},
		{80, true},
		{8080, true},
		{65535, true},
		{0, false},
		{-1, false},
		{65536, false},
		{100000, false},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.port)), func(t *testing.T) {
			result := IsValidPort(tt.port)
			if result != tt.valid {
				t.Errorf("IsValidPort(%d) = %v, want %v", tt.port, result, tt.valid)
			}
		})
	}
}

func TestIsValidIPAddress(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"127.0.0.1", true},
		{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"::1", true},
		{"", false},
		{"256.1.1.1", false},
		{"192.168.1", false},
		{"192.168.1.1.1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsValidIPAddress(tt.ip)
			if result != tt.valid {
				t.Errorf("IsValidIPAddress(%q) = %v, want %v", tt.ip, result, tt.valid)
			}
		})
	}
}

func TestMergeStringMaps(t *testing.T) {
	map1 := map[string]string{"a": "1", "b": "2"}
	map2 := map[string]string{"b": "3", "c": "4"}
	map3 := map[string]string{"c": "5", "d": "6"}

	result := MergeStringMaps(map1, map2, map3)

	expected := map[string]string{
		"a": "1",
		"b": "3",
		"c": "5",
		"d": "6",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("MergeStringMaps() = %v, want %v", result, expected)
	}
}

func TestDeepCopy(t *testing.T) {
	type TestStruct struct {
		Name   string
		Values []int
		Nested map[string]string
	}

	original := TestStruct{
		Name:   "test",
		Values: []int{1, 2, 3},
		Nested: map[string]string{"key": "value"},
	}

	copied, err := DeepCopy(original)
	if err != nil {
		t.Errorf("DeepCopy() unexpected error: %v", err)
		return
	}

	// Verify deep copy
	if !reflect.DeepEqual(original, copied) {
		t.Errorf("DeepCopy() result not equal to original")
	}

	// Modify original to ensure it's a deep copy
	original.Values[0] = 999
	original.Nested["key"] = "modified"

	if copied.Values[0] == 999 {
		t.Errorf("DeepCopy() was not deep - slice was affected")
	}

	if copied.Nested["key"] == "modified" {
		t.Errorf("DeepCopy() was not deep - map was affected")
	}
}

func TestMinMaxClampInt(t *testing.T) {
	if MinInt(5, 3) != 3 {
		t.Errorf("MinInt(5, 3) = %d, want 3", MinInt(5, 3))
	}

	if MaxInt(5, 3) != 5 {
		t.Errorf("MaxInt(5, 3) = %d, want 5", MaxInt(5, 3))
	}

	if ClampInt(10, 1, 5) != 5 {
		t.Errorf("ClampInt(10, 1, 5) = %d, want 5", ClampInt(10, 1, 5))
	}

	if ClampInt(-1, 1, 5) != 1 {
		t.Errorf("ClampInt(-1, 1, 5) = %d, want 1", ClampInt(-1, 1, 5))
	}

	if ClampInt(3, 1, 5) != 3 {
		t.Errorf("ClampInt(3, 1, 5) = %d, want 3", ClampInt(3, 1, 5))
	}
}
