package authclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/martinlindhe/base36"
	"golang.org/x/crypto/blake2s"
	"gopkg.in/square/go-jose.v2"
)

// predefined cache errors
var (
	ErrExpired  = errors.New("expired")
	ErrInvalid  = errors.New("invalid")
	ErrNotFound = errors.New("not found")
)

// A Cache loads and stores JWTs.
type Cache interface {
	Load(serverURL *url.URL) (rawJWT string, err error)
	Store(serverURL *url.URL, rawJWT string) error
}

// A LocalCache stores files in the config directory.
type LocalCache struct {
}

// Load loads a raw JWT from the local cache.
func (cache *LocalCache) Load(serverURL *url.URL) (rawJWT string, err error) {
	dir, err := cache.cacheDirectory()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, cache.fileName(serverURL))
	rawBS, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	rawJWT = string(rawBS)

	tok, err := jose.ParseSigned(rawJWT)
	if err != nil {
		return "", ErrInvalid
	}

	var claims struct {
		Expiry int64 `json:"exp"`
	}
	err = json.Unmarshal(tok.UnsafePayloadWithoutVerification(), &claims)
	if err != nil {
		return "", ErrInvalid
	}

	expiresAt := time.Unix(claims.Expiry, 0)
	if expiresAt.Before(time.Now()) {
		return "", ErrExpired
	}

	return rawJWT, nil
}

// Store stores a raw JWT in the local cache.
func (cache *LocalCache) Store(serverURL *url.URL, rawJWT string) error {
	dir, err := cache.cacheDirectory()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, cache.fileName(serverURL))
	err = ioutil.WriteFile(path, []byte(rawJWT), 0600)
	if err != nil {
		return err
	}

	return nil
}

func (cache *LocalCache) configDirectory() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("error getting user config directory: %w", err)
	}

	dir := filepath.Join(cfgDir, "pomerium-cli")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", fmt.Errorf("error creating user config directory: %w", err)
	}

	return dir, nil
}

func (cache *LocalCache) cacheDirectory() (string, error) {
	root, err := cache.configDirectory()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(root, "cache", "jwts")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return "", fmt.Errorf("error creating user cache directory: %w", err)
	}

	return "", nil
}

func (cache *LocalCache) hash(str string) string {
	h := blake2s.Sum256([]byte(str))
	return base36.EncodeBytes(h[:])
}

func (cache *LocalCache) fileName(serverURL *url.URL) string {
	return cache.hash(serverURL.String()) + ".jwt"
}
