package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/satori/go.uuid"

	"github.com/Sirupsen/logrus"
)

type Options struct {
	PasswordFile   string
	RepositoryPath string
	SourcePath     string
	Log            *logrus.Logger
}

type Backup struct {
	passwordFile   string
	repositoryPath string
	sourcePath     string
	resticPath     string
	log            *logrus.Logger
	created        bool
}

func New(opts *Options) (*Backup, error) {
	o := opts
	if o == nil {
		o = &Options{}
	}
	if o.SourcePath == "" {
		return nil, fmt.Errorf("a SourcePath has to be specified")
	}
	if o.PasswordFile == "" {
		o.PasswordFile = filepath.Join(o.SourcePath, "backups.passwd")
	}
	if o.RepositoryPath == "" {
		o.RepositoryPath = fmt.Sprintf("%s_backups", o.SourcePath)
	}
	b := Backup{
		repositoryPath: o.RepositoryPath,
		sourcePath:     o.SourcePath,
		passwordFile:   o.PasswordFile,
		log:            o.Log,
	}

	if b.log == nil {
		b.log = logrus.New()
		b.log.SetLevel(logrus.ErrorLevel)
	}
	return &b, nil
}

func (b *Backup) Available() bool {
	if b.resticPath != "" {
		return true
	}
	path, err := exec.LookPath("restic")
	if err != nil {
		return false
	}
	if path == "" {
		return false
	}
	b.resticPath = path
	return true
}

func (b *Backup) Init() error {
	if err := b.ensurePasswordFile(); err != nil {
		return err
	}
	if err := b.ensureRepository(); err != nil {
		return err
	}
	return nil
}

func (b *Backup) Created() bool {
	return b.created
}

func (b *Backup) createEnv() []string {
	return []string{
		fmt.Sprintf("RESTIC_REPOSITORY=%s", b.repositoryPath),
		fmt.Sprintf("RESTIC_PASSWORD_FILE=%s", b.passwordFile),
	}
}

func (b *Backup) CreateSnapshot() error {
	cmd := exec.Command(b.resticPath, "backup", b.sourcePath)
	cmd.Env = b.createEnv()
	return cmd.Run()
}

func (b *Backup) Restore(id string) error {
	cmd := exec.Command(b.resticPath, "restore", "--target", filepath.Dir(b.sourcePath), id)
	cmd.Env = b.createEnv()
	return cmd.Run()
}

type ByTime []Snapshot

func (s ByTime) Len() int {
	return len(s)
}

func (s ByTime) Less(i int, j int) bool {
	return s[i].RawTime < s[j].RawTime
}

func (s ByTime) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

func (b *Backup) Snapshots() ([]Snapshot, error) {
	cmd := exec.Command(b.resticPath, "snapshots", "--json")
	cmd.Env = b.createEnv()
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var result []Snapshot
	if err := json.NewDecoder(&buffer).Decode(&result); err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(ByTime(result)))
	return result, nil
}

func (b *Backup) ensurePasswordFile() error {
	stats, err := os.Stat(b.passwordFile)
	if err != nil {
		if os.IsNotExist(err) {
			id := uuid.NewV4()
			return ioutil.WriteFile(b.passwordFile, []byte(id.String()), 0600)
		}
		return err
	}
	if stats.IsDir() {
		return fmt.Errorf("%s is a directory and not a file", b.passwordFile)
	}
	return nil
}

func (b *Backup) ensureRepository() error {
	configFile := filepath.Join(b.repositoryPath, "config")
	stats, err := os.Stat(b.repositoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return b.createRepository()
		}
		return err
	}
	if !stats.IsDir() {
		return fmt.Errorf("%s is not a directory", b.repositoryPath)
	}
	if _, err := os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			return b.createRepository()
		}
		return err
	}
	return nil
}

func (b *Backup) createRepository() error {
	cmd := exec.Command(b.resticPath, "init")
	cmd.Env = b.createEnv()
	if err := cmd.Run(); err != nil {
		return err
	}
	b.created = true
	return nil
}
