package cli

import (
	"testing"
)

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SSH URL unchanged",
			input:    "git@github.com:user/repo.git",
			expected: "git@github.com:user/repo.git",
		},
		{
			name:     "HTTPS URL without credentials",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "HTTPS URL with credentials removed",
			input:    "https://username:password@github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "HTTPS URL with username only removed",
			input:    "https://username@github.com/user/repo.git",
			expected: "https://github.com/user/repo.git",
		},
		{
			name:     "git protocol URL unchanged",
			input:    "git://github.com/user/repo.git",
			expected: "git://github.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeGitURL(tt.input)
			if err != nil {
				t.Errorf("normalizeGitURL() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("normalizeGitURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUrlsMatch(t *testing.T) {
	tests := []struct {
		name   string
		url1   string
		url2   string
		expect bool
	}{
		{
			name:   "identical HTTPS URLs",
			url1:   "https://github.com/user/repo.git",
			url2:   "https://github.com/user/repo.git",
			expect: true,
		},
		{
			name:   "HTTPS URLs with and without credentials",
			url1:   "https://token@github.com/user/repo.git",
			url2:   "https://github.com/user/repo.git",
			expect: true,
		},
		{
			name:   "HTTPS URLs with different credentials",
			url1:   "https://token1@github.com/user/repo.git",
			url2:   "https://token2@github.com/user/repo.git",
			expect: true,
		},
		{
			name:   "identical SSH URLs",
			url1:   "git@github.com:user/repo.git",
			url2:   "git@github.com:user/repo.git",
			expect: true,
		},
		{
			name:   "different repositories",
			url1:   "https://github.com/user/repo1.git",
			url2:   "https://github.com/user/repo2.git",
			expect: false,
		},
		{
			name:   "SSH vs HTTPS same repo",
			url1:   "git@github.com:user/repo.git",
			url2:   "https://github.com/user/repo.git",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlsMatch(tt.url1, tt.url2)
			if result != tt.expect {
				t.Errorf("urlsMatch(%v, %v) = %v, want %v", tt.url1, tt.url2, result, tt.expect)
			}
		})
	}
}
