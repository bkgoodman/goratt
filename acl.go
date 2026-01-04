package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ACLEntry represents a single ACL entry from the API.
type ACLEntry struct {
	Tagid      string `json:"tagid"`
	TagIdent   string `json:"tag_ident"`
	Allowed    string `json:"allowed"`
	Warning    string `json:"warning"`
	Member     string `json:"member"`
	Nickname   string `json:"nickname"`
	Plan       string `json:"plan"`
	LastAccess string `json:"last_accessed"`
	Level      int    `json:"level"`
	RawTagID   string `json:"raw_tag_id"`
}

// ACLRecord is the in-memory representation of an ACL entry.
type ACLRecord struct {
	Tag      uint64
	Level    int
	Member   string
	Nickname string
	Warning  string
	Allowed  bool
}

// ACLManager handles ACL list management.
type ACLManager struct {
	mu       sync.RWMutex
	tags     []ACLRecord
	cfg      *Config
	tagFile  string
	onUpdate func() // callback when ACL is updated
}

// NewACLManager creates a new ACL manager.
func NewACLManager(cfg *Config) *ACLManager {
	return &ACLManager{
		cfg:     cfg,
		tagFile: cfg.TagFile,
	}
}

// SetUpdateCallback sets a callback to be called when ACL is updated.
func (a *ACLManager) SetUpdateCallback(fn func()) {
	a.onUpdate = fn
}

// Lookup finds a tag in the ACL list.
func (a *ACLManager) Lookup(tagID uint64) (ACLRecord, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, tag := range a.tags {
		if tag.Tag == tagID {
			return tag, true
		}
	}
	return ACLRecord{}, false
}

// FetchFromAPI downloads the ACL list from the API.
func (a *ACLManager) FetchFromAPI() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	caCert, err := ioutil.ReadFile(a.cfg.API.CAFile)
	if err != nil {
		return fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	httpClient := &http.Client{Transport: transport}

	url := fmt.Sprintf("%s/api/v1/resources/%s/acl", a.cfg.API.URL, a.cfg.Resource)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(a.cfg.API.Username + ":" + a.cfg.API.Password))
	req.Header.Add("Authorization", "Basic "+auth)

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("make request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var items []ACLEntry
	if err := json.Unmarshal(body, &items); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	// Write to temp file
	file, err := os.Create(a.tagFile + ".tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	a.tags = a.tags[:0]
	for _, item := range items {
		number, err := strconv.ParseUint(item.RawTagID, 10, 64)
		if err != nil {
			continue
		}

		allowed := item.Allowed == "allowed"
		a.tags = append(a.tags, ACLRecord{
			Tag:      number,
			Level:    item.Level,
			Member:   item.Member,
			Nickname: item.Nickname,
			Warning:  item.Warning,
			Allowed:  allowed,
		})

		access := "denied"
		if allowed {
			access = "allowed"
		}
		// Format: tag access level member nickname warning (tab-separated for nickname/warning which may have spaces)
		fmt.Fprintf(file, "%d\t%s\t%d\t%s\t%s\t%s\n", number, access, item.Level, item.Member, item.Nickname, item.Warning)
	}
	file.Close()

	if err := os.Rename(a.tagFile+".tmp", a.tagFile); err != nil {
		return fmt.Errorf("rename tag file: %w", err)
	}

	if a.onUpdate != nil {
		a.onUpdate()
	}

	return nil
}

// LoadFromFile loads the ACL list from the tag file.
// Creates the file if it doesn't exist.
func (a *ACLManager) LoadFromFile() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure parent directory exists
	dir := filepath.Dir(a.tagFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create tag file directory: %w", err)
	}

	// Try to open the file, create if it doesn't exist
	file, err := os.OpenFile(a.tagFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open tag file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	a.tags = a.tags[:0]
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")

		// Support both old format (space-separated, 4 fields) and new format (tab-separated, 6 fields)
		var tag uint64
		var level int
		var member, access, nickname, warning string

		if len(parts) >= 6 {
			// New tab-separated format
			tag, _ = strconv.ParseUint(parts[0], 10, 64)
			access = parts[1]
			level, _ = strconv.Atoi(parts[2])
			member = parts[3]
			nickname = parts[4]
			warning = parts[5]
		} else {
			// Old space-separated format (backward compatibility)
			_, err := fmt.Sscanf(line, "%d %s %d %s", &tag, &access, &level, &member)
			if err != nil {
				continue
			}
		}

		a.tags = append(a.tags, ACLRecord{
			Tag:      tag,
			Level:    level,
			Member:   member,
			Nickname: nickname,
			Warning:  warning,
			Allowed:  access == "allowed",
		})
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning reading tag file: %v", err)
	}

	return nil
}
