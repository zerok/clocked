# Clocked: A simple time-tracker

Welcome to clocked, a small, simple task-based time tracker. The motivation
for creating this was that I wanted to have something I could use to track
my working time offline and at the end of the day easily sync up to JIRA.
I've tried taskwarrior and OrgMode in the past but none of them offered me the
UX and flexibility I wanted. So here we are ;-)

## How to get started

Getting started with clocked is quite simple. Just install it and start adding
tasks to it. Every screen will show you a list of all the available keyboard
commands. If you don't want to sync your tasks with JIRA, then there is even
nothing for you to configure :-)

## Synchronizing your work-time with JIRA worklogs

If you do want to sync with JIRA, you will have to create a
`$HOME/.clocked/config.yml` file and put your JIRA's URL and username into it:

```
jiraURL: https://jira.company.com
jiraUsername: jdoe
```

When you start clocked for the next time then it will ask you for your JIRA
password and store it into a macOS keychain.

Once that is all done, make sure to create tasks that have the same code as
the tasks you have in JIRA. Then hit `^s` to enter the summary view to see
all the tasks you've worked on today. From there hit `^j` to enter the 
sync-view and `s` to actually start the synchronization.

This will delete all your worklogs of the selected date and create new ones
for your tasks as tracked by clocked.


## Backups using restic

If you have [restic][] installed, clocked will create a snapshot after every
change to a task. If you create a new task, a snapshot will be made. If you
clock in or out, a new snapshot will be made. By default the backup repository
is stored in `$HOME/.clocked_backups` and its password is saved in
`$HOME/.clocked/backups.passwd`

## Command-line arguments

- `--log-file <path/to/file>` specifies a path to a logfile clocked should
  write to.
- `--store <path/to/folder>` specifies where clocked should store its files.
  Default: `$HOME/.clocked`

[restic]: https://restic.github.io/
