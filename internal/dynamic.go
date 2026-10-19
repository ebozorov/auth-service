package internal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Dynamic holds the settings that change without a restart (docs/DECISION.md #25).
// They come from a YAML file only, never from env.
type Dynamic struct {
	SignInRequestTTL time.Duration `yaml:"sign_in_request_ttl"`
	SignInUserLimit  int           `yaml:"sign_in_user_limit"`
	SignInUserWindow time.Duration `yaml:"sign_in_user_window"`
	SignInIPLimit    int           `yaml:"sign_in_ip_limit"`
	SignInIPWindow   time.Duration `yaml:"sign_in_ip_window"`
	AccessTokenTTL   time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `yaml:"refresh_token_ttl"`
}

// DefaultDynamic is used for keys missing from the file, or when there is no file.
func DefaultDynamic() Dynamic {
	return Dynamic{
		SignInRequestTTL: 10 * time.Minute,
		SignInUserLimit:  5,
		SignInUserWindow: time.Hour,
		SignInIPLimit:    20,
		SignInIPWindow:   time.Hour,
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  720 * time.Hour,
	}
}

// reloadDelay joins the burst of events one save makes into one reload.
const reloadDelay = 200 * time.Millisecond

// DynamicConfig keeps the current settings. Code calls Get every time it needs
// a value, so a reload takes effect right away.
type DynamicConfig struct {
	path    string
	current atomic.Pointer[Dynamic]
}

// LoadDynamic reads the file once. A missing file gives the defaults;
// a broken one is an error, so the server does not start with it.
func LoadDynamic(path string) (*DynamicConfig, error) {
	d, err := readDynamic(path)
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("dynamic config: %s yo'q, standart qiymatlar olinadi", path)
		def := DefaultDynamic()
		d, err = &def, nil
	}
	if err != nil {
		return nil, err
	}
	c := &DynamicConfig{path: path}
	c.current.Store(d)
	return c, nil
}

// Get returns the current settings.
func (c *DynamicConfig) Get() Dynamic {
	return *c.current.Load()
}

// Watch reloads the file whenever it changes, until the process ends. It watches
// the directory, not the file: a Kubernetes ConfigMap is updated by swapping a
// symlink, and editors save by renaming, so the file itself is replaced.
func (c *DynamicConfig) Watch() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(filepath.Dir(c.path)); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer w.Close()
		var timer *time.Timer
		for {
			select {
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				if timer == nil {
					timer = time.AfterFunc(reloadDelay, c.reload)
				} else {
					timer.Reset(reloadDelay)
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("dynamic config: watch: %v", err)
			}
		}
	}()
	return nil
}

// reload swaps in the new settings. A missing or broken file keeps the old ones.
func (c *DynamicConfig) reload() {
	d, err := readDynamic(c.path)
	if err != nil {
		log.Printf("dynamic config: %v; eski qiymatlar qoldi", err)
		return
	}
	old := c.current.Swap(d)
	for _, change := range diffDynamic(*old, *d) {
		log.Printf("dynamic config: %s", change)
	}
}

func readDynamic(path string) (*Dynamic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d := DefaultDynamic()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // a typo in a key name is an error, not a silent default
	if err := dec.Decode(&d); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &d, nil
}

func (d Dynamic) validate() error {
	var errs []error
	v := reflect.ValueOf(d)
	for i := range v.NumField() {
		key := v.Type().Field(i).Tag.Get("yaml")
		switch f := v.Field(i).Interface().(type) {
		case time.Duration:
			if f <= 0 {
				errs = append(errs, fmt.Errorf("%s: must be a positive duration", key))
			}
		case int:
			if f <= 0 {
				errs = append(errs, fmt.Errorf("%s: must be a positive number", key))
			}
		}
	}
	return errors.Join(errs...)
}

// diffDynamic lists the changed keys as "key: old → new".
func diffDynamic(old, cur Dynamic) []string {
	var changes []string
	o, n := reflect.ValueOf(old), reflect.ValueOf(cur)
	for i := range o.NumField() {
		if a, b := o.Field(i).Interface(), n.Field(i).Interface(); a != b {
			changes = append(changes, fmt.Sprintf("%s: %v → %v", o.Type().Field(i).Tag.Get("yaml"), a, b))
		}
	}
	return changes
}
