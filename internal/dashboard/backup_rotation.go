package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var managedBackupName = regexp.MustCompile(`^dashboard-[0-9]{8}T[0-9]{6}Z-[0-9]+\.db$`)

type backupRotation struct {
	Valid    int      `json:"valid"`
	Invalid  int      `json:"invalid"`
	Retained int      `json:"retained"`
	Removed  []string `json:"removed"`
	Applied  bool     `json:"applied"`
}

// Rotation counts only integrity-checked backups. Corruption and failed exports
// cannot evict the last good copies. Unmanaged files and symlinks are untouched.
func pruneBackups(directory string, keep int, apply bool) (backupRotation, error) {
	result := backupRotation{Removed: []string{}, Applied: apply}
	if keep < 2 || keep > 365 {
		return result, errors.New("--keep must be between 2 and 365")
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return result, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return result, errors.New("backup directory must be a private directory (0700)")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return result, err
	}
	if len(entries) > 1000 {
		return result, errors.New("backup directory exceeds 1000 files; inspect manually")
	}
	var valid []string
	for _, entry := range entries {
		if !managedBackupName.MatchString(entry.Name()) || !entry.Type().IsRegular() {
			continue
		}
		name := filepath.Join(directory, entry.Name())
		if err := verifyDatabase(name); err != nil {
			result.Invalid++
			continue
		}
		valid = append(valid, entry.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(valid)))
	result.Valid = len(valid)
	result.Retained = len(valid)
	for _, name := range valid[min(keep, len(valid)):] {
		if apply {
			if err := os.Remove(filepath.Join(directory, name)); err != nil {
				return result, err
			}
		}
		result.Removed = append(result.Removed, name)
		result.Retained--
	}
	if apply && len(result.Removed) > 0 {
		directoryFile, err := os.Open(directory)
		if err != nil {
			return result, err
		}
		defer directoryFile.Close()
		if err = directoryFile.Sync(); err != nil {
			return result, err
		}
	}
	return result, nil
}
func printRotation(result backupRotation) {
	encoded, _ := json.Marshal(result)
	fmt.Println(string(encoded))
}

// Verify a DB/key pair without migrating or modifying either file.
func verifyBackupKey(database, keyFile string) error {
	vault, err := loadVault(keyFile, false)
	if err != nil {
		return err
	}
	db, err := existingDatabase(database)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query("SELECT id,kind,encrypted_secret FROM connections WHERE owner_id='local' AND length(encrypted_secret)>0")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, kind string
		var encrypted []byte
		if err = rows.Scan(&id, &kind, &encrypted); err != nil {
			return err
		}
		if _, err = vault.decrypt(id, kind, encrypted); err != nil {
			return ErrConnectionLocked
		}
	}
	return rows.Err()
}
