package main

import (
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	termbox "github.com/nsf/termbox-go"
	"github.com/ogier/pflag"
	"github.com/zerok/clocked/internal/database"
	"github.com/zerok/clocked/internal/form"
)

func generateNewTaskForm() *form.Form {
	return form.NewForm([]form.Field{
		{
			Code:       "code",
			Label:      "Code:",
			IsRequired: true,
		}, {
			Code:       "title",
			Label:      "Title:",
			IsRequired: true,
		},
	})
}

func main() {
	log := logrus.New()
	fp, err := os.OpenFile("clocked.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.WithError(err).Fatal("Failed to open logfile")
	}
	defer fp.Close()
	log.Out = fp

	var verbose bool
	var storageFolder string
	pflag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	pflag.StringVar(&storageFolder, "store", filepath.Join(os.Getenv("HOME"), ".clocked"), "Path where clocked will store its data")
	pflag.Parse()

	app := newApplication()
	db, err := database.NewDatabase(storageFolder, log)
	if err != nil {
		log.WithError(err).Fatalf("Failed to load databse from %s", storageFolder)
	}
	if err := db.LoadState(); err != nil {
		log.WithError(err).Fatalf("Failed to load database")
	}
	app.db = db
	app.log = log

	if verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	err = termbox.Init()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize application")
	}
	defer termbox.Close()
	app.handleResize()
	app.mode = selectionMode
	app.start()
}
