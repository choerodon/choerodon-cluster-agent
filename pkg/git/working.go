// Copyright 2016 Weaveworks Ltd.
// Use of this source code is governed by a Apache License Version 2.0 license
// that can be found at https://github.com/weaveworks/flux/blob/master/LICENSE

package git

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Config holds some values we use when working in the working clone of
// a repo.
type Config struct {
	Branch          string // branch we're syncing to
	Path            string // path within the repo containing files we care about
	SyncTag         string
	NotesRef        string
	UserName        string
	UserEmail       string
	SetAuthor       bool
	SkipMessage     string
	DevOpsTag       string
	GitUrl          string
	GitPollInterval time.Duration
}

// Checkout is a local working clone of the remote repo. It is
// intended to be used for one-off "transactions", e.g,. committing
// changes then pushing upstream. It has no locking.
type Checkout struct {
	dir          string
	config       Config
	upstream     Remote
	realNotesRef string // cache the notes ref, since we use it to push as well
}

type Commit struct {
	Revision string
	Message  string
}

// CommitAction - struct holding commit information
type CommitAction struct {
	Author  string
	Message string
}

func (r *Repo) Clone(ctx context.Context, conf Config) (*Checkout, error) {
	upstream := r.Origin()
	repoDir, err := r.workingClone(ctx, conf.DevOpsTag)
	if err != nil {
		return nil, err
	}

	if err := config(ctx, repoDir, conf.UserName, conf.UserEmail); err != nil {
		os.RemoveAll(repoDir)
		return nil, err
	}

	// We'll need the notes ref for pushing it, so make sure we have
	// it. This assumes we're syncing it (otherwise we'll likely get conflicts)
	realNotesRef, err := getNotesRef(ctx, repoDir, conf.NotesRef)
	if err != nil {
		os.RemoveAll(repoDir)
		return nil, err
	}

	return &Checkout{
		dir:          repoDir,
		upstream:     upstream,
		realNotesRef: realNotesRef,
		config:       conf,
	}, nil
}

// Clean a Checkout up (remove the clone)
func (c *Checkout) Clean() {
	if c.dir != "" {
		os.RemoveAll(c.dir)
	}
}

// Dir returns the path to the repo
func (c *Checkout) Dir() string {
	return c.dir
}

// ManifestDir returns the path to the manifests files
func (c *Checkout) ManifestDir() string {
	return filepath.Join(c.dir, c.config.Path)
}

// CommitAndPush commits changes made in this checkout, along with any
// extra data as a note, and pushes the commit and note to the remote repo.
func (c *Checkout) CommitAndPush(ctx context.Context, commitAction CommitAction, note interface{}) error {
	if !check(ctx, c.dir, c.config.Path) {
		return ErrNoChanges
	}

	commitAction.Message += c.config.SkipMessage

	if err := commit(ctx, c.dir, commitAction); err != nil {
		return err
	}

	if note != nil {
		rev, err := refRevision(ctx, c.dir, "HEAD")
		if err != nil {
			return err
		}
		if err := addNote(ctx, c.dir, rev, c.config.NotesRef, note); err != nil {
			return err
		}
	}

	refs := []string{c.config.Branch}
	ok, err := refExists(ctx, c.dir, c.realNotesRef)
	if ok {
		refs = append(refs, c.realNotesRef)
	} else if err != nil {
		return err
	}

	if err := push(ctx, c.dir, c.upstream.URL, refs); err != nil {
		return PushError(c.upstream.URL, err)
	}
	return nil
}

// GetNote gets a note for the revision specified, or nil if there is no such note.
func (c *Checkout) GetNote(ctx context.Context, rev string, note interface{}) (bool, error) {
	return getNote(ctx, c.dir, c.realNotesRef, rev, note)
}

func (c *Checkout) HeadRevision(ctx context.Context) (string, error) {
	return refRevision(ctx, c.dir, "HEAD")
}

func (c *Checkout) SyncRevision(ctx context.Context) (string, error) {
	return refRevision(ctx, c.dir, c.config.SyncTag)
}

func (c *Checkout) DevOpsSyncRevision(ctx context.Context) (string, error) {
	return refRevision(ctx, c.dir, c.config.DevOpsTag)
}

func (c *Checkout) MoveSyncTagAndPush(ctx context.Context, ref, msg string) error {
	return moveTagAndPush(ctx, c.dir, c.config.SyncTag, ref, msg, c.upstream.URL)
}

// ChangedFiles does a git diff listing changed files
func (c *Checkout) ChangedFiles(ctx context.Context, ref string) ([]string, []string, error) {
	list, err := changedFiles(ctx, c.dir, c.config.Path, ref)
	absolutePath := make([]string, len(list))
	if err == nil {
		for i, file := range list {
			absolutePath[i] = filepath.Join(c.dir, file)
		}
	}
	return absolutePath, list, err
}

func (c *Checkout) FileLastCommit(ctx context.Context, file string) (string, error) {
	return fileLastCommit(ctx, c.dir, c.config.Path, file)
}

func (c *Checkout) NoteRevList(ctx context.Context) (map[string]struct{}, error) {
	return noteRevList(ctx, c.dir, c.realNotesRef)
}
