package rag

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UserProfile represents a persistent user profile
type UserProfile struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	PreferredName  string                 `json:"preferred_name"`
	Preferences    map[string]interface{} `json:"preferences"`
	WorkContext    string                 `json:"work_context"`
	PersonalInfo   map[string]string      `json:"personal_info"`
	ConversationIDs []string              `json:"conversation_ids"`
	CreatedAt      time.Time              `json:"created_at"`
	LastSeenAt     time.Time              `json:"last_seen_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// UserProfileManager manages persistent user profiles
type UserProfileManager struct {
	profiles    map[string]*UserProfile
	mu          sync.RWMutex
	persistPath string
	autoSave    bool
}

// NewUserProfileManager creates a new user profile manager
func NewUserProfileManager(persistPath string) *UserProfileManager {
	manager := &UserProfileManager{
		profiles:    make(map[string]*UserProfile),
		persistPath: persistPath,
		autoSave:    true,
	}

	// Try to load existing profiles
	if err := manager.LoadFromDisk(); err != nil {
		log.Printf("Warning: Could not load existing user profiles from %s: %v", persistPath, err)
	} else {
		log.Printf("Loaded %d user profiles from %s", len(manager.profiles), persistPath)
	}

	return manager
}

// GetOrCreateUserProfile gets an existing user profile or creates a new one
func (upm *UserProfileManager) GetOrCreateUserProfile(userID string) *UserProfile {
	upm.mu.Lock()
	defer upm.mu.Unlock()

	// Check if profile already exists
	if profile, exists := upm.profiles[userID]; exists {
		profile.LastSeenAt = time.Now()
		if upm.autoSave {
			go upm.SaveToDisk()
		}
		return profile
	}

	// Create new profile
	profile := &UserProfile{
		ID:             userID,
		Name:           "",
		PreferredName:  "",
		Preferences:    make(map[string]interface{}),
		WorkContext:    "",
		PersonalInfo:   make(map[string]string),
		ConversationIDs: make([]string, 0),
		CreatedAt:      time.Now(),
		LastSeenAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}

	upm.profiles[userID] = profile

	if upm.autoSave {
		go upm.SaveToDisk()
	}

	log.Printf("Created new user profile for user: %s", userID)
	return profile
}

// UpdateUserProfile updates user profile information
func (upm *UserProfileManager) UpdateUserProfile(userID string, updates map[string]interface{}) error {
	upm.mu.Lock()
	defer upm.mu.Unlock()

	profile, exists := upm.profiles[userID]
	if !exists {
		return fmt.Errorf("user profile not found: %s", userID)
	}

	// Update fields based on the updates map
	if name, ok := updates["name"].(string); ok && name != "" {
		profile.Name = name
		profile.UpdatedAt = time.Now()
	}

	if preferredName, ok := updates["preferred_name"].(string); ok && preferredName != "" {
		profile.PreferredName = preferredName
		profile.UpdatedAt = time.Now()
	}

	if workContext, ok := updates["work_context"].(string); ok && workContext != "" {
		profile.WorkContext = workContext
		profile.UpdatedAt = time.Now()
	}

	// Update preferences
	if preferences, ok := updates["preferences"].(map[string]interface{}); ok {
		for key, value := range preferences {
			profile.Preferences[key] = value
		}
		profile.UpdatedAt = time.Now()
	}

	// Update personal info
	if personalInfo, ok := updates["personal_info"].(map[string]string); ok {
		for key, value := range personalInfo {
			profile.PersonalInfo[key] = value
		}
		profile.UpdatedAt = time.Now()
	}

	profile.LastSeenAt = time.Now()

	if upm.autoSave {
		go upm.SaveToDisk()
	}

	log.Printf("Updated user profile for user: %s", userID)
	return nil
}

// AddConversationToProfile adds a conversation ID to user's profile
func (upm *UserProfileManager) AddConversationToProfile(userID, conversationID string) error {
	upm.mu.Lock()
	defer upm.mu.Unlock()

	profile, exists := upm.profiles[userID]
	if !exists {
		return fmt.Errorf("user profile not found: %s", userID)
	}

	// Check if conversation ID already exists
	for _, existingID := range profile.ConversationIDs {
		if existingID == conversationID {
			return nil // Already exists
		}
	}

	// Add new conversation ID
	profile.ConversationIDs = append(profile.ConversationIDs, conversationID)
	profile.UpdatedAt = time.Now()
	profile.LastSeenAt = time.Now()

	// Keep only last 50 conversations to prevent unlimited growth
	if len(profile.ConversationIDs) > 50 {
		profile.ConversationIDs = profile.ConversationIDs[len(profile.ConversationIDs)-50:]
	}

	if upm.autoSave {
		go upm.SaveToDisk()
	}

	return nil
}

// GetUserConversations returns all conversation IDs for a user
func (upm *UserProfileManager) GetUserConversations(userID string) []string {
	upm.mu.RLock()
	defer upm.mu.RUnlock()

	if profile, exists := upm.profiles[userID]; exists {
		// Return a copy to avoid race conditions
		conversations := make([]string, len(profile.ConversationIDs))
		copy(conversations, profile.ConversationIDs)
		return conversations
	}

	return []string{}
}

// ExtractUserInfoFromText extracts user information from conversation text
func (upm *UserProfileManager) ExtractUserInfoFromText(userID, text string) {
	profile := upm.GetOrCreateUserProfile(userID)
	
	// Simple heuristics to extract user information
	updates := make(map[string]interface{})
	personalInfo := make(map[string]string)
	
	// Look for name patterns
	namePatterns := []string{"my name is", "i'm ", "i am ", "call me ", "name's "}
	for _, pattern := range namePatterns {
		if idx := findCaseInsensitive(text, pattern); idx != -1 {
			nameStart := idx + len(pattern)
			if nameStart < len(text) {
				words := extractWords(text[nameStart:])
				if len(words) > 0 {
					name := cleanName(words[0])
					if len(name) > 1 {
						if profile.Name == "" {
							updates["name"] = name
						}
						if profile.PreferredName == "" {
							updates["preferred_name"] = name
						}
						personalInfo["extracted_name"] = name
					}
				}
			}
		}
	}

	// Look for work context
	workPatterns := []string{"i work", "my job", "i'm a ", "i am a "}
	for _, pattern := range workPatterns {
		if idx := findCaseInsensitive(text, pattern); idx != -1 {
			workStart := idx + len(pattern)
			if workStart < len(text) {
				workText := text[workStart:min(workStart+100, len(text))]
				personalInfo["work_mention"] = workText
			}
		}
	}

	// Update profile if we found information
	if len(updates) > 0 {
		updates["personal_info"] = personalInfo
		upm.UpdateUserProfile(userID, updates)
	}
}

// SaveToDisk saves user profiles to disk
func (upm *UserProfileManager) SaveToDisk() error {
	upm.mu.RLock()
	defer upm.mu.RUnlock()

	// Create data directory if it doesn't exist
	dir := filepath.Dir(upm.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Convert profiles map to slice for JSON serialization
	profileSlice := make([]*UserProfile, 0, len(upm.profiles))
	for _, profile := range upm.profiles {
		profileSlice = append(profileSlice, profile)
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(profileSlice, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles to JSON: %w", err)
	}

	// Write to file atomically
	tempPath := upm.persistPath + ".tmp"
	if err := ioutil.WriteFile(tempPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, upm.persistPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("User profiles saved to %s (%d profiles)", upm.persistPath, len(upm.profiles))
	return nil
}

// LoadFromDisk loads user profiles from disk
func (upm *UserProfileManager) LoadFromDisk() error {
	upm.mu.Lock()
	defer upm.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(upm.persistPath); os.IsNotExist(err) {
		return fmt.Errorf("profiles file does not exist: %s", upm.persistPath)
	}

	// Read file
	jsonData, err := ioutil.ReadFile(upm.persistPath)
	if err != nil {
		return fmt.Errorf("failed to read profiles file: %w", err)
	}

	// Unmarshal JSON
	var profileSlice []*UserProfile
	if err := json.Unmarshal(jsonData, &profileSlice); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	// Restore profiles
	upm.profiles = make(map[string]*UserProfile)
	for _, profile := range profileSlice {
		upm.profiles[profile.ID] = profile
	}

	return nil
}

// Helper functions
func findCaseInsensitive(text, pattern string) int {
	textLower := strings.ToLower(text)
	patternLower := strings.ToLower(pattern)
	return strings.Index(textLower, patternLower)
}

func extractWords(text string) []string {
	words := strings.Fields(text)
	return words
}

func cleanName(name string) string {
	name = strings.Trim(name, ".,!?;:")
	return name
}

