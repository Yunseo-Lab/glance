package dashboard

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func existingDatabase(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("database must be a regular file")
	}
	u := url.URL{Scheme: "file", Path: abs, RawQuery: "mode=ro&_busy_timeout=5000"}
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
func verifyDatabase(path string) error {
	db, err := existingDatabase(path)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var integrity string
	if err = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return errors.New("SQLite integrity check failed")
	}
	var version int
	if err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version != 1 {
		return fmt.Errorf("unsupported database version %d", version)
	}
	var data []byte
	if err = db.QueryRowContext(ctx, "SELECT document FROM boards WHERE id='home' AND owner_id='local'").Scan(&data); err != nil {
		return err
	}
	var b Board
	if err = json.Unmarshal(data, &b); err != nil {
		return err
	}
	return b.Validate()
}

// An exclusive link prevents overwrites even if the output is created mid-backup.
func publishDatabase(temp, output string) error {
	f, err := os.OpenFile(temp, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Chmod(temp, 0600); err != nil {
		return err
	}
	if err = os.Link(temp, output); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(output))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func backupDatabase(source, output string) error {
	if _, err := os.Lstat(output); err == nil {
		return errors.New("output already exists; refusing to overwrite")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := verifyDatabase(source); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(output), ".glance-backup-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Close(); err != nil {
		return err
	}
	db, err := existingDatabase(source)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", name); err != nil {
		return err
	}
	if err = verifyDatabase(name); err != nil {
		return err
	}
	return publishDatabase(name, output)
}
func restoreDatabase(source, output string) error {
	if _, err := os.Lstat(output); err == nil {
		return errors.New("restore target already exists; choose a new database path")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := verifyDatabase(source); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	temp, err := os.CreateTemp(filepath.Dir(output), ".glance-restore-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	_, err = io.Copy(temp, in)
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = verifyDatabase(name); err != nil {
		return err
	}
	return publishDatabase(name, output)
}
func maintenance(command string, args []string) int {
	flags := flag.NewFlagSet("glance dashboard "+command, flag.ContinueOnError)
	defaultDB := os.Getenv("GLANCE_DATABASE_PATH")
	if defaultDB == "" {
		defaultDB = "data/dashboard.db"
	}
	db := flags.String("db", defaultDB, "Source database (restore: new destination)")
	output := flags.String("output", "", "New backup file")
	input := flags.String("input", "", "Backup file to restore")
	address := flags.String("url", "http://127.0.0.1:8080/api/readyz", "Health check URL")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		return 1
	}
	var err error
	switch command {
	case "backup":
		if *output == "" {
			err = errors.New("--output is required")
		} else {
			err = backupDatabase(*db, *output)
		}
	case "restore":
		if *input == "" {
			err = errors.New("--input is required")
		} else {
			err = restoreDatabase(*input, *db)
		}
	case "verify-backup":
		if *input == "" {
			err = errors.New("--input is required")
		} else {
			err = verifyDatabase(*input)
		}
	case "healthcheck":
		var token string
		token, err = loadOwnerToken()
		if err != nil {
			break
		}
		var req *http.Request
		req, err = http.NewRequest(http.MethodGet, *address, nil)
		if err != nil {
			break
		}
		host := req.URL.Hostname()
		ip := net.ParseIP(host)
		if req.URL.User != nil || req.URL.Scheme != "http" && req.URL.Scheme != "https" || host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			err = errors.New("health checks require a loopback HTTP(S) address")
			break
		}
		if token != "" {
			req.SetBasicAuth("healthcheck", token)
		}
		var response *http.Response
		response, err = (&http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
		if err == nil {
			response.Body.Close()
			if response.StatusCode != 200 {
				err = fmt.Errorf("service not ready: HTTP %d", response.StatusCode)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if command == "backup" {
		file, e := os.Open(*output)
		if e != nil {
			return 1
		}
		defer file.Close()
		sum := sha256.New()
		if _, err = io.Copy(sum, file); err != nil {
			return 1
		}
		fmt.Printf("backup created; SHA-256 %x\n", sum.Sum(nil))
	} else {
		fmt.Println(command + ": ok")
	}
	return 0
}
