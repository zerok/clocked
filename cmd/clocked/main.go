package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	termbox "github.com/nsf/termbox-go"
	"github.com/spf13/pflag"
	"github.com/zerok/clocked/internal/backup"
	"github.com/zerok/clocked/internal/config"
	"github.com/zerok/clocked/internal/database"
	"github.com/zerok/clocked/internal/jira"
)

func main() {
	var verbose bool
	var storageFolder string
	var logFile string
	pflag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	pflag.StringVar(&logFile, "log-file", "", "Path to a logfile")
	pflag.StringVar(&storageFolder, "store", filepath.Join(os.Getenv("HOME"), ".clocked"), "Path where clocked will store its data")
	pflag.Parse()

	log := logrus.New()
	if logFile != "" {
		fp, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			log.WithError(err).Fatal("Failed to open logfile")
		}
		defer fp.Close()
		log.Out = fp
	}

	if verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	if err := ensureStorageFolder(storageFolder); err != nil {
		log.WithError(err).Fatalf("Failed to create storage folder %s", storageFolder)
	}

	cfg, err := config.Load(filepath.Join(storageFolder, "config.yml"))
	if err != nil {
		log.WithError(err).Fatalf("Failed to load configuration file")
	}

	db, err := database.NewDatabase(storageFolder, log)
	if err != nil {
		log.WithError(err).Fatalf("Failed to load database from %s", storageFolder)
	}
	if err := db.LoadState(); err != nil {
		log.WithError(err).Fatalf("Failed to load database")
	}

	bk, err := backup.New(&backup.Options{
		SourcePath: storageFolder,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to configure backup")
	}
	if !bk.Available() {
		log.Info("Backing up not possible. Most likely restic is not installed.")
	} else {
		if err := bk.Init(); err != nil {
			log.WithError(err).Fatalf("Failed to initialize backup")
		}
		if bk.Created() && !db.Empty() {
			if err := bk.CreateSnapshot(); err != nil {
				log.WithError(err).Fatalf("Failed to create initial snapshot")
			}
		}
	}

	app := newApplication()
	app.backup = bk
	app.db = db
	app.log = log
	if cfg.JIRAURL != "" && cfg.JIRAPassword != "" && cfg.JIRAUsername != "" {
		app.jiraClient = jira.NewClient(cfg.JIRAURL, cfg.JIRAUsername, cfg.JIRAPassword)
	}

	if logFile == "" {
		log.SetLevel(logrus.FatalLevel)
		log.Out = ioutil.Discard
	}

	err = termbox.Init()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize application")
	}
	defer termbox.Close()
	app.handleResize()
	app.start()
}

func ensureStorageFolder(storageFolder string) error {
	return os.MkdirAll(storageFolder, 0700)
}
