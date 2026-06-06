package cacheutil

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"time"
)

type DataVersion struct {
	Value        string
	LastModified time.Time
}

func ReadFileWithVersion(path string) ([]byte, DataVersion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, DataVersion{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, DataVersion{}, err
	}

	return data, VersionFromBytes(data, info.ModTime()), nil
}

func VersionFromBytes(data []byte, lastModified time.Time) DataVersion {
	sum := sha256.Sum256(data)
	return DataVersion{
		Value:        hex.EncodeToString(sum[:16]),
		LastModified: lastModified.UTC().Truncate(time.Second),
	}
}

func (version DataVersion) ETag(scope string) string {
	if version.Value == "" {
		return ""
	}

	scopeSum := sha256.Sum256([]byte(scope))
	return `"` + version.Value + "-" + hex.EncodeToString(scopeSum[:8]) + `"`
}

func HashKey(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte{0})
		hash.Write([]byte(strings.TrimSpace(part)))
	}

	sum := hash.Sum(nil)
	return hex.EncodeToString(sum[:16])
}
